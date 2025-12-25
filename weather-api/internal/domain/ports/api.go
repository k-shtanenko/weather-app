package ports

import (
	"context"
	"io"
	"time"

	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
)

type ReportService interface {
	GenerateDailyReport(ctx context.Context, cityID int, date time.Time) (entities.ExcelReportEntity, error)
	GenerateWeeklyReport(ctx context.Context, cityID int, year int, week int) (entities.ExcelReportEntity, error)
	GenerateMonthlyReport(ctx context.Context, cityID int, year int, month time.Month) (entities.ExcelReportEntity, error)
	GetReport(ctx context.Context, reportID string) (entities.ExcelReportEntity, error)
	DownloadReport(ctx context.Context, reportID string) (io.ReadCloser, string, error)
	HealthCheck(ctx context.Context) error
}

type CacheService interface {
	GetReportFromCache(ctx context.Context, reportType entities.ReportType, cityID int, periodStart, periodEnd time.Time) (entities.APICacheEntity, error)
	CacheReport(ctx context.Context, reportType entities.ReportType, cityID int, periodStart, periodEnd time.Time, data []byte, contentType, fileName string, ttl time.Duration) error
	HealthCheck(ctx context.Context) error
}

type APIServer interface {
	Start() error
	Stop(ctx context.Context) error
}
