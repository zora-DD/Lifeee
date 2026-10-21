package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func (c *Control) ListActivityLogs(ctx context.Context, req *proto.ListActivityLogsRequest) (*proto.ListActivityLogsResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListActivityLogs").
		With(hlog.Uint64("created_by", req.GetCreatedBy())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))
	svc := newActivityLogService()
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

func (c *Control) GetActivityLog(ctx context.Context, req *proto.GetActivityLogRequest) (*proto.ActivityLogItem, error) {
	logger := hlog.FromContext(ctx).Named("GetActivityLog").With(hlog.String("id", req.GetId()))
	resp, err := newActivityLogService().Get(ctx, req)
	if err != nil {
		logger.Error("Get failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) CreateActivityLog(ctx context.Context, req *proto.CreateActivityLogRequest) (*proto.ActivityLogItem, error) {
	logger := hlog.FromContext(ctx).Named("CreateActivityLog")
	resp, err := newActivityLogService().Create(ctx, req)
	if err != nil {
		logger.Error("Create failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) UpdateActivityLog(ctx context.Context, req *proto.UpdateActivityLogRequest) (*proto.ActivityLogItem, error) {
	logger := hlog.FromContext(ctx).Named("UpdateActivityLog").With(hlog.String("id", req.GetId()))
	resp, err := newActivityLogService().Update(ctx, req)
	if err != nil {
		logger.Error("Update failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) DeleteActivityLog(ctx context.Context, req *proto.DeleteActivityLogRequest) (*proto.DeleteActivityLogResponse, error) {
	logger := hlog.FromContext(ctx).Named("DeleteActivityLog").With(hlog.String("id", req.GetId()))
	resp, err := newActivityLogService().Delete(ctx, req)
	if err != nil {
		logger.Error("Delete failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) GetActivityLogStats(ctx context.Context, req *proto.ActivityLogStatsRequest) (*proto.ActivityLogStatsResponse, error) {
	logger := hlog.FromContext(ctx).Named("GetActivityLogStats").
		With(hlog.String("scope", req.GetScope())).
		With(hlog.Int32("year", req.GetYear())).
		With(hlog.Int32("month", req.GetMonth()))
	resp, err := newActivityLogService().Stats(ctx, req)
	if err != nil {
		logger.Error("Stats failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func newActivityLogService() *service.ActivityLogService {
	return service.NewActivityLogService(database.MySQL(), dao.NewActivityLogDao())
}
