package ports

import (
	"context"
	"time"

	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
)

type Cache interface {
	Get(ctx context.Context, key string) (entities.APICacheEntity, error)
	Set(ctx context.Context, key string, data entities.APICacheEntity, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPattern(ctx context.Context, pattern string) error
	HealthCheck(ctx context.Context) error
	Close() error
}
