package model

import (
	"strings"
	"sync"
	"time"
)

// Naive cache for Weather Requests
type WeatherCache struct {
	sync.RWMutex
	Weather map[string]weatherCacheValue `json:"weather"`
}

type weatherCacheValue struct {
	Weather_c Weather `json:"weather_c"`
	Weather_f Weather `json:"weather_f"`
	Valid_until time.Time `json:"valid_until"`
}

func NewCache() WeatherCache {
	m := make(map[string]weatherCacheValue)

	return WeatherCache{
		Weather: m,
	}
}

// cacheKey normalizes the requested city so " Rome " and "rome" share an entry.
// Entries are keyed by the request, not the city name the API returns, since
// different requests (eg "Rome,IT" and "Rome,US") can return the same name.
func cacheKey(city string) string {
	return strings.ToLower(strings.TrimSpace(city))
}

// Check the cache if already contain the city requested
func (w* WeatherCache) GetWeather(city string, fahrenheit bool) (bool, *Weather) {
	w.RLock()
	defer w.RUnlock()
	weatherCache := w.Weather[cacheKey(city)]
	if weatherCache.Weather_c.City == ""  {
		return false, nil 
	}

	if time.Now().Compare(weatherCache.Valid_until) > 0 {
		return false, nil
	}

	if fahrenheit {
		return true, &weatherCache.Weather_f 

	}

	return true, &weatherCache.Weather_c

}

// Create a new entry on the cache for the requested city
func (w* WeatherCache) SetWeather(city string, weather Weather, fahrenheit bool) {
	weather_c := weather
	weather_f := weather

	if fahrenheit {
		current_temp_c := 5.0/9.0 * (weather.CurrentTemp - 32.0)
		feels_like_c := 5.0/9.0 * (weather.FeelsLike - 32.0)
		max_temp_c := 5.0/9.0 * (weather.MaxTemp - 32.0)
		min_temp_c := 5.0/9.0 * (weather.MinTemp -32.0)

		weather_c.CurrentTemp = current_temp_c
		weather_c.FeelsLike = feels_like_c
		weather_c.MaxTemp = max_temp_c
		weather_c.MinTemp = min_temp_c
	} else {
		current_temp_f := weather.CurrentTemp * 9.0/5.0 + 32
		feels_like_f :=  weather.FeelsLike * 9.0/5.0 + 32
		max_temp_f := weather.MaxTemp * 9.0/5.0 + 32
		min_temp_f := weather.MinTemp * 9.0/5.0 + 32

		weather_f.CurrentTemp = current_temp_f
		weather_f.FeelsLike = feels_like_f
		weather_f.MaxTemp = max_temp_f
		weather_f.MinTemp = min_temp_f
	}

	cache := weatherCacheValue{
		Weather_c: weather_c,
		Weather_f: weather_f,
		Valid_until: time.Now().Add(time.Duration(time.Minute * 30)),
	}


	w.Lock()
	defer w.Unlock()
	w.Weather[cacheKey(city)] = cache
}



type Cache struct {
	Entries []weatherCacheValue `json:"entries"`
}

func GetCache(cache *WeatherCache) (Cache, error) {
	retCache := Cache{}

	retCache.Entries = make([]weatherCacheValue, 0)

	cache.RLock()
	defer cache.RUnlock()
	for _, value := range cache.Weather {
		retCache.Entries = append(retCache.Entries, value)
	}

	return retCache, nil
}
