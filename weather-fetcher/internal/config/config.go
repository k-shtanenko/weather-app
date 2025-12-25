package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App         AppConfig
	OpenWeather OpenWeatherConfig
	Kafka       KafkaConfig
	Scheduler   SchedulerConfig
	HealthCheck HealthCheckConfig
}

type AppConfig struct {
	Name     string `mapstructure:"name"`
	Env      string `mapstructure:"env"`
	LogLevel string `mapstructure:"log_level"`
}

type OpenWeatherConfig struct {
	APIKey  string   `mapstructure:"api_key"`
	BaseURL string   `mapstructure:"base_url"`
	Units   string   `mapstructure:"units"`
	Lang    string   `mapstructure:"lang"`
	CityIDs []string `mapstructure:"city_ids"`
}

type KafkaConfig struct {
	Broker       string `mapstructure:"broker"`
	Topic        string `mapstructure:"topic"`
	RequiredAcks int16  `mapstructure:"required_acks"`
	MaxRetries   int    `mapstructure:"max_retries"`
}

type SchedulerConfig struct {
	Interval time.Duration `mapstructure:"interval"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

type HealthCheckConfig struct {
	APITimeout    time.Duration `mapstructure:"api_timeout"`
	KafkaTimeout  time.Duration `mapstructure:"kafka_timeout"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
	MaxRetries    int           `mapstructure:"max_retries"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/weather-fetcher/")

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
		}
	}

	if apiKey := os.Getenv("WEATHER_API_KEY"); apiKey != "" {
		v.Set("openweather.api_key", apiKey)
	}

	if apiKey := os.Getenv("OPENWEATHER_UNITS"); apiKey != "" {
		v.Set("openweather.units", apiKey)
	}

	if apiKey := os.Getenv("OPENWEATHER_LANG"); apiKey != "" {
		v.Set("openweather.lang", apiKey)
	}

	if baseUrl := os.Getenv("OPENWEATHER_BASE_URL"); baseUrl != "" {
		v.Set("openweather.base_url", baseUrl)
	}

	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		v.Set("kafka.broker", broker)
	}

	if topic := os.Getenv("KAFKA_WEATHER_TOPIC"); topic != "" {
		v.Set("kafka.topic", topic)
	}

	if cityIDs := os.Getenv("CITY_IDS"); cityIDs != "" {
		cityIDList := strings.Split(cityIDs, ",")
		for i, cityID := range cityIDList {
			cityIDList[i] = strings.TrimSpace(cityID)
		}
		v.Set("openweather.city_ids", cityIDList)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("ошибка десериализации конфигурации: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("невалидная конфигурация: %w", err)
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.OpenWeather.APIKey == "" {
		return fmt.Errorf("API ключ OpenWeather не может быть пустым")
	}

	if len(cfg.OpenWeather.CityIDs) == 0 {
		return fmt.Errorf("список CityIDs не может быть пустым")
	}

	if cfg.Kafka.Broker == "" {
		return fmt.Errorf("список Kafka брокеров не может быть пустым")
	}

	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("Kafka топик не может быть пустым")
	}

	if cfg.Scheduler.Interval <= 0 {
		return fmt.Errorf("интервал планировщика должен быть положительным")
	}

	return nil
}
