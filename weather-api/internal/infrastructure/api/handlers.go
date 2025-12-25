package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/entities"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/ports"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type APIHandler struct {
	reportService ports.ReportService
	cacheService  ports.CacheService
	logger        logger.Logger
}

func NewAPIHandler(reportService ports.ReportService, cacheService ports.CacheService) *APIHandler {
	return &APIHandler{
		reportService: reportService,
		cacheService:  cacheService,
		logger:        logger.New("info", "development").WithField("component", "api_handler"),
	}
}

// GetDailyReport godoc
// @Summary Generate daily weather report
// @Description Генерирует дневной отчет по городу
// @Tags reports
// @Accept json
// @Produce json
// @Param city_id query int true "ID города"
// @Param date query string true "Дата в формате YYYY-MM-DD"
// @Success 200 {object} ReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/daily [get]
func (h *APIHandler) GetDailyReport(c *gin.Context) {
	ctx := c.Request.Context()

	cityID, _ := strconv.Atoi(c.Query("city_id"))
	if cityID <= 0 {
		h.respondError(c, http.StatusBadRequest, "valid city_id parameter is required")
		return
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		h.respondError(c, http.StatusBadRequest, "date parameter is required")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		return
	}

	periodStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	periodEnd := periodStart.Add(24 * time.Hour).Add(-time.Second)

	cacheEntry, err := h.cacheService.GetReportFromCache(ctx, entities.ReportTypeDaily, cityID, periodStart, periodEnd)
	if err != nil {
		h.logger.Errorf("Cache error: %v", err)
	}

	if cacheEntry != nil {
		h.logger.Debugf("Serving from cache: %s", cacheEntry.GetCacheKey())
		c.Data(http.StatusOK, cacheEntry.GetContentType(), cacheEntry.GetData())
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", cacheEntry.GetFileName()))
		return
	}

	report, err := h.reportService.GenerateDailyReport(ctx, cityID, date)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	reader, fileName, err := h.reportService.DownloadReport(ctx, report.GetID())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to download report: %v", err))
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to read report data: %v", err))
		return
	}

	if err := h.cacheService.CacheReport(ctx, entities.ReportTypeDaily, cityID, periodStart, periodEnd, data,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileName, 24*time.Hour); err != nil {
		h.logger.Warnf("Failed to cache report: %v", err)
	}

	c.JSON(http.StatusOK, ReportResponse{
		ID:          report.GetID(),
		ReportType:  string(report.GetReportType()),
		CityID:      report.GetCityID(),
		PeriodStart: report.GetPeriodStart(),
		PeriodEnd:   report.GetPeriodEnd(),
		FileName:    report.GetFileName(),
		FileSize:    report.GetFileSize(),
		DownloadURL: report.GetDownloadURL(),
		GeneratedAt: report.GetGeneratedAt(),
		ExpiresAt:   report.GetExpiresAt(),
	})
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
}

// GetWeeklyReport godoc
// @Summary Generate weekly weather report
// @Description Генерирует недельный отчет по городу
// @Tags reports
// @Accept json
// @Produce json
// @Param city_id query int true "City ID"
// @Param year query int true "Год"
// @Param week query int true "Номер недели (1-52)"
// @Success 200 {object} ReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/weekly [get]
func (h *APIHandler) GetWeeklyReport(c *gin.Context) {
	ctx := c.Request.Context()

	cityID, _ := strconv.Atoi(c.Query("city_id"))
	if cityID <= 0 {
		h.respondError(c, http.StatusBadRequest, "valid city_id parameter is required")
		return
	}

	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2000 || year > 2100 {
		h.respondError(c, http.StatusBadRequest, "valid year parameter is required (2000-2100)")
		return
	}

	week, err := strconv.Atoi(c.Query("week"))
	if err != nil || week < 1 || week > 53 {
		h.respondError(c, http.StatusBadRequest, "valid week parameter is required (1-53)")
		return
	}

	periodStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	periodStart = periodStart.AddDate(0, 0, (week-1)*7)
	periodEnd := periodStart.AddDate(0, 0, 6)

	cacheEntry, err := h.cacheService.GetReportFromCache(ctx, entities.ReportTypeWeekly, cityID, periodStart, periodEnd)
	if err != nil {
		h.logger.Errorf("Cache error: %v", err)
	}

	if cacheEntry != nil {
		h.logger.Debugf("Serving from cache: %s", cacheEntry.GetCacheKey())
		c.Data(http.StatusOK, cacheEntry.GetContentType(), cacheEntry.GetData())
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", cacheEntry.GetFileName()))
		return
	}

	report, err := h.reportService.GenerateWeeklyReport(ctx, cityID, year, week)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	reader, fileName, err := h.reportService.DownloadReport(ctx, report.GetID())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to download report: %v", err))
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to read report data: %v", err))
		return
	}

	if err := h.cacheService.CacheReport(ctx, entities.ReportTypeWeekly, cityID, periodStart, periodEnd, data,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileName, 7*24*time.Hour); err != nil {
		h.logger.Warnf("Failed to cache report: %v", err)
	}

	c.JSON(http.StatusOK, ReportResponse{
		ID:          report.GetID(),
		ReportType:  string(report.GetReportType()),
		CityID:      report.GetCityID(),
		PeriodStart: report.GetPeriodStart(),
		PeriodEnd:   report.GetPeriodEnd(),
		FileName:    report.GetFileName(),
		FileSize:    report.GetFileSize(),
		DownloadURL: report.GetDownloadURL(),
		GeneratedAt: report.GetGeneratedAt(),
		ExpiresAt:   report.GetExpiresAt(),
	})
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
}

