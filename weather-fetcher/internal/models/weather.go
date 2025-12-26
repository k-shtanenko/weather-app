package models

import (
	"time"
)

type WeatherEntity interface {
	Validate() error
	ToMap() map[string]interface{}
	GetCityID() int
	GetSource() string
}

type Weather struct {
	CityID      int       `json:"city_id"`
	CityName    string    `json:"city_name"`
	Country     string    `json:"country"`
	Temperature float64   `json:"temperature"`
	FeelsLike   float64   `json:"feels_like"`
	Humidity    int       `json:"humidity"`
	Pressure    int       `json:"pressure"`
	WindSpeed   float64   `json:"wind_speed"`
	WindDeg     int       `json:"wind_deg"`
	Clouds      int       `json:"clouds"`
	Weather     string    `json:"weather"`
	WeatherIcon string    `json:"weather_icon"`
	Visibility  int       `json:"visibility"`
	Sunrise     time.Time `json:"sunrise"`
	Sunset      time.Time `json:"sunset"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
}

func (w *Weather) GetCityID() int    { return w.CityID }
func (w *Weather) GetSource() string { return w.Source }

func (w *Weather) Validate() error {
	if w.CityID == 0 {
		return ErrInvalidCityID
	}
	if w.CityName == "" {
		return ErrInvalidCityName
	}
	if w.Humidity < 0 || w.Humidity > 100 {
		return ErrInvalidHumidity
	}
	if w.Pressure < 800 || w.Pressure > 1200 {
		return ErrInvalidPressure
	}
	if w.WindSpeed < 0 {
		return ErrInvalidWindSpeed
	}
	if w.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	return nil
}

func (w *Weather) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"city_id":      w.CityID,
		"city_name":    w.CityName,
		"country":      w.Country,
		"temperature":  w.Temperature,
		"feels_like":   w.FeelsLike,
		"humidity":     w.Humidity,
		"pressure":     w.Pressure,
		"wind_speed":   w.WindSpeed,
		"wind_deg":     w.WindDeg,
		"clouds":       w.Clouds,
		"weather":      w.Weather,
		"weather_icon": w.WeatherIcon,
		"visibility":   w.Visibility,
		"sunrise":      w.Sunrise.Format(time.RFC3339),
		"sunset":       w.Sunset.Format(time.RFC3339),
		"timestamp":    w.Timestamp.Format(time.RFC3339),
		"source":       w.Source,
	}
}

func FromMap(data map[string]interface{}) (*Weather, error) {
	timestamp, _ := time.Parse(time.RFC3339, data["timestamp"].(string))
	sunrise, _ := time.Parse(time.RFC3339, data["sunrise"].(string))
	sunset, _ := time.Parse(time.RFC3339, data["sunset"].(string))

	weather := &Weather{
		CityID:      int(data["city_id"].(float64)),
		CityName:    data["city_name"].(string),
		Country:     data["country"].(string),
		Temperature: data["temperature"].(float64),
		FeelsLike:   data["feels_like"].(float64),
		Humidity:    int(data["humidity"].(float64)),
		Pressure:    int(data["pressure"].(float64)),
		WindSpeed:   data["wind_speed"].(float64),
		WindDeg:     int(data["wind_deg"].(float64)),
		Clouds:      int(data["clouds"].(float64)),
		Weather:     data["weather"].(string),
		WeatherIcon: data["weather_icon"].(string),
		Visibility:  int(data["visibility"].(float64)),
		Sunrise:     sunrise,
		Sunset:      sunset,
		Timestamp:   timestamp,
		Source:      data["source"].(string),
	}

	if err := weather.Validate(); err != nil {
		return nil, err
	}

	return weather, nil
}

var (
	ErrInvalidCityID    = ValidationError{Field: "city_id", Reason: "не может быть пустым"}
	ErrInvalidCityName  = ValidationError{Field: "city_name", Reason: "не может быть пустым"}
	ErrInvalidHumidity  = ValidationError{Field: "humidity", Reason: "должно быть между 0 и 100"}
	ErrInvalidPressure  = ValidationError{Field: "pressure", Reason: "должно быть между 800 и 1200"}
	ErrInvalidWindSpeed = ValidationError{Field: "wind_speed", Reason: "не может быть отрицательным"}
	ErrInvalidTimestamp = ValidationError{Field: "timestamp", Reason: "не может быть нулевым"}
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Reason
}
