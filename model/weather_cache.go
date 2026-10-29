package model

import (
	"strings"
	"sync"
	"time"
)

// CacheStrategy controls how weather entries are keyed.
type CacheStrategy uint8

const (
	// CacheByCoordinates keys weather entries by floored coordinates.
	CacheByCoordinates CacheStrategy = iota
	// CacheByCity preserves the legacy normalized-city key strategy.
	CacheByCity
)

type cacheStrategy interface {
	name() string
	getWeather(city string, fahrenheit bool) (bool, *Weather)
	getWeatherByCoordinates(coordinates Coordinates, fahrenheit bool) (bool, *Weather)
	setWeather(city string, weather Weather, fahrenheit bool)
	entries() []weatherCacheValue
}

// WeatherCache provides a common API for the available cache strategies.
type WeatherCache struct {
	strategy cacheStrategy
	stats    *cacheStats
}

type cacheStats struct {
	sync.Mutex
	hits     uint64
	requests uint64
}

type weatherCacheValue struct {
	Weather_c   Weather   `json:"weather_c"`
	Weather_f   Weather   `json:"weather_f"`
	Valid_until time.Time `json:"valid_until"`
}

type coordinateCache struct {
	sync.RWMutex
	weatherByCoord  map[Coordinates]weatherCacheValue
	cityCoordinates map[string]Coordinates
}

type cityCache struct {
	sync.RWMutex
	weatherByCity map[string]weatherCacheValue
}

func (*coordinateCache) name() string {
	return "coordinates"
}

func (*cityCache) name() string {
	return "city"
}

// NewCache creates a cache using coordinate keys by default.
func NewCache() WeatherCache {
	return NewCacheWithStrategy(CacheByCoordinates)
}

// NewCacheWithStrategy creates a cache using the requested key strategy.
// Unknown strategies default to coordinate keys.
func NewCacheWithStrategy(strategy CacheStrategy) WeatherCache {
	switch strategy {
	case CacheByCity:
		return WeatherCache{
			strategy: &cityCache{weatherByCity: map[string]weatherCacheValue{}},
			stats:    &cacheStats{},
		}
	default:
		return WeatherCache{
			strategy: &coordinateCache{
				weatherByCoord:  map[Coordinates]weatherCacheValue{},
				cityCoordinates: map[string]Coordinates{},
			},
			stats: &cacheStats{},
		}
	}
}

// cacheKey normalizes the requested city so " Rome " and "rome" share an entry.
func cacheKey(city string) string {
	return strings.ToLower(strings.TrimSpace(city))
}

// GetWeather looks up a city. Its first return value is true on a cache hit and
// false on a miss.
func (w *WeatherCache) GetWeather(city string, fahrenheit bool) (bool, *Weather) {
	hit, weather := w.strategy.getWeather(city, fahrenheit)
	w.recordLookup(hit)
	return hit, weather
}

// GetWeatherByCoordinates looks up floored coordinates. Its first return value
// is true on a cache hit and false on a miss. The city-key strategy always
// returns a miss here.
func (w *WeatherCache) GetWeatherByCoordinates(coordinates Coordinates, fahrenheit bool) (bool, *Weather) {
	hit, weather := w.strategy.getWeatherByCoordinates(coordinates, fahrenheit)
	w.recordLookup(hit)
	return hit, weather
}

// SetWeather stores a result using the configured cache strategy.
func (w *WeatherCache) SetWeather(city string, weather Weather, fahrenheit bool) {
	w.strategy.setWeather(city, weather, fahrenheit)
}

func (w *WeatherCache) recordLookup(hit bool) {
	w.stats.Lock()
	defer w.stats.Unlock()

	w.stats.requests++
	if hit {
		w.stats.hits++
	}
}

func (c *coordinateCache) getWeather(city string, fahrenheit bool) (bool, *Weather) {
	c.RLock()
	defer c.RUnlock()

	coordinates, ok := c.cityCoordinates[cacheKey(city)]
	if !ok {
		return false, nil
	}
	weather, ok := c.weatherByCoord[coordinates]
	return cachedWeather(weather, ok, fahrenheit)
}

