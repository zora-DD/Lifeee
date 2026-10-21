package control

import (
	"context"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/proto"
	"life-core/service"
	"life-core/third_party/minimax"
)

// GetTaskAISummary 返回今日任务事实与 AI 建议。
func (c *Control) GetTaskAISummary(ctx context.Context, req *proto.GetTaskAISummaryRequest) (*proto.TaskAISummaryResponse, error) {
	logger := hlog.FromContext(ctx).Named("GetTaskAISummary").With(hlog.Bool("refresh", req.GetRefresh()))
	response, err := c.getAISummaryService().TaskSummary(ctx, req)
	if err != nil {
		logger.Error("TaskSummary failed", hlog.Err(err))
		return nil, err
	}
	return response, nil
}

// GetLifeAISummary 返回当前用户指定周期的生活总结。
func (c *Control) GetLifeAISummary(ctx context.Context, req *proto.GetLifeAISummaryRequest) (*proto.LifeAISummaryResponse, error) {
	logger := hlog.FromContext(ctx).Named("GetLifeAISummary").
		With(hlog.String("scope", req.GetScope())).
		With(hlog.Int32("year", req.GetYear())).
		With(hlog.Int32("month", req.GetMonth())).
		With(hlog.Bool("refresh", req.GetRefresh()))
	response, err := c.getAISummaryService().LifeSummary(ctx, req)
	if err != nil {
		logger.Error("LifeSummary failed", hlog.Err(err))
		return nil, err
	}
	return response, nil
}

func (c *Control) getAISummaryService() *service.AISummaryService {
	c.aiSummaryOnce.Do(func() {
		var organizer service.AISummaryOrganizer
		cfg := config.Get()
		if cfg != nil && cfg.Voice.Enabled {
			if client, err := minimax.NewMiniMaxClient(cfg.Voice.MiniMax); err == nil {
				organizer = client
			}
		}
		c.aiSummary = service.NewAISummaryService(
			database.MySQL(),
			dao.NewAISummaryDao(),
			organizer,
			service.NewAISummaryCache(),
		)
	})
	return c.aiSummary
}
