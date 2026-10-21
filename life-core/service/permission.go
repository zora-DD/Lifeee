package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/model"
	"life-core/proto"
)

// UserPermissionRepository 定义模块权限持久化能力。
type UserPermissionRepository interface {
	ListByUserID(context.Context, *gorm.DB, uint64) ([]model.UserModulePermission, error)
	UpsertAll(context.Context, *gorm.DB, []model.UserModulePermission) error
}

type PermissionUserRepository interface {
	GetByID(context.Context, *gorm.DB, uint64) (*model.User, error)
}

// PermissionService 负责查询和配置用户模块权限。
type PermissionService struct {
	db          *gorm.DB
	users       PermissionUserRepository
	permissions UserPermissionRepository
}

func NewPermissionService(db *gorm.DB, users PermissionUserRepository, permissions UserPermissionRepository) *PermissionService {
	return &PermissionService{db: db, users: users, permissions: permissions}
}

func (s *PermissionService) GetMine(ctx context.Context, _ *proto.GetMyPermissionsRequest) (*proto.GetMyPermissionsResponse, error) {
	user, ok := authctx.UserFromContext(ctx)
	if !ok {
		return nil, hecode.ErrUnauthorized
	}
	items, err := s.listEffective(ctx, user)
	if err != nil {
		return nil, err
	}
	return &proto.GetMyPermissionsResponse{Permissions: items}, nil
}

func (s *PermissionService) GetUser(ctx context.Context, req *proto.GetUserPermissionsRequest) (*proto.GetUserPermissionsResponse, error) {
	if err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, err := parsePermissionUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	return s.getUserResponse(ctx, userID)
}

func (s *PermissionService) Modify(ctx context.Context, req *proto.ModifyUserPermissionsRequest) (*proto.GetUserPermissionsResponse, error) {
	admin, err := s.adminFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	userID, err := parsePermissionUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	target, err := s.users.GetByID(ctx, s.db, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: user %d", hecode.ErrNotFound, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get permission user: %v", hecode.ErrDBQuery, err)
	}
	if target.Role == model.UserRoleAdmin {
		return nil, fmt.Errorf("%w: administrator permissions cannot be modified", hecode.ErrInvalidParam)
	}
	rows, err := buildPermissionRows(userID, admin.ID, req.GetPermissions())
	if err != nil {
		return nil, err
	}
	if err := s.permissions.UpsertAll(ctx, s.db, rows); err != nil {
		return nil, fmt.Errorf("%w: save user permissions: %v", hecode.ErrDBUpdate, err)
	}
	return &proto.GetUserPermissionsResponse{User: userToProto(target), Permissions: permissionRowsToProto(rows)}, nil
}

func (s *PermissionService) getUserResponse(ctx context.Context, userID uint64) (*proto.GetUserPermissionsResponse, error) {
	user, err := s.users.GetByID(ctx, s.db, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: user %d", hecode.ErrNotFound, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get permission user: %v", hecode.ErrDBQuery, err)
	}
	permissions, err := s.listEffective(ctx, user)
	if err != nil {
		return nil, err
	}
	return &proto.GetUserPermissionsResponse{User: userToProto(user), Permissions: permissions}, nil
}

func (s *PermissionService) listEffective(ctx context.Context, user *model.User) ([]*proto.ModulePermissionItem, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	if user.Role == model.UserRoleAdmin {
		return defaultPermissionProto(model.PermissionLevelWrite), nil
	}
	rows, err := s.permissions.ListByUserID(ctx, s.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: list user permissions: %v", hecode.ErrDBQuery, err)
	}
	return completePermissionProto(rows), nil
}

func (s *PermissionService) checkDependencies() error {
	if s == nil || s.db == nil || s.users == nil || s.permissions == nil {
		return fmt.Errorf("%w: permission service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *PermissionService) checkAdmin(ctx context.Context) error {
	_, err := s.adminFromContext(ctx)
	return err
}

func (s *PermissionService) adminFromContext(ctx context.Context) (*model.User, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
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

func parsePermissionUserID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid user id", hecode.ErrInvalidParam)
	}
	return id, nil
}

func buildPermissionRows(userID, grantedBy uint64, items []*proto.ModulePermissionItem) ([]model.UserModulePermission, error) {
	levels := make(map[string]string, len(items))
	for _, item := range items {
		if item == nil || !isPermissionModule(item.GetModule()) || !isPermissionLevel(item.GetLevel()) {
			return nil, fmt.Errorf("%w: invalid module permission", hecode.ErrInvalidParam)
		}
		if _, exists := levels[item.GetModule()]; exists {
			return nil, fmt.Errorf("%w: duplicate module permission", hecode.ErrInvalidParam)
		}
		levels[item.GetModule()] = item.GetLevel()
	}
	if len(levels) != len(model.PermissionModules()) {
		return nil, fmt.Errorf("%w: all module permissions are required", hecode.ErrInvalidParam)
	}
	rows := make([]model.UserModulePermission, 0, len(levels))
	for _, module := range model.PermissionModules() {
		rows = append(rows, model.UserModulePermission{
			UserID: userID, Module: module, Level: levels[module], GrantedBy: grantedBy,
		})
	}
	return rows, nil
}

func completePermissionProto(rows []model.UserModulePermission) []*proto.ModulePermissionItem {
	levels := make(map[string]string, len(rows))
	for _, row := range rows {
		if isPermissionModule(row.Module) && isPermissionLevel(row.Level) {
			levels[row.Module] = row.Level
		}
	}
	items := make([]*proto.ModulePermissionItem, 0, len(model.PermissionModules()))
	for _, module := range model.PermissionModules() {
		level := levels[module]
		if level == "" {
			level = model.PermissionLevelNone
		}
		items = append(items, &proto.ModulePermissionItem{Module: module, Level: level})
	}
	return items
}

func defaultPermissionProto(level string) []*proto.ModulePermissionItem {
	rows := make([]model.UserModulePermission, 0, len(model.PermissionModules()))
	for _, module := range model.PermissionModules() {
		rows = append(rows, model.UserModulePermission{Module: module, Level: level})
	}
	return permissionRowsToProto(rows)
}

func permissionRowsToProto(rows []model.UserModulePermission) []*proto.ModulePermissionItem {
	items := make([]*proto.ModulePermissionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &proto.ModulePermissionItem{Module: row.Module, Level: row.Level})
	}
	return items
}

func isPermissionModule(module string) bool {
	for _, allowed := range model.PermissionModules() {
		if module == allowed {
			return true
		}
	}
	return false
}

func isPermissionLevel(level string) bool {
	return level == model.PermissionLevelNone || level == model.PermissionLevelRead || level == model.PermissionLevelWrite
}
