package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

type aiSummaryRepositoryStub struct {
	tasks             []model.Task
	snapshot          *dao.LifeSummarySnapshot
	taskErr           error
	lifeErr           error
	taskCalls         int
	lifeCalls         int
	lastUserID        uint64
	lastFrom          time.Time
	lastTo            time.Time
	persisted         *model.TaskAISummary
	getPersistedCalls int
	upsertCalls       int
}

func (s *aiSummaryRepositoryStub) GetTaskSummaryByDate(_ context.Context, _ *gorm.DB, _ time.Time) (*model.TaskAISummary, error) {
	s.getPersistedCalls++
	if s.persisted == nil {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *s.persisted
	copy.PayloadJSON = append([]byte(nil), s.persisted.PayloadJSON...)
	return &copy, nil
}

func (s *aiSummaryRepositoryStub) UpsertTaskSummary(_ context.Context, _ *gorm.DB, row *model.TaskAISummary) error {
	s.upsertCalls++
	copy := *row
	copy.PayloadJSON = append([]byte(nil), row.PayloadJSON...)
	s.persisted = &copy
	return nil
}

func (s *aiSummaryRepositoryStub) ListTasksForSummary(_ context.Context, _ *gorm.DB, from, to time.Time) ([]model.Task, error) {
	s.taskCalls++
	s.lastFrom, s.lastTo = from, to
	return s.tasks, s.taskErr
}

func (s *aiSummaryRepositoryStub) GetLifeSnapshot(_ context.Context, _ *gorm.DB, userID uint64, from, to time.Time) (*dao.LifeSummarySnapshot, error) {
	s.lifeCalls++
	s.lastUserID, s.lastFrom, s.lastTo = userID, from, to
	return s.snapshot, s.lifeErr
}

type aiSummaryOrganizerStub struct {
	taskDraft     *SummaryDraft
	lifeDraft     *SummaryDraft
	taskErr       error
	lifeErr       error
	taskCalls     int
	lifeCalls     int
	lastTaskFacts TaskSummaryFact
	lastLifeFacts LifeSummaryFact
}

func (s *aiSummaryOrganizerStub) SummarizeTasks(_ context.Context, facts TaskSummaryFact) (*SummaryDraft, error) {
	s.taskCalls++
	s.lastTaskFacts = facts
	return s.taskDraft, s.taskErr
}

func (s *aiSummaryOrganizerStub) SummarizeLife(_ context.Context, facts LifeSummaryFact) (*SummaryDraft, error) {
	s.lifeCalls++
	s.lastLifeFacts = facts
	return s.lifeDraft, s.lifeErr
}

func aiSummaryTestContext(userID uint64) context.Context {
	return authctx.WithUser(context.Background(), &model.User{ID: userID})
}

func timePointer(value time.Time) *time.Time { return &value }

func TestTaskAISummaryClassificationAndCache(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, location)
	repository := &aiSummaryRepositoryStub{tasks: []model.Task{
		{ID: 1, Title: "处理逾期事项", Status: model.TaskStatusTodo, Priority: model.TaskPriorityLow, DueAt: timePointer(now.Add(-time.Hour))},
		{ID: 2, Title: "今天提交", Status: model.TaskStatusTodo, Priority: model.TaskPriorityMedium, DueAt: timePointer(now.Add(time.Hour))},
		{ID: 3, Title: "后天出发", Status: model.TaskStatusTodo, Priority: model.TaskPriorityLow, DueAt: timePointer(now.Add(48 * time.Hour))},
		{ID: 4, Title: "重要但不紧急", Status: model.TaskStatusTodo, Priority: model.TaskPriorityHigh},
		{ID: 5, Title: "普通事项", Status: model.TaskStatusTodo, Priority: model.TaskPriorityLow},
		{ID: 6, Title: "今天完成", Status: model.TaskStatusDone, Priority: model.TaskPriorityLow, UpdatedAt: now.Add(-time.Hour)},
	}}
	organizer := &aiSummaryOrganizerStub{taskDraft: &SummaryDraft{
		Summary:     "先处理逾期事项，再完成今天截止的任务。",
		Suggestions: []string{"先完成最紧急的一项"},
	}}
	service := NewAISummaryService(new(gorm.DB), repository, organizer, NewAISummaryCache())
	service.now = func() time.Time { return now }
	ctx := aiSummaryTestContext(7)

	first, err := service.TaskSummary(ctx, &proto.GetTaskAISummaryRequest{})
	if err != nil {
		t.Fatalf("TaskSummary() error = %v", err)
	}
	if first.TodoCount != 5 || first.DoneTodayCount != 1 || first.OverdueCount != 1 || first.DueTodayCount != 1 || first.UpcomingCount != 1 || first.HighPriorityCount != 1 {
		t.Fatalf("任务统计不正确: %+v", first)
	}
	if len(first.FocusTasks) != 3 {
		t.Fatalf("重点任务数量 = %d, want 3", len(first.FocusTasks))
	}
	if first.FocusTasks[0].Id != "1" || first.FocusTasks[1].Id != "2" || first.FocusTasks[2].Id != "4" {
		t.Fatalf("重点任务排序不正确: %+v", first.FocusTasks)
	}
	if !first.AiGenerated || first.Cached {
		t.Fatalf("首次生成标记不正确: aiGenerated=%v cached=%v", first.AiGenerated, first.Cached)
	}
	if repository.upsertCalls != 1 || repository.persisted == nil {
		t.Fatalf("任务总结未持久化: upsertCalls=%d", repository.upsertCalls)
	}
	if organizer.lastTaskFacts.TodoCount != 5 || len(organizer.lastTaskFacts.FocusTasks) != 3 {
		t.Fatalf("传给模型的事实不正确: %+v", organizer.lastTaskFacts)
	}

	second, err := service.TaskSummary(ctx, &proto.GetTaskAISummaryRequest{})
	if err != nil {
		t.Fatalf("第二次 TaskSummary() error = %v", err)
	}
	if !second.Cached || organizer.taskCalls != 1 {
		t.Fatalf("同日同数据未命中缓存: cached=%v calls=%d", second.Cached, organizer.taskCalls)
	}

	refreshed, err := service.TaskSummary(ctx, &proto.GetTaskAISummaryRequest{Refresh: true})
	if err != nil {
		t.Fatalf("刷新 TaskSummary() error = %v", err)
	}
	if refreshed.Cached || organizer.taskCalls != 2 {
		t.Fatalf("强制刷新未重新生成: cached=%v calls=%d", refreshed.Cached, organizer.taskCalls)
	}
}

