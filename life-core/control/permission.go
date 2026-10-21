package control

import (
	"context"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func newPermissionService() *service.PermissionService {
	return service.NewPermissionService(database.MySQL(), dao.NewUserDao(), dao.NewUserModulePermissionDao())
}

func (c *Control) GetMyPermissions(ctx context.Context, req *proto.GetMyPermissionsRequest) (*proto.GetMyPermissionsResponse, error) {
	return newPermissionService().GetMine(ctx, req)
}

func (c *Control) GetUserPermissions(ctx context.Context, req *proto.GetUserPermissionsRequest) (*proto.GetUserPermissionsResponse, error) {
	return newPermissionService().GetUser(ctx, req)
}

func (c *Control) ModifyUserPermissions(ctx context.Context, req *proto.ModifyUserPermissionsRequest) (*proto.GetUserPermissionsResponse, error) {
	return newPermissionService().Modify(ctx, req)
}
