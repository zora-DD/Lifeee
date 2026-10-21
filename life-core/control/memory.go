package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func (c *Control) ListMemories(ctx context.Context, req *proto.ListMemoriesRequest) (*proto.ListMemoriesResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListMemories").
		With(hlog.Int32("year", req.GetYear())).
		With(hlog.Int32("page", req.GetPage())).
		With(hlog.Int32("page_size", req.GetPageSize()))
	resp, err := newMemoryService().List(ctx, req)
	if err != nil {
		logger.Error("List failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func (c *Control) CreateMemory(ctx context.Context, req *proto.CreateMemoryRequest) (*proto.MemoryItem, error) {
	logger := hlog.FromContext(ctx).Named("CreateMemory").With(hlog.String("title", req.GetTitle()))
	resp, err := newMemoryService().Create(ctx, req)
	if err != nil {
		logger.Error("Create failed", hlog.Err(err))
		return nil, err
	}
	return resp, nil
}

func newMemoryService() *service.MemoryService {
	return service.NewMemoryService(database.MySQL(), dao.NewMemoryDao(), dao.NewMemoryPhotoDao())
}
