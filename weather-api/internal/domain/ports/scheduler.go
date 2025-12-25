package ports

import (
	"context"
	"time"
)

type Task func(ctx context.Context) error

type Scheduler interface {
	Schedule(ctx context.Context, name string, interval time.Duration, task func(ctx context.Context) error) error
	Stop()
	HealthCheck(ctx context.Context) error
}
