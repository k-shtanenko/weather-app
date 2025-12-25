package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k-shtanenko/weather-app/weather-api/internal/application"
	"github.com/k-shtanenko/weather-app/weather-api/internal/config"
	"github.com/k-shtanenko/weather-app/weather-api/internal/domain/ports"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/api"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/cache"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/database"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/excel"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/messaging"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/scheduler"
	"github.com/k-shtanenko/weather-app/weather-api/internal/infrastructure/storage"
	"github.com/k-shtanenko/weather-app/weather-api/internal/pkg/logger"
)

type App struct {
	config        *config.Config
	logger        logger.Logger
	weatherRepo   ports.WeatherRepository
	reportRepo    ports.ReportRepository
	weatherProc   *application.WeatherProcessor
	reportService *application.ReportService
	cacheService  *application.CacheService
	kafkaConsumer ports.Consumer
	scheduler     ports.Scheduler
	apiServer     api.API
}

func Bootstrap() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	appLogger := logger.New(cfg.App.LogLevel, cfg.App.Env).WithField("service", cfg.App.Name)
	appLogger.Infof("Starting %s in %s mode", cfg.App.Name, cfg.App.Env)

	app := &App{
		config: cfg,
		logger: appLogger,
	}

	if err := app.initComponents(); err != nil {
		appLogger.Fatalf("Failed to initialize components: %v", err)
	}

	if err := app.start(); err != nil {
		appLogger.Fatalf("Failed to start application: %v", err)
	}

	app.waitForShutdown()
}

func (a *App) initComponents() error {
	a.logger.Info("Initializing components...")

	a.logger.Info("Initializing PostgreSQL repositories...")
	weatherRepo, err := database.NewPostgresWeatherRepository(
		a.config.Postgres.CoordinatorHost,
		a.config.Postgres.ShardHosts,
		a.config.Postgres.Port,
		a.config.Postgres.User,
		a.config.Postgres.Password,
		a.config.Postgres.Database,
	)
	if err != nil {
		return fmt.Errorf("failed to create weather repository: %w", err)
	}
	a.weatherRepo = weatherRepo

	reportRepo, err := database.NewPostgresReportRepository(
		a.config.Postgres.CoordinatorHost,
		a.config.Postgres.Port,
		a.config.Postgres.User,
		a.config.Postgres.Password,
		a.config.Postgres.Database,
	)
	if err != nil {
		return fmt.Errorf("failed to create report repository: %w", err)
	}
	a.reportRepo = reportRepo

	a.logger.Info("Initializing Redis cache...")
	redisCache, err := cache.NewRedisCache(
		a.config.Redis.Host,
		a.config.Redis.Port,
		a.config.Redis.Password,
		a.config.Redis.DB,
	)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}
	a.cacheService = application.NewCacheService(redisCache, a.config.Cache.MaxCacheSizeMB)

	a.logger.Info("Initializing Minio storage...")
	minioStorage, err := storage.NewMinioStorage(
		a.config.Minio.Endpoint,
		a.config.Minio.AccessKey,
		a.config.Minio.SecretKey,
		a.config.Minio.UseSSL,
	)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	reportStorage := storage.NewMinioReportStorage(minioStorage, a.config.Minio.Bucket)

	a.logger.Info("Initializing Excel generator...")
	excelGen := excel.NewExcelGeneratorImpl()

	a.logger.Info("Initializing application services...")
	a.weatherProc = application.NewWeatherProcessor(weatherRepo)
	a.reportService = application.NewReportService(
		weatherRepo,
		reportRepo,
		excelGen,
		reportStorage,
		a.cacheService,
	)

	a.logger.Info("Initializing Kafka consumer...")
	kafkaConsumer, err := messaging.NewKafkaConsumer(
		a.config.Kafka.Broker,
		a.config.Kafka.Topic,
		a.config.Kafka.GroupID,
		a.config.Kafka.BatchSize,
	)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	a.kafkaConsumer = kafkaConsumer

	a.logger.Info("Initializing scheduler...")
	a.scheduler = scheduler.NewCronScheduler()

	a.logger.Info("Initializing API server...")
	apiMiddleware := api.NewMiddleware(a.config.API.RateLimit, a.config.API.RateLimitWindow)
	a.apiServer = api.NewAPIServer(a.reportService, a.cacheService, apiMiddleware, a.config)

	a.logger.Info("All components initialized successfully")
	return nil
}

func (a *App) start() error {
	a.logger.Info("Starting application...")

	ctx := context.Background()

	a.logger.Info("Starting Kafka consumer...")
	if err := a.kafkaConsumer.Consume(ctx, a.weatherProc.ProcessWeatherData); err != nil {
		return fmt.Errorf("failed to start Kafka consumer: %w", err)
	}

	a.logger.Info("Setting up scheduler...")
	if err := a.setupScheduler(ctx); err != nil {
		return fmt.Errorf("failed to setup scheduler: %w", err)
	}

	a.logger.Info("Starting API server...")
	if err := a.apiServer.Start(); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	go func() {
		time.Sleep(a.config.HealthCheck.StartupDelay)
		a.runHealthChecks(ctx)
	}()

	a.logger.Info("Application started successfully")
	return nil
}

