package ports

import (
	"context"
	"time"

	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
)

type WeatherRepository interface {
	Save(ctx context.Context, data entities.WeatherDataEntity) error
	FindByCityIDAndDateRange(ctx context.Context, cityID int, start, end time.Time) ([]entities.WeatherDataEntity, error)
	GetDailyAggregate(ctx context.Context, cityID int, date time.Time) (entities.DailyAggregateEntity, error)
	SaveDailyAggregate(ctx context.Context, aggregate entities.DailyAggregateEntity) error
	UpdateDailyAggregate(ctx context.Context, aggregate entities.DailyAggregateEntity) error
	GetCitiesWithData(ctx context.Context) ([]int, error)
	CleanupOldData(ctx context.Context, retentionDays int) error
	HealthCheck(ctx context.Context) error
	Close() error
}

type ReportRepository interface {
	SaveReport(ctx context.Context, report entities.ExcelReportEntity) error
	FindReportByTypeAndPeriod(ctx context.Context, reportType entities.ReportType, cityID int, start, end time.Time) (entities.ExcelReportEntity, error)
	FindReportByID(ctx context.Context, reportID string) (entities.ExcelReportEntity, error)
	CleanupExpiredReports(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}
