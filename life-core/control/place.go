package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func (c *Control) ListPlaces(ctx context.Context, req *proto.ListPlacesRequest) (*proto.ListPlacesResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListPlaces").
		With(hlog.String("category", req.GetCategory())).
		With(hlog.String("status", req.GetStatus())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))
	svc := newPlaceService()
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

func (c *Control) GetPlace(ctx context.Context, req *proto.GetPlaceRequest) (*proto.PlaceItem, error) {
	logger := hlog.FromContext(ctx).Named("GetPlace").With(hlog.String("id", req.GetId()))
	resp, err := newPlaceService().Get(ctx, req)
	if err != nil {
		logger.Error("Get failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) CreatePlace(ctx context.Context, req *proto.CreatePlaceRequest) (*proto.PlaceItem, error) {
	logger := hlog.FromContext(ctx).Named("CreatePlace").With(hlog.String("name", req.GetName()))
	resp, err := newPlaceService().Create(ctx, req)
	if err != nil {
		logger.Error("Create failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) UpdatePlace(ctx context.Context, req *proto.UpdatePlaceRequest) (*proto.PlaceItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdatePlace").With(hlog.String("id", req.GetId()))
	resp, err := newPlaceService().Update(ctx, req)
	if err != nil {
		logger.Error("Update failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) DeletePlace(ctx context.Context, req *proto.DeletePlaceRequest) (*proto.DeletePlaceResponse, error) {
	logger := hlog.FromContext(ctx).Named("DeletePlace").With(hlog.String("id", req.GetId()))
	resp, err := newPlaceService().Delete(ctx, req)
	if err != nil {
		logger.Error("Delete failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func newPlaceService() *service.PlaceService {
	return service.NewPlaceService(database.MySQL(), dao.NewPlaceDao())
}
