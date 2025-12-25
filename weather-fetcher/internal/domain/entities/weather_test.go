package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeather_Validate(t *testing.T) {
	now := time.Now()

	t.Run("valid weather", func(t *testing.T) {
		weather := &Weather{
			CityID:      123,
			CityName:    "Test City",
			Country:     "RU",
			Temperature: 20.5,
			FeelsLike:   19.8,
			Humidity:    65,
			Pressure:    1013,
			WindSpeed:   5.2,
			WindDeg:     180,
			Clouds:      40,
			Weather:     "cloudy",
			WeatherIcon: "04d",
			Visibility:  10000,
			Sunrise:     now,
			Sunset:      now.Add(12 * time.Hour),
			Timestamp:   now,
			Source:      "test",
		}

		err := weather.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid city ID", func(t *testing.T) {
		weather := &Weather{
			CityID:    0,
			CityName:  "Test City",
			Pressure:  1013,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "city_id: не может быть пустым", err.Error())
	})

	t.Run("invalid city name", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "",
			Pressure:  1013,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "city_name: не может быть пустым", err.Error())
	})

	t.Run("invalid humidity too low", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "Test City",
			Pressure:  1013,
			Humidity:  -1,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "humidity: должно быть между 0 и 100", err.Error())
	})

	t.Run("invalid humidity too high", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "Test City",
			Pressure:  1013,
			Humidity:  101,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "humidity: должно быть между 0 и 100", err.Error())
	})

	t.Run("invalid pressure too low", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "Test City",
			Pressure:  799,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "pressure: должно быть между 800 и 1200", err.Error())
	})

	t.Run("invalid pressure too high", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "Test City",
			Pressure:  1201,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "pressure: должно быть между 800 и 1200", err.Error())
	})

	t.Run("invalid wind speed", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "Test City",
			Pressure:  1013,
			WindSpeed: -1.0,
			Timestamp: now,
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "wind_speed: не может быть отрицательным", err.Error())
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		weather := &Weather{
			CityID:    123,
			CityName:  "Test City",
			Pressure:  1013,
			Timestamp: time.Time{},
			Source:    "test",
		}

		err := weather.Validate()
		assert.Error(t, err)
		assert.Equal(t, "timestamp: не может быть нулевым", err.Error())
	})
}

func TestWeather_GetCityID(t *testing.T) {
	weather := &Weather{CityID: 123}
	assert.Equal(t, 123, weather.GetCityID())
}

func TestWeather_GetSource(t *testing.T) {
	weather := &Weather{Source: "test"}
	assert.Equal(t, "test", weather.GetSource())
}

func TestWeather_ToMap(t *testing.T) {
	now := time.Now()
	sunrise := now.Add(-6 * time.Hour)
	sunset := now.Add(6 * time.Hour)

	weather := &Weather{
		CityID:      123,
		CityName:    "Test City",
		Country:     "RU",
		Temperature: 20.5,
		FeelsLike:   19.8,
		Humidity:    65,
		Pressure:    1013,
		WindSpeed:   5.2,
		WindDeg:     180,
		Clouds:      40,
		Weather:     "cloudy",
		WeatherIcon: "04d",
		Visibility:  10000,
		Sunrise:     sunrise,
		Sunset:      sunset,
		Timestamp:   now,
		Source:      "test",
	}

	result := weather.ToMap()

	assert.Equal(t, 123, result["city_id"])
	assert.Equal(t, "Test City", result["city_name"])
	assert.Equal(t, "RU", result["country"])
	assert.Equal(t, 20.5, result["temperature"])
	assert.Equal(t, 19.8, result["feels_like"])
	assert.Equal(t, 65, result["humidity"])
	assert.Equal(t, 1013, result["pressure"])
	assert.Equal(t, 5.2, result["wind_speed"])
	assert.Equal(t, 180, result["wind_deg"])
	assert.Equal(t, 40, result["clouds"])
	assert.Equal(t, "cloudy", result["weather"])
	assert.Equal(t, "04d", result["weather_icon"])
	assert.Equal(t, 10000, result["visibility"])
	assert.Equal(t, sunrise.Format(time.RFC3339), result["sunrise"])
	assert.Equal(t, sunset.Format(time.RFC3339), result["sunset"])
	assert.Equal(t, now.Format(time.RFC3339), result["timestamp"])
	assert.Equal(t, "test", result["source"])
}

