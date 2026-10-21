package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func (c *Control) ListRestaurants(ctx context.Context, req *proto.ListRestaurantsRequest) (*proto.ListRestaurantsResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListRestaurants").
		With(hlog.String("category", req.GetCategory())).
		With(hlog.String("preference", req.GetPreference())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))
	svc := newRestaurantService()
	if err := svc.InitAndCheckListParam(ctx, req); err != nil {
		logger.Error("InitAndCheckListParam failed", hlog.Err(err))
		return nil, err
	}
	resp, err := svc.List(ctx, req)
	if err != nil {
		logger.Error("List failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) GetRestaurant(ctx context.Context, req *proto.GetRestaurantRequest) (*proto.RestaurantItem, error) {
	logger := hlog.FromContext(ctx).Named("GetRestaurant").With(hlog.String("id", req.GetId()))
	resp, err := newRestaurantService().Get(ctx, req)
	if err != nil {
		logger.Error("Get failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) CreateRestaurant(ctx context.Context, req *proto.CreateRestaurantRequest) (*proto.RestaurantItem, error) {
	logger := hlog.FromContext(ctx).Named("CreateRestaurant").With(hlog.String("name", req.GetName()))
	resp, err := newRestaurantService().Create(ctx, req)
	if err != nil {
		logger.Error("Create failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) UpdateRestaurant(ctx context.Context, req *proto.UpdateRestaurantRequest) (*proto.RestaurantItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdateRestaurant").With(hlog.String("id", req.GetId()))
	resp, err := newRestaurantService().Update(ctx, req)
	if err != nil {
		logger.Error("Update failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) DeleteRestaurant(ctx context.Context, req *proto.DeleteRestaurantRequest) (*proto.DeleteRestaurantResponse, error) {
	logger := hlog.FromContext(ctx).Named("DeleteRestaurant").With(hlog.String("id", req.GetId()))
	resp, err := newRestaurantService().Delete(ctx, req)
	if err != nil {
		logger.Error("Delete failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) RecommendRestaurant(ctx context.Context, req *proto.RecommendRestaurantRequest) (*proto.RestaurantItem, error) {
	logger := hlog.FromContext(ctx).Named("RecommendRestaurant").With(hlog.String("category", req.GetCategory()))
	resp, err := newRestaurantService().Recommend(ctx, req)
	if err != nil {
		logger.Error("Recommend failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) ListRestaurantConsumptions(
	ctx context.Context,
	req *proto.ListRestaurantConsumptionsRequest,
) (*proto.ListRestaurantConsumptionsResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListRestaurantConsumptions").
		With(hlog.String("restaurant_id", req.GetRestaurantId()))
	resp, err := newRestaurantService().ListConsumptions(ctx, req)
	if err != nil {
		logger.Error("ListConsumptions failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) CreateRestaurantConsumption(
	ctx context.Context,
	req *proto.CreateRestaurantConsumptionRequest,
) (*proto.RestaurantConsumptionItem, error) {
	logger := hlog.FromContext(ctx).Named("CreateRestaurantConsumption").
		With(hlog.String("restaurant_id", req.GetRestaurantId()))
	resp, err := newRestaurantService().CreateConsumption(ctx, req)
	if err != nil {
		logger.Error("CreateConsumption failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) UpdateRestaurantConsumption(
	ctx context.Context,
	req *proto.UpdateRestaurantConsumptionRequest,
) (*proto.RestaurantConsumptionItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdateRestaurantConsumption").
		With(hlog.String("restaurant_id", req.GetRestaurantId())).
		With(hlog.String("id", req.GetId()))
	resp, err := newRestaurantService().UpdateConsumption(ctx, req)
	if err != nil {
		logger.Error("UpdateConsumption failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) DeleteRestaurantConsumption(
	ctx context.Context,
	req *proto.DeleteRestaurantConsumptionRequest,
) (*proto.DeleteRestaurantConsumptionResponse, error) {
	logger := hlog.FromContext(ctx).Named("DeleteRestaurantConsumption").
		With(hlog.String("restaurant_id", req.GetRestaurantId())).
		With(hlog.String("id", req.GetId()))
	resp, err := newRestaurantService().DeleteConsumption(ctx, req)
	if err != nil {
		logger.Error("DeleteConsumption failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func newRestaurantService() *service.RestaurantService {
	return service.NewRestaurantService(
		database.MySQL(),
		dao.NewRestaurantDao(),
		dao.NewRestaurantConsumptionDao(),
	)
}
