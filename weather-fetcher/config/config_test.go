package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	originalAPIKey := os.Getenv("WEATHER_API_KEY")
	originalBaseURL := os.Getenv("OPENWEATHER_BASE_URL")
	originalBroker := os.Getenv("KAFKA_BROKER")
	originalTopic := os.Getenv("KAFKA_WEATHER_TOPIC")
	originalCityIDs := os.Getenv("CITY_IDS")

	defer func() {
		os.Setenv("WEATHER_API_KEY", originalAPIKey)
		os.Setenv("OPENWEATHER_BASE_URL", originalBaseURL)
		os.Setenv("KAFKA_BROKER", originalBroker)
		os.Setenv("KAFKA_WEATHER_TOPIC", originalTopic)
		os.Setenv("CITY_IDS", originalCityIDs)
	}()

	configContent := `
app:
  name: "weather-fetcher-test"
  env: "test"
  log_level: "debug"

openweather:
  api_key: "test-api-key"
  base_url: "https://api.openweathermap.org"
  units: "metric"
  lang: "en"
  city_ids: 
    - "123"
    - "456"

kafka:
  broker: "localhost:9092"
  topic: "weather-test"
  required_acks: 1
  max_retries: 3

scheduler:
  interval: "60s"
  timeout: "10s"

healthcheck:
  api_timeout: "1s"
  kafka_timeout: "2s"
  retry_interval: "500ms"
  max_retries: 2
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	t.Run("load from file", func(t *testing.T) {
		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "weather-fetcher-test", cfg.App.Name)
		assert.Equal(t, "test", cfg.App.Env)
		assert.Equal(t, "debug", cfg.App.LogLevel)

		assert.Equal(t, "test-api-key", cfg.OpenWeather.APIKey)
		assert.Equal(t, "https://api.openweathermap.org", cfg.OpenWeather.BaseURL)
		assert.Equal(t, []string{"123", "456"}, cfg.OpenWeather.CityIDs)
		assert.Equal(t, "metric", cfg.OpenWeather.Units)
		assert.Equal(t, "en", cfg.OpenWeather.Lang)

		assert.Equal(t, "localhost:9092", cfg.Kafka.Broker)
		assert.Equal(t, "weather-test", cfg.Kafka.Topic)
		assert.Equal(t, int16(1), cfg.Kafka.RequiredAcks)
		assert.Equal(t, 3, cfg.Kafka.MaxRetries)

		assert.Equal(t, time.Minute, cfg.Scheduler.Interval)
		assert.Equal(t, 10*time.Second, cfg.Scheduler.Timeout)

		assert.Equal(t, time.Second, cfg.HealthCheck.APITimeout)
		assert.Equal(t, 2*time.Second, cfg.HealthCheck.KafkaTimeout)
		assert.Equal(t, 500*time.Millisecond, cfg.HealthCheck.RetryInterval)
		assert.Equal(t, 2, cfg.HealthCheck.MaxRetries)
	})

	t.Run("override with environment variables", func(t *testing.T) {
		os.Setenv("WEATHER_API_KEY", "env-api-key")
		os.Setenv("OPENWEATHER_BASE_URL", "https://env.api.url")
		os.Setenv("KAFKA_BROKER", "env-broker:9092")
		os.Setenv("KAFKA_WEATHER_TOPIC", "env-topic")
		os.Setenv("CITY_IDS", "789,101112")
		os.Setenv("OPENWEATHER_UNITS", "imperial")
		os.Setenv("OPENWEATHER_LANG", "ru")

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, "env-api-key", cfg.OpenWeather.APIKey)
		assert.Equal(t, "https://env.api.url", cfg.OpenWeather.BaseURL)
		assert.Equal(t, "env-broker:9092", cfg.Kafka.Broker)
		assert.Equal(t, "env-topic", cfg.Kafka.Topic)
		assert.Equal(t, []string{"789", "101112"}, cfg.OpenWeather.CityIDs)
		assert.Equal(t, "imperial", cfg.OpenWeather.Units)
		assert.Equal(t, "ru", cfg.OpenWeather.Lang)
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "test-key",
				CityIDs: []string{"123", "456"},
			},
			Kafka: KafkaConfig{
				Broker: "localhost:9092",
				Topic:  "weather",
			},
			Scheduler: SchedulerConfig{
				Interval: time.Minute,
			},
		}

		err := validateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("missing API key", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "",
				CityIDs: []string{"123"},
			},
			Kafka: KafkaConfig{
				Broker: "localhost:9092",
				Topic:  "weather",
			},
			Scheduler: SchedulerConfig{
				Interval: time.Minute,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API ключ OpenWeather не может быть пустым")
	})

	t.Run("empty city IDs", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "test-key",
				CityIDs: []string{},
			},
			Kafka: KafkaConfig{
				Broker: "localhost:9092",
				Topic:  "weather",
			},
			Scheduler: SchedulerConfig{
				Interval: time.Minute,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "список CityIDs не может быть пустым")
	})

	t.Run("missing Kafka broker", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "test-key",
				CityIDs: []string{"123"},
			},
			Kafka: KafkaConfig{
				Broker: "",
				Topic:  "weather",
			},
			Scheduler: SchedulerConfig{
				Interval: time.Minute,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "список Kafka брокеров не может быть пустым")
	})

	t.Run("missing Kafka topic", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "test-key",
				CityIDs: []string{"123"},
			},
			Kafka: KafkaConfig{
				Broker: "localhost:9092",
				Topic:  "",
			},
			Scheduler: SchedulerConfig{
				Interval: time.Minute,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Kafka топик не может быть пустым")
	})

	t.Run("invalid scheduler interval", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "test-key",
				CityIDs: []string{"123"},
			},
			Kafka: KafkaConfig{
				Broker: "localhost:9092",
				Topic:  "weather",
			},
			Scheduler: SchedulerConfig{
				Interval: 0,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "интервал планировщика должен быть положительным")
	})

	t.Run("negative scheduler interval", func(t *testing.T) {
		cfg := &Config{
			OpenWeather: OpenWeatherConfig{
				APIKey:  "test-key",
				CityIDs: []string{"123"},
			},
			Kafka: KafkaConfig{
				Broker: "localhost:9092",
				Topic:  "weather",
			},
			Scheduler: SchedulerConfig{
				Interval: -time.Minute,
			},
		}

		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "интервал планировщика должен быть положительным")
	})
}

func TestValidateConfig_AllValidations(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "missing API key",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "", CityIDs: []string{"123"}},
				Kafka:       KafkaConfig{Broker: "broker", Topic: "topic"},
				Scheduler:   SchedulerConfig{Interval: time.Minute},
			},
			wantErr:     true,
			errContains: "API ключ OpenWeather не может быть пустым",
		},
		{
			name: "empty city IDs",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "key", CityIDs: []string{}},
				Kafka:       KafkaConfig{Broker: "broker", Topic: "topic"},
				Scheduler:   SchedulerConfig{Interval: time.Minute},
			},
			wantErr:     true,
			errContains: "список CityIDs не может быть пустым",
		},
		{
			name: "missing Kafka broker",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "key", CityIDs: []string{"123"}},
				Kafka:       KafkaConfig{Broker: "", Topic: "topic"},
				Scheduler:   SchedulerConfig{Interval: time.Minute},
			},
			wantErr:     true,
			errContains: "список Kafka брокеров не может быть пустым",
		},
		{
			name: "missing Kafka topic",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "key", CityIDs: []string{"123"}},
				Kafka:       KafkaConfig{Broker: "broker", Topic: ""},
				Scheduler:   SchedulerConfig{Interval: time.Minute},
			},
			wantErr:     true,
			errContains: "Kafka топик не может быть пустым",
		},
		{
			name: "invalid scheduler interval zero",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "key", CityIDs: []string{"123"}},
				Kafka:       KafkaConfig{Broker: "broker", Topic: "topic"},
				Scheduler:   SchedulerConfig{Interval: 0},
			},
			wantErr:     true,
			errContains: "интервал планировщика должен быть положительным",
		},
		{
			name: "invalid scheduler interval negative",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "key", CityIDs: []string{"123"}},
				Kafka:       KafkaConfig{Broker: "broker", Topic: "topic"},
				Scheduler:   SchedulerConfig{Interval: -time.Minute},
			},
			wantErr:     true,
			errContains: "интервал планировщика должен быть положительным",
		},
		{
			name: "all valid",
			cfg: &Config{
				OpenWeather: OpenWeatherConfig{APIKey: "key", CityIDs: []string{"123"}},
				Kafka:       KafkaConfig{Broker: "broker", Topic: "topic"},
				Scheduler:   SchedulerConfig{Interval: time.Minute},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(tc.cfg)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad_WithEmptyCityIDsEnvVar(t *testing.T) {
	originalCityIDs := os.Getenv("CITY_IDS")
	defer func() { os.Setenv("CITY_IDS", originalCityIDs) }()

	os.Setenv("CITY_IDS", "")

	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	configContent := `
