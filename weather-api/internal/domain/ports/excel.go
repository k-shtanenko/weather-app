package ports

import (
	"context"
	"time"

	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
)

type ExcelGenerator interface {
	GenerateWeatherReport(
		ctx context.Context,
		reportType entities.ReportType,
		cityID int,
		cityName string,
		periodStart, periodEnd time.Time,
		weatherData []entities.WeatherDataEntity,
		aggregates []entities.DailyAggregateEntity,
	) ([]byte, error)
}
