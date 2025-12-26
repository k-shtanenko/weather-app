package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/models"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWeatherService_Start(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		interval := 5 * time.Minute
		cityIDs := []string{"123", "456"}

		mockScheduler.On("Schedule", mock.Anything, interval, mock.Anything).Return(nil)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.Start(ctx, interval)

		assert.NoError(t, err)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("scheduler error", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		interval := 5 * time.Minute
		cityIDs := []string{"123"}

		expectedErr := errors.New("scheduler error")
		mockScheduler.On("Schedule", mock.Anything, interval, mock.Anything).Return(expectedErr)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.Start(ctx, interval)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start scheduler")
		mockScheduler.AssertExpectations(t)
	})
}

func TestWeatherService_Stop(t *testing.T) {
	mockScheduler := &testutils.MockScheduler{}
	mockFetcher := &testutils.MockFetcher{}
	mockProducer := &testutils.MockProducer{}

	cityIDs := []string{"123"}

	mockScheduler.On("Stop").Once()
	mockProducer.On("Close").Return(nil)

	service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

	service.Stop()

	mockScheduler.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

func TestWeatherService_HealthCheck(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123"}

		mockFetcher.On("HealthCheck", mock.Anything).Return(nil)
		mockProducer.On("HealthCheck", mock.Anything).Return(nil)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.HealthCheck(ctx)

		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("fetcher health check fails", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123"}

		expectedErr := errors.New("fetcher error")
		mockFetcher.On("HealthCheck", mock.Anything).Return(expectedErr)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetcher health check failed")
		mockFetcher.AssertExpectations(t)
	})

	t.Run("producer health check fails", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123"}

		mockFetcher.On("HealthCheck", mock.Anything).Return(nil)
		expectedErr := errors.New("producer error")
		mockProducer.On("HealthCheck", mock.Anything).Return(expectedErr)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "producer health check failed")
		mockFetcher.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})
}

func TestWeatherService_FetchAndSendWeather(t *testing.T) {
	t.Run("successful fetch and send", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123", "456"}
		weathers := []models.WeatherEntity{
			&models.Weather{CityID: 123, CityName: "City1", Timestamp: time.Now(), Source: "test"},
			&models.Weather{CityID: 456, CityName: "City2", Timestamp: time.Now(), Source: "test"},
		}

		mockFetcher.On("FetchBatch", mock.Anything, cityIDs).Return(weathers, nil)
		mockProducer.On("ProduceBatch", mock.Anything, weathers).Return(nil)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.fetchAndSendWeather(ctx)

		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("fetch batch error", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123"}
		expectedErr := errors.New("fetch error")

		mockFetcher.On("FetchBatch", mock.Anything, cityIDs).Return(nil, expectedErr)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.fetchAndSendWeather(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fetch weather data")
		mockFetcher.AssertExpectations(t)
	})

	t.Run("produce batch error", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123"}
		weathers := []models.WeatherEntity{
			&models.Weather{CityID: 123, CityName: "City1", Timestamp: time.Now(), Source: "test"},
		}
		expectedErr := errors.New("produce error")

		mockFetcher.On("FetchBatch", mock.Anything, cityIDs).Return(weathers, nil)
		mockProducer.On("ProduceBatch", mock.Anything, weathers).Return(expectedErr)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.fetchAndSendWeather(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "send to Kafka")
		mockFetcher.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})
}

func TestWeatherServiceFactory(t *testing.T) {
	factory := NewWeatherServiceFactory()

	mockScheduler := &testutils.MockScheduler{}
	mockFetcher := &testutils.MockFetcher{}
	mockProducer := &testutils.MockProducer{}
	cityIDs := []string{"123"}

	service := factory.Create(mockFetcher, mockProducer, mockScheduler, cityIDs)

	assert.NotNil(t, service)
	assert.IsType(t, &WeatherService{}, service)
}

