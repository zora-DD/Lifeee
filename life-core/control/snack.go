package control

import (
	"context"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func snackService() *service.SnackService {
	return service.NewSnackService(database.MySQL(), dao.NewSnackDao())
}

func (c *Control) ListSnacks(ctx context.Context, req *proto.ListSnacksRequest) (*proto.ListSnacksResponse, error) {
	return snackService().List(ctx, req)
}

func (c *Control) GetSnack(ctx context.Context, req *proto.GetSnackRequest) (*proto.SnackItem, error) {
	return snackService().Get(ctx, req)
}

func (c *Control) CreateSnack(ctx context.Context, req *proto.CreateSnackRequest) (*proto.SnackItem, error) {
	return snackService().Create(ctx, req)
}

func (c *Control) UpdateSnack(ctx context.Context, req *proto.UpdateSnackRequest) (*proto.SnackItem, error) {
	return snackService().Update(ctx, req)
}

func (c *Control) DeleteSnack(ctx context.Context, req *proto.DeleteSnackRequest) (*proto.DeleteSnackResponse, error) {
	return snackService().Delete(ctx, req)
}

func (c *Control) TasteSnack(ctx context.Context, req *proto.TasteSnackRequest) (*proto.SnackItem, error) {
	return snackService().Taste(ctx, req)
}

func (c *Control) RecommendSnack(ctx context.Context, req *proto.RecommendSnackRequest) (*proto.SnackItem, error) {
	return snackService().Recommend(ctx, req)
}
