package models

import (
	"time"
)

type DailyAggregateEntity interface {
	GetID() string
	GetCityID() int
	GetDate() time.Time
	GetAvgTemperature() float64
	GetMaxTemperature() float64
	GetMinTemperature() float64
	GetAvgHumidity() int
	GetAvgPressure() int
	GetAvgWindSpeed() float64
	GetDominantWeather() string
	GetTotalRecords() int
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

type DailyAggregate struct {
	ID              string    `json:"id" db:"id"`
	CityID          int       `json:"city_id" db:"city_id"`
	Date            time.Time `json:"date" db:"date"`
	AvgTemperature  float64   `json:"avg_temperature" db:"avg_temperature"`
	MaxTemperature  float64   `json:"max_temperature" db:"max_temperature"`
	MinTemperature  float64   `json:"min_temperature" db:"min_temperature"`
	AvgHumidity     int       `json:"avg_humidity" db:"avg_humidity"`
	AvgPressure     int       `json:"avg_pressure" db:"avg_pressure"`
	AvgWindSpeed    float64   `json:"avg_wind_speed" db:"avg_wind_speed"`
	DominantWeather string    `json:"dominant_weather" db:"dominant_weather"`
	TotalRecords    int       `json:"total_records" db:"total_records"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

func (d *DailyAggregate) GetID() string              { return d.ID }
func (d *DailyAggregate) GetCityID() int             { return d.CityID }
func (d *DailyAggregate) GetDate() time.Time         { return d.Date }
func (d *DailyAggregate) GetAvgTemperature() float64 { return d.AvgTemperature }
func (d *DailyAggregate) GetMaxTemperature() float64 { return d.MaxTemperature }
func (d *DailyAggregate) GetMinTemperature() float64 { return d.MinTemperature }
func (d *DailyAggregate) GetAvgHumidity() int        { return d.AvgHumidity }
func (d *DailyAggregate) GetAvgPressure() int        { return d.AvgPressure }
func (d *DailyAggregate) GetAvgWindSpeed() float64   { return d.AvgWindSpeed }
func (d *DailyAggregate) GetDominantWeather() string { return d.DominantWeather }
func (d *DailyAggregate) GetTotalRecords() int       { return d.TotalRecords }
func (d *DailyAggregate) GetCreatedAt() time.Time    { return d.CreatedAt }
func (d *DailyAggregate) GetUpdatedAt() time.Time    { return d.UpdatedAt }
