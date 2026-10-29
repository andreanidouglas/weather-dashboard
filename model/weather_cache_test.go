package model

import (
	"testing"
	"time"
)

func TestWeatherCacheCoordinateStrategy(t *testing.T) {
	cache := NewCache()
	t.Cleanup(cache.Close)
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
	if entries.Strategy != "coordinates" {
		t.Errorf("coordinate cache strategy = %q, want %q", entries.Strategy, "coordinates")
	}
	if entries.Hits != 2 || entries.Requests != 4 {
		t.Errorf("coordinate cache stats = hits:%d requests:%d, want hits:2 requests:4", entries.Hits, entries.Requests)
	}
}

func TestWeatherCacheCityStrategy(t *testing.T) {
	cache := NewCacheWithStrategy(CacheByCity)
	t.Cleanup(cache.Close)
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
	if entries.Strategy != "city" {
		t.Errorf("city cache strategy = %q, want %q", entries.Strategy, "city")
	}
	if entries.Hits != 2 || entries.Requests != 3 {
		t.Errorf("city cache stats = hits:%d requests:%d, want hits:2 requests:3", entries.Hits, entries.Requests)
	}
}

func TestCacheStrategiesDeleteExpiredEntries(t *testing.T) {
	tests := []struct {
		name     string
		strategy CacheStrategy
	}{
		{name: "coordinates", strategy: CacheByCoordinates},
		{name: "city", strategy: CacheByCity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCacheWithStrategy(tt.strategy)
			t.Cleanup(cache.Close)

			weatherByCity := map[string]Weather{
				"expired": {
					Coordinates: Coordinates{Lat: 10.25, Lon: 20.5},
					City:        "Expired",
				},
				"boundary": {
					Coordinates: Coordinates{Lat: 11.25, Lon: 21.5},
					City:        "Boundary",
				},
				"valid": {
					Coordinates: Coordinates{Lat: 12.25, Lon: 22.5},
					City:        "Valid",
				},
			}
			for city, weather := range weatherByCity {
				cache.SetWeather(city, weather, false)
			}

			scanTime := time.Now()
			setCacheExpiry(&cache, "expired", weatherByCity["expired"].Coordinates, scanTime.Add(-time.Nanosecond))
			setCacheExpiry(&cache, "boundary", weatherByCity["boundary"].Coordinates, scanTime)
			setCacheExpiry(&cache, "valid", weatherByCity["valid"].Coordinates, scanTime.Add(time.Hour))

			cache.strategy.deleteExpired(scanTime)

			entries, err := GetCache(&cache)
			if err != nil {
				t.Fatalf("get cache entries: %v", err)
			}
			if got := len(entries.Entries); got != 2 {
				t.Errorf("cache has %d entries after sweep, want 2", got)
			}

			if hit, _ := cache.GetWeather("expired", false); hit {
				t.Error("expired entry was still a cache hit")
			}
			if !cacheContainsEntry(&cache, "boundary", weatherByCity["boundary"].Coordinates) {
				t.Error("entry expiring exactly at scan time was deleted")
			}
			if hit, _ := cache.GetWeather("valid", false); !hit {
				t.Error("unexpired entry was deleted")
			}

			if coordinateCache, ok := cache.strategy.(*coordinateCache); ok {
				coordinateCache.RLock()
				defer coordinateCache.RUnlock()
				if _, exists := coordinateCache.cityCoordinates[cacheKey("expired")]; exists {
					t.Error("expired city-to-coordinate alias was not deleted")
				}
			}
		})
	}
}

func setCacheExpiry(cache *WeatherCache, city string, coordinates Coordinates, validUntil time.Time) {
	switch strategy := cache.strategy.(type) {
	case *coordinateCache:
		strategy.Lock()
		defer strategy.Unlock()
		key := floorCoordinates(coordinates.Lat, coordinates.Lon)
		entry := strategy.weatherByCoord[key]
		entry.Valid_until = validUntil
		strategy.weatherByCoord[key] = entry
	case *cityCache:
		strategy.Lock()
		defer strategy.Unlock()
		entry := strategy.weatherByCity[cacheKey(city)]
		entry.Valid_until = validUntil
		strategy.weatherByCity[cacheKey(city)] = entry
	}
}

func cacheContainsEntry(cache *WeatherCache, city string, coordinates Coordinates) bool {
	switch strategy := cache.strategy.(type) {
	case *coordinateCache:
		strategy.RLock()
		defer strategy.RUnlock()
		_, exists := strategy.weatherByCoord[floorCoordinates(coordinates.Lat, coordinates.Lon)]
		return exists
	case *cityCache:
		strategy.RLock()
		defer strategy.RUnlock()
		_, exists := strategy.weatherByCity[cacheKey(city)]
		return exists
	default:
		return false
	}
}