func TestFromMap(t *testing.T) {
	now := time.Now()
	sunrise := now.Add(-6 * time.Hour)
	sunset := now.Add(6 * time.Hour)

	data := map[string]interface{}{
		"city_id":      123.0,
		"city_name":    "Test City",
		"country":      "RU",
		"temperature":  20.5,
		"feels_like":   19.8,
		"humidity":     65.0,
		"pressure":     1013.0,
		"wind_speed":   5.2,
		"wind_deg":     180.0,
		"clouds":       40.0,
		"weather":      "cloudy",
		"weather_icon": "04d",
		"visibility":   10000.0,
		"sunrise":      sunrise.Format(time.RFC3339),
		"sunset":       sunset.Format(time.RFC3339),
		"timestamp":    now.Format(time.RFC3339),
		"source":       "test",
	}

	weather, err := FromMap(data)
	require.NoError(t, err)

	assert.Equal(t, 123, weather.CityID)
	assert.Equal(t, "Test City", weather.CityName)
	assert.Equal(t, "RU", weather.Country)
	assert.Equal(t, 20.5, weather.Temperature)
	assert.Equal(t, 19.8, weather.FeelsLike)
	assert.Equal(t, 65, weather.Humidity)
	assert.Equal(t, 1013, weather.Pressure)
	assert.Equal(t, 5.2, weather.WindSpeed)
	assert.Equal(t, 180, weather.WindDeg)
	assert.Equal(t, 40, weather.Clouds)
	assert.Equal(t, "cloudy", weather.Weather)
	assert.Equal(t, "04d", weather.WeatherIcon)
	assert.Equal(t, 10000, weather.Visibility)
	assert.Equal(t, sunrise.Format(time.RFC3339), weather.Sunrise.Format(time.RFC3339))
	assert.Equal(t, sunset.Format(time.RFC3339), weather.Sunset.Format(time.RFC3339))
	assert.Equal(t, now.Format(time.RFC3339), weather.Timestamp.Format(time.RFC3339))
	assert.Equal(t, "test", weather.Source)
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:  "city_id",
		Reason: "не может быть пустым",
	}

	assert.Equal(t, "city_id: не может быть пустым", err.Error())
}

func TestWeatherEntity_Interface(t *testing.T) {
	var _ WeatherEntity = (*Weather)(nil)
}