func TestTaskAISummaryUsesPersistedResultAfterRestart(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, location)
	repository := &aiSummaryRepositoryStub{tasks: []model.Task{{ID: 1, Title: "当天任务", Status: model.TaskStatusTodo, UpdatedAt: now}}}
	organizer := &aiSummaryOrganizerStub{taskDraft: &SummaryDraft{Summary: "已生成", Suggestions: []string{"去完成"}}}
	firstService := NewAISummaryService(new(gorm.DB), repository, organizer, NewAISummaryCache())
	firstService.now = func() time.Time { return now }
	if _, err := firstService.GenerateDailyTaskSummary(context.Background(), false); err != nil {
		t.Fatalf("首次预生成 error = %v", err)
	}

	secondService := NewAISummaryService(new(gorm.DB), repository, organizer, NewAISummaryCache())
	secondService.now = func() time.Time { return now }
	result, err := secondService.GenerateDailyTaskSummary(context.Background(), false)
	if err != nil {
		t.Fatalf("重启后读取持久化结果 error = %v", err)
	}
	if !result.Cached || organizer.taskCalls != 1 || repository.upsertCalls != 1 {
		t.Fatalf("未复用持久化结果: cached=%v modelCalls=%d upsertCalls=%d", result.Cached, organizer.taskCalls, repository.upsertCalls)
	}

	repository.tasks[0].Title = "事实已变化"
	changed, err := secondService.GenerateDailyTaskSummary(context.Background(), false)
	if err != nil {
		t.Fatalf("事实变化后重新生成 error = %v", err)
	}
	if changed.Cached || organizer.taskCalls != 2 || repository.upsertCalls != 2 {
		t.Fatalf("事实变化后未重新生成: cached=%v modelCalls=%d upsertCalls=%d", changed.Cached, organizer.taskCalls, repository.upsertCalls)
	}
}

