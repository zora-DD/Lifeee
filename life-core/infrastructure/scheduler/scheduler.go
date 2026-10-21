// Package scheduler 提供跨平台的进程内定时任务调度和任务注册。
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vaynedu/hollow/pkg/hlog"
)

var (
	ErrNilJob         = errors.New("scheduler: nil job")
	ErrDuplicateJob   = errors.New("scheduler: duplicate job name")
	ErrAlreadyStarted = errors.New("scheduler: cannot register after start")
)

// Job 是所有定时任务需要实现的最小接口。
type Job interface {
	Name() string
	Spec() string
	Run(context.Context) error
}

type registeredJob struct {
	job        Job
	timeout    time.Duration
	runOnStart bool
	running    atomic.Bool
}

// Scheduler 管理任务注册、执行、重叠保护和优雅关闭。
type Scheduler struct {
	cron    *cron.Cron
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	started bool
	jobs    map[string]*registeredJob
	startup sync.WaitGroup
}

// New 创建使用指定时区的调度器。
func New(location *time.Location) (*Scheduler, error) {
	if location == nil {
		return nil, errors.New("scheduler: nil location")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cron: cron.New(cron.WithLocation(location)),
		ctx:  ctx, cancel: cancel,
		jobs: make(map[string]*registeredJob),
	}, nil
}

// Register 注册任务。timeout 限制单次执行时间，runOnStart 用于启动补跑。
func (s *Scheduler) Register(job Job, timeout time.Duration, runOnStart bool) error {
	if s == nil || job == nil {
		return ErrNilJob
	}
	name := job.Name()
	if name == "" {
		return errors.New("scheduler: empty job name")
	}
	if timeout <= 0 {
		return errors.New("scheduler: timeout must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return ErrAlreadyStarted
	}
	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateJob, name)
	}
	entry := &registeredJob{job: job, timeout: timeout, runOnStart: runOnStart}
	if _, err := s.cron.AddFunc(job.Spec(), func() { s.execute(entry) }); err != nil {
		return fmt.Errorf("scheduler: register %s: %w", name, err)
	}
	s.jobs[name] = entry
	return nil
}

// Start 启动 cron，并异步执行需要启动补跑的任务。
func (s *Scheduler) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	startupJobs := make([]*registeredJob, 0)
	for _, entry := range s.jobs {
		if entry.runOnStart {
			startupJobs = append(startupJobs, entry)
		}
	}
	s.cron.Start()
	for _, entry := range startupJobs {
		s.startup.Add(1)
		go func(item *registeredJob) {
			defer s.startup.Done()
			s.execute(item)
		}(entry)
	}
	s.mu.Unlock()
}

func (s *Scheduler) execute(entry *registeredJob) {
	if !entry.running.CompareAndSwap(false, true) {
		hlog.L().Named("scheduler").Info("job skipped because previous run is still active", hlog.String("job", entry.job.Name()))
		return
	}
	defer entry.running.Store(false)

	startedAt := time.Now()
	logger := hlog.L().Named("scheduler").With(hlog.String("job", entry.job.Name()))
	ctx, cancel := context.WithTimeout(s.ctx, entry.timeout)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("job panicked",
				hlog.String("panic", fmt.Sprint(recovered)),
				hlog.String("stack", string(debug.Stack())),
				hlog.Int64("duration_ms", time.Since(startedAt).Milliseconds()))
		}
	}()

	if err := entry.job.Run(ctx); err != nil {
		logger.Error("job failed", hlog.Err(err), hlog.Int64("duration_ms", time.Since(startedAt).Milliseconds()))
		return
	}
	logger.Info("job completed", hlog.Int64("duration_ms", time.Since(startedAt).Milliseconds()))
}

// Shutdown 停止接收新的触发，并等待正在执行的任务结束。
func (s *Scheduler) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	cronStopped := s.cron.Stop()
	done := make(chan struct{})
	go func() {
		<-cronStopped.Done()
		s.startup.Wait()
		close(done)
	}()
	select {
	case <-done:
		s.cancel()
		return nil
	case <-ctx.Done():
		s.cancel()
		return ctx.Err()
	}
}
