package service

import (
	"context"
	"errors"
	"testing"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/model"
	"life-core/proto"
)

type permissionRepositoryStub struct {
	rows  []model.UserModulePermission
	saved []model.UserModulePermission
}

func (s *permissionRepositoryStub) ListByUserID(context.Context, *gorm.DB, uint64) ([]model.UserModulePermission, error) {
	return append([]model.UserModulePermission(nil), s.rows...), nil
}

func (s *permissionRepositoryStub) UpsertAll(_ context.Context, _ *gorm.DB, rows []model.UserModulePermission) error {
	s.saved = append([]model.UserModulePermission(nil), rows...)
	s.rows = append([]model.UserModulePermission(nil), rows...)
	return nil
}

type permissionUserRepositoryStub struct {
	user *model.User
}

func (s *permissionUserRepositoryStub) GetByID(context.Context, *gorm.DB, uint64) (*model.User, error) {
	if s.user == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return s.user, nil
}

func TestPermissionService(t *testing.T) {
	admin := &model.User{ID: 1, Status: model.UserStatusApproved, Role: model.UserRoleAdmin}
	target := &model.User{ID: 8, Status: model.UserStatusApproved, Role: model.UserRoleUser}
	repository := new(permissionRepositoryStub)
	service := NewPermissionService(new(gorm.DB), &permissionUserRepositoryStub{user: target}, repository)
	ctx := authctx.WithUser(context.Background(), admin)

	permissions := []*proto.ModulePermissionItem{
		{Module: model.PermissionModuleTask, Level: model.PermissionLevelNone},
		{Module: model.PermissionModuleActivity, Level: model.PermissionLevelNone},
		{Module: model.PermissionModuleBill, Level: model.PermissionLevelRead},
		{Module: model.PermissionModuleRestaurant, Level: model.PermissionLevelWrite},
		{Module: model.PermissionModulePlace, Level: model.PermissionLevelRead},
	}
	result, err := service.Modify(ctx, &proto.ModifyUserPermissionsRequest{UserId: "8", Permissions: permissions})
	if err != nil {
		t.Fatalf("Modify() error=%v", err)
	}
	if len(repository.saved) != 5 || repository.saved[3].GrantedBy != 1 || len(result.Permissions) != 5 {
		t.Fatalf("保存权限不完整: saved=%+v result=%+v", repository.saved, result)
	}

	mineCtx := authctx.WithUser(context.Background(), target)
	mine, err := service.GetMine(mineCtx, &proto.GetMyPermissionsRequest{})
	if err != nil || len(mine.Permissions) != 5 || mine.Permissions[2].Level != model.PermissionLevelRead {
		t.Fatalf("GetMine() result=%+v error=%v", mine, err)
	}

	_, err = service.Modify(ctx, &proto.ModifyUserPermissionsRequest{UserId: "8", Permissions: permissions[:2]})
	if !errors.Is(err, hecode.ErrInvalidParam) {
		t.Fatalf("不完整权限应返回参数错误，actual=%v", err)
	}
}

func TestPermissionServiceAdminAlwaysHasFullAccess(t *testing.T) {
	admin := &model.User{ID: 1, Status: model.UserStatusApproved, Role: model.UserRoleAdmin}
	service := NewPermissionService(new(gorm.DB), &permissionUserRepositoryStub{user: admin}, new(permissionRepositoryStub))
	result, err := service.GetMine(authctx.WithUser(context.Background(), admin), &proto.GetMyPermissionsRequest{})
	if err != nil || len(result.Permissions) != 5 {
		t.Fatalf("管理员权限 result=%+v error=%v", result, err)
	}
	for _, item := range result.Permissions {
		if item.Level != model.PermissionLevelWrite {
			t.Fatalf("管理员模块 %s 权限=%s", item.Module, item.Level)
		}
	}
}
