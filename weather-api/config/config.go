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
	Kafka       KafkaConfig
	Postgres    PostgresConfig
	Redis       RedisConfig
	Minio       MinioConfig
	Scheduler   SchedulerConfig
	Cache       CacheConfig
	Reports     ReportsConfig
	API         APIConfig
	HealthCheck HealthCheckConfig
}

type AppConfig struct {
	Name            string        `mapstructure:"name"`
	Env             string        `mapstructure:"env"`
	LogLevel        string        `mapstructure:"log_level"`
	Port            int           `mapstructure:"port"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type KafkaConfig struct {
	Broker          string        `mapstructure:"broker"`
	Topic           string        `mapstructure:"topic"`
	GroupID         string        `mapstructure:"group_id"`
	ConsumerTimeout time.Duration `mapstructure:"consumer_timeout"`
	MaxRetries      int           `mapstructure:"max_retries"`
	BatchSize       int           `mapstructure:"batch_size"`
}

type PostgresConfig struct {
	CoordinatorHost    string        `mapstructure:"coordinator_host"`
	CoordinatorPort    int           `mapstructure:"coordinator_port"`
	ShardHosts         []string      `mapstructure:"shard_hosts"`
	Port               int           `mapstructure:"port"`
	User               string        `mapstructure:"user"`
	Password           string        `mapstructure:"password"`
	Database           string        `mapstructure:"database"`
	SSLMode            string        `mapstructure:"ssl_mode"`
	MaxConnections     int           `mapstructure:"max_connections"`
	MaxIdleConnections int           `mapstructure:"max_idle_connections"`
	ConnectionTimeout  time.Duration `mapstructure:"connection_timeout"`
	MigrationPath      string        `mapstructure:"migration_path"`
}

type RedisConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	Password           string        `mapstructure:"password"`
	DB                 int           `mapstructure:"db"`
	PoolSize           int           `mapstructure:"pool_size"`
	MinIdleConnections int           `mapstructure:"min_idle_connections"`
	MaxRetries         int           `mapstructure:"max_retries"`
	Timeout            time.Duration `mapstructure:"timeout"`
}

type MinioConfig struct {
	Endpoint  string        `mapstructure:"endpoint"`
	AccessKey string        `mapstructure:"access_key"`
	SecretKey string        `mapstructure:"secret_key"`
	Bucket    string        `mapstructure:"bucket"`
	UseSSL    bool          `mapstructure:"use_ssl"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type SchedulerConfig struct {
	KafkaPollInterval        time.Duration `mapstructure:"kafka_poll_interval"`
	ReportGenerationInterval time.Duration `mapstructure:"report_generation_interval"`
	CleanupInterval          time.Duration `mapstructure:"cleanup_interval"`
	Timeout                  time.Duration `mapstructure:"timeout"`
}

type CacheConfig struct {
	DailyReportTTL   time.Duration `mapstructure:"daily_report_ttl"`
	WeeklyReportTTL  time.Duration `mapstructure:"weekly_report_ttl"`
	MonthlyReportTTL time.Duration `mapstructure:"monthly_report_ttl"`
	MaxCacheSizeMB   int           `mapstructure:"max_cache_size_mb"`
}

type ReportsConfig struct {
	DailyRetentionDays   int      `mapstructure:"daily_retention_days"`
	WeeklyRetentionDays  int      `mapstructure:"weekly_retention_days"`
	MonthlyRetentionDays int      `mapstructure:"monthly_retention_days"`
	IncludeCities        []string `mapstructure:"include_cities"`
}

type APIConfig struct {
	BasePath           string        `mapstructure:"base_path"`
	EnableSwagger      bool          `mapstructure:"enable_swagger"`
	CorsAllowedOrigins []string      `mapstructure:"cors_allowed_origins"`
	RateLimit          int           `mapstructure:"rate_limit"`
	RateLimitWindow    time.Duration `mapstructure:"rate_limit_window"`
}

type HealthCheckConfig struct {
	Timeout      time.Duration `mapstructure:"timeout"`
	Interval     time.Duration `mapstructure:"interval"`
	StartupDelay time.Duration `mapstructure:"startup_delay"`
}

func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/weather-api/")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
		}
	}

	overrideFromEnv(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("ошибка десериализации конфигурации: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("невалидная конфигурация: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "weather-api")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.shutdown_timeout", "30s")

	v.SetDefault("kafka.broker", "kafka:9093")
	v.SetDefault("kafka.topic", "weather-data-raw")
	v.SetDefault("kafka.group_id", "weather-api-group")
	v.SetDefault("kafka.consumer_timeout", "5m")
	v.SetDefault("kafka.max_retries", 3)
	v.SetDefault("kafka.batch_size", 100)

	v.SetDefault("postgres.coordinator_host", "postgres-coordinator")
	v.SetDefault("postgres.coordinator_port", 5432)
	v.SetDefault("postgres.shard_hosts", []string{"postgres-shard-1", "postgres-shard-2", "postgres-shard-3"})
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.user", "weather_user")
	v.SetDefault("postgres.password", "weather_pass")
	v.SetDefault("postgres.database", "weather_db")
	v.SetDefault("postgres.ssl_mode", "disable")
	v.SetDefault("postgres.max_connections", 20)
	v.SetDefault("postgres.max_idle_connections", 5)
	v.SetDefault("postgres.connection_timeout", "30s")
	v.SetDefault("postgres.migration_path", "./migrations")

	v.SetDefault("redis.host", "redis")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "redis_pass")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_connections", 2)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.timeout", "5s")

	v.SetDefault("minio.endpoint", "minio:9000")
	v.SetDefault("minio.access_key", "minioadmin")
	v.SetDefault("minio.secret_key", "minioadmin")
	v.SetDefault("minio.bucket", "weather-reports")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.timeout", "30s")

	v.SetDefault("scheduler.kafka_poll_interval", "5m")
	v.SetDefault("scheduler.report_generation_interval", "1h")
	v.SetDefault("scheduler.cleanup_interval", "24h")
	v.SetDefault("scheduler.timeout", "10m")

	v.SetDefault("cache.daily_report_ttl", "24h")
	v.SetDefault("cache.weekly_report_ttl", "168h")  // 7 дней
	v.SetDefault("cache.monthly_report_ttl", "720h") // 30 дней
	v.SetDefault("cache.max_cache_size_mb", 100)

	v.SetDefault("reports.daily_retention_days", 30)
	v.SetDefault("reports.weekly_retention_days", 90)
	v.SetDefault("reports.monthly_retention_days", 365)
	v.SetDefault("reports.include_cities", []string{
		"524901", "498817", "551487", "1486209", "1496747",
		"2013348", "2023469", "1508291", "511565", "479123",
	})

	v.SetDefault("api.base_path", "/api/v1")
	v.SetDefault("api.enable_swagger", true)
	v.SetDefault("api.cors_allowed_origins", []string{"*"})
	v.SetDefault("api.rate_limit", 100)
	v.SetDefault("api.rate_limit_window", "1m")

	v.SetDefault("healthcheck.timeout", "10s")
	v.SetDefault("healthcheck.interval", "30s")
	v.SetDefault("healthcheck.startup_delay", "30s")
}

