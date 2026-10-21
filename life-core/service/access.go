package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

const (
	defaultAccessRequestPageSize = 20
	maxAccessRequestPageSize     = 100
)

// AccessUserRepository 定义访问审批依赖的用户数据操作。
type AccessUserRepository interface {
	GetByID(context.Context, *gorm.DB, uint64) (*model.User, error)
	List(context.Context, *gorm.DB, dao.UserListQuery) ([]model.User, int64, error)
	UpdateStatus(context.Context, *gorm.DB, uint64, string) error
	UpdateApplicationNote(context.Context, *gorm.DB, uint64, string) error
}

// AccessService 负责管理员访问审批。
type AccessService struct {
	db          *gorm.DB
	users       AccessUserRepository
	permissions UserPermissionRepository
}

func NewAccessService(db *gorm.DB, users AccessUserRepository, permissions ...UserPermissionRepository) *AccessService {
	service := &AccessService{db: db, users: users}
	if len(permissions) > 0 {
		service.permissions = permissions[0]
	}
	return service
}

// Status 返回当前登录用户的访问状态。
func (s *AccessService) Status(ctx context.Context, _ *proto.GetAccessStatusRequest) (*proto.UserItem, error) {
	return CurrentUser(ctx)
}

// Apply 保存待审批用户填写的身份备注，便于管理员识别现实身份。
func (s *AccessService) Apply(ctx context.Context, req *proto.ApplyAccessRequest) (*proto.UserItem, error) {
	user, ok := authctx.UserFromContext(ctx)
	if !ok || user == nil {
		return nil, hecode.ErrUnauthorized
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	note := strings.TrimSpace(req.GetNote())
	if note == "" || utf8.RuneCountInString(note) > 200 {
		return nil, fmt.Errorf("%w: application note is required and must not exceed 200 characters", hecode.ErrInvalidParam)
	}
	if user.Role == model.UserRoleAdmin {
		return nil, fmt.Errorf("%w: administrator does not need to apply", hecode.ErrInvalidParam)
	}
	if user.Status == model.UserStatusDisabled {
		return nil, fmt.Errorf("%w: disabled user cannot apply", hecode.ErrPermissionDenied)
	}
	if err := s.users.UpdateApplicationNote(ctx, s.db, user.ID, note); err != nil {
		return nil, fmt.Errorf("%w: update application note: %v", hecode.ErrDBUpdate, err)
	}
	if user.Status == model.UserStatusRejected {
		if err := s.users.UpdateStatus(ctx, s.db, user.ID, model.UserStatusPending); err != nil {
			return nil, fmt.Errorf("%w: reset application status: %v", hecode.ErrDBUpdate, err)
		}
	}
	updated, err := s.users.GetByID(ctx, s.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: get updated user: %v", hecode.ErrDBQuery, err)
	}
	return userToProto(updated), nil
}

// List 返回管理员可审批的用户列表。
func (s *AccessService) List(ctx context.Context, req *proto.ListAccessRequestsRequest) (*proto.ListAccessRequestsResponse, error) {
	if err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	status := strings.TrimSpace(req.GetStatus())
	if status == "" {
		status = model.UserStatusPending
	}
	if !isUserStatus(status) {
		return nil, fmt.Errorf("%w: unsupported user status", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultAccessRequestPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxAccessRequestPageSize {
		return nil, fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	rows, total, err := s.users.List(ctx, s.db, dao.UserListQuery{
		Status: status, Role: model.UserRoleUser, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: list access requests: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.UserItem, 0, len(rows))
	for i := range rows {
		items = append(items, userToProto(&rows[i]))
	}
	return &proto.ListAccessRequestsResponse{
		Items: items, Page: int32(page), PageSize: int32(pageSize), Total: total,
		HasMore: int64(page)*int64(pageSize) < total,
	}, nil
}

// Modify 审批通过或拒绝一个普通用户。
func (s *AccessService) Modify(ctx context.Context, req *proto.ModifyAccessRequestRequest) (*proto.UserItem, error) {
	admin, err := s.adminFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, err := strconv.ParseUint(strings.TrimSpace(req.GetUserId()), 10, 64)
	if err != nil || userID == 0 {
		return nil, fmt.Errorf("%w: invalid user id", hecode.ErrInvalidParam)
	}
	status := strings.TrimSpace(req.GetStatus())
	if status != model.UserStatusApproved && status != model.UserStatusRejected {
		return nil, fmt.Errorf("%w: status must be approved or rejected", hecode.ErrInvalidParam)
	}
	target, err := s.users.GetByID(ctx, s.db, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: user %d", hecode.ErrNotFound, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get access request: %v", hecode.ErrDBQuery, err)
	}
	if target.ID == admin.ID || target.Role == model.UserRoleAdmin {
		return nil, fmt.Errorf("%w: administrator status cannot be reviewed", hecode.ErrInvalidParam)
	}
	if status == model.UserStatusApproved && s.permissions != nil {
		rows, listErr := s.permissions.ListByUserID(ctx, s.db, target.ID)
		if listErr != nil {
			return nil, fmt.Errorf("%w: list initial permissions: %v", hecode.ErrDBQuery, listErr)
		}
		if len(rows) == 0 {
			rows = make([]model.UserModulePermission, 0, len(model.PermissionModules()))
			for _, module := range model.PermissionModules() {
				rows = append(rows, model.UserModulePermission{
					UserID: target.ID, Module: module, Level: model.PermissionLevelWrite, GrantedBy: admin.ID,
				})
			}
			if saveErr := s.permissions.UpsertAll(ctx, s.db, rows); saveErr != nil {
				return nil, fmt.Errorf("%w: create initial permissions: %v", hecode.ErrDBInsert, saveErr)
			}
		}
	}
	if err := s.users.UpdateStatus(ctx, s.db, target.ID, status); err != nil {
		return nil, fmt.Errorf("%w: modify access request: %v", hecode.ErrDBUpdate, err)
	}
	target.Status = status
	return userToProto(target), nil
}

func (s *AccessService) checkAdmin(ctx context.Context) error {
	_, err := s.adminFromContext(ctx)
	return err
}

func (s *AccessService) adminFromContext(ctx context.Context) (*model.User, error) {
	if s == nil || s.db == nil || s.users == nil {
		return nil, fmt.Errorf("%w: access service dependencies are not initialized", hecode.ErrService)
	}
	user, ok := authctx.UserFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	if user.Status != model.UserStatusApproved || user.Role != model.UserRoleAdmin {
		return nil, hecode.ErrPermissionDenied
	}
	return user, nil
}

func isUserStatus(status string) bool {
	switch status {
	case model.UserStatusPending, model.UserStatusApproved, model.UserStatusRejected, model.UserStatusDisabled:
		return true
	default:
		return false
	}
}
