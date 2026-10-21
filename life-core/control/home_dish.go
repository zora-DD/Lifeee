package control

import (
	"context"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func homeDishService() *service.HomeDishService {
	return service.NewHomeDishService(database.MySQL(), dao.NewHomeDishDao())
}
func (c *Control) ListHomeDishes(ctx context.Context, req *proto.ListHomeDishesRequest) (*proto.ListHomeDishesResponse, error) {
	return homeDishService().List(ctx, req)
}
func (c *Control) GetHomeDish(ctx context.Context, req *proto.GetHomeDishRequest) (*proto.HomeDishItem, error) {
	return homeDishService().Get(ctx, req)
}
func (c *Control) CreateHomeDish(ctx context.Context, req *proto.CreateHomeDishRequest) (*proto.HomeDishItem, error) {
	return homeDishService().Create(ctx, req)
}
func (c *Control) UpdateHomeDish(ctx context.Context, req *proto.UpdateHomeDishRequest) (*proto.HomeDishItem, error) {
	return homeDishService().Update(ctx, req)
}
func (c *Control) DeleteHomeDish(ctx context.Context, req *proto.DeleteHomeDishRequest) (*proto.DeleteHomeDishResponse, error) {
	return homeDishService().Delete(ctx, req)
}
func (c *Control) CookHomeDish(ctx context.Context, req *proto.CookHomeDishRequest) (*proto.HomeDishItem, error) {
	return homeDishService().Cook(ctx, req)
}
func (c *Control) RecommendHomeDish(ctx context.Context, req *proto.RecommendHomeDishRequest) (*proto.HomeDishItem, error) {
	return homeDishService().Recommend(ctx, req)
}
