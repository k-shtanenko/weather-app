package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-shtanenko/weather-app/weather-api/config"
	"github.com/k-shtanenko/weather-app/weather-api/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-api/internal/models"
)

type WeatherRepository interface {
	Save(ctx context.Context, data models.WeatherDataEntity) error
	FindByCityIDAndDateRange(ctx context.Context, cityID int, start, end time.Time) ([]models.WeatherDataEntity, error)
	GetDailyAggregate(ctx context.Context, cityID int, date time.Time) (models.DailyAggregateEntity, error)
	SaveDailyAggregate(ctx context.Context, aggregate models.DailyAggregateEntity) error
	UpdateDailyAggregate(ctx context.Context, aggregate models.DailyAggregateEntity) error
	GetCitiesWithData(ctx context.Context) ([]int, error)
	CleanupOldData(ctx context.Context, retentionDays int) error
	HealthCheck(ctx context.Context) error
	Close() error
}

type ReportRepository interface {
	SaveReport(ctx context.Context, report models.ExcelReportEntity) error
	FindReportByTypeAndPeriod(ctx context.Context, reportType models.ReportType, cityID int, start, end time.Time) (models.ExcelReportEntity, error)
	FindReportByID(ctx context.Context, reportID string) (models.ExcelReportEntity, error)
	CleanupExpiredReports(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

type PostgresWeatherRepository struct {
	bucketManager *BucketManager
	logger        logger.Logger
}

type PostgresReportRepository struct {
	bucketManager *BucketManager
	logger        logger.Logger
}

func NewPostgresWeatherRepository(coordinatorHost string, shardHosts []string, port int, user, password, database string) (*PostgresWeatherRepository, error) {
	log := logger.New("info", "development").WithField("component", "postgres_weather_repository")

	cfg := config.PostgresConfig{
		CoordinatorHost:    coordinatorHost,
		ShardHosts:         shardHosts,
		Port:               port,
		User:               user,
		Password:           password,
		Database:           database,
		SSLMode:            "disable",
		MaxConnections:     20,
		MaxIdleConnections: 5,
		ConnectionTimeout:  30 * time.Second,
	}

	bucketManager, err := NewBucketManager(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket manager: %w", err)
	}

	log.Infof("Created weather repository with %d buckets", bucketManager.totalBuckets)
	return &PostgresWeatherRepository{
		bucketManager: bucketManager,
		logger:        log,
	}, nil
}

func (r *PostgresWeatherRepository) getPool(cityID int) *pgxpool.Pool {
	return r.bucketManager.GetPoolForUser(cityID)
}

func (r *PostgresWeatherRepository) Save(ctx context.Context, data models.WeatherDataEntity) error {
	pool := r.getPool(data.GetCityID())

	query := `
		INSERT INTO weather_data (
			id, city_id, city_name, country, temperature, feels_like,
			humidity, pressure, wind_speed, wind_deg, clouds,
			weather_description, weather_icon, visibility,
			sunrise, sunset, recorded_at, source, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (id, city_id) DO UPDATE SET
			temperature = EXCLUDED.temperature,
			humidity = EXCLUDED.humidity,
			pressure = EXCLUDED.pressure,
			wind_speed = EXCLUDED.wind_speed,
			weather_description = EXCLUDED.weather_description
	`

	_, err := pool.Exec(ctx, query,
		data.GetID(),
		data.GetCityID(),
		data.GetCityName(),
		data.GetCountry(),
		data.GetTemperature(),
		data.GetFeelsLike(),
		data.GetHumidity(),
		data.GetPressure(),
		data.GetWindSpeed(),
		data.GetWindDeg(),
		data.GetClouds(),
		data.GetWeatherDescription(),
		data.GetWeatherIcon(),
		data.GetVisibility(),
		data.GetSunrise(),
		data.GetSunset(),
		data.GetRecordedAt(),
		data.GetSource(),
		data.GetCreatedAt(),
	)

	return err
}

func (r *PostgresWeatherRepository) FindByCityIDAndDateRange(ctx context.Context, cityID int, start, end time.Time) ([]models.WeatherDataEntity, error) {
	pool := r.getPool(cityID)

	query := `
		SELECT id, city_id, city_name, country, temperature, feels_like,
			humidity, pressure, wind_speed, wind_deg, clouds,
			weather_description, weather_icon, visibility,
			sunrise, sunset, recorded_at, source, created_at
		FROM weather_data
		WHERE city_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
		ORDER BY recorded_at ASC
	`

	rows, err := pool.Query(ctx, query, cityID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query weather data: %w", err)
	}
	defer rows.Close()

	var results []models.WeatherDataEntity
	for rows.Next() {
		var data models.WeatherData
		err := rows.Scan(
			&data.ID,
			&data.CityID,
			&data.CityName,
			&data.Country,
			&data.Temperature,
			&data.FeelsLike,
			&data.Humidity,
			&data.Pressure,
			&data.WindSpeed,
			&data.WindDeg,
			&data.Clouds,
			&data.WeatherDescription,
			&data.WeatherIcon,
			&data.Visibility,
			&data.Sunrise,
			&data.Sunset,
			&data.RecordedAt,
			&data.Source,
			&data.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weather data: %w", err)
		}
		results = append(results, &data)
	}

	return results, nil
}

func (r *PostgresWeatherRepository) GetDailyAggregate(ctx context.Context, cityID int, date time.Time) (models.DailyAggregateEntity, error) {
	pool := r.getPool(cityID)

	query := `
		SELECT id, city_id, date, avg_temperature, max_temperature, min_temperature,
			avg_humidity, avg_pressure, avg_wind_speed, dominant_weather,
			total_records, created_at, updated_at
		FROM daily_aggregates
		WHERE city_id = $1 AND date = $2
	`

	var aggregate models.DailyAggregate
	err := pool.QueryRow(ctx, query, cityID, date).Scan(
		&aggregate.ID,
		&aggregate.CityID,
		&aggregate.Date,
		&aggregate.AvgTemperature,
		&aggregate.MaxTemperature,
		&aggregate.MinTemperature,
		&aggregate.AvgHumidity,
		&aggregate.AvgPressure,
		&aggregate.AvgWindSpeed,
		&aggregate.DominantWeather,
		&aggregate.TotalRecords,
		&aggregate.CreatedAt,
		&aggregate.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get daily aggregate: %w", err)
	}

	return &aggregate, nil
}

func (r *PostgresWeatherRepository) SaveDailyAggregate(ctx context.Context, aggregate models.DailyAggregateEntity) error {
	pool := r.getPool(aggregate.GetCityID())

	query := `
		INSERT INTO daily_aggregates (
			id, city_id, date, avg_temperature, max_temperature, min_temperature,
			avg_humidity, avg_pressure, avg_wind_speed, dominant_weather,
			total_records, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := pool.Exec(ctx, query,
		aggregate.GetID(),
		aggregate.GetCityID(),
		aggregate.GetDate(),
		aggregate.GetAvgTemperature(),
		aggregate.GetMaxTemperature(),
		aggregate.GetMinTemperature(),
		aggregate.GetAvgHumidity(),
		aggregate.GetAvgPressure(),
		aggregate.GetAvgWindSpeed(),
		aggregate.GetDominantWeather(),
		aggregate.GetTotalRecords(),
		aggregate.GetCreatedAt(),
		aggregate.GetUpdatedAt(),
	)

	return err
}

func (r *PostgresWeatherRepository) UpdateDailyAggregate(ctx context.Context, aggregate models.DailyAggregateEntity) error {
	pool := r.getPool(aggregate.GetCityID())

	query := `
		UPDATE daily_aggregates SET
			avg_temperature = $1,
			max_temperature = $2,
			min_temperature = $3,
			avg_humidity = $4,
			avg_pressure = $5,
			avg_wind_speed = $6,
			dominant_weather = $7,
			total_records = $8,
			updated_at = NOW()
		WHERE city_id = $9 AND date = $10
	`

	result, err := pool.Exec(ctx, query,
		aggregate.GetAvgTemperature(),
		aggregate.GetMaxTemperature(),
		aggregate.GetMinTemperature(),
		aggregate.GetAvgHumidity(),
		aggregate.GetAvgPressure(),
		aggregate.GetAvgWindSpeed(),
		aggregate.GetDominantWeather(),
		aggregate.GetTotalRecords(),
		aggregate.GetCityID(),
		aggregate.GetDate(),
	)

	if err != nil {
		return fmt.Errorf("failed to update daily aggregate: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("daily aggregate not found for update")
	}

	return nil
}

func (r *PostgresWeatherRepository) GetCitiesWithData(ctx context.Context) ([]int, error) {
	var allCities []int
	seenCities := make(map[int]bool)

	for i := 0; i < r.bucketManager.totalBuckets; i++ {
		pool := r.bucketManager.buckets[i].Pool

		query := `
			SELECT DISTINCT city_id FROM weather_data
			WHERE recorded_at >= NOW() - INTERVAL '30 days'
		`

		rows, err := pool.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to query cities from bucket %d: %w", i, err)
		}

		for rows.Next() {
			var cityID int
			if err := rows.Scan(&cityID); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan city_id: %w", err)
			}
			if !seenCities[cityID] {
				seenCities[cityID] = true
				allCities = append(allCities, cityID)
			}
		}
		rows.Close()
	}

	return allCities, nil
}

func (r *PostgresWeatherRepository) CleanupOldData(ctx context.Context, retentionDays int) error {
	for i := 0; i < r.bucketManager.totalBuckets; i++ {
		pool := r.bucketManager.buckets[i].Pool
		query := `DELETE FROM weather_data WHERE recorded_at < NOW() - INTERVAL '1 day' * $1`
		_, err := pool.Exec(ctx, query, retentionDays)
		if err != nil {
			return fmt.Errorf("failed to cleanup bucket %d: %w", i, err)
		}

		aggQuery := `DELETE FROM daily_aggregates WHERE date < NOW() - INTERVAL '1 day' * $1`
		_, err = pool.Exec(ctx, aggQuery, retentionDays*2)
		if err != nil {
			return fmt.Errorf("failed to cleanup aggregates in bucket %d: %w", i, err)
		}
	}

	return nil
}

func (r *PostgresWeatherRepository) HealthCheck(ctx context.Context) error {
	for i := 0; i < r.bucketManager.totalBuckets; i++ {
		pool := r.bucketManager.buckets[i].Pool
		if err := pool.Ping(ctx); err != nil {
			return fmt.Errorf("bucket %d ping failed: %w", i, err)
		}
	}
	return nil
}

func (r *PostgresWeatherRepository) Close() error {
	r.bucketManager.Close()
	return nil
}

func NewPostgresReportRepository(host string, port int, user, password, database string) (*PostgresReportRepository, error) {
	log := logger.New("info", "development").WithField("component", "postgres_report_repository")

	cfg := config.PostgresConfig{
		CoordinatorHost:    host,
		ShardHosts:         []string{host},
		Port:               port,
		User:               user,
		Password:           password,
		Database:           database,
		SSLMode:            "disable",
		MaxConnections:     20,
		MaxIdleConnections: 5,
		ConnectionTimeout:  30 * time.Second,
	}

	bucketManager, err := NewBucketManager(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket manager: %w", err)
	}

	log.Info("Created report repository")
	return &PostgresReportRepository{
		bucketManager: bucketManager,
		logger:        log,
	}, nil
}

func (r *PostgresReportRepository) getPool() *pgxpool.Pool {
	return r.bucketManager.GetPoolForUser(0)
}

func (r *PostgresReportRepository) SaveReport(ctx context.Context, report models.ExcelReportEntity) error {
	pool := r.getPool()

	query := `
		INSERT INTO excel_reports (
			id, report_type, city_id, period_start, period_end,
			file_name, file_size, storage_path, download_url,
			checksum, generated_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := pool.Exec(ctx, query,
		report.GetID(),
		report.GetReportType(),
		report.GetCityID(),
		report.GetPeriodStart(),
		report.GetPeriodEnd(),
		report.GetFileName(),
		report.GetFileSize(),
		report.GetStoragePath(),
		report.GetDownloadURL(),
		report.GetChecksum(),
		report.GetGeneratedAt(),
		report.GetExpiresAt(),
	)

	return err
}

func (r *PostgresReportRepository) FindReportByTypeAndPeriod(ctx context.Context, reportType models.ReportType, cityID int, start, end time.Time) (models.ExcelReportEntity, error) {
	pool := r.getPool()

	query := `
		SELECT id, report_type, city_id, period_start, period_end,
			file_name, file_size, storage_path, download_url,
			checksum, generated_at, expires_at
		FROM excel_reports
		WHERE report_type = $1 AND city_id = $2
			AND period_start = $3 AND period_end = $4
		ORDER BY generated_at DESC
		LIMIT 1
	`

	var report models.ExcelReport
	err := pool.QueryRow(ctx, query, reportType, cityID, start, end).Scan(
		&report.ID,
		&report.ReportType,
		&report.CityID,
		&report.PeriodStart,
		&report.PeriodEnd,
		&report.FileName,
		&report.FileSize,
		&report.StoragePath,
		&report.DownloadURL,
		&report.Checksum,
		&report.GeneratedAt,
		&report.ExpiresAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find report: %w", err)
	}

	return &report, nil
}

func (r *PostgresReportRepository) FindReportByID(ctx context.Context, reportID string) (models.ExcelReportEntity, error) {
	pool := r.getPool()

	query := `
		SELECT id, report_type, city_id, period_start, period_end,
			file_name, file_size, storage_path, download_url,
			checksum, generated_at, expires_at
		FROM excel_reports
		WHERE id = $1
	`

	var report models.ExcelReport
	err := pool.QueryRow(ctx, query, reportID).Scan(
		&report.ID,
		&report.ReportType,
		&report.CityID,
		&report.PeriodStart,
		&report.PeriodEnd,
		&report.FileName,
		&report.FileSize,
		&report.StoragePath,
		&report.DownloadURL,
		&report.Checksum,
		&report.GeneratedAt,
		&report.ExpiresAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find report by ID: %w", err)
	}

	return &report, nil
}

func (r *PostgresReportRepository) CleanupExpiredReports(ctx context.Context) error {
	pool := r.getPool()
	query := `DELETE FROM excel_reports WHERE expires_at < NOW()`
	_, err := pool.Exec(ctx, query)
	return err
}

func (r *PostgresReportRepository) HealthCheck(ctx context.Context) error {
	return r.getPool().Ping(ctx)
}

func (r *PostgresReportRepository) Close() error {
	r.bucketManager.Close()
	return nil
}
