package model

import (
	"time"
	"sync"
)


type WeatherCache struct {
	sync.RWMutex
	weather map[string]weatherCacheValue
}

type weatherCacheValue struct {
	weather_c Weather
	weather_f Weather
	valid_until time.Time
}

func NewCache() WeatherCache {
	m := make(map[string]weatherCacheValue)

	return WeatherCache{
		weather: m,
	}

}

func (w* WeatherCache) GetWeather(city string, fahreinheit bool) (bool, *Weather) {
	w.RLock()
	defer w.Unlock()
	weatherCache := w.weather[city]
	if weatherCache.weather_c.City == ""  {
		return false, nil 
	}

	if time.Now().Compare(weatherCache.valid_until) > 0 {
		return false, nil
	}

	if fahreinheit {
		return true, &weatherCache.weather_f 

	}

	return true, &weatherCache.weather_c

}

func (w* WeatherCache) SetWeather(weather Weather, fahreinheit bool) {
	weather_c := weather
	weather_f := weather

	if fahreinheit {
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
		weather_c: weather_c,
		weather_f: weather_f,
		valid_until: time.Now().Add(time.Duration(time.Minute * 30)),
	}


	w.Lock()
	defer w.Unlock()
	w.weather[weather.City] = cache
}
