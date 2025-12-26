package testutils

import (
	"context"
	"time"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/models"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/scheduler"
	"github.com/stretchr/testify/mock"
)

type MockFetcher struct {
	mock.Mock
}

func (m *MockFetcher) Fetch(ctx context.Context, cityID string) (models.WeatherEntity, error) {
	args := m.Called(ctx, cityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(models.WeatherEntity), args.Error(1)
}

func (m *MockFetcher) FetchBatch(ctx context.Context, cityIDs []string) ([]models.WeatherEntity, error) {
	args := m.Called(ctx, cityIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.WeatherEntity), args.Error(1)
}

func (m *MockFetcher) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Produce(ctx context.Context, weather models.WeatherEntity) error {
	args := m.Called(ctx, weather)
	return args.Error(0)
}

func (m *MockProducer) ProduceBatch(ctx context.Context, weathers []models.WeatherEntity) error {
	args := m.Called(ctx, weathers)
	return args.Error(0)
}

func (m *MockProducer) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Schedule(ctx context.Context, interval time.Duration, task scheduler.Task) error {
	args := m.Called(ctx, interval, task)
	return args.Error(0)
}

func (m *MockScheduler) Stop() {
	m.Called()
}