app:
  name: "test"
  env: "test"
  log_level: "info"

openweather:
  api_key: "test-key"
  base_url: "http://test.com"
  units: "metric"
  lang: "en"
  city_ids: ["123"]

kafka:
  broker: "localhost:9092"
  topic: "weather"
  required_acks: 1
  max_retries: 3

scheduler:
  interval: "60s"
  timeout: "30s"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	viper.Reset()
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"123"}, cfg.OpenWeather.CityIDs)
}

func TestLoad_ViperReadInConfigError_NonConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
invalid: !!binary |
  This is not valid base64
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	os.Setenv("WEATHER_API_KEY", "test-key")
	os.Setenv("CITY_IDS", "123")
	os.Setenv("KAFKA_BROKER", "localhost:9092")
	os.Setenv("KAFKA_WEATHER_TOPIC", "weather")
	os.Setenv("OPENWEATHER_BASE_URL", "http://test.com")
	os.Setenv("OPENWEATHER_UNITS", "metric")
	os.Setenv("OPENWEATHER_LANG", "en")
	os.Setenv("SCHEDULER_INTERVAL", "60s")

	defer func() {
		os.Unsetenv("WEATHER_API_KEY")
		os.Unsetenv("CITY_IDS")
		os.Unsetenv("KAFKA_BROKER")
		os.Unsetenv("KAFKA_WEATHER_TOPIC")
		os.Unsetenv("OPENWEATHER_BASE_URL")
		os.Unsetenv("OPENWEATHER_UNITS")
		os.Unsetenv("OPENWEATHER_LANG")
		os.Unsetenv("SCHEDULER_INTERVAL")
	}()

	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_ConfigFileNotFoundButEnvVarsValid(t *testing.T) {
	originalAPIKey := os.Getenv("WEATHER_API_KEY")
	originalCityIDs := os.Getenv("CITY_IDS")
	originalBroker := os.Getenv("KAFKA_BROKER")
	originalTopic := os.Getenv("KAFKA_WEATHER_TOPIC")
	originalBaseURL := os.Getenv("OPENWEATHER_BASE_URL")
	originalUnits := os.Getenv("OPENWEATHER_UNITS")
	originalLang := os.Getenv("OPENWEATHER_LANG")

	defer func() {
		os.Setenv("WEATHER_API_KEY", originalAPIKey)
		os.Setenv("CITY_IDS", originalCityIDs)
		os.Setenv("KAFKA_BROKER", originalBroker)
		os.Setenv("KAFKA_WEATHER_TOPIC", originalTopic)
		os.Setenv("OPENWEATHER_BASE_URL", originalBaseURL)
		os.Setenv("OPENWEATHER_UNITS", originalUnits)
		os.Setenv("OPENWEATHER_LANG", originalLang)
	}()

	os.Setenv("WEATHER_API_KEY", "test-key-12345")
	os.Setenv("CITY_IDS", "524901,498817")
	os.Setenv("KAFKA_BROKER", "localhost:9092")
	os.Setenv("KAFKA_WEATHER_TOPIC", "weather-test")
	os.Setenv("OPENWEATHER_BASE_URL", "https://api.openweathermap.org")
	os.Setenv("OPENWEATHER_UNITS", "metric")
	os.Setenv("OPENWEATHER_LANG", "en")
	os.Setenv("SCHEDULER_INTERVAL", "60s")

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	viper.Reset()
	cfg, err := Load()

	if err != nil {
		assert.Contains(t, err.Error(), "интервал планировщика должен быть положительным")
	} else {
		assert.NotNil(t, cfg)
		assert.True(t, cfg.Scheduler.Interval > 0)
	}
}

