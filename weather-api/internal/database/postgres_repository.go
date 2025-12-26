package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	coordinatorPool *pgxpool.Pool
	shardPools      []*pgxpool.Pool
	logger          logger.Logger
}

type PostgresReportRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewPostgresWeatherRepository(coordinatorHost string, shardHosts []string, port int, user, password, database string) (*PostgresWeatherRepository, error) {
	log := logger.New("info", "development").WithField("component", "postgres_weather_repository")

	coordinatorConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, coordinatorHost, port, database)

	coordinatorPool, err := pgxpool.New(context.Background(), coordinatorConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create coordinator pool: %w", err)
	}

	shardPools := make([]*pgxpool.Pool, len(shardHosts))
	for i, shardHost := range shardHosts {
		shardConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			user, password, shardHost, port, database)

		shardPool, err := pgxpool.New(context.Background(), shardConnStr)
		if err != nil {
			return nil, fmt.Errorf("failed to create shard %d pool: %w", i, err)
		}
		shardPools[i] = shardPool
	}

	if err := coordinatorPool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("coordinator ping failed: %w", err)
	}

	for i, pool := range shardPools {
		if err := pool.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("shard %d ping failed: %w", i, err)
		}
	}

	log.Infof("Created weather repository with %d shards", len(shardPools))
	return &PostgresWeatherRepository{
		coordinatorPool: coordinatorPool,
		shardPools:      shardPools,
		logger:          log,
	}, nil
}

func (r *PostgresWeatherRepository) getShardPool(cityID int) *pgxpool.Pool {
	shardIndex := cityID % len(r.shardPools)
	return r.shardPools[shardIndex]
}

func (r *PostgresWeatherRepository) Save(ctx context.Context, data models.WeatherDataEntity) error {
	pool := r.getShardPool(data.GetCityID())

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
	pool := r.getShardPool(cityID)

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
	query := `
		SELECT id, city_id, date, avg_temperature, max_temperature, min_temperature,
			avg_humidity, avg_pressure, avg_wind_speed, dominant_weather,
			total_records, created_at, updated_at
		FROM daily_aggregates
		WHERE city_id = $1 AND date = $2
	`

	var aggregate models.DailyAggregate
	err := r.coordinatorPool.QueryRow(ctx, query, cityID, date).Scan(
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
	query := `
		INSERT INTO daily_aggregates (
			id, city_id, date, avg_temperature, max_temperature, min_temperature,
			avg_humidity, avg_pressure, avg_wind_speed, dominant_weather,
			total_records, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.coordinatorPool.Exec(ctx, query,
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

	result, err := r.coordinatorPool.Exec(ctx, query,
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
	query := `
		SELECT DISTINCT city_id FROM weather_data
		WHERE recorded_at >= NOW() - INTERVAL '30 days'
		ORDER BY city_id
	`

	rows, err := r.coordinatorPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cities: %w", err)
	}
	defer rows.Close()

	var cities []int
	for rows.Next() {
		var cityID int
		if err := rows.Scan(&cityID); err != nil {
			return nil, fmt.Errorf("failed to scan city_id: %w", err)
		}
		cities = append(cities, cityID)
	}

	return cities, nil
}

func (r *PostgresWeatherRepository) CleanupOldData(ctx context.Context, retentionDays int) error {
	for i, pool := range r.shardPools {
		query := `DELETE FROM weather_data WHERE recorded_at < NOW() - INTERVAL '1 day' * $1`
		_, err := pool.Exec(ctx, query, retentionDays)
		if err != nil {
			return fmt.Errorf("failed to cleanup shard %d: %w", i, err)
		}
	}

	aggQuery := `DELETE FROM daily_aggregates WHERE date < NOW() - INTERVAL '1 day' * $1`
	_, err := r.coordinatorPool.Exec(ctx, aggQuery, retentionDays*2)
	return err
}

func (r *PostgresWeatherRepository) HealthCheck(ctx context.Context) error {
	if err := r.coordinatorPool.Ping(ctx); err != nil {
		return fmt.Errorf("coordinator ping failed: %w", err)
	}

	for i, pool := range r.shardPools {
		if err := pool.Ping(ctx); err != nil {
			return fmt.Errorf("shard %d ping failed: %w", i, err)
		}
	}

	return nil
}

func (r *PostgresWeatherRepository) Close() error {
	r.coordinatorPool.Close()
	for _, pool := range r.shardPools {
		pool.Close()
	}
	return nil
}

func NewPostgresReportRepository(host string, port int, user, password, database string) (*PostgresReportRepository, error) {
	log := logger.New("info", "development").WithField("component", "postgres_report_repository")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, database)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	log.Info("Created report repository")
	return &PostgresReportRepository{
		pool:   pool,
		logger: log,
	}, nil
}

func (r *PostgresReportRepository) SaveReport(ctx context.Context, report models.ExcelReportEntity) error {
	query := `
		INSERT INTO excel_reports (
			id, report_type, city_id, period_start, period_end,
			file_name, file_size, storage_path, download_url,
			checksum, generated_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(ctx, query,
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
	err := r.pool.QueryRow(ctx, query, reportType, cityID, start, end).Scan(
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
	query := `
		SELECT id, report_type, city_id, period_start, period_end,
			file_name, file_size, storage_path, download_url,
			checksum, generated_at, expires_at
		FROM excel_reports
		WHERE id = $1
	`

	var report models.ExcelReport
	err := r.pool.QueryRow(ctx, query, reportID).Scan(
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
	query := `DELETE FROM excel_reports WHERE expires_at < NOW()`
	_, err := r.pool.Exec(ctx, query)
	return err
}

func (r *PostgresReportRepository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *PostgresReportRepository) Close() error {
	r.pool.Close()
	return nil
}
