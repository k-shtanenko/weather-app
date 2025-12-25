package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/k-shtanenko/weather-app/weather-api/docs"
	"github.com/k-shtanenko/weather-app/weather-api/internal/config"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type API interface {
	setupRoutes()
	Start() error
	Stop(ctx context.Context) error
}

type APIServer struct {
	server     *http.Server
	router     *gin.Engine
	handler    *APIHandler
	middleware *Middleware
	config     *config.Config
	logger     logger.Logger
}

func NewAPIServer(reportService ReportService, cacheService CacheService, middleware *Middleware, cfg *config.Config) *APIServer {
	gin.SetMode(gin.ReleaseMode)
	if cfg.App.Env == "development" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	handler := NewAPIHandler(reportService, cacheService)

	return &APIServer{
		router:     router,
		handler:    handler,
		middleware: middleware,
		config:     cfg,
		logger:     logger.New(cfg.App.LogLevel, cfg.App.Env).WithField("component", "api_server"),
	}
}

func (s *APIServer) setupRoutes() {
	api := s.router.Group(s.config.API.BasePath)

	api.Use(s.middleware.Recovery())
	api.Use(s.middleware.Logging())
	api.Use(s.middleware.CORS())
	api.Use(s.middleware.RateLimit())
	api.Use(s.middleware.Cache())

	api.GET("/health", s.handler.HealthCheck)

	reports := api.Group("/reports")
	{
		reports.GET("/daily", s.handler.GetDailyReport)
		reports.GET("/weekly", s.handler.GetWeeklyReport)
		reports.GET("/monthly", s.handler.GetMonthlyReport)
		reports.GET("/:id/download", s.handler.DownloadReport)
	}

	if s.config.API.EnableSwagger {
		url := ginSwagger.URL("/swagger/doc.json")
		s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
		s.logger.Info("Swagger documentation enabled at /swagger/index.html")
	}

	s.router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": fmt.Sprintf("Route %s not found", c.Request.URL.Path),
		})
	})
}

func (s *APIServer) Start() error {
	s.setupRoutes()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.App.Port),
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		s.logger.Infof("Starting API server on port %d", s.config.App.Port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	return nil
}

func (s *APIServer) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down API server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.App.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server gracefully: %w", err)
	}

	s.logger.Info("API server stopped")
	return nil
}