func TestValidateConfig_EmptyCityIDsFromEnvVar(t *testing.T) {
	originalCityIDs := os.Getenv("CITY_IDS")
	defer func() { os.Setenv("CITY_IDS", originalCityIDs) }()

	os.Setenv("CITY_IDS", "")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
app:
  name: "test"
  env: "test"
  log_level: "info"

openweather:
  api_key: "test-key"
  base_url: "http://test.com"
  units: "metric"
  lang: "en"

kafka:
  broker: "localhost:9092"
  topic: "weather"
  required_acks: 1
  max_retries: 3

scheduler:
  interval: "60s"
  timeout: "30s"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	viper.Reset()
	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "список CityIDs не может быть пустым")
}

func TestLoad_ConfigFileReadError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
invalid: !!binary |
  This is not valid base64
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	os.Setenv("WEATHER_API_KEY", "test-key")
	os.Setenv("CITY_IDS", "123")
	os.Setenv("KAFKA_BROKER", "localhost:9092")
	os.Setenv("KAFKA_WEATHER_TOPIC", "weather")
	os.Setenv("OPENWEATHER_BASE_URL", "http://test.com")
	os.Setenv("OPENWEATHER_UNITS", "metric")
	os.Setenv("OPENWEATHER_LANG", "en")
	os.Setenv("SCHEDULER_INTERVAL", "60s")

	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, strings.Contains(err.Error(), "ошибка чтения") ||
		strings.Contains(err.Error(), "read") ||
		strings.Contains(err.Error(), "config"))
}