func (c *coordinateCache) getWeatherByCoordinates(coordinates Coordinates, fahrenheit bool) (bool, *Weather) {
	c.RLock()
	defer c.RUnlock()

	key := floorCoordinates(coordinates.Lat, coordinates.Lon)
	weather, ok := c.weatherByCoord[key]
	return cachedWeather(weather, ok, fahrenheit)
}

func (c *coordinateCache) setWeather(city string, weather Weather, fahrenheit bool) {
	cache, coordinates := newWeatherCacheValue(weather, fahrenheit)

	c.Lock()
	defer c.Unlock()

	c.weatherByCoord[coordinates] = cache
	if key := cacheKey(city); key != "" {
		c.cityCoordinates[key] = coordinates
	}
}

func (c *coordinateCache) entries() []weatherCacheValue {
	c.RLock()
	defer c.RUnlock()

	entries := make([]weatherCacheValue, 0, len(c.weatherByCoord))
	for _, value := range c.weatherByCoord {
		entries = append(entries, value)
	}
	return entries
}

func (c *cityCache) getWeather(city string, fahrenheit bool) (bool, *Weather) {
	c.RLock()
	defer c.RUnlock()

	weather, ok := c.weatherByCity[cacheKey(city)]
	return cachedWeather(weather, ok, fahrenheit)
}

func (c *cityCache) getWeatherByCoordinates(_ Coordinates, _ bool) (bool, *Weather) {
	return false, nil
}

func (c *cityCache) setWeather(city string, weather Weather, fahrenheit bool) {
	cache, _ := newWeatherCacheValue(weather, fahrenheit)

	c.Lock()
	defer c.Unlock()
	c.weatherByCity[cacheKey(city)] = cache
}

func (c *cityCache) entries() []weatherCacheValue {
	c.RLock()
	defer c.RUnlock()

	entries := make([]weatherCacheValue, 0, len(c.weatherByCity))
	for _, value := range c.weatherByCity {
		entries = append(entries, value)
	}
	return entries
}

func cachedWeather(cache weatherCacheValue, exists bool, fahrenheit bool) (bool, *Weather) {
	if !exists || time.Now().After(cache.Valid_until) {
		return false, nil
	}
	if fahrenheit {
		return true, &cache.Weather_f
	}
	return true, &cache.Weather_c
}

func newWeatherCacheValue(weather Weather, fahrenheit bool) (weatherCacheValue, Coordinates) {
	weather.Coordinates = floorCoordinates(weather.Lat, weather.Lon)
	weatherC := weather
	weatherF := weather

	if fahrenheit {
		weatherC.CurrentTemp = 5.0 / 9.0 * (weather.CurrentTemp - 32.0)
		weatherC.FeelsLike = 5.0 / 9.0 * (weather.FeelsLike - 32.0)
		weatherC.MaxTemp = 5.0 / 9.0 * (weather.MaxTemp - 32.0)
		weatherC.MinTemp = 5.0 / 9.0 * (weather.MinTemp - 32.0)
	} else {
		weatherF.CurrentTemp = weather.CurrentTemp*9.0/5.0 + 32
		weatherF.FeelsLike = weather.FeelsLike*9.0/5.0 + 32
		weatherF.MaxTemp = weather.MaxTemp*9.0/5.0 + 32
		weatherF.MinTemp = weather.MinTemp*9.0/5.0 + 32
	}

	return weatherCacheValue{
		Weather_c:   weatherC,
		Weather_f:   weatherF,
		Valid_until: time.Now().Add(30 * time.Minute),
	}, weather.Coordinates
}

// Cache is the snapshot returned by the cache inspection endpoint. Requests
// counts cache lookups; hits counts successful lookups since cache creation.
type Cache struct {
	Strategy string              `json:"strategy"`
	Hits     uint64              `json:"hits"`
	Requests uint64              `json:"requests"`
	Entries  []weatherCacheValue `json:"entries"`
}

func GetCache(cache *WeatherCache) (Cache, error) {
	hits, requests := cache.stats.snapshot()
	return Cache{
		Strategy: cache.strategy.name(),
		Hits:     hits,
		Requests: requests,
		Entries:  cache.strategy.entries(),
	}, nil
}

func (s *cacheStats) snapshot() (uint64, uint64) {
	s.Lock()
	defer s.Unlock()
	return s.hits, s.requests
}
