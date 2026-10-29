package model

import "testing"

func TestWeatherCacheCoordinateStrategy(t *testing.T) {
	cache := NewCache()
	if hit, weather := cache.GetWeather("unknown", false); hit || weather != nil {
		t.Fatalf("missing city lookup = (%v, %v), want a miss", hit, weather)
	}

	cache.SetWeather("Rome, Lazio", Weather{
		Coordinates: Coordinates{Lat: 41.9028, Lon: 12.4964},
		City:        "Rome",
		CurrentTemp: 20,
	}, false)

	if hit, weather := cache.GetWeather("rome", false); hit || weather != nil {
		t.Fatalf("unrequested canonical city lookup = (%v, %v), want a miss", hit, weather)
	}

	cache.SetWeather("Roma", Weather{
		Coordinates: Coordinates{Lat: 41.9099, Lon: 12.4999},
		City:        "Roma",
		CurrentTemp: 21,
	}, false)

	if hit, weather := cache.GetWeatherByCoordinates(Coordinates{Lat: 41.9099, Lon: 12.4999}, false); !hit {
		t.Fatal("coordinate lookup was a miss, want a hit")
	} else if weather.CurrentTemp != 21 || weather.Coordinates != (Coordinates{Lat: 41.90, Lon: 12.49}) {
		t.Errorf("coordinate lookup = %+v, want latest weather with floored coordinates", weather)
	}

	if hit, weather := cache.GetWeather("rome, lazio", false); !hit || weather.CurrentTemp != 21 {
		t.Errorf("first city alias lookup = (%v, %v), want shared coordinate entry", hit, weather)
	}

	entries, err := GetCache(&cache)
	if err != nil {
		t.Fatalf("get cache entries: %v", err)
	}
	if got := len(entries.Entries); got != 1 {
		t.Errorf("coordinate cache has %d entries, want 1", got)
	}
}

func TestWeatherCacheCityStrategy(t *testing.T) {
	cache := NewCacheWithStrategy(CacheByCity)
	rome := Weather{
		Coordinates: Coordinates{Lat: 41.9028, Lon: 12.4964},
		City:        "Rome",
		CurrentTemp: 20,
	}
	roma := Weather{
		Coordinates: Coordinates{Lat: 41.9028, Lon: 12.4964},
		City:        "Roma",
		CurrentTemp: 21,
	}
	cache.SetWeather("Rome", rome, false)
	cache.SetWeather("Roma", roma, false)

	for _, tt := range []struct {
		name     string
		wantTemp float64
	}{
		{name: "Rome", wantTemp: 20},
		{name: "Roma", wantTemp: 21},
	} {
		t.Run(tt.name, func(t *testing.T) {
			hit, weather := cache.GetWeather(tt.name, false)
			if !hit || weather.CurrentTemp != tt.wantTemp {
				t.Errorf("city lookup = (%v, %v), want hit with temperature %v", hit, weather, tt.wantTemp)
			}
		})
	}

	if hit, weather := cache.GetWeatherByCoordinates(rome.Coordinates, false); hit || weather != nil {
		t.Errorf("coordinate lookup = (%v, %v), want a miss with city strategy", hit, weather)
	}

	entries, err := GetCache(&cache)
	if err != nil {
		t.Fatalf("get cache entries: %v", err)
	}
	if got := len(entries.Entries); got != 2 {
		t.Errorf("city cache has %d entries, want 2", got)
	}
}
