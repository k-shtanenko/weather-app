package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/k-shtanenko/weather-app/weather-api/internal/database"
	"github.com/k-shtanenko/weather-app/weather-api/internal/excel"
	"github.com/k-shtanenko/weather-app/weather-api/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-api/internal/models"
	"github.com/k-shtanenko/weather-app/weather-api/internal/storage"
)

type Report interface {
	GenerateDailyReport(ctx context.Context, cityID int, date time.Time) (models.ExcelReportEntity, error)
	GenerateWeeklyReport(ctx context.Context, cityID int, year int, week int) (models.ExcelReportEntity, error)
	GenerateMonthlyReport(ctx context.Context, cityID int, year int, month time.Month) (models.ExcelReportEntity, error)
	GetReport(ctx context.Context, reportID string) (models.ExcelReportEntity, error)
	DownloadReport(ctx context.Context, reportID string) (io.ReadCloser, string, error)
	HealthCheck(ctx context.Context) error
}

type ReportService struct {
	weatherRepo database.WeatherRepository
	reportRepo  database.ReportRepository
	excelGen    excel.ExcelGenerator
	storage     storage.ReportStorage
	cache       *CacheService
	logger      logger.Logger
}

func NewReportService(
	weatherRepo database.WeatherRepository,
	reportRepo database.ReportRepository,
	excelGen excel.ExcelGenerator,
	storage storage.ReportStorage,
	cache *CacheService,
) *ReportService {
	return &ReportService{
		weatherRepo: weatherRepo,
		reportRepo:  reportRepo,
		excelGen:    excelGen,
		storage:     storage,
		cache:       cache,
		logger:      logger.New("info", "development").WithField("component", "report_service"),
	}
}

func (s *ReportService) GenerateDailyReport(ctx context.Context, cityID int, date time.Time) (models.ExcelReportEntity, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Second)
	return s.generateReport(ctx, models.ReportTypeDaily, cityID, startOfDay, endOfDay)
}

func (s *ReportService) GenerateWeeklyReport(ctx context.Context, cityID int, year int, week int) (models.ExcelReportEntity, error) {
	periodStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	periodStart = periodStart.AddDate(0, 0, (week-1)*7)
	periodEnd := periodStart.AddDate(0, 0, 6)
	return s.generateReport(ctx, models.ReportTypeWeekly, cityID, periodStart, periodEnd)
}

func (s *ReportService) GenerateMonthlyReport(ctx context.Context, cityID int, year int, month time.Month) (models.ExcelReportEntity, error) {
	periodStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, -1)
	return s.generateReport(ctx, models.ReportTypeMonthly, cityID, periodStart, periodEnd)
}

func (s *ReportService) generateReport(ctx context.Context, reportType models.ReportType, cityID int, periodStart, periodEnd time.Time) (models.ExcelReportEntity, error) {
	if cacheEntry, err := s.cache.GetReportFromCache(ctx, reportType, cityID, periodStart, periodEnd); err == nil && cacheEntry != nil {
		expiresAt := cacheEntry.GetExpiresAt()
		return &models.ExcelReport{
			ID:          cacheEntry.GetID(),
			ReportType:  reportType,
			CityID:      cityID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			FileName:    cacheEntry.GetFileName(),
			FileSize:    int64(len(cacheEntry.GetData())),
			GeneratedAt: cacheEntry.GetCreatedAt(),
			ExpiresAt:   &expiresAt,
		}, nil
	}

	existingReport, err := s.reportRepo.FindReportByTypeAndPeriod(ctx, reportType, cityID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing report: %w", err)
	}
	if existingReport != nil {
		if existingReport.GetExpiresAt() == nil || time.Now().Before(*existingReport.GetExpiresAt()) {
			return existingReport, nil
		}
	}

	weatherData, err := s.weatherRepo.FindByCityIDAndDateRange(ctx, cityID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather data: %w", err)
	}

	cityName := ""
	if len(weatherData) > 0 {
		cityName = weatherData[0].GetCityName()
	}

	excelData, err := s.excelGen.GenerateWeatherReport(ctx, reportType, cityID, cityName, periodStart, periodEnd, weatherData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate excel: %w", err)
	}

	return s.saveReport(ctx, reportType, cityID, periodStart, periodEnd, excelData)
}

func (s *ReportService) saveReport(ctx context.Context, reportType models.ReportType, cityID int, periodStart, periodEnd time.Time, excelData []byte) (models.ExcelReportEntity, error) {
	reportID := uuid.New().String()
	fileName := fmt.Sprintf("%s_%d_%s_%s.xlsx",
		reportType,
		cityID,
		periodStart.Format("20060102"),
		periodEnd.Format("20060102"),
	)

	report := &models.ExcelReport{
		ID:          reportID,
		ReportType:  reportType,
		CityID:      cityID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		FileName:    fileName,
		FileSize:    int64(len(excelData)),
		StoragePath: fmt.Sprintf("%s/%s", reportType, fileName),
		GeneratedAt: time.Now(),
		DownloadURL: fmt.Sprintf("/api/v1/reports/%s/download", reportID),
	}

	reader := bytes.NewReader(excelData)
	if err := s.storage.UploadReport(ctx, report, reader); err != nil {
		return nil, fmt.Errorf("failed to upload report to storage: %w", err)
	}

	if err := s.reportRepo.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report metadata: %w", err)
	}

	ttl := 24 * time.Hour
	switch reportType {
	case models.ReportTypeWeekly:
		ttl = 7 * 24 * time.Hour
	case models.ReportTypeMonthly:
		ttl = 30 * 24 * time.Hour
	}

	if err := s.cache.CacheReport(ctx, reportType, cityID, periodStart, periodEnd, excelData,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileName, ttl); err != nil {
		s.logger.Warnf("Failed to cache report: %v", err)
	}

	return report, nil
}

func (s *ReportService) GetReport(ctx context.Context, reportID string) (models.ExcelReportEntity, error) {
	return s.reportRepo.FindReportByID(ctx, reportID)
}

func (s *ReportService) DownloadReport(ctx context.Context, reportID string) (io.ReadCloser, string, error) {
	report, err := s.GetReport(ctx, reportID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get report: %w", err)
	}
	if report == nil {
		return nil, "", fmt.Errorf("report %s not found", reportID)
	}

	reader, err := s.storage.DownloadReport(ctx, report)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download report: %w", err)
	}

	return reader, report.GetFileName(), nil
}

func (s *ReportService) CleanupExpiredReports(ctx context.Context) error {
	return s.reportRepo.CleanupExpiredReports(ctx)
}

func (s *ReportService) HealthCheck(ctx context.Context) error {
	if err := s.weatherRepo.HealthCheck(ctx); err != nil {
		return fmt.Errorf("weather repository health check failed: %w", err)
	}
	if err := s.reportRepo.HealthCheck(ctx); err != nil {
		return fmt.Errorf("report repository health check failed: %w", err)
	}
	return nil
}
