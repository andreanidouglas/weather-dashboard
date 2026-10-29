package model

import (
	"strings"
	"sync"
	"time"
)

const cacheCleanupInterval = 3 * time.Second

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
	getWeather(city string) (bool, *Weather)
	getWeatherByCoordinates(coordinates Coordinates) (bool, *Weather)
	setWeather(city string, weather Weather)
	entries() []weatherCacheValue
	deleteExpired(now time.Time)
}

// WeatherCache provides a common API for the available cache strategies.
type WeatherCache struct {
	strategy cacheStrategy
	stats    *cacheStats
	sweeper  *cacheSweeper
}

type cacheSweeper struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

type cacheStats struct {
	sync.Mutex
	hits     uint64
	requests uint64
}

type weatherCacheValue struct {
	Weather    Weather   `json:"weather"`
	ValidUntil time.Time `json:"valid_until"`
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
	var implementation cacheStrategy
	switch strategy {
	case CacheByCity:
		implementation = &cityCache{weatherByCity: map[string]weatherCacheValue{}}
	default:
		implementation = &coordinateCache{
			weatherByCoord:  map[Coordinates]weatherCacheValue{},
			cityCoordinates: map[string]Coordinates{},
		}
	}

	cache := WeatherCache{
		strategy: implementation,
		stats:    &cacheStats{},
		sweeper: &cacheSweeper{
			stop: make(chan struct{}),
			done: make(chan struct{}),
		},
	}
	go cache.runSweeper()
	return cache
}

// cacheKey normalizes the requested city so " Rome " and "rome" share an entry.
func cacheKey(city string) string {
	return strings.ToLower(strings.TrimSpace(city))
}

// GetWeather looks up a city. Its first return value is true on a cache hit and
// false on a miss.
func (w *WeatherCache) GetWeather(city string) (bool, *Weather) {
	hit, weather := w.strategy.getWeather(city)
	w.recordLookup(hit)
	return hit, weather
}

// GetWeatherByCoordinates looks up floored coordinates. Its first return value
// is true on a cache hit and false on a miss. The city-key strategy always
// returns a miss here.
func (w *WeatherCache) GetWeatherByCoordinates(coordinates Coordinates) (bool, *Weather) {
	hit, weather := w.strategy.getWeatherByCoordinates(coordinates)
	w.recordLookup(hit)
	return hit, weather
}

// SetWeather stores a result using the configured cache strategy.
func (w *WeatherCache) SetWeather(city string, weather Weather) {
	w.strategy.setWeather(city, weather)
}

// Close stops the periodic expired-entry sweeper.
func (w *WeatherCache) Close() {
	w.sweeper.once.Do(func() {
		close(w.sweeper.stop)
	})
	<-w.sweeper.done
}

func (w *WeatherCache) runSweeper() {
	ticker := time.NewTicker(cacheCleanupInterval)
	defer ticker.Stop()
	defer close(w.sweeper.done)

	for {
		select {
		case now := <-ticker.C:
			w.strategy.deleteExpired(now)
		case <-w.sweeper.stop:
			return
		}
	}
}

func (w *WeatherCache) recordLookup(hit bool) {
	w.stats.Lock()
	defer w.stats.Unlock()

	w.stats.requests++
	if hit {
		w.stats.hits++
	}
}

func (c *coordinateCache) getWeather(city string) (bool, *Weather) {
	c.RLock()
	defer c.RUnlock()

	coordinates, ok := c.cityCoordinates[cacheKey(city)]
	if !ok {
		return false, nil
	}
	weather, ok := c.weatherByCoord[coordinates]
	return cachedWeather(weather, ok)
}

func (c *coordinateCache) getWeatherByCoordinates(coordinates Coordinates) (bool, *Weather) {
	c.RLock()
	defer c.RUnlock()

	key := floorCoordinates(coordinates.Lat, coordinates.Lon)
	weather, ok := c.weatherByCoord[key]
	return cachedWeather(weather, ok)
}

func (c *coordinateCache) setWeather(city string, weather Weather) {
	cache, coordinates := newWeatherCacheValue(weather)

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

func (c *coordinateCache) deleteExpired(now time.Time) {
	c.Lock()
	defer c.Unlock()

	for coordinates, entry := range c.weatherByCoord {
		if entry.ValidUntil.Before(now) {
			delete(c.weatherByCoord, coordinates)
		}
	}
	for city, coordinates := range c.cityCoordinates {
		if _, exists := c.weatherByCoord[coordinates]; !exists {
			delete(c.cityCoordinates, city)
		}
	}
}

func (c *cityCache) getWeather(city string) (bool, *Weather) {
	c.RLock()
	defer c.RUnlock()

	weather, ok := c.weatherByCity[cacheKey(city)]
	return cachedWeather(weather, ok)
}

func (c *cityCache) getWeatherByCoordinates(_ Coordinates) (bool, *Weather) {
	return false, nil
}

func (c *cityCache) setWeather(city string, weather Weather) {
	cache, _ := newWeatherCacheValue(weather)

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

func (c *cityCache) deleteExpired(now time.Time) {
	c.Lock()
	defer c.Unlock()

	for city, entry := range c.weatherByCity {
		if entry.ValidUntil.Before(now) {
			delete(c.weatherByCity, city)
		}
	}
}

func cachedWeather(cache weatherCacheValue, exists bool) (bool, *Weather) {
	if !exists || time.Now().After(cache.ValidUntil) {
		return false, nil
	}
	return true, &cache.Weather
}

func newWeatherCacheValue(weather Weather) (weatherCacheValue, Coordinates) {
	weather.Coordinates = floorCoordinates(weather.Coordinates.Lat, weather.Coordinates.Lon)

	var weatherC, weatherF Temperature
	switch {
	case weather.Celsius.Unit == UnitCelsius && weather.Fahrenheit.Unit == UnitFahrenheit:
		weatherC = weather.Celsius
		weatherF = weather.Fahrenheit
	case weather.Celsius.Unit == UnitCelsius:
		weatherC = weather.Celsius
		weatherF = weatherC.ConvertTo(UnitFahrenheit)
	case weather.Fahrenheit.Unit == UnitFahrenheit:
		weatherF = weather.Fahrenheit
		weatherC = weatherF.ConvertTo(UnitCelsius)
	default:
		weatherC = weather.Celsius
		weatherC.Unit = UnitCelsius
		weatherF = weatherC.ConvertTo(UnitFahrenheit)
	}
	weather.Celsius = weatherC
	weather.Fahrenheit = weatherF

	return weatherCacheValue{
		Weather:    weather,
		ValidUntil: time.Now().Add(30 * time.Minute),
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