func TestLifeAISummaryUsesCurrentUserAndFactFingerprint(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, location)
	repository := &aiSummaryRepositoryStub{snapshot: &dao.LifeSummarySnapshot{
		TaskCreatedCount: 8, TaskDoneCount: 6, ActivityCount: 12, ActivityMinutes: 320,
		BillCount: 9, ExpenseTotal: "1280.5", RestaurantVisitCount: 3, VisitedPlaceCount: 2,
	}}
	organizer := &aiSummaryOrganizerStub{lifeDraft: &SummaryDraft{
		Summary: "这个月保持了稳定记录。", Highlights: []string{"完成了 6 项任务"}, Suggestions: []string{"继续保持"},
	}}
	service := NewAISummaryService(new(gorm.DB), repository, organizer, NewAISummaryCache())
	service.now = func() time.Time { return now }
	ctx := aiSummaryTestContext(88)
	req := &proto.GetLifeAISummaryRequest{Scope: "month", Year: 2026, Month: 9}

	first, err := service.LifeSummary(ctx, req)
	if err != nil {
		t.Fatalf("LifeSummary() error = %v", err)
	}
	if repository.lastUserID != 88 {
		t.Fatalf("查询用户 = %d, want 88", repository.lastUserID)
	}
	if first.ExpenseTotal != "1280.50" || first.TaskDoneCount != 6 || first.ActivityMinutes != 320 {
		t.Fatalf("生活统计不正确: %+v", first)
	}
	if first.PeriodLabel != "2026年9月" || !first.AiGenerated || first.Cached {
		t.Fatalf("生活总结标记不正确: %+v", first)
	}
	if organizer.lastLifeFacts.BillCount != 9 || organizer.lastLifeFacts.ExpenseTotal != "1280.50" {
		t.Fatalf("传给模型的生活事实不正确: %+v", organizer.lastLifeFacts)
	}

	second, err := service.LifeSummary(ctx, req)
	if err != nil {
		t.Fatalf("第二次 LifeSummary() error = %v", err)
	}
	if !second.Cached || organizer.lifeCalls != 1 {
		t.Fatalf("生活总结未命中缓存: cached=%v calls=%d", second.Cached, organizer.lifeCalls)
	}

	repository.snapshot.ActivityCount++
	changed, err := service.LifeSummary(ctx, req)
	if err != nil {
		t.Fatalf("数据变化后 LifeSummary() error = %v", err)
	}
	if changed.Cached || organizer.lifeCalls != 2 {
		t.Fatalf("事实变化后仍错误复用缓存: cached=%v calls=%d", changed.Cached, organizer.lifeCalls)
	}
}

func TestAISummaryFallbackAndValidation(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.Local)
	repository := &aiSummaryRepositoryStub{snapshot: &dao.LifeSummarySnapshot{ExpenseTotal: "0"}}
	organizer := &aiSummaryOrganizerStub{taskErr: errors.New("模型不可用"), lifeErr: errors.New("模型不可用")}
	service := NewAISummaryService(new(gorm.DB), repository, organizer, NewAISummaryCache())
	service.now = func() time.Time { return now }
	ctx := aiSummaryTestContext(1)

	taskResult, err := service.TaskSummary(ctx, nil)
	if err != nil {
		t.Fatalf("降级任务总结 error = %v", err)
	}
	if taskResult.AiGenerated || taskResult.Summary == "" {
		t.Fatalf("任务降级结果不正确: %+v", taskResult)
	}

	_, err = service.LifeSummary(ctx, &proto.GetLifeAISummaryRequest{Scope: "week", Year: 2026})
	if err == nil {
		t.Fatal("不支持的周期应返回错误")
	}

	_, err = service.TaskSummary(context.Background(), nil)
	if err == nil {
		t.Fatal("未登录请求应返回错误")
	}
}