func TestFromMap_InvalidData(t *testing.T) {
	t.Run("missing required field - city_id", func(t *testing.T) {
		now := time.Now()
		data := map[string]interface{}{
			"city_name":    "Test City",
			"country":      "RU",
			"temperature":  20.5,
			"feels_like":   19.8,
			"humidity":     65.0,
			"pressure":     1013.0,
			"wind_speed":   5.2,
			"wind_deg":     180.0,
			"clouds":       40.0,
			"weather":      "cloudy",
			"weather_icon": "04d",
			"visibility":   10000.0,
			"sunrise":      now.Format(time.RFC3339),
			"sunset":       now.Format(time.RFC3339),
			"timestamp":    now.Format(time.RFC3339),
			"source":       "test",
		}

		assert.Panics(t, func() {
			FromMap(data)
		})
	})

	t.Run("invalid timestamp format", func(t *testing.T) {
		now := time.Now()
		data := map[string]interface{}{
			"city_id":      123.0,
			"city_name":    "Test City",
			"country":      "RU",
			"temperature":  20.5,
			"feels_like":   19.8,
			"humidity":     65.0,
			"pressure":     1013.0,
			"wind_speed":   5.2,
			"wind_deg":     180.0,
			"clouds":       40.0,
			"weather":      "cloudy",
			"weather_icon": "04d",
			"visibility":   10000.0,
			"sunrise":      now.Format(time.RFC3339),
			"sunset":       now.Format(time.RFC3339),
			"timestamp":    "invalid timestamp",
			"source":       "test",
		}

		weather, err := FromMap(data)

		if err == nil {
			err = weather.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "timestamp")
		} else {
			assert.Error(t, err)
		}
	})

	t.Run("type conversion error - неправильный тип для city_id", func(t *testing.T) {
		now := time.Now()
		data := map[string]interface{}{
			"city_id":      "not a number",
			"city_name":    "Test City",
			"country":      "RU",
			"temperature":  20.5,
			"feels_like":   19.8,
			"humidity":     65.0,
			"pressure":     1013.0,
			"wind_speed":   5.2,
			"wind_deg":     180.0,
			"clouds":       40.0,
			"weather":      "cloudy",
			"weather_icon": "04d",
			"visibility":   10000.0,
			"sunrise":      now.Format(time.RFC3339),
			"sunset":       now.Format(time.RFC3339),
			"timestamp":    now.Format(time.RFC3339),
			"source":       "test",
		}

		assert.Panics(t, func() {
			FromMap(data)
		})
	})

	t.Run("nil value in map", func(t *testing.T) {
		data := map[string]interface{}{
			"city_id":      nil,
			"city_name":    "Test City",
			"country":      "RU",
			"temperature":  20.5,
			"feels_like":   19.8,
			"humidity":     65.0,
			"pressure":     1013.0,
			"wind_speed":   5.2,
			"wind_deg":     180.0,
			"clouds":       40.0,
			"weather":      "cloudy",
			"weather_icon": "04d",
			"visibility":   10000.0,
			"sunrise":      time.Now().Format(time.RFC3339),
			"sunset":       time.Now().Format(time.RFC3339),
			"timestamp":    time.Now().Format(time.RFC3339),
			"source":       "test",
		}

		assert.Panics(t, func() {
			FromMap(data)
		})
	})
}

func TestValidationErrors_Constants(t *testing.T) {
	assert.Equal(t, "city_id: не может быть пустым", ErrInvalidCityID.Error())
	assert.Equal(t, "city_name: не может быть пустым", ErrInvalidCityName.Error())
	assert.Equal(t, "humidity: должно быть между 0 и 100", ErrInvalidHumidity.Error())
	assert.Equal(t, "pressure: должно быть между 800 и 1200", ErrInvalidPressure.Error())
	assert.Equal(t, "wind_speed: не может быть отрицательным", ErrInvalidWindSpeed.Error())
	assert.Equal(t, "timestamp: не может быть нулевым", ErrInvalidTimestamp.Error())
}

func TestWeather_ToMap_EdgeCases(t *testing.T) {
	now := time.Now()
	sunrise := now.Add(-6 * time.Hour)
	sunset := now.Add(6 * time.Hour)

	weather := &Weather{
		CityID:      123,
		CityName:    "Test City",
		Country:     "RU",
		Temperature: 20.5,
		FeelsLike:   19.8,
		Humidity:    65,
		Pressure:    1013,
		WindSpeed:   5.2,
		WindDeg:     180,
		Clouds:      40,
		Weather:     "",
		WeatherIcon: "",
		Visibility:  10000,
		Sunrise:     sunrise,
		Sunset:      sunset,
		Timestamp:   now,
		Source:      "test",
	}

	result := weather.ToMap()

	assert.Equal(t, 123, result["city_id"])
	assert.Equal(t, "Test City", result["city_name"])
	assert.Equal(t, "RU", result["country"])
	assert.Equal(t, 20.5, result["temperature"])
	assert.Equal(t, 19.8, result["feels_like"])
	assert.Equal(t, 65, result["humidity"])
	assert.Equal(t, 1013, result["pressure"])
	assert.Equal(t, 5.2, result["wind_speed"])
	assert.Equal(t, 180, result["wind_deg"])
	assert.Equal(t, 40, result["clouds"])
	assert.Equal(t, "", result["weather"])
	assert.Equal(t, "", result["weather_icon"])
	assert.Equal(t, 10000, result["visibility"])
	assert.Equal(t, sunrise.Format(time.RFC3339), result["sunrise"])
	assert.Equal(t, sunset.Format(time.RFC3339), result["sunset"])
	assert.Equal(t, now.Format(time.RFC3339), result["timestamp"])
	assert.Equal(t, "test", result["source"])
}
