package ports

import (
	"context"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/domain/entities"
)

type Producer interface {
	Produce(ctx context.Context, weather entities.WeatherEntity) error
	ProduceBatch(ctx context.Context, weathers []entities.WeatherEntity) error
	HealthCheck(ctx context.Context) error
	Close() error
}

type ProducerFactory interface {
	CreateProducer(broker string, topic string, requiredAcks int16, maxRetries int) (Producer, error)
}
