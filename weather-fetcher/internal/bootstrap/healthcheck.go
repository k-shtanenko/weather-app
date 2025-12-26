package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/k-shtanenko/weather-app/weather-fetcher/api"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/producer"
)

type HealthCheck interface {
	CheckAll(ctx context.Context) error
	checkWithRetry(ctx context.Context, checkFunc func(context.Context) error, serviceName string, timeout time.Duration) error
	checkAPI(ctx context.Context) error
	checkKafka(ctx context.Context) error
}

type HealthChecker struct {
	fetcher  api.Fetcher
	producer producer.Producer
	logger   logger.Logger
	config   struct {
		apiTimeout    time.Duration
		kafkaTimeout  time.Duration
		retryInterval time.Duration
		maxRetries    int
	}
}

func NewHealthChecker(fetcher api.Fetcher, producer producer.Producer, apiTimeout, kafkaTimeout, retryInterval time.Duration, maxRetries int) HealthCheck {
	return &HealthChecker{
		fetcher:  fetcher,
		producer: producer,
		logger:   logger.New("info", "development").WithField("component", "health_checker"),
		config: struct {
			apiTimeout    time.Duration
			kafkaTimeout  time.Duration
			retryInterval time.Duration
			maxRetries    int
		}{
			apiTimeout:    apiTimeout,
			kafkaTimeout:  kafkaTimeout,
			retryInterval: retryInterval,
			maxRetries:    maxRetries,
		},
	}
}

func (h *HealthChecker) CheckAll(ctx context.Context) error {
	h.logger.Info("Starting health checks for all dependencies")

	if err := h.checkWithRetry(ctx, h.checkAPI, "OpenWeather API", h.config.apiTimeout); err != nil {
		return fmt.Errorf("OpenWeather API health check failed: %w", err)
	}

	if err := h.checkWithRetry(ctx, h.checkKafka, "Kafka", h.config.kafkaTimeout); err != nil {
		return fmt.Errorf("Kafka health check failed: %w", err)
	}

	h.logger.Info("All health checks passed successfully")
	return nil
}

func (h *HealthChecker) checkWithRetry(ctx context.Context, checkFunc func(context.Context) error, serviceName string, timeout time.Duration) error {
	var lastErr error

	for i := 0; i < h.config.maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			h.logger.Debugf("Checking %s (attempt %d/%d)", serviceName, i+1, h.config.maxRetries)

			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			err := checkFunc(checkCtx)
			cancel()

			if err == nil {
				h.logger.Infof("%s health check passed", serviceName)
				return nil
			}

			lastErr = err
			h.logger.Warnf("%s health check failed (attempt %d/%d): %v", serviceName, i+1, h.config.maxRetries, err)

			if i < h.config.maxRetries-1 {
				h.logger.Debugf("Retrying %s check in %v", serviceName, h.config.retryInterval)
				time.Sleep(h.config.retryInterval)
			}
		}
	}

	return fmt.Errorf("all %d attempts failed, last error: %w", h.config.maxRetries, lastErr)
}

func (h *HealthChecker) checkAPI(ctx context.Context) error {
	return h.fetcher.HealthCheck(ctx)
}

func (h *HealthChecker) checkKafka(ctx context.Context) error {
	return h.producer.HealthCheck(ctx)
}
