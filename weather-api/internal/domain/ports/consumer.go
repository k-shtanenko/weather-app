package ports

import (
	"context"

	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
)

type Consumer interface {
	Consume(ctx context.Context, handler func(ctx context.Context, message entities.WeatherDataEntity) error) error
	Close() error
	HealthCheck(ctx context.Context) error
}

type MessageHandler func(ctx context.Context, message entities.WeatherDataEntity) error