func TestWeatherService_Stop_ProducerCloseError(t *testing.T) {
	mockScheduler := &testutils.MockScheduler{}
	mockFetcher := &testutils.MockFetcher{}
	mockProducer := &testutils.MockProducer{}

	cityIDs := []string{"123"}

	mockScheduler.On("Stop").Once()
	mockProducer.On("Close").Return(assert.AnError)

	service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

	service.Stop()

	mockScheduler.AssertExpectations(t)
	mockProducer.AssertExpectations(t)
}

func TestWeatherService_HealthCheck_ContextCancelled(t *testing.T) {
	mockScheduler := &testutils.MockScheduler{}
	mockFetcher := &testutils.MockFetcher{}
	mockProducer := &testutils.MockProducer{}

	cityIDs := []string{"123"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockFetcher.On("HealthCheck", mock.Anything).Return(ctx.Err())
	mockProducer.On("HealthCheck", mock.Anything).Return(ctx.Err())

	service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

	err := service.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetcher health check failed")
	mockFetcher.AssertExpectations(t)
	mockProducer.AssertNotCalled(t, "HealthCheck", mock.Anything)
}

func TestNewWeatherService_Constructor(t *testing.T) {
	mockScheduler := &testutils.MockScheduler{}
	mockFetcher := &testutils.MockFetcher{}
	mockProducer := &testutils.MockProducer{}

	cityIDs := []string{"123", "456"}

	service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

	assert.NotNil(t, service)
	assert.IsType(t, &WeatherService{}, service)

	ws := service.(*WeatherService)
	assert.Equal(t, cityIDs, ws.cityIDs)
	assert.NotNil(t, ws.logger)
}

func TestWeatherService_InterfaceImplementation(t *testing.T) {
	var _ Service = (*WeatherService)(nil)
	var _ ServiceFactory = (*WeatherServiceFactory)(nil)
}

func TestWeatherServiceFactory_Create(t *testing.T) {
	factory := &WeatherServiceFactory{}

	mockScheduler := &testutils.MockScheduler{}
	mockFetcher := &testutils.MockFetcher{}
	mockProducer := &testutils.MockProducer{}
	cityIDs := []string{"123"}

	service := factory.Create(mockFetcher, mockProducer, mockScheduler, cityIDs)

	assert.NotNil(t, service)
	assert.IsType(t, &WeatherService{}, service)
}

func TestWeatherService_FetchAndSendWeather_EdgeCases(t *testing.T) {
	t.Run("empty city IDs", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{}

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		weathers := []models.WeatherEntity{}
		mockFetcher.On("FetchBatch", mock.Anything, cityIDs).Return(weathers, nil)
		mockProducer.On("ProduceBatch", mock.Anything, weathers).Return(nil)

		err := service.fetchAndSendWeather(ctx)

		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})

	t.Run("partial success with some failures", func(t *testing.T) {
		ctx := context.Background()
		mockScheduler := &testutils.MockScheduler{}
		mockFetcher := &testutils.MockFetcher{}
		mockProducer := &testutils.MockProducer{}

		cityIDs := []string{"123", "456", "789"}
		weathers := []models.WeatherEntity{
			&models.Weather{
				CityID:    123,
				CityName:  "City1",
				Timestamp: time.Now(),
				Source:    "test",
				Pressure:  1013,
			},
			&models.Weather{
				CityID:    789,
				CityName:  "City3",
				Timestamp: time.Now(),
				Source:    "test",
				Pressure:  1013,
			},
		}

		mockFetcher.On("FetchBatch", mock.Anything, cityIDs).Return(weathers, nil)
		mockProducer.On("ProduceBatch", mock.Anything, weathers).Return(nil)

		service := NewWeatherService(mockFetcher, mockProducer, mockScheduler, cityIDs)

		err := service.fetchAndSendWeather(ctx)

		assert.NoError(t, err)
		mockFetcher.AssertExpectations(t)
		mockProducer.AssertExpectations(t)
	})
}