func TestLoad_ConfigFileNotFound_NoEnvVars(t *testing.T) {
	originalVars := map[string]string{
		"WEATHER_API_KEY":      os.Getenv("WEATHER_API_KEY"),
		"CITY_IDS":             os.Getenv("CITY_IDS"),
		"KAFKA_BROKER":         os.Getenv("KAFKA_BROKER"),
		"KAFKA_WEATHER_TOPIC":  os.Getenv("KAFKA_WEATHER_TOPIC"),
		"OPENWEATHER_BASE_URL": os.Getenv("OPENWEATHER_BASE_URL"),
	}

	defer func() {
		for k, v := range originalVars {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	os.Unsetenv("WEATHER_API_KEY")
	os.Unsetenv("CITY_IDS")
	os.Unsetenv("KAFKA_BROKER")
	os.Unsetenv("KAFKA_WEATHER_TOPIC")
	os.Unsetenv("OPENWEATHER_BASE_URL")

	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	viper.Reset()
	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_EmptyConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := ``

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	os.Setenv("WEATHER_API_KEY", "test-key")
	os.Setenv("CITY_IDS", "123")
	os.Setenv("KAFKA_BROKER", "localhost:9092")
	os.Setenv("KAFKA_WEATHER_TOPIC", "weather")
	os.Setenv("OPENWEATHER_BASE_URL", "http://test.com")

	viper.Reset()
	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_WithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `invalid yaml: : : :`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	os.Setenv("WEATHER_API_KEY", "test-key")
	os.Setenv("CITY_IDS", "123")
	os.Setenv("KAFKA_BROKER", "localhost:9092")
	os.Setenv("KAFKA_WEATHER_TOPIC", "weather")
	os.Setenv("OPENWEATHER_BASE_URL", "http://test.com")
	os.Setenv("OPENWEATHER_UNITS", "metric")
	os.Setenv("OPENWEATHER_LANG", "en")

	viper.Reset()
	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_UnmarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
app:
  name: "test"
  env: "test"
  log_level: "info"

openweather:
  api_key: "test-key"
  base_url: "http://test.com"
  units: "metric"
  lang: "en"
  city_ids: ["123"]

kafka:
  broker: "localhost:9092"
  topic: "weather"
  required_acks: 1
  max_retries: 3

scheduler:
  interval: "invalid-duration"  # Это вызовет ошибку при Unmarshal в time.Duration
  timeout: "30s"

healthcheck:
  api_timeout: "1s"
  kafka_timeout: "2s"
  retry_interval: "500ms"
  max_retries: 2
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	viper.Reset()
	cfg, err := Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "десериализации")
}
