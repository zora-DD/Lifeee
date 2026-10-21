package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func (c *Control) ListDailyRoutines(ctx context.Context, req *proto.ListDailyRoutinesRequest) (*proto.ListDailyRoutinesResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListDailyRoutines").With(hlog.String("date", req.GetDate()))
	resp, err := newDailyRoutineService().List(ctx, req)
	if err != nil {
		logger.Error("List failed", hlog.Err(err))
	}
	return resp, err
}

func (c *Control) CreateDailyRoutine(ctx context.Context, req *proto.CreateDailyRoutineRequest) (*proto.DailyRoutineItem, error) {
	logger := hlog.FromContext(ctx).Named("CreateDailyRoutine").With(hlog.String("title", req.GetTitle()))
	resp, err := newDailyRoutineService().Create(ctx, req)
	if err != nil {
		logger.Error("Create failed", hlog.Err(err))
	}
	return resp, err
}

func (c *Control) UpdateDailyRoutine(ctx context.Context, req *proto.UpdateDailyRoutineRequest) (*proto.DailyRoutineItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdateDailyRoutine").With(hlog.String("id", req.GetId()))
	resp, err := newDailyRoutineService().Update(ctx, req)
	if err != nil {
		logger.Error("Update failed", hlog.Err(err))
	}
	return resp, err
}

func (c *Control) DeleteDailyRoutine(ctx context.Context, req *proto.DeleteDailyRoutineRequest) (*proto.DeleteDailyRoutineResponse, error) {
	logger := hlog.FromContext(ctx).Named("DeleteDailyRoutine").With(hlog.String("id", req.GetId()))
	resp, err := newDailyRoutineService().Delete(ctx, req)
	if err != nil {
		logger.Error("Delete failed", hlog.Err(err))
	}
	return resp, err
}

func (c *Control) CheckinDailyRoutine(ctx context.Context, req *proto.CheckinDailyRoutineRequest) (*proto.DailyRoutineItem, error) {
	logger := hlog.FromContext(ctx).Named("CheckinDailyRoutine").With(hlog.String("id", req.GetId()))
	resp, err := newDailyRoutineService().Checkin(ctx, req)
	if err != nil {
		logger.Error("Checkin failed", hlog.Err(err))
	}
	return resp, err
}

func newDailyRoutineService() *service.DailyRoutineService {
	return service.NewDailyRoutineService(database.MySQL(), dao.NewDailyRoutineDao())
}
