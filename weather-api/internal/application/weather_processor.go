package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/ports"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type WeatherProcessor struct {
	weatherRepo ports.WeatherRepository
	logger      logger.Logger
}

func NewWeatherProcessor(weatherRepo ports.WeatherRepository) *WeatherProcessor {
	return &WeatherProcessor{
		weatherRepo: weatherRepo,
		logger:      logger.New("info", "development").WithField("component", "weather_processor"),
	}
}

func (s *WeatherProcessor) ProcessWeatherData(ctx context.Context, weather entities.WeatherDataEntity) error {
	if err := weather.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := s.weatherRepo.Save(ctx, weather); err != nil {
		return fmt.Errorf("failed to save weather data: %w", err)
	}

	// Обновляем агрегат
	return s.updateDailyAggregate(ctx, weather)
}

func (s *WeatherProcessor) updateDailyAggregate(ctx context.Context, weather entities.WeatherDataEntity) error {
	date := time.Date(
		weather.GetRecordedAt().Year(),
		weather.GetRecordedAt().Month(),
		weather.GetRecordedAt().Day(),
		0, 0, 0, 0,
		weather.GetRecordedAt().Location(),
	)

	existingAgg, err := s.weatherRepo.GetDailyAggregate(ctx, weather.GetCityID(), date)
	if err != nil {
		return fmt.Errorf("failed to get daily aggregate: %w", err)
	}

	if existingAgg == nil {
		aggregate := &entities.DailyAggregate{
			ID:              uuid.New().String(),
			CityID:          weather.GetCityID(),
			Date:            date,
			AvgTemperature:  weather.GetTemperature(),
			MaxTemperature:  weather.GetTemperature(),
			MinTemperature:  weather.GetTemperature(),
			AvgHumidity:     weather.GetHumidity(),
			AvgPressure:     weather.GetPressure(),
			AvgWindSpeed:    weather.GetWindSpeed(),
			DominantWeather: weather.GetWeatherDescription(),
			TotalRecords:    1,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		return s.weatherRepo.SaveDailyAggregate(ctx, aggregate)
	}

	// Упрощенное обновление агрегата
	updatedAggregate := &entities.DailyAggregate{
		ID:              existingAgg.GetID(),
		CityID:          existingAgg.GetCityID(),
		Date:            existingAgg.GetDate(),
		AvgTemperature:  (existingAgg.GetAvgTemperature()*float64(existingAgg.GetTotalRecords()) + weather.GetTemperature()) / float64(existingAgg.GetTotalRecords()+1),
		MaxTemperature:  max(existingAgg.GetMaxTemperature(), weather.GetTemperature()),
		MinTemperature:  min(existingAgg.GetMinTemperature(), weather.GetTemperature()),
		AvgHumidity:     (existingAgg.GetAvgHumidity()*existingAgg.GetTotalRecords() + weather.GetHumidity()) / (existingAgg.GetTotalRecords() + 1),
		AvgPressure:     (existingAgg.GetAvgPressure()*existingAgg.GetTotalRecords() + weather.GetPressure()) / (existingAgg.GetTotalRecords() + 1),
		AvgWindSpeed:    (existingAgg.GetAvgWindSpeed()*float64(existingAgg.GetTotalRecords()) + weather.GetWindSpeed()) / float64(existingAgg.GetTotalRecords()+1),
		DominantWeather: existingAgg.GetDominantWeather(), // Упрощенно, оставляем предыдущее
		TotalRecords:    existingAgg.GetTotalRecords() + 1,
		CreatedAt:       existingAgg.GetCreatedAt(),
		UpdatedAt:       time.Now(),
	}

	return s.weatherRepo.UpdateDailyAggregate(ctx, updatedAggregate)
}

func (s *WeatherProcessor) HealthCheck(ctx context.Context) error {
	return s.weatherRepo.HealthCheck(ctx)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
