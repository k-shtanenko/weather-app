package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/domain/entities"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/domain/ports"
	"github.com/k-shtanenko/weather-app/weather-fetcher/internal/pkg/logger"
)

type OpenWeatherFetcher struct {
	client  *http.Client
	baseURL string
	apiKey  string
	units   string
	lang    string
	logger  logger.Logger
}

func NewOpenWeatherFetcher(baseURL, apiKey, units, lang string) ports.Fetcher {
	return &OpenWeatherFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		units:   units,
		lang:    lang,
		logger:  logger.New("info", "development").WithField("component", "openweather_fetcher"),
	}
}

type OpenWeatherResponse struct {
	Coord struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Base string `json:"base"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
		SeaLevel  int     `json:"sea_level"`
		GrndLevel int     `json:"grnd_level"`
	} `json:"main"`
	Visibility int `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Dt  int `json:"dt"`
	Sys struct {
		Type    int    `json:"type"`
		ID      int    `json:"id"`
		Country string `json:"country"`
		Sunrise int    `json:"sunrise"`
		Sunset  int    `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cod      int    `json:"cod"`
}

func (f *OpenWeatherFetcher) Fetch(ctx context.Context, cityID string) (entities.WeatherEntity, error) {
	f.logger.Debugf("Fetching weather for city ID: %s", cityID)

	url := fmt.Sprintf("%s/weather?id=%s&appid=%s&units=%s&lang=%s",
		f.baseURL, cityID, f.apiKey, f.units, f.lang)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp OpenWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	weather := f.convertToWeather(&apiResp)

	f.logger.Debugf("Successfully fetched weather for %s (ID: %d)", weather.CityName, weather.CityID)
	return weather, nil
}

func (f *OpenWeatherFetcher) FetchBatch(ctx context.Context, cityIDs []string) ([]entities.WeatherEntity, error) {
	f.logger.Debugf("Fetching weather batch for %d cities", len(cityIDs))

	var results []entities.WeatherEntity
	var errors []error

	for _, cityID := range cityIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			weather, err := f.Fetch(ctx, cityID)
			if err != nil {
				f.logger.Warnf("Failed to fetch weather for city %s: %v", cityID, err)
				errors = append(errors, fmt.Errorf("city %s: %w", cityID, err))
				continue
			}
			results = append(results, weather)
		}
	}

	if len(results) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("all requests failed: %v", errors)
	}

	f.logger.Infof("Successfully fetched weather for %d out of %d cities", len(results), len(cityIDs))
	return results, nil
}

func (f *OpenWeatherFetcher) HealthCheck(ctx context.Context) error {
	testCityID := "524901" // Москва
	url := fmt.Sprintf("%s/weather?id=%s&appid=%s", f.baseURL, testCityID, f.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API health check failed with status: %d", resp.StatusCode)
	}

	f.logger.Debug("OpenWeatherMap API health check passed")
	return nil
}

func (f *OpenWeatherFetcher) convertToWeather(resp *OpenWeatherResponse) *entities.Weather {
	weatherDesc := ""
	if len(resp.Weather) > 0 {
		weatherDesc = resp.Weather[0].Description
	}

	weatherIcon := ""
	if len(resp.Weather) > 0 {
		weatherIcon = resp.Weather[0].Icon
	}

	return &entities.Weather{
		CityID:      resp.ID,
		CityName:    resp.Name,
		Country:     resp.Sys.Country,
		Temperature: resp.Main.Temp,
		FeelsLike:   resp.Main.FeelsLike,
		Humidity:    resp.Main.Humidity,
		Pressure:    resp.Main.Pressure,
		WindSpeed:   resp.Wind.Speed,
		WindDeg:     resp.Wind.Deg,
		Clouds:      resp.Clouds.All,
		Weather:     weatherDesc,
		WeatherIcon: weatherIcon,
		Visibility:  resp.Visibility,
		Sunrise:     time.Unix(int64(resp.Sys.Sunrise), 0),
		Sunset:      time.Unix(int64(resp.Sys.Sunset), 0),
		Timestamp:   time.Now(),
		Source:      "openweathermap",
	}
}

type OpenWeatherFetcherFactory struct {
	logger logger.Logger
}

func NewOpenWeatherFetcherFactory() ports.FetcherFactory {
	return &OpenWeatherFetcherFactory{
		logger: logger.New("info", "development").WithField("component", "openweather_fetcher_factory"),
	}
}

func (f *OpenWeatherFetcherFactory) CreateFetcher(baseURL, apiKey, units, lang string) ports.Fetcher {
	f.logger.Infof("Creating OpenWeatherFetcher with baseURL: %s", baseURL)
	return NewOpenWeatherFetcher(baseURL, apiKey, units, lang)
}
