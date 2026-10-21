package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
)

func (c *Control) ListBills(ctx context.Context, req *proto.ListBillsRequest) (*proto.ListBillsResponse, error) {
	logger := hlog.FromContext(ctx).Named("ListBills")
	svc := newBillService()
	if err := svc.InitAndCheckListParam(ctx, req); err != nil {
		logger.Error("InitAndCheckListParam failed", hlog.Err(err))
		return nil, err
	}
	resp, err := svc.List(ctx, req)
	if err != nil {
		logger.Error("List failed", hlog.Err(err))
	}
	return resp, err
}

func (c *Control) GetBill(ctx context.Context, req *proto.GetBillRequest) (*proto.BillItem, error) {
	return newBillService().Get(ctx, req)
}

func (c *Control) CreateBill(ctx context.Context, req *proto.CreateBillRequest) (*proto.BillItem, error) {
	return newBillService().Create(ctx, req)
}

func (c *Control) UpdateBill(ctx context.Context, req *proto.UpdateBillRequest) (*proto.BillItem, error) {
	return newBillService().Update(ctx, req)
}

func (c *Control) DeleteBill(ctx context.Context, req *proto.DeleteBillRequest) (*proto.DeleteBillResponse, error) {
	return newBillService().Delete(ctx, req)
}

func (c *Control) GetBillStats(ctx context.Context, req *proto.BillStatsRequest) (*proto.BillStatsResponse, error) {
	return newBillService().Stats(ctx, req)
}

func newBillService() *service.BillService {
	return service.NewBillService(database.MySQL(), dao.NewBillDao())
}
