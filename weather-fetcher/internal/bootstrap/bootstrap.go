package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/k-shtanenko/weather-app/weather-fetcher/api"
	"github.com/k-shtanenko/weather-app/weather-fetcher/config"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/producer"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/scheduler"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/services"
)

type BootstrapInterface interface {
	PrintConfigInfo()
	Run() error
	initDependencies() (api.Fetcher, producer.Producer, scheduler.Scheduler, error)
}

type Bootstrap struct {
	config *config.Config
	logger logger.Logger
}

func NewBootstrap() (*Bootstrap, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	log := logger.New(cfg.App.LogLevel, cfg.App.Env).WithField("service", cfg.App.Name)

	return &Bootstrap{
		config: cfg,
		logger: log,
	}, nil
}

func (b *Bootstrap) Run() error {
	b.logger.Info("Starting weather-fetcher service")
	b.logger.Infof("Environment: %s", b.config.App.Env)
	b.logger.Infof("Log level: %s", b.config.App.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalChan
		b.logger.Infof("Received signal: %v. Shutting down...", sig)
		cancel()
	}()

	fetcher, producer, scheduler, err := b.initDependencies()
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	b.logger.Info("Performing initial health checks...")
	healthChecker := NewHealthChecker(
		fetcher,
		producer,
		b.config.HealthCheck.APITimeout,
		b.config.HealthCheck.KafkaTimeout,
		b.config.HealthCheck.RetryInterval,
		b.config.HealthCheck.MaxRetries,
	)

	if err := healthChecker.CheckAll(ctx); err != nil {
		return fmt.Errorf("initial health checks failed: %w", err)
	}

	serviceFactory := services.NewWeatherServiceFactory()
	service := serviceFactory.Create(
		fetcher,
		producer,
		scheduler,
		b.config.OpenWeather.CityIDs,
	)

	b.logger.Infof("Starting service with %d cities, interval: %v",
		len(b.config.OpenWeather.CityIDs), b.config.Scheduler.Interval)

	if err := service.Start(ctx, b.config.Scheduler.Interval); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	<-ctx.Done()

	b.logger.Info("Stopping service...")
	service.Stop()

	b.logger.Info("Service stopped gracefully")
	return nil
}

func (b *Bootstrap) initDependencies() (
	api.Fetcher,
	producer.Producer,
	scheduler.Scheduler,
	error,
) {
	b.logger.Info("Initializing dependencies...")

	fetcherFactory := api.NewOpenWeatherFetcherFactory()
	fetcher := fetcherFactory.CreateFetcher(
		b.config.OpenWeather.BaseURL,
		b.config.OpenWeather.APIKey,
		b.config.OpenWeather.Units,
		b.config.OpenWeather.Lang,
	)
	b.logger.Info("OpenWeather fetcher initialized")

	producerFactory := producer.NewKafkaProducerFactory()
	producer, err := producerFactory.CreateProducer(
		b.config.Kafka.Broker,
		b.config.Kafka.Topic,
		b.config.Kafka.RequiredAcks,
		b.config.Kafka.MaxRetries,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	b.logger.Infof("Kafka producer initialized for topic: %s", b.config.Kafka.Topic)

	schedulerFactory := scheduler.NewCronSchedulerFactory()
	scheduler := schedulerFactory.CreateScheduler(b.config.Scheduler.Timeout)
	b.logger.Info("Scheduler initialized")

	return fetcher, producer, scheduler, nil
}

func (b *Bootstrap) PrintConfigInfo() {
	b.logger.Infof("Service Name: %s", b.config.App.Name)
	b.logger.Infof("Environment: %s", b.config.App.Env)
	b.logger.Infof("OpenWeather API Base URL: %s", b.config.OpenWeather.BaseURL)
	b.logger.Infof("Kafka Brokers: %v", b.config.Kafka.Broker)
	b.logger.Infof("Kafka Topic: %s", b.config.Kafka.Topic)
	b.logger.Infof("Cities to monitor: %v", b.config.OpenWeather.CityIDs)
	b.logger.Infof("Update interval: %v", b.config.Scheduler.Interval)
}
