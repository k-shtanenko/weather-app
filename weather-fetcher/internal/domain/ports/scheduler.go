package ports

import (
	"context"
	"time"
)

type Task func(ctx context.Context) error

type Scheduler interface {
	Schedule(ctx context.Context, interval time.Duration, task Task) error
	Stop()
}

type SchedulerFactory interface {
	CreateScheduler(timeout time.Duration) Scheduler
}
