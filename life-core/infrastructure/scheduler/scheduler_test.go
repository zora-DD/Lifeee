package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type schedulerJobStub struct {
	name       string
	spec       string
	runs       atomic.Int32
	started    chan struct{}
	release    chan struct{}
	err        error
	panicOnRun bool
}

func (j *schedulerJobStub) Name() string { return j.name }
func (j *schedulerJobStub) Spec() string { return j.spec }
func (j *schedulerJobStub) Run(ctx context.Context) error {
	j.runs.Add(1)
	if j.started != nil {
		select {
		case j.started <- struct{}{}:
		default:
		}
	}
	if j.panicOnRun {
		panic("test panic")
	}
	if j.release != nil {
		select {
		case <-j.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return j.err
}

func TestSchedulerRunOnStartAndSkipOverlapping(t *testing.T) {
	s, err := New(time.Local)
	if err != nil {
		t.Fatal(err)
	}
	job := &schedulerJobStub{
		name: "blocking", spec: "@every 1h",
		started: make(chan struct{}, 1), release: make(chan struct{}),
	}
	if err := s.Register(job, time.Minute, true); err != nil {
		t.Fatal(err)
	}
	s.Start()
	select {
	case <-job.started:
	case <-time.After(time.Second):
		t.Fatal("启动补跑未执行")
	}

	entry := s.jobs[job.name]
	s.execute(entry)
	if job.runs.Load() != 1 {
		t.Fatalf("重叠任务未跳过，执行次数=%d", job.runs.Load())
	}
	close(job.release)
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerValidationAndPanicRecovery(t *testing.T) {
	s, err := New(time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Register(nil, time.Second, false); !errors.Is(err, ErrNilJob) {
		t.Fatalf("nil job error=%v", err)
	}
	invalid := &schedulerJobStub{name: "invalid", spec: "not-a-cron"}
	if err := s.Register(invalid, time.Second, false); err == nil {
		t.Fatal("非法 cron 表达式应注册失败")
	}
	panicJob := &schedulerJobStub{name: "panic", spec: "@every 1h", panicOnRun: true}
	if err := s.Register(panicJob, time.Second, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Register(panicJob, time.Second, false); !errors.Is(err, ErrDuplicateJob) {
		t.Fatalf("重复任务 error=%v", err)
	}
	// execute 内部必须恢复 panic，不能让调度进程退出。
	s.execute(s.jobs[panicJob.name])
	if panicJob.runs.Load() != 1 {
		t.Fatalf("panic 任务执行次数=%d", panicJob.runs.Load())
	}
	s.Start()
	if err := s.Register(&schedulerJobStub{name: "late", spec: "@every 1h"}, time.Second, false); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("启动后注册 error=%v", err)
	}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerTimeoutAndShutdown(t *testing.T) {
	s, err := New(time.Local)
	if err != nil {
		t.Fatal(err)
	}
	job := &schedulerJobStub{name: "timeout", spec: "@every 1h", release: make(chan struct{})}
	if err := s.Register(job, 20*time.Millisecond, true); err != nil {
		t.Fatal(err)
	}
	s.Start()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error=%v", err)
	}
	if job.runs.Load() != 1 {
		t.Fatalf("超时任务执行次数=%d", job.runs.Load())
	}
}

type summaryGeneratorStub struct {
	refresh bool
	calls   int
}

func (s *summaryGeneratorStub) GenerateDailyTaskSummary(_ context.Context, refresh bool) error {
	s.calls++
	s.refresh = refresh
	return nil
}

func TestDailyTaskSummaryJobDoesNotForceRefresh(t *testing.T) {
	generator := new(summaryGeneratorStub)
	job := &DailyTaskSummaryJob{spec: "0 6 * * *", generator: generator}
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if generator.calls != 1 || generator.refresh {
		t.Fatalf("每日总结调用不正确: calls=%d refresh=%v", generator.calls, generator.refresh)
	}
}
