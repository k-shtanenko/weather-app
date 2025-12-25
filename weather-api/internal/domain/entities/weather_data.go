package entities

import (
	"time"
)

type WeatherDataEntity interface {
	GetID() string
	GetCityID() int
	GetCityName() string
	GetCountry() string
	GetTemperature() float64
	GetFeelsLike() float64
	GetHumidity() int
	GetPressure() int
	GetWindSpeed() float64
	GetWindDeg() int
	GetClouds() int
	GetWeatherDescription() string
	GetWeatherIcon() string
	GetVisibility() int
	GetSunrise() time.Time
	GetSunset() time.Time
	GetRecordedAt() time.Time
	GetSource() string
	GetCreatedAt() time.Time
	Validate() error
	ToMap() map[string]interface{}
}

type WeatherData struct {
	ID                 string    `json:"id" db:"id"`
	CityID             int       `json:"city_id" db:"city_id"`
	CityName           string    `json:"city_name" db:"city_name"`
	Country            string    `json:"country" db:"country"`
	Temperature        float64   `json:"temperature" db:"temperature"`
	FeelsLike          float64   `json:"feels_like" db:"feels_like"`
	Humidity           int       `json:"humidity" db:"humidity"`
	Pressure           int       `json:"pressure" db:"pressure"`
	WindSpeed          float64   `json:"wind_speed" db:"wind_speed"`
	WindDeg            int       `json:"wind_deg" db:"wind_deg"`
	Clouds             int       `json:"clouds" db:"clouds"`
	WeatherDescription string    `json:"weather_description" db:"weather_description"`
	WeatherIcon        string    `json:"weather_icon" db:"weather_icon"`
	Visibility         int       `json:"visibility" db:"visibility"`
	Sunrise            time.Time `json:"sunrise" db:"sunrise"`
	Sunset             time.Time `json:"sunset" db:"sunset"`
	RecordedAt         time.Time `json:"recorded_at" db:"recorded_at"`
	Source             string    `json:"source" db:"source"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

func (w *WeatherData) GetID() string                 { return w.ID }
func (w *WeatherData) GetCityID() int                { return w.CityID }
func (w *WeatherData) GetCityName() string           { return w.CityName }
func (w *WeatherData) GetCountry() string            { return w.Country }
func (w *WeatherData) GetTemperature() float64       { return w.Temperature }
func (w *WeatherData) GetFeelsLike() float64         { return w.FeelsLike }
func (w *WeatherData) GetHumidity() int              { return w.Humidity }
func (w *WeatherData) GetPressure() int              { return w.Pressure }
func (w *WeatherData) GetWindSpeed() float64         { return w.WindSpeed }
func (w *WeatherData) GetWindDeg() int               { return w.WindDeg }
func (w *WeatherData) GetClouds() int                { return w.Clouds }
func (w *WeatherData) GetWeatherDescription() string { return w.WeatherDescription }
func (w *WeatherData) GetWeatherIcon() string        { return w.WeatherIcon }
func (w *WeatherData) GetVisibility() int            { return w.Visibility }
func (w *WeatherData) GetSunrise() time.Time         { return w.Sunrise }
func (w *WeatherData) GetSunset() time.Time          { return w.Sunset }
func (w *WeatherData) GetRecordedAt() time.Time      { return w.RecordedAt }
func (w *WeatherData) GetSource() string             { return w.Source }
func (w *WeatherData) GetCreatedAt() time.Time       { return w.CreatedAt }

func (w *WeatherData) Validate() error {
	if w.CityID == 0 {
		return ValidationError{Field: "city_id", Reason: "не может быть пустым"}
	}
	if w.CityName == "" {
		return ValidationError{Field: "city_name", Reason: "не может быть пустым"}
	}
	if w.Temperature < -100 || w.Temperature > 100 {
		return ValidationError{Field: "temperature", Reason: "должно быть между -100 и 100"}
	}
	if w.RecordedAt.IsZero() {
		return ValidationError{Field: "recorded_at", Reason: "не может быть нулевым"}
	}
	return nil
}

func (w *WeatherData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":                  w.ID,
		"city_id":             w.CityID,
		"city_name":           w.CityName,
		"country":             w.Country,
		"temperature":         w.Temperature,
		"feels_like":          w.FeelsLike,
		"humidity":            w.Humidity,
		"pressure":            w.Pressure,
		"wind_speed":          w.WindSpeed,
		"wind_deg":            w.WindDeg,
		"clouds":              w.Clouds,
		"weather_description": w.WeatherDescription,
		"weather_icon":        w.WeatherIcon,
		"visibility":          w.Visibility,
		"sunrise":             w.Sunrise.Format(time.RFC3339),
		"sunset":              w.Sunset.Format(time.RFC3339),
		"recorded_at":         w.RecordedAt.Format(time.RFC3339),
		"source":              w.Source,
		"created_at":          w.CreatedAt.Format(time.RFC3339),
	}
}
