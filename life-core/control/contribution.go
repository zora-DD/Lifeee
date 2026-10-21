package control

import (
	"context"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

// ListContributions 返回当前用户创建的内容。
func (c *Control) ListContributions(ctx context.Context, req *proto.ListContributionsRequest) (*proto.ListContributionsResponse, error) {
	return service.NewContributionService(database.MySQL(), dao.NewContentDao()).List(ctx, req)
}
