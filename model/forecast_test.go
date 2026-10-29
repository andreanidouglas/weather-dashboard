package model

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGetForecast(t *testing.T) {
	type dailyForecast struct {
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
	}

	days := make([]dailyForecast, 8)
	for i := range days {
		days[i].Date = time.Date(2030, time.February, 1+i, 12, 0, 0, 0, time.UTC).Unix()
		days[i].Temp.Min = float64(i)
		days[i].Temp.Max = float64(i + 10)
		days[i].Humidity = 50 + i
		days[i].Pop = 0.25
		days[i].Weather = append(days[i].Weather, struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		}{Main: "Clouds", Description: "scattered clouds"})
	}
	responseBody, err := json.Marshal(struct {
		TimezoneOffset int             `json:"timezone_offset"`
		Daily          []dailyForecast `json:"daily"`
	}{TimezoneOffset: 0, Daily: days})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	originalForecastDataURL := ForecastDataURL
	ForecastDataURL = server.URL
	t.Cleanup(func() {
		ForecastDataURL = originalForecastDataURL
	})

	forecast, err := GetForecast(context.Background(), ForecastRequest{
		Lat:        41.9,
		Lon:        12.5,
		Fahrenheit: true,
	}, &ApiContext{Key: "test-key"})
	if err != nil {
		t.Fatalf("get forecast: %v", err)
	}

	query, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse upstream query: %v", err)
	}
	for key, want := range map[string]string{
		"lat":     "41.9",
		"lon":     "12.5",
		"units":   "imperial",
		"exclude": "current,minutely,hourly,alerts",
		"appid":   "test-key",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("upstream %s = %q, want %q", key, got, want)
		}
	}

	if len(forecast.Days) != 7 {
		t.Fatalf("forecast days = %d, want 7", len(forecast.Days))
	}
	firstDay := forecast.Days[0]
	if firstDay.Date != "Sat, Feb 2" || firstDay.MinTemp != 1 || firstDay.MaxTemp != 11 {
		t.Errorf("first forecast day = %+v, want Feb 2 with low 1 and high 11", firstDay)
	}
	if firstDay.Condition != "Clouds" || firstDay.Description != "scattered clouds" ||
		firstDay.Humidity != 51 || firstDay.Precipitation != 0.25 {
		t.Errorf("first forecast details = %+v", firstDay)
	}
}

func TestGetForecastRejectsInvalidCoordinates(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{name: "latitude out of range", lat: 91, lon: 0},
		{name: "longitude out of range", lat: 0, lon: -181},
		{name: "not a number", lat: math.NaN(), lon: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := ForecastRequest{Lat: tt.lat, Lon: tt.lon}
			apiContext := &ApiContext{Key: "test-key"}
			if _, err := GetForecast(context.Background(), request, apiContext); err == nil {
				t.Fatal("expected invalid coordinates error")
			}
		})
	}
}
