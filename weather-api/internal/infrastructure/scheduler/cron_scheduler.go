package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type CronScheduler struct {
	cron   *cron.Cron
	jobs   map[string]cron.EntryID
	mu     sync.RWMutex
	logger logger.Logger
}

func NewCronScheduler() *CronScheduler {
	c := cron.New(cron.WithSeconds())

	scheduler := &CronScheduler{
		cron:   c,
		jobs:   make(map[string]cron.EntryID),
		logger: logger.New("info", "development").WithField("component", "cron_scheduler"),
	}

	c.Start()
	scheduler.logger.Info("Cron scheduler started")

	return scheduler
}

func (c *CronScheduler) Schedule(ctx context.Context, name string, interval time.Duration, task func(ctx context.Context) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.jobs[name]; exists {
		return fmt.Errorf("job with name '%s' already exists", name)
	}

	cronExpr := intervalToCron(interval)
	c.logger.Infof("Scheduling job '%s' with interval %v (cron: %s)", name, interval, cronExpr)

	entryID, err := c.cron.AddFunc(cronExpr, func() {
		c.runTask(name, task)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule job '%s': %w", name, err)
	}

	c.jobs[name] = entryID
	c.logger.Infof("Job '%s' scheduled with entry ID: %d", name, entryID)
	return nil
}

func (c *CronScheduler) runTask(name string, task func(ctx context.Context) error) {
	startTime := time.Now()
	c.logger.Infof("Starting scheduled job: %s", name)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := task(ctx); err != nil {
		c.logger.Errorf("Job '%s' failed after %v: %v", name, time.Since(startTime), err)
		return
	}

	duration := time.Since(startTime)
	c.logger.Infof("Job '%s' completed successfully in %v", name, duration)
}

func (c *CronScheduler) Stop() {
	c.logger.Info("Stopping cron scheduler...")
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := c.cron.Stop()
	<-ctx.Done()

	c.jobs = make(map[string]cron.EntryID)
	c.logger.Info("Cron scheduler stopped")
}

func (c *CronScheduler) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.cron.Entries()) == 0 && len(c.jobs) > 0 {
		return fmt.Errorf("cron has no entries but jobs are registered")
	}

	for name, entryID := range c.jobs {
		if entry := c.cron.Entry(entryID); entry.ID != entryID {
			return fmt.Errorf("job '%s' not found in cron", name)
		}
	}

	return nil
}

func intervalToCron(interval time.Duration) string {
	if interval <= 0 {
		return "@every 1m"
	}

	seconds := int(interval.Seconds())

	if seconds < 60 {
		if seconds < 10 {
			seconds = 10
		}
		return fmt.Sprintf("*/%d * * * * *", seconds)
	}

	minutes := int(interval.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("0 */%d * * * *", minutes)
}
