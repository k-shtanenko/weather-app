package ports

import (
	"context"
	"io"

	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
)

type Storage interface {
	Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, contentType string) error
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, bucket, key string) error
	Exists(ctx context.Context, bucket, key string) (bool, error)
	HealthCheck(ctx context.Context) error
}

type ReportStorage interface {
	UploadReport(ctx context.Context, report entities.ExcelReportEntity, data io.Reader) error
	DownloadReport(ctx context.Context, report entities.ExcelReportEntity) (io.ReadCloser, error)
	DeleteReport(ctx context.Context, report entities.ExcelReportEntity) error
}
