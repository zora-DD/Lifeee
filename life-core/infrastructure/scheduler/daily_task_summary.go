package scheduler

import (
	"context"
	"errors"

	"life-core/service"
)

const dailyTaskSummaryJobName = "daily_task_summary"

// TaskSummaryGenerator 抽象每日总结能力，便于调度层独立测试。
type TaskSummaryGenerator interface {
	GenerateDailyTaskSummary(context.Context, bool) error
}

type aiSummaryServiceAdapter struct {
	service *service.AISummaryService
}

func (a aiSummaryServiceAdapter) GenerateDailyTaskSummary(ctx context.Context, refresh bool) error {
	_, err := a.service.GenerateDailyTaskSummary(ctx, refresh)
	return err
}

// DailyTaskSummaryJob 每天预生成一份全局公开任务总结。
type DailyTaskSummaryJob struct {
	spec      string
	generator TaskSummaryGenerator
}

func NewDailyTaskSummaryJob(spec string, summaryService *service.AISummaryService) *DailyTaskSummaryJob {
	return &DailyTaskSummaryJob{spec: spec, generator: aiSummaryServiceAdapter{service: summaryService}}
}

func (j *DailyTaskSummaryJob) Name() string { return dailyTaskSummaryJobName }
func (j *DailyTaskSummaryJob) Spec() string { return j.spec }

func (j *DailyTaskSummaryJob) Run(ctx context.Context) error {
	if j == nil || j.generator == nil {
		return errors.New("scheduler: daily task summary generator is not initialized")
	}
	// 不强制刷新：当天事实一致时直接复用数据库，启动补跑不会重复调用 AI。
	return j.generator.GenerateDailyTaskSummary(ctx, false)
}
