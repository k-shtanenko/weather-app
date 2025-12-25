package application

import (
	"context"
	"fmt"
	"time"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/domain/ports"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/pkg/logger"
)

type Service interface {
	Start(ctx context.Context, interval time.Duration) error
	Stop()
	HealthCheck(ctx context.Context) error
	GetStats() map[string]interface{}
	fetchAndSendWeather(ctx context.Context) error
}

type ServiceFactory interface {
	Create(
		fetcher ports.Fetcher,
		producer ports.Producer,
		scheduler ports.Scheduler,
		cityIDs []string,
	) Service
}

type WeatherService struct {
	fetcher   ports.Fetcher
	producer  ports.Producer
	scheduler ports.Scheduler
	logger    logger.Logger
	cityIDs   []string
}

func NewWeatherService(
	fetcher ports.Fetcher,
	producer ports.Producer,
	scheduler ports.Scheduler,
	cityIDs []string,
) Service {
	return &WeatherService{
		fetcher:   fetcher,
		producer:  producer,
		scheduler: scheduler,
		logger:    logger.New("info", "development").WithField("component", "weather_service"),
		cityIDs:   cityIDs,
	}
}

func (s *WeatherService) Start(ctx context.Context, interval time.Duration) error {
	s.logger.Infof("Starting weather service for %d cities with interval: %v", len(s.cityIDs), interval)

	if err := s.scheduler.Schedule(ctx, interval, s.fetchAndSendWeather); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	s.logger.Info("Weather service started successfully")
	return nil
}

func (s *WeatherService) Stop() {
	s.logger.Info("Stopping weather service")
	s.scheduler.Stop()

	if err := s.producer.Close(); err != nil {
		s.logger.Errorf("Failed to close producer: %v", err)
	}

	s.logger.Info("Weather service stopped")
}

func (s *WeatherService) fetchAndSendWeather(ctx context.Context) error {
	startTime := time.Now()
	s.logger.Info("Starting weather data fetch and send cycle")

	weathers, err := s.fetcher.FetchBatch(ctx, s.cityIDs)
	if err != nil {
		s.logger.Errorf("Failed to fetch weather data: %v", err)
		return fmt.Errorf("fetch weather data: %w", err)
	}

	s.logger.Infof("Successfully fetched weather data for %d cities", len(weathers))

	if err := s.producer.ProduceBatch(ctx, weathers); err != nil {
		s.logger.Errorf("Failed to send weather data to Kafka: %v", err)
		return fmt.Errorf("send to Kafka: %w", err)
	}

	duration := time.Since(startTime)
	s.logger.Infof("Weather data cycle completed successfully in %v. Processed %d records.", duration, len(weathers))

	return nil
}

func (s *WeatherService) HealthCheck(ctx context.Context) error {
	s.logger.Debug("Performing health check")

	if err := s.fetcher.HealthCheck(ctx); err != nil {
		return fmt.Errorf("fetcher health check failed: %w", err)
	}
	s.logger.Debug("Fetcher health check passed")

	if err := s.producer.HealthCheck(ctx); err != nil {
		return fmt.Errorf("producer health check failed: %w", err)
	}
	s.logger.Debug("Producer health check passed")

	s.logger.Info("All health checks passed")
	return nil
}

func (s *WeatherService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"cities_count": len(s.cityIDs),
		"status":       "running",
		"timestamp":    time.Now().Format(time.RFC3339),
	}
}

type WeatherServiceFactory struct{}

func NewWeatherServiceFactory() *WeatherServiceFactory {
	return &WeatherServiceFactory{}
}

func (f *WeatherServiceFactory) Create(
	fetcher ports.Fetcher,
	producer ports.Producer,
	scheduler ports.Scheduler,
	cityIDs []string,
) Service {
	return NewWeatherService(fetcher, producer, scheduler, cityIDs)
}
