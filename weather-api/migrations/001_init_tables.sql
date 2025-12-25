CREATE TABLE IF NOT EXISTS weather_data (
    id TEXT NOT NULL,
    city_id INTEGER NOT NULL,
    city_name VARCHAR(100) NOT NULL,
    country VARCHAR(10) NOT NULL,
    temperature DECIMAL(5,2) NOT NULL,
    feels_like DECIMAL(5,2),
    humidity INTEGER CHECK (humidity >= 0 AND humidity <= 100),
    pressure INTEGER CHECK (pressure >= 800 AND pressure <= 1200),
    wind_speed DECIMAL(5,2) CHECK (wind_speed >= 0),
    wind_deg INTEGER CHECK (wind_deg >= 0 AND wind_deg < 360),
    clouds INTEGER CHECK (clouds >= 0 AND clouds <= 100),
    weather_description VARCHAR(100),
    weather_icon VARCHAR(10),
    visibility INTEGER,
    sunrise TIMESTAMP WITH TIME ZONE,
    sunset TIMESTAMP WITH TIME ZONE,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    source VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    PRIMARY KEY (id, city_id)
) PARTITION BY HASH(city_id);

CREATE TABLE IF NOT EXISTS weather_data_partition_0 
PARTITION OF weather_data 
FOR VALUES WITH (MODULUS 3, REMAINDER 0);

CREATE TABLE IF NOT EXISTS weather_data_partition_1 
PARTITION OF weather_data 
FOR VALUES WITH (MODULUS 3, REMAINDER 1);

CREATE TABLE IF NOT EXISTS weather_data_partition_2 
PARTITION OF weather_data 
FOR VALUES WITH (MODULUS 3, REMAINDER 2);

CREATE INDEX IF NOT EXISTS idx_partition_0_recorded_at ON weather_data_partition_0(recorded_at);
CREATE INDEX IF NOT EXISTS idx_partition_1_recorded_at ON weather_data_partition_1(recorded_at);
CREATE INDEX IF NOT EXISTS idx_partition_2_recorded_at ON weather_data_partition_2(recorded_at);

CREATE TABLE IF NOT EXISTS daily_aggregates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id INTEGER NOT NULL,
    date DATE NOT NULL,
    avg_temperature DECIMAL(5,2),
    max_temperature DECIMAL(5,2),
    min_temperature DECIMAL(5,2),
    avg_humidity INTEGER,
    avg_pressure INTEGER,
    avg_wind_speed DECIMAL(5,2),
    dominant_weather VARCHAR(100),
    total_records INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(city_id, date)
);

CREATE INDEX IF NOT EXISTS idx_aggregates_city_date ON daily_aggregates(city_id, date);
CREATE INDEX IF NOT EXISTS idx_aggregates_date ON daily_aggregates(date);

CREATE TABLE IF NOT EXISTS excel_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type VARCHAR(20) NOT NULL CHECK (report_type IN ('daily', 'weekly', 'monthly')),
    city_id INTEGER,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    download_url VARCHAR(500),
    checksum VARCHAR(64),
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_reports_type_city ON excel_reports(report_type, city_id);
CREATE INDEX IF NOT EXISTS idx_reports_period ON excel_reports(period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_reports_generated ON excel_reports(generated_at);
CREATE INDEX IF NOT EXISTS idx_reports_city_period ON excel_reports(city_id, period_start, period_end);

CREATE TABLE IF NOT EXISTS api_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key VARCHAR(255) UNIQUE NOT NULL,
    cache_type VARCHAR(50) NOT NULL,
    data BYTEA NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    file_name VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    hit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cache_key ON api_cache(cache_key);
CREATE INDEX IF NOT EXISTS idx_cache_expires ON api_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_cache_type ON api_cache(cache_type);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_daily_aggregates_updated_at 
    BEFORE UPDATE ON daily_aggregates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE VIEW weather_summary AS
SELECT 
    city_id,
    city_name,
    COUNT(*) as total_records,
    MIN(recorded_at) as first_record,
    MAX(recorded_at) as last_record,
    AVG(temperature) as avg_temperature,
    MIN(temperature) as min_temperature,
    MAX(temperature) as max_temperature,
    AVG(humidity) as avg_humidity,
    AVG(pressure) as avg_pressure
FROM weather_data
GROUP BY city_id, city_name;