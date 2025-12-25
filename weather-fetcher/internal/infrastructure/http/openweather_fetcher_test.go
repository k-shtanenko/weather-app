package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/weather-app/weather-fetcher/internal/domain/entities"
	"github.com/weather-app/weather-fetcher/internal/domain/ports"
)

func TestOpenWeatherFetcher_Fetch(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.String(), "id=123")
			assert.Contains(t, r.URL.String(), "appid=test-key")
			assert.Contains(t, r.URL.String(), "units=metric")
			assert.Contains(t, r.URL.String(), "lang=en")

			response := map[string]interface{}{
				"id":   123,
				"name": "Test City",
				"sys": map[string]interface{}{
					"country": "RU",
					"sunrise": 1672531200,
					"sunset":  1672560000,
				},
				"main": map[string]interface{}{
					"temp":       20.5,
					"feels_like": 19.8,
					"pressure":   1013,
					"humidity":   65,
				},
				"weather": []map[string]interface{}{
					{"description": "cloudy", "icon": "04d"},
				},
				"visibility": 10000,
				"wind": map[string]interface{}{
					"speed": 5.2,
					"deg":   180,
				},
				"clouds": map[string]interface{}{
					"all": 40,
				},
				"cod": 200,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		weather, err := fetcher.Fetch(ctx, "123")

		require.NoError(t, err)
		assert.NotNil(t, weather)
		assert.Equal(t, 123, weather.GetCityID())
		assert.Equal(t, "openweathermap", weather.GetSource())
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"cod":     "404",
				"message": "city not found",
			})
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		weather, err := fetcher.Fetch(ctx, "999")

		assert.Error(t, err)
		assert.Nil(t, weather)
		assert.Contains(t, err.Error(), "API returned status 404")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		weather, err := fetcher.Fetch(ctx, "123")

		assert.Error(t, err)
		assert.Nil(t, weather)
		assert.Contains(t, err.Error(), "failed to decode response")
	})
}

func TestOpenWeatherFetcher_FetchBatch(t *testing.T) {
	t.Run("successful batch fetch", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			response := map[string]interface{}{
				"id":   requestCount,
				"name": "Test City",
				"sys": map[string]interface{}{
					"country": "RU",
					"sunrise": 1672531200,
					"sunset":  1672560000,
				},
				"main": map[string]interface{}{
					"temp":       20.5,
					"feels_like": 19.8,
					"pressure":   1013,
					"humidity":   65,
				},
				"weather": []map[string]interface{}{
					{"description": "cloudy", "icon": "04d"},
				},
				"visibility": 10000,
				"wind": map[string]interface{}{
					"speed": 5.2,
					"deg":   180,
				},
				"clouds": map[string]interface{}{
					"all": 40,
				},
				"cod": 200,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		cityIDs := []string{"123", "456", "789"}
		weathers, err := fetcher.FetchBatch(ctx, cityIDs)

		require.NoError(t, err)
		assert.Equal(t, 3, len(weathers))
		assert.Equal(t, 3, requestCount)
	})

	t.Run("partial failure in batch", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount == 2 {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			response := map[string]interface{}{
				"id":   requestCount,
				"name": "Test City",
				"sys": map[string]interface{}{
					"country": "RU",
					"sunrise": 1672531200,
					"sunset":  1672560000,
				},
				"main": map[string]interface{}{
					"temp":       20.5,
					"feels_like": 19.8,
					"pressure":   1013,
					"humidity":   65,
				},
				"weather": []map[string]interface{}{
					{"description": "cloudy", "icon": "04d"},
				},
				"visibility": 10000,
				"wind": map[string]interface{}{
					"speed": 5.2,
					"deg":   180,
				},
				"clouds": map[string]interface{}{
					"all": 40,
				},
				"cod": 200,
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		cityIDs := []string{"123", "456", "789"}
		weathers, err := fetcher.FetchBatch(ctx, cityIDs)

		require.NoError(t, err)
		assert.Equal(t, 2, len(weathers))
	})

	t.Run("all requests fail in batch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		cityIDs := []string{"123", "456"}
		weathers, err := fetcher.FetchBatch(ctx, cityIDs)

		assert.Error(t, err)
		assert.Nil(t, weathers)
		assert.Contains(t, err.Error(), "all requests failed")
	})

	t.Run("context cancelled during batch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"cod": 200})
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		cityIDs := []string{"123", "456"}
		weathers, err := fetcher.FetchBatch(ctx, cityIDs)

		assert.Error(t, err)
		assert.Nil(t, weathers)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestOpenWeatherFetcher_HealthCheck(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.String(), "id=524901")
			response := map[string]interface{}{
				"cod": 200,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		err := fetcher.HealthCheck(ctx)

		assert.NoError(t, err)
	})

	t.Run("health check fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

		ctx := context.Background()
		err := fetcher.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API health check failed with status")
	})
}

func TestOpenWeatherFetcherFactory(t *testing.T) {
	factory := NewOpenWeatherFetcherFactory()

	fetcher := factory.CreateFetcher("https://api.test.com", "test-key", "metric", "en")

	assert.NotNil(t, fetcher)
}

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestOpenWeatherFetcherFactory_CreateFetcher(t *testing.T) {
	factory := NewOpenWeatherFetcherFactory()

	fetcher := factory.CreateFetcher(
		"https://api.test.com",
		"test-key",
		"metric",
		"en",
	)

	assert.NotNil(t, fetcher)
	assert.Implements(t, (*ports.Fetcher)(nil), fetcher)
}

func TestOpenWeatherFetcherFactory_InterfaceImplementation(t *testing.T) {
	var _ ports.FetcherFactory = (*OpenWeatherFetcherFactory)(nil)
}

func TestOpenWeatherFetcher_Fetch_EmptyWeatherArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"id": 123,
			"name": "Test City",
			"sys": {"country": "RU", "sunrise": 1672531200, "sunset": 1672560000},
			"main": {"temp": 20.5, "feels_like": 19.8, "pressure": 1013, "humidity": 65},
			"weather": [],
			"visibility": 10000,
			"wind": {"speed": 5.2, "deg": 180},
			"clouds": {"all": 40},
			"cod": 200
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	require.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 123, weather.GetCityID())
	assert.Equal(t, "openweathermap", weather.GetSource())
	assert.Equal(t, "", weather.(*entities.Weather).Weather)
	assert.Equal(t, "", weather.(*entities.Weather).WeatherIcon)
}

func TestOpenWeatherFetcher_Fetch_MultipleWeatherItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"id": 456,
			"name": "Test City 2",
			"sys": {"country": "US", "sunrise": 1672531200, "sunset": 1672560000},
			"main": {"temp": 25.0, "feels_like": 24.5, "pressure": 1015, "humidity": 70},
			"weather": [
				{"id": 800, "main": "Clear", "description": "clear sky", "icon": "01d"},
				{"id": 801, "main": "Clouds", "description": "few clouds", "icon": "02d"}
			],
			"visibility": 15000,
			"wind": {"speed": 3.5, "deg": 90},
			"clouds": {"all": 20},
			"cod": 200
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "456")

	require.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 456, weather.GetCityID())
	assert.Equal(t, "Test City 2", weather.(*entities.Weather).CityName)
	assert.Equal(t, "clear sky", weather.(*entities.Weather).Weather)
	assert.Equal(t, "01d", weather.(*entities.Weather).WeatherIcon)
}

func TestOpenWeatherFetcher_Fetch_WithGustWind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"id": 789,
			"name": "Windy City",
			"sys": {"country": "CA", "sunrise": 1672531200, "sunset": 1672560000},
			"main": {"temp": 15.0, "feels_like": 12.5, "pressure": 1005, "humidity": 80},
			"weather": [{"id": 802, "main": "Clouds", "description": "scattered clouds", "icon": "03d"}],
			"visibility": 8000,
			"wind": {"speed": 8.5, "deg": 270, "gust": 12.3},
			"clouds": {"all": 60},
			"cod": 200
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "789")

	require.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 789, weather.GetCityID())
	assert.Equal(t, 8.5, weather.(*entities.Weather).WindSpeed)
}

func TestOpenWeatherFetcher_Fetch_HTTPRequestError(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("://invalid-url", "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestOpenWeatherFetcher_HealthCheck_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	err := fetcher.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API health check failed with status")
}

func TestOpenWeatherFetcher_Fetch_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")
	fetcher.(*OpenWeatherFetcher).client.Timeout = 1 * time.Second

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to execute request")
}

func TestOpenWeatherFetcher_FetchBatch_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cityIDs := []string{"123", "456"}
	weathers, err := fetcher.FetchBatch(ctx, cityIDs)

	assert.Error(t, err)
	assert.Nil(t, weathers)
	assert.Equal(t, context.Canceled, err)
}

func TestOpenWeatherFetcher_HealthCheck_NetworkTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")
	fetcher.(*OpenWeatherFetcher).client.Timeout = 1 * time.Second

	ctx := context.Background()
	err := fetcher.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute health check")
}

func TestOpenWeatherFetcher_Fetch_EmptyResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestOpenWeatherFetcher_Fetch_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"cod":     "500",
			"message": "Internal server error",
		})
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "API returned status 500")
}

func TestOpenWeatherFetcher_Fetch_InvalidResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestOpenWeatherFetcher_Fetch_Non200StatusWithBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "API returned status 500")
}

func TestOpenWeatherFetcher_HealthCheck_RequestCreationError(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("://invalid-url", "test-key", "metric", "en")

	ctx := context.Background()
	err := fetcher.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create health check request")
}

func TestOpenWeatherFetcher_Fetch_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":         123,
			"name":       "Test City",
			"sys":        map[string]interface{}{"country": "RU", "sunrise": 1672531200, "sunset": 1672560000},
			"main":       map[string]interface{}{"temp": 20.5, "feels_like": 19.8, "pressure": 1013, "humidity": 65},
			"weather":    []map[string]interface{}{{"description": "cloudy", "icon": "04d"}},
			"visibility": 10000,
			"wind":       map[string]interface{}{"speed": 5.2, "deg": 180},
			"clouds":     map[string]interface{}{"all": 40},
			"cod":        200,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 123, weather.GetCityID())
	assert.Equal(t, "Test City", weather.(*entities.Weather).CityName)
}

func TestOpenWeatherFetcher_HealthCheck_InvalidURL(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("://invalid-url", "test-key", "metric", "en")

	ctx := context.Background()
	err := fetcher.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create health check request")
}

func TestOpenWeatherFetcher_Fetch_HTTPClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to execute request")
}

func TestOpenWeatherFetcher_Fetch_ResponseBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
}

func TestOpenWeatherFetcher_Fetch_InvalidResponseStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "API returned status 500")
}

func TestOpenWeatherFetcher_Fetch_InvalidJSONInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestOpenWeatherFetcherFactory_CreateFetcher_Logger(t *testing.T) {
	factory := NewOpenWeatherFetcherFactory()

	fetcher := factory.CreateFetcher(
		"https://api.test.com",
		"test-key",
		"metric",
		"en",
	)

	assert.NotNil(t, fetcher)
}

func TestOpenWeatherFetcher_convertToWeather_EmptyArrays(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("http://test.com", "key", "metric", "en").(*OpenWeatherFetcher)

	resp := &OpenWeatherResponse{
		ID:   123,
		Name: "Test City",
		Sys: struct {
			Type    int    `json:"type"`
			ID      int    `json:"id"`
			Country string `json:"country"`
			Sunrise int    `json:"sunrise"`
			Sunset  int    `json:"sunset"`
		}{
			Country: "RU",
			Sunrise: 1672531200,
			Sunset:  1672560000,
		},
		Main: struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			TempMin   float64 `json:"temp_min"`
			TempMax   float64 `json:"temp_max"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
			SeaLevel  int     `json:"sea_level"`
			GrndLevel int     `json:"grnd_level"`
		}{
			Temp:      20.5,
			FeelsLike: 19.8,
			Pressure:  1013,
			Humidity:  65,
		},
		Weather: []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		}{},
		Visibility: 10000,
		Wind: struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
			Gust  float64 `json:"gust"`
		}{
			Speed: 5.2,
			Deg:   180,
		},
		Clouds: struct {
			All int `json:"all"`
		}{
			All: 40,
		},
	}

	weather := fetcher.convertToWeather(resp)

	assert.NotNil(t, weather)
	assert.Equal(t, 123, weather.GetCityID())
}

func TestOpenWeatherFetcher_FetchBatch_ContextCancelledImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"cod":200}`))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	cityIDs := []string{"123", "456"}
	weathers, err := fetcher.FetchBatch(ctx, cityIDs)

	assert.Error(t, err)
	assert.Nil(t, weathers)
	assert.Equal(t, context.Canceled, err)
}

func TestOpenWeatherFetcher_FetchBatch_SingleCitySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"id": 123,
			"name": "Test City",
			"sys": {"country": "RU", "sunrise": 1672531200, "sunset": 1672560000},
			"main": {"temp": 20.5, "feels_like": 19.8, "pressure": 1013, "humidity": 65},
			"weather": [{"description": "cloudy", "icon": "04d"}],
			"visibility": 10000,
			"wind": {"speed": 5.2, "deg": 180},
			"clouds": {"all": 40},
			"cod": 200
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	cityIDs := []string{"123"}
	weathers, err := fetcher.FetchBatch(ctx, cityIDs)

	require.NoError(t, err)
	assert.Equal(t, 1, len(weathers))
	assert.Equal(t, 123, weathers[0].GetCityID())
}

func TestOpenWeatherFetcher_Fetch_ErrorCreatingRequest(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("://invalid-url", "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestOpenWeatherFetcher_Fetch_RequestCreationError(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("://invalid-url", "test-key", "metric", "en")

	ctx := context.Background()
	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestOpenWeatherFetcher_HealthCheck_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")
	fetcher.(*OpenWeatherFetcher).client.Timeout = 100 * time.Millisecond

	ctx := context.Background()
	err := fetcher.HealthCheck(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute health check")
}

func TestOpenWeatherFetcherFactory_Logger(t *testing.T) {
	factory := NewOpenWeatherFetcherFactory()

	fetcher := factory.CreateFetcher(
		"https://api.test.com",
		"test-key",
		"metric",
		"en",
	)

	assert.NotNil(t, fetcher)
}

func TestOpenWeatherFetcher_convertToWeather_MissingWeatherData(t *testing.T) {
	fetcher := NewOpenWeatherFetcher("http://test.com", "key", "metric", "en").(*OpenWeatherFetcher)

	resp := &OpenWeatherResponse{
		ID:   123,
		Name: "Test City",
		Sys: struct {
			Type    int    `json:"type"`
			ID      int    `json:"id"`
			Country string `json:"country"`
			Sunrise int    `json:"sunrise"`
			Sunset  int    `json:"sunset"`
		}{
			Country: "RU",
			Sunrise: 1672531200,
			Sunset:  1672560000,
		},
		Main: struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			TempMin   float64 `json:"temp_min"`
			TempMax   float64 `json:"temp_max"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
			SeaLevel  int     `json:"sea_level"`
			GrndLevel int     `json:"grnd_level"`
		}{
			Temp:      20.5,
			FeelsLike: 19.8,
			Pressure:  1013,
			Humidity:  65,
		},
		Weather:    nil,
		Visibility: 10000,
		Wind: struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
			Gust  float64 `json:"gust"`
		}{
			Speed: 5.2,
			Deg:   180,
		},
		Clouds: struct {
			All int `json:"all"`
		}{
			All: 40,
		},
	}

	weather := fetcher.convertToWeather(resp)

	assert.NotNil(t, weather)
	assert.Equal(t, 123, weather.GetCityID())
}

func TestOpenWeatherFetcher_FetchBatch_AllFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx := context.Background()
	cityIDs := []string{"123", "456", "789"}
	weathers, err := fetcher.FetchBatch(ctx, cityIDs)

	assert.Error(t, err)
	assert.Nil(t, weathers)
	assert.Contains(t, err.Error(), "all requests failed")
}

func TestOpenWeatherFetcher_Fetch_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	fetcher := NewOpenWeatherFetcher(server.URL, "test-key", "metric", "en")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	weather, err := fetcher.Fetch(ctx, "123")

	assert.Error(t, err)
	assert.Nil(t, weather)
}
