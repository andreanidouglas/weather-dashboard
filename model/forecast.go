package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var ForecastDataURL = "https://api.openweathermap.org/data/3.0/onecall"

const forecastHTTPTimeout = 10 * time.Second

var forecastHTTPClient = &http.Client{Timeout: forecastHTTPTimeout}

type Forecast struct {
	Days []ForecastDay `json:"days"`
}

// ForecastRequest identifies the location and units for a forecast lookup.
type ForecastRequest struct {
	Lat        float64
	Lon        float64
	Fahrenheit bool
}

type ForecastDay struct {
	Date          string  `json:"date"`
	MinTemp       float64 `json:"min_temp"`
	MaxTemp       float64 `json:"max_temp"`
	Condition     string  `json:"condition"`
	Description   string  `json:"description"`
	Humidity      int     `json:"humidity"`
	Precipitation float64 `json:"precipitation"`
}

type openWeatherForecast struct {
	TimezoneOffset int `json:"timezone_offset"`
	Daily          []struct {
		Date int64 `json:"dt"`
		Temp struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		} `json:"temp"`
		Humidity int     `json:"humidity"`
		Pop      float64 `json:"pop"`
		Weather  []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"daily"`
}

// GetForecast returns the next seven complete daily forecasts for coordinates.
func GetForecast(ctx context.Context, request ForecastRequest, apiContext *ApiContext) (*Forecast, error) {
	if math.IsNaN(request.Lat) || math.IsInf(request.Lat, 0) || request.Lat < -90 || request.Lat > 90 {
		return nil, fmt.Errorf("invalid latitude")
	}
	if math.IsNaN(request.Lon) || math.IsInf(request.Lon, 0) || request.Lon < -180 || request.Lon > 180 {
		return nil, fmt.Errorf("invalid longitude")
	}

	endpoint, err := url.Parse(ForecastDataURL)
	if err != nil {
		return nil, fmt.Errorf("parse forecast endpoint: %w", err)
	}

	units := "metric"
	if request.Fahrenheit {
		units = "imperial"
	}

	query := endpoint.Query()
	query.Set("lat", strconv.FormatFloat(request.Lat, 'f', -1, 64))
	query.Set("lon", strconv.FormatFloat(request.Lon, 'f', -1, 64))
	query.Set("units", units)
	query.Set("exclude", "current,minutely,hourly,alerts")
	query.Set("appid", apiContext.Key)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create forecast request: %w", err)
	}

	res, err := forecastHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request forecast: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forecast API returned status %d", res.StatusCode)
	}

	var response openWeatherForecast
	decoder := json.NewDecoder(io.LimitReader(res.Body, 1<<20))
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode forecast response: %w", err)
	}
	if len(response.Daily) < 8 {
		return nil, fmt.Errorf("forecast response did not include seven future days")
	}

	location := time.FixedZone("forecast", response.TimezoneOffset)
	days := make([]ForecastDay, 0, 7)
	for _, daily := range response.Daily[1:8] {
		day := ForecastDay{
			Date:          time.Unix(daily.Date, 0).In(location).Format("Mon, Jan 2"),
			MinTemp:       daily.Temp.Min,
			MaxTemp:       daily.Temp.Max,
			Humidity:      daily.Humidity,
			Precipitation: daily.Pop,
		}
		if len(daily.Weather) > 0 {
			day.Condition = daily.Weather[0].Main
			day.Description = daily.Weather[0].Description
		}
		days = append(days, day)
	}

	return &Forecast{Days: days}, nil
}