// GetMonthlyReport godoc
// @Summary Generate monthly weather report
// @Description Генерирует месячный отчет по городу
// @Tags reports
// @Accept json
// @Produce json
// @Param city_id query int true "City ID"
// @Param year query int true "Год"
// @Param month query int true "Месяц (1-12)"
// @Success 200 {object} ReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/monthly [get]
func (h *APIHandler) GetMonthlyReport(c *gin.Context) {
	ctx := c.Request.Context()

	cityID, _ := strconv.Atoi(c.Query("city_id"))
	if cityID <= 0 {
		h.respondError(c, http.StatusBadRequest, "valid city_id parameter is required")
		return
	}

	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 2000 || year > 2100 {
		h.respondError(c, http.StatusBadRequest, "valid year parameter is required (2000-2100)")
		return
	}

	month, err := strconv.Atoi(c.Query("month"))
	if err != nil || month < 1 || month > 12 {
		h.respondError(c, http.StatusBadRequest, "valid month parameter is required (1-12)")
		return
	}

	periodStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, -1)

	cacheEntry, err := h.cacheService.GetReportFromCache(ctx, entities.ReportTypeMonthly, cityID, periodStart, periodEnd)
	if err != nil {
		h.logger.Errorf("Cache error: %v", err)
	}

	if cacheEntry != nil {
		h.logger.Debugf("Serving from cache: %s", cacheEntry.GetCacheKey())
		c.Data(http.StatusOK, cacheEntry.GetContentType(), cacheEntry.GetData())
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", cacheEntry.GetFileName()))
		return
	}

	report, err := h.reportService.GenerateMonthlyReport(ctx, cityID, year, time.Month(month))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	reader, fileName, err := h.reportService.DownloadReport(ctx, report.GetID())
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to download report: %v", err))
		return
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to read report data: %v", err))
		return
	}

	if err := h.cacheService.CacheReport(ctx, entities.ReportTypeMonthly, cityID, periodStart, periodEnd, data,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileName, 30*24*time.Hour); err != nil {
		h.logger.Warnf("Failed to cache report: %v", err)
	}

	c.JSON(http.StatusOK, ReportResponse{
		ID:          report.GetID(),
		ReportType:  string(report.GetReportType()),
		CityID:      report.GetCityID(),
		PeriodStart: report.GetPeriodStart(),
		PeriodEnd:   report.GetPeriodEnd(),
		FileName:    report.GetFileName(),
		FileSize:    report.GetFileSize(),
		DownloadURL: report.GetDownloadURL(),
		GeneratedAt: report.GetGeneratedAt(),
		ExpiresAt:   report.GetExpiresAt(),
	})
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Проверяет состояние сервиса
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *APIHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	healthStatus := HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
		Time:    time.Now(),
		Services: map[string]string{
			"api":    "healthy",
			"cache":  "unknown",
			"report": "unknown",
		},
	}

	if err := h.cacheService.HealthCheck(ctx); err != nil {
		healthStatus.Status = "degraded"
		healthStatus.Services["cache"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		healthStatus.Services["cache"] = "healthy"
	}

	if err := h.reportService.HealthCheck(ctx); err != nil {
		healthStatus.Status = "degraded"
		healthStatus.Services["report"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		healthStatus.Services["report"] = "healthy"
	}

	c.JSON(http.StatusOK, healthStatus)
}

func (h *APIHandler) respondError(c *gin.Context, status int, message string) {
	h.logger.Errorf("HTTP %d: %s", status, message)
	c.JSON(status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Time:    time.Now(),
	})
}

// DownloadReport godoc
// @Summary Download report by ID
// @Description Скачать Excel отчет по id
// @Tags reports
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param id path string true "ID отчета"
// @Success 200 {file} file Excel file
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /reports/{id}/download [get]
func (h *APIHandler) DownloadReport(c *gin.Context) {
	ctx := c.Request.Context()
	reportID := c.Param("id")

	if reportID == "" {
		h.respondError(c, http.StatusBadRequest, "report ID is required")
		return
	}

	reader, fileName, err := h.reportService.DownloadReport(ctx, reportID)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to download report: %v", err))
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	_, err = io.Copy(c.Writer, reader)
	if err != nil {
		h.logger.Errorf("Failed to stream report: %v", err)
	}
}

type ReportResponse struct {
	ID          string     `json:"id"`
	ReportType  string     `json:"report_type"`
	CityID      int        `json:"city_id,omitempty"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	FileName    string     `json:"file_name"`
	FileSize    int64      `json:"file_size"`
	DownloadURL string     `json:"download_url,omitempty"`
	GeneratedAt time.Time  `json:"generated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type ErrorResponse struct {
	Error   string    `json:"error"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Time     time.Time         `json:"time"`
	Services map[string]string `json:"services"`
}