func overrideFromEnv(v *viper.Viper) {
	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		v.Set("kafka.broker", broker)
	}
	if topic := os.Getenv("KAFKA_WEATHER_TOPIC"); topic != "" {
		v.Set("kafka.topic", topic)
	}
	if groupID := os.Getenv("KAFKA_GROUP_ID"); groupID != "" {
		v.Set("kafka.group_id", groupID)
	}

	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		v.Set("postgres.coordinator_host", host)
	}
	if shardHosts := os.Getenv("POSTGRES_SHARD_HOSTS"); shardHosts != "" {
		shardList := strings.Split(shardHosts, ",")
		for i, shard := range shardList {
			shardList[i] = strings.TrimSpace(shard)
		}
		v.Set("postgres.shard_hosts", shardList)
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		v.Set("postgres.user", user)
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		v.Set("postgres.password", password)
	}

	if host := os.Getenv("REDIS_HOST"); host != "" {
		v.Set("redis.host", host)
	}
	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		v.Set("redis.password", password)
	}

	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		v.Set("minio.endpoint", endpoint)
	}
	if accessKey := os.Getenv("MINIO_ACCESS_KEY"); accessKey != "" {
		v.Set("minio.access_key", accessKey)
	}
	if secretKey := os.Getenv("MINIO_SECRET_KEY"); secretKey != "" {
		v.Set("minio.secret_key", secretKey)
	}
	if bucket := os.Getenv("MINIO_BUCKET"); bucket != "" {
		v.Set("minio.bucket", bucket)
	}

	if env := os.Getenv("APP_ENV"); env != "" {
		v.Set("app.env", env)
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		v.Set("app.log_level", logLevel)
	}
}

func validateConfig(cfg *Config) error {
	if len(cfg.Kafka.Broker) == 0 {
		return fmt.Errorf("список Kafka брокеров не может быть пустым")
	}
	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("Kafka топик не может быть пустым")
	}
	if cfg.Postgres.CoordinatorHost == "" {
		return fmt.Errorf("PostgreSQL хост не может быть пустым")
	}
	if cfg.Postgres.User == "" {
		return fmt.Errorf("PostgreSQL пользователь не может быть пустым")
	}
	if cfg.Redis.Host == "" {
		return fmt.Errorf("Redis хост не может быть пустым")
	}
	if cfg.Minio.Endpoint == "" {
		return fmt.Errorf("Minio endpoint не может быть пустым")
	}
	if cfg.Minio.Bucket == "" {
		return fmt.Errorf("Minio bucket не может быть пустым")
	}

	if cfg.Scheduler.KafkaPollInterval <= 0 {
		return fmt.Errorf("интервал опроса Kafka должен быть положительным")
	}
	if cfg.Scheduler.ReportGenerationInterval <= 0 {
		return fmt.Errorf("интервал генерации отчетов должен быть положительным")
	}

	if cfg.Cache.DailyReportTTL <= 0 {
		return fmt.Errorf("TTL дневных отчетов должен быть положительным")
	}

	return nil
}
