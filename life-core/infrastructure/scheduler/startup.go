package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vaynedu/hollow/pkg/hlog"

	"life-core/config"
	"life-core/dao"
	"life-core/infrastructure/database"
	"life-core/service"
	"life-core/third_party/minimax"
)

var (
	defaultMu        sync.Mutex
	defaultScheduler *Scheduler
)

// Startup 根据配置注册并启动全部定时任务。
func Startup() error {
	cfg := config.Get()
	if cfg == nil {
		return errors.New("scheduler: config is not loaded")
	}
	if !cfg.Scheduler.Enabled {
		hlog.L().Named("scheduler.Startup").Info("scheduler disabled")
		return nil
	}
	location, err := time.LoadLocation(cfg.Scheduler.Timezone)
	if err != nil {
		return err
	}
	instance, err := New(location)
	if err != nil {
		return err
	}
	if cfg.Scheduler.DailyTaskSummary.Enabled {
		db := database.MySQL()
		if db == nil {
			return errors.New("scheduler: mysql is required by daily task summary")
		}
		var organizer service.AISummaryOrganizer
		if cfg.Voice.Enabled {
			client, clientErr := minimax.NewMiniMaxClient(cfg.Voice.MiniMax)
			if clientErr != nil {
				return clientErr
			}
			organizer = client
		}
		summaryService := service.NewAISummaryService(db, dao.NewAISummaryDao(), organizer, service.NewAISummaryCache())
		jobCfg := cfg.Scheduler.DailyTaskSummary
		if err := instance.Register(NewDailyTaskSummaryJob(jobCfg.Spec, summaryService), jobCfg.Timeout, jobCfg.RunOnStart); err != nil {
			return err
		}
	}
	defaultMu.Lock()
	defaultScheduler = instance
	defaultMu.Unlock()
	instance.Start()
	hlog.L().Named("scheduler.Startup").Info("scheduler started", hlog.String("timezone", cfg.Scheduler.Timezone))
	return nil
}

// Shutdown 优雅停止全局调度器。
func Shutdown(ctx context.Context) error {
	defaultMu.Lock()
	instance := defaultScheduler
	defaultScheduler = nil
	defaultMu.Unlock()
	if instance == nil {
		return nil
	}
	return instance.Shutdown(ctx)
}
