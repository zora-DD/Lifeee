package control

import (
	"context"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func newFavoriteService() *service.FavoriteService {
	return service.NewFavoriteService(database.MySQL(), dao.NewFavoriteDao(), dao.NewContentDao())
}

// ListFavorites 返回当前用户的收藏列表。
func (c *Control) ListFavorites(ctx context.Context, req *proto.ListFavoritesRequest) (*proto.ListFavoritesResponse, error) {
	return newFavoriteService().List(ctx, req)
}

// GetFavoriteStatus 返回指定内容的收藏状态。
func (c *Control) GetFavoriteStatus(ctx context.Context, req *proto.GetFavoriteStatusRequest) (*proto.FavoriteStatusResponse, error) {
	return newFavoriteService().Status(ctx, req)
}

// ModifyFavorite 收藏或取消收藏指定内容。
func (c *Control) ModifyFavorite(ctx context.Context, req *proto.ModifyFavoriteRequest) (*proto.FavoriteStatusResponse, error) {
	return newFavoriteService().Modify(ctx, req)
}
