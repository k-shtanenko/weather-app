package ports

import (
	"context"

	"github.com/weather-app/weather-fetcher/internal/domain/entities"
)

type Fetcher interface {
	Fetch(ctx context.Context, cityID string) (entities.WeatherEntity, error)
	FetchBatch(ctx context.Context, cityIDs []string) ([]entities.WeatherEntity, error)
	HealthCheck(ctx context.Context) error
}

type FetcherFactory interface {
	CreateFetcher(baseURL, apiKey, units, lang string) Fetcher
}
