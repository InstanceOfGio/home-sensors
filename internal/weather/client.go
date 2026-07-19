package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"home-sensors/internal/models"
)

const forecastURL = "https://api.open-meteo.com/v1/forecast"

type rawResponse struct {
	Hourly struct {
		Time                     []int64   `json:"time"`
		Temperature              []float64 `json:"temperature_2m"`
		PrecipitationProbability []float64 `json:"precipitation_probability"`
		WindSpeed                []float64 `json:"wind_speed_10m"`
		Humidity                 []float64 `json:"relative_humidity_2m"`
	} `json:"hourly"`
}

// FetchToday returns the hourly forecast for the rest of the current local
// day at the given coordinates, using the free Open-Meteo API (no API key
// required).
func FetchToday(ctx context.Context, lat, lon float64) ([]models.WeatherPoint, error) {
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	q.Set("hourly", "temperature_2m,precipitation_probability,wind_speed_10m,relative_humidity_2m")
	q.Set("timezone", "auto")
	q.Set("timeformat", "unixtime")
	q.Set("forecast_days", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, forecastURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo: unexpected status %d", resp.StatusCode)
	}

	var raw rawResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode open-meteo response: %w", err)
	}

	now := time.Now()
	points := make([]models.WeatherPoint, 0, len(raw.Hourly.Time))
	for i, ts := range raw.Hourly.Time {
		t := time.Unix(ts, 0)
		if t.Before(now.Truncate(time.Hour)) {
			continue
		}
		points = append(points, models.WeatherPoint{
			Time:                     t,
			Temperature:              raw.Hourly.Temperature[i],
			PrecipitationProbability: raw.Hourly.PrecipitationProbability[i],
			WindSpeed:                raw.Hourly.WindSpeed[i],
			Humidity:                 raw.Hourly.Humidity[i],
		})
	}
	return points, nil
}