func (a *App) setupScheduler(ctx context.Context) error {
	if err := a.scheduler.Schedule(ctx, "daily_report_generation",
		a.config.Scheduler.ReportGenerationInterval,
		func(ctx context.Context) error {
			a.logger.Info("Running scheduled daily report generation")
			yesterday := time.Now().AddDate(0, 0, -1)

			cities, err := a.weatherRepo.GetCitiesWithData(ctx)
			if err != nil {
				a.logger.Errorf("Failed to get cities: %v", err)
				return err
			}

			for _, cityID := range cities {
				if _, err := a.reportService.GenerateDailyReport(ctx, cityID, yesterday); err != nil {
					a.logger.Warnf("Failed to generate daily report for city %d: %v", cityID, err)
				}
			}

			a.logger.Infof("Generated daily reports for %d cities", len(cities))
			return nil
		}); err != nil {
		return fmt.Errorf("failed to schedule daily report generation: %w", err)
	}

	if err := a.scheduler.Schedule(ctx, "data_cleanup",
		a.config.Scheduler.CleanupInterval,
		func(ctx context.Context) error {
			a.logger.Info("Running scheduled data cleanup")

			if err := a.weatherRepo.CleanupOldData(ctx, a.config.Reports.DailyRetentionDays); err != nil {
				a.logger.Errorf("Failed to cleanup old weather data: %v", err)
			}

			if err := a.reportService.CleanupExpiredReports(ctx); err != nil {
				a.logger.Errorf("Failed to cleanup expired reports: %v", err)
			}

			if err := a.cacheService.CleanupExpiredCache(ctx); err != nil {
				a.logger.Errorf("Failed to cleanup expired cache: %v", err)
			}

			a.logger.Info("Data cleanup completed")
			return nil
		}); err != nil {
		return fmt.Errorf("failed to schedule data cleanup: %w", err)
	}

	return nil
}

func (a *App) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(a.config.HealthCheck.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.performHealthChecks(ctx)
		}
	}
}

func (a *App) performHealthChecks(ctx context.Context) {
	a.logger.Debug("Running health checks...")

	checks := []struct {
		name  string
		check func(context.Context) error
	}{
		{"weather_repository", a.weatherRepo.HealthCheck},
		{"report_repository", a.reportRepo.HealthCheck},
		{"kafka_consumer", a.kafkaConsumer.HealthCheck},
		{"scheduler", a.scheduler.HealthCheck},
		{"weather_processor", a.weatherProc.HealthCheck},
		{"report_service", a.reportService.HealthCheck},
		{"cache_service", a.cacheService.HealthCheck},
	}

	for _, check := range checks {
		checkCtx, cancel := context.WithTimeout(ctx, a.config.HealthCheck.Timeout)
		if err := check.check(checkCtx); err != nil {
			a.logger.Errorf("Health check failed for %s: %v", check.name, err)
		} else {
			a.logger.Debugf("Health check passed for %s", check.name)
		}
		cancel()
	}
}

func (a *App) waitForShutdown() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	sig := <-signalChan
	a.logger.Infof("Received signal: %v. Shutting down...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), a.config.App.ShutdownTimeout)
	defer cancel()

	a.shutdownComponents(ctx)

	a.logger.Info("Application shutdown completed")
}

func (a *App) shutdownComponents(ctx context.Context) {
	if a.apiServer != nil {
		a.logger.Info("Stopping API server...")
		if err := a.apiServer.Stop(ctx); err != nil {
			a.logger.Errorf("Failed to stop API server: %v", err)
		}
	}

	if a.scheduler != nil {
		a.logger.Info("Stopping scheduler...")
		a.scheduler.Stop()
	}

	if a.kafkaConsumer != nil {
		a.logger.Info("Stopping Kafka consumer...")
		if err := a.kafkaConsumer.Close(); err != nil {
			a.logger.Errorf("Failed to close Kafka consumer: %v", err)
		}
	}

	if a.weatherRepo != nil {
		a.logger.Info("Closing weather repository...")
		if err := a.weatherRepo.Close(); err != nil {
			a.logger.Errorf("Failed to close weather repository: %v", err)
		}
	}

	if a.reportRepo != nil {
		a.logger.Info("Closing report repository...")
		if closer, ok := a.reportRepo.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				a.logger.Errorf("Failed to close report repository: %v", err)
			}
		}
	}

	if a.cacheService != nil {
		a.logger.Info("Cache service cleanup completed")
	}
}
