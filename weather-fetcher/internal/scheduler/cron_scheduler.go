package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/logger"
	"github.com/robfig/cron/v3"
)

type Task func(ctx context.Context) error

type Scheduler interface {
	Schedule(ctx context.Context, interval time.Duration, task Task) error
	Stop()
}

type SchedulerFactory interface {
	CreateScheduler(timeout time.Duration) Scheduler
}

type CronScheduler struct {
	cron    *cron.Cron
	timeout time.Duration
	logger  logger.Logger
	entries map[cron.EntryID]context.CancelFunc
}

func NewCronScheduler(timeout time.Duration) Scheduler {
	return &CronScheduler{
		cron:    cron.New(cron.WithSeconds()),
		timeout: timeout,
		logger:  logger.New("info", "development").WithField("component", "cron_scheduler"),
		entries: make(map[cron.EntryID]context.CancelFunc),
	}
}

func (s *CronScheduler) Schedule(ctx context.Context, interval time.Duration, task Task) error {
	s.logger.Infof("Scheduling task with interval: %v", interval)

	cronExpr := intervalToCron(interval)
	s.logger.Debugf("Converted interval %v to cron expression: %s", interval, cronExpr)

	entryID, _ := s.cron.AddFunc(cronExpr, s.wrapTask(ctx, task))

	_, cancel := context.WithCancel(ctx)
	s.entries[entryID] = cancel

	s.logger.Infof("Task scheduled with entry ID: %d", entryID)

	if len(s.cron.Entries()) == 1 {
		s.cron.Start()
		s.logger.Info("Cron scheduler started")
	}

	return nil
}

func (s *CronScheduler) wrapTask(ctx context.Context, task Task) func() {
	return func() {
		startTime := time.Now()
		s.logger.Debug("Starting scheduled task")

		taskCtx, cancel := context.WithTimeout(ctx, s.timeout)
		defer cancel()

		// Выполняем задачу
		if err := task(taskCtx); err != nil {
			s.logger.Errorf("Task failed: %v", err)
			return
		}

		duration := time.Since(startTime)
		s.logger.Debugf("Task completed successfully in %v", duration)
	}
}

func (s *CronScheduler) Stop() {
	s.logger.Info("Stopping cron scheduler")

	for entryID, cancel := range s.entries {
		s.logger.Debugf("Cancelling task with entry ID: %d", entryID)
		cancel()
	}

	ctx := s.cron.Stop()
	<-ctx.Done()

	s.logger.Info("Cron scheduler stopped")
}

func intervalToCron(interval time.Duration) string {
	if interval == 0 {
		return "0 */5 * * * *" // Каждые 5 минут
	}

	seconds := int(interval.Seconds())

	if seconds < 10 {
		seconds = 10
	}

	return fmt.Sprintf("*/%d * * * * *", seconds)
}

type CronSchedulerFactory struct {
	logger logger.Logger
}

func NewCronSchedulerFactory() SchedulerFactory {
	return &CronSchedulerFactory{
		logger: logger.New("info", "development").WithField("component", "cron_scheduler_factory"),
	}
}

func (f *CronSchedulerFactory) CreateScheduler(timeout time.Duration) Scheduler {
	f.logger.Infof("Creating CronScheduler with timeout: %v", timeout)
	return NewCronScheduler(timeout)
}
