package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/k-shtanenko/weather-app/weather-api/internal/cache"
	"github.com/k-shtanenko/weather-app/weather-api/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-api/internal/models"
)

type Cache interface {
	GetReportFromCache(ctx context.Context, reportType models.ReportType, cityID int, periodStart, periodEnd time.Time) (models.APICacheEntity, error)
	CacheReport(ctx context.Context, reportType models.ReportType, cityID int, periodStart, periodEnd time.Time, data []byte, contentType, fileName string, ttl time.Duration) error
	HealthCheck(ctx context.Context) error
}

type CacheService struct {
	cache          cache.Cache
	logger         logger.Logger
	maxCacheSizeMB int
}

func NewCacheService(cache cache.Cache, maxCacheSizeMB int) *CacheService {
	return &CacheService{
		cache:          cache,
		logger:         logger.New("info", "development").WithField("component", "cache_service"),
		maxCacheSizeMB: maxCacheSizeMB,
	}
}

func (s *CacheService) GetReportFromCache(ctx context.Context, reportType models.ReportType, cityID int, periodStart, periodEnd time.Time) (models.APICacheEntity, error) {
	key := generateReportKey(reportType, cityID, periodStart, periodEnd)
	return s.cache.Get(ctx, key)
}

func (s *CacheService) CacheReport(ctx context.Context, reportType models.ReportType, cityID int, periodStart, periodEnd time.Time, data []byte, contentType, fileName string, ttl time.Duration) error {
	key := generateReportKey(reportType, cityID, periodStart, periodEnd)

	cacheEntry := &models.APICache{
		ID:             uuid.New().String(),
		CacheKey:       key,
		CacheType:      "report",
		Data:           data,
		ContentType:    contentType,
		FileName:       fileName,
		ExpiresAt:      time.Now().Add(ttl),
		HitCount:       0,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	return s.cache.Set(ctx, key, cacheEntry, ttl)
}

func (s *CacheService) CleanupExpiredCache(ctx context.Context) error {
	pattern := "report:*"
	return s.cache.DeleteByPattern(ctx, pattern)
}

func (s *CacheService) HealthCheck(ctx context.Context) error {
	return s.cache.HealthCheck(ctx)
}

func generateReportKey(reportType models.ReportType, cityID int, periodStart, periodEnd time.Time) string {
	return fmt.Sprintf("report:%s:city:%d:%s:%s",
		reportType,
		cityID,
		periodStart.Format("20060102"),
		periodEnd.Format("20060102"))
}
