package model

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetWeatherCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"coord":{"lat":41.9028,"lon":12.4964},
			"weather":[{"main":"Clear"}],
			"main":{"temp":20,"temp_min":18,"temp_max":22,"feels_like":20,"humidity":50},
			"sys":{"country":"IT"},
			"timezone":7200,
			"name":"Rome"
		}`))
	}))
	defer server.Close()

	originalWeatherDataURL := WeatherDataURL
	WeatherDataURL = server.URL
	t.Cleanup(func() {
		WeatherDataURL = originalWeatherDataURL
	})

	tests := []struct {
		name         string
		unit         TemperatureUnit
		current      float64
		other        TemperatureUnit
		otherCurrent float64
		get          func() (*Weather, error)
	}{
		{
			name:         "city request in celsius",
			unit:         UnitCelsius,
			current:      20,
			other:        UnitFahrenheit,
			otherCurrent: 68,
			get: func() (*Weather, error) {
				return GetWeather(WeatherRequest{City: "Rome"}, &ApiContext{Key: "test-key"})
			},
		},
		{
			name:         "coordinate request in celsius",
			unit:         UnitCelsius,
			current:      20,
			other:        UnitFahrenheit,
			otherCurrent: 68,
			get: func() (*Weather, error) {
				return GetWeatherByLatLon(LatLon{Lat: 41.9028, Lon: 12.4964}, ApiContext{Key: "test-key"})
			},
		},
		{
			name:         "city request in fahrenheit",
			unit:         UnitFahrenheit,
			current:      20,
			other:        UnitCelsius,
			otherCurrent: -6.666666666666667,
			get: func() (*Weather, error) {
				return GetWeather(WeatherRequest{City: "Rome", Fahrenheit: true}, &ApiContext{Key: "test-key"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather, err := tt.get()
			if err != nil {
				t.Fatalf("get weather: %v", err)
			}
			if got, want := weather.Coordinates, (Coordinates{Lat: 41.90, Lon: 12.49}); got != want {
				t.Errorf("weather coordinates = %+v, want %+v", got, want)
			}
			temperature := weather.TemperatureFor(tt.unit == UnitFahrenheit)
			if temperature.Unit != tt.unit || temperature.Current != tt.current {
				t.Errorf("weather temperature = %+v, want unit %q and current %v", temperature, tt.unit, tt.current)
			}
			otherTemperature := weather.TemperatureFor(tt.unit == UnitCelsius)
			if otherTemperature.Unit != tt.other || math.Abs(otherTemperature.Current-tt.otherCurrent) > 1e-9 {
				t.Errorf("other weather temperature = %+v, want unit %q and current %v",
					otherTemperature,
					tt.other,
					tt.otherCurrent,
				)
			}

			// Coordinates is comparable and can be used as a map key.
			cacheKeys := map[Coordinates]struct{}{weather.Coordinates: {}}
			if _, ok := cacheKeys[Coordinates{Lat: 41.90, Lon: 12.49}]; !ok {
				t.Error("weather coordinates were not usable as a matching map key")
			}
		})
	}
}

func TestGetWeatherRejectsInvalidResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "invalid JSON",
			body: `{`,
		},
		{
			name: "missing weather conditions",
			body: `{"coord":{"lat":10,"lon":20},"main":{"temp":20},"weather":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			originalWeatherDataURL := WeatherDataURL
			WeatherDataURL = server.URL
			t.Cleanup(func() {
				WeatherDataURL = originalWeatherDataURL
			})

			if _, err := GetWeather(WeatherRequest{City: "Rome"}, &ApiContext{Key: "test-key"}); err == nil {
				t.Fatal("GetWeather succeeded for invalid upstream response, want an error")
			}
		})
	}
}

func TestFloorCoordinates(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
		want Coordinates
	}{
		{
			name: "positive coordinates",
			lat:  41.9028,
			lon:  12.4964,
			want: Coordinates{Lat: 41.90, Lon: 12.49},
		},
		{
			name: "negative coordinates",
			lat:  -41.9028,
			lon:  -12.4964,
			want: Coordinates{Lat: -41.91, Lon: -12.50},
		},
		{
			name: "coordinates already at hundredth precision",
			lat:  41.90,
			lon:  12.49,
			want: Coordinates{Lat: 41.90, Lon: 12.49},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := floorCoordinates(tt.lat, tt.lon); got != tt.want {
				t.Errorf("floorCoordinates(%v, %v) = %+v, want %+v", tt.lat, tt.lon, got, tt.want)
			}
		})
	}
}
