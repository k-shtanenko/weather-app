package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type MinioStorage struct {
	client *minio.Client
	logger logger.Logger
}

type MinioReportStorage struct {
	storage *MinioStorage
	bucket  string
	logger  logger.Logger
}

func NewMinioStorage(endpoint, accessKey, secretKey string, useSSL bool) (*MinioStorage, error) {
	log := logger.New("info", "development").WithField("component", "minio_storage")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Minio buckets: %w", err)
	}

	log.Info("Minio storage initialized successfully")
	return &MinioStorage{
		client: client,
		logger: log,
	}, nil
}

func (m *MinioStorage) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, contentType string) error {
	exists, err := m.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		m.logger.Infof("Created bucket: %s", bucket)
	}

	_, err = m.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	m.logger.Debugf("Uploaded file to bucket: %s, key: %s", bucket, key)
	return nil
}

func (m *MinioStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	object, err := m.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	_, err = object.Stat()
	if err != nil {
		object.Close()
		return nil, fmt.Errorf("object not found: %w", err)
	}

	return object, nil
}

func (m *MinioStorage) Delete(ctx context.Context, bucket, key string) error {
	if err := m.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (m *MinioStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := m.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object: %w", err)
	}
	return true, nil
}

func (m *MinioStorage) HealthCheck(ctx context.Context) error {
	_, err := m.client.ListBuckets(ctx)
	return err
}

func NewMinioReportStorage(storage *MinioStorage, bucket string) *MinioReportStorage {
	return &MinioReportStorage{
		storage: storage,
		bucket:  bucket,
		logger:  logger.New("info", "development").WithField("component", "minio_report_storage"),
	}
}

func (m *MinioReportStorage) UploadReport(ctx context.Context, report entities.ExcelReportEntity, data io.Reader) error {
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	key := m.getStorageKey(report)

	if err := m.storage.Upload(ctx, m.bucket, key, data, report.GetFileSize(), contentType); err != nil {
		return fmt.Errorf("failed to upload report: %w", err)
	}

	m.logger.Infof("Uploaded report %s to Minio, key: %s", report.GetID(), key)
	return nil
}

func (m *MinioReportStorage) DownloadReport(ctx context.Context, report entities.ExcelReportEntity) (io.ReadCloser, error) {
	key := m.getStorageKey(report)
	return m.storage.Download(ctx, m.bucket, key)
}

func (m *MinioReportStorage) DeleteReport(ctx context.Context, report entities.ExcelReportEntity) error {
	key := m.getStorageKey(report)
	return m.storage.Delete(ctx, m.bucket, key)
}

func (m *MinioReportStorage) getStorageKey(report entities.ExcelReportEntity) string {
	return fmt.Sprintf("%s/%d/%s",
		report.GetReportType(),
		report.GetCityID(),
		report.GetFileName(),
	)
}
