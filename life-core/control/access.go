package control

import (
	"context"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func newAccessService() *service.AccessService {
	return service.NewAccessService(database.MySQL(), dao.NewUserDao(), dao.NewUserModulePermissionDao())
}

func (c *Control) GetAccessStatus(ctx context.Context, req *proto.GetAccessStatusRequest) (*proto.UserItem, error) {
	return newAccessService().Status(ctx, req)
}

func (c *Control) ApplyAccess(ctx context.Context, req *proto.ApplyAccessRequest) (*proto.UserItem, error) {
	return newAccessService().Apply(ctx, req)
}

func (c *Control) ListAccessRequests(ctx context.Context, req *proto.ListAccessRequestsRequest) (*proto.ListAccessRequestsResponse, error) {
	return newAccessService().List(ctx, req)
}

func (c *Control) ModifyAccessRequest(ctx context.Context, req *proto.ModifyAccessRequestRequest) (*proto.UserItem, error) {
	return newAccessService().Modify(ctx, req)
}
