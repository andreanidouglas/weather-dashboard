package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/router"
)

const (
	cityCount     = 1000
	totalRequests = 10000
)

type requestKind uint8

const (
	cityRequest requestKind = iota
	coordinateRequest
)

type location struct {
	name        string
	coordinates model.Coordinates
}

type benchmarkRequest struct {
	city        string
	coordinates model.Coordinates
}

type workload struct {
	name      string
	kind      requestKind
	locations []location
	requests  []benchmarkRequest
}

type weatherPayload struct {
	Coordinates model.Coordinates `json:"coord"`
	Conditions  []condition       `json:"weather"`
	Main        mainWeather       `json:"main"`
	System      system            `json:"sys"`
	Timezone    int               `json:"timezone"`
	Name        string            `json:"name"`
}

type condition struct {
	Main string `json:"main"`
}

type mainWeather struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Humidity  int     `json:"humidity"`
}

type system struct {
	Country string `json:"country"`
}

type result struct {
	strategy string
	hits     int64
	misses   int64
	entries  int
	elapsed  time.Duration
}

func main() {
	seed := flag.Int64("seed", 1, "seed for the reproducible random request sequence")
	flag.Parse()

	log.SetOutput(io.Discard)
	workloads := makeWorkloads(*seed)
	strategies := []struct {
		name     string
		strategy model.CacheStrategy
	}{
		{name: "coordinates", strategy: model.CacheByCoordinates},
		{name: "city", strategy: model.CacheByCity},
	}

	fmt.Printf("Workloads: %d random requests across %d cities per scenario (seed=%d)\n", totalRequests, cityCount, *seed)
	for _, testWorkload := range workloads {
		fmt.Printf("\n%s\n", testWorkload.name)
		fmt.Printf("%-14s %8s %8s %8s %8s %10s\n", "strategy", "hits", "misses", "total", "hit rate", "entries")
		for _, cacheStrategy := range strategies {
			benchmarkResult, err := runStrategy(
				cacheStrategy.name,
				cacheStrategy.strategy,
				testWorkload,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "benchmark %s strategy: %v\n", cacheStrategy.name, err)
				os.Exit(1)
			}
			printResult(benchmarkResult)
		}
	}
}

func makeWorkloads(seed int64) []workload {
	random := rand.New(rand.NewSource(seed))
	uniqueLocations := makeLocations(cityCount, cityCount)
	aliasedLocations := makeLocations(cityCount, 100)

	return []workload{
		makeCityWorkload("City names: 1,000 unique locations", uniqueLocations, random),
		makeCityWorkload("City aliases: 1,000 names in 100 coordinate buckets", aliasedLocations, random),
		makeCoordinateWorkload("Coordinate endpoint: 1,000 locations with sub-cent jitter", uniqueLocations, random),
	}
}

func makeLocations(count, bucketCount int) []location {
	locations := make([]location, 0, count)
	for i := range count {
		bucket := i * bucketCount / count
		locations = append(locations, location{
			name: fmt.Sprintf("City%04d", i),
			coordinates: model.Coordinates{
				Lat: 10.25 + float64(bucket/100),
				Lon: 20.5 + float64(bucket%100),
			},
		})
	}
	return locations
}

func makeCityWorkload(name string, locations []location, random *rand.Rand) workload {
	requests := make([]benchmarkRequest, totalRequests)
	for i := range totalRequests {
		city := locations[random.Intn(len(locations))]
		requests[i] = benchmarkRequest{city: city.name}
	}
	return workload{
		name:      name,
		kind:      cityRequest,
		locations: locations,
		requests:  requests,
	}
}

func makeCoordinateWorkload(name string, locations []location, random *rand.Rand) workload {
	requests := make([]benchmarkRequest, totalRequests)
	for i := range totalRequests {
		city := locations[random.Intn(len(locations))]
		jitter := float64(random.Intn(9)+1) / 1000
		requests[i] = benchmarkRequest{
			coordinates: model.Coordinates{
				Lat: city.coordinates.Lat + jitter,
				Lon: city.coordinates.Lon + jitter,
			},
		}
	}
	return workload{
		name:      name,
		kind:      coordinateRequest,
		locations: locations,
		requests:  requests,
	}
}

func runStrategy(name string, strategy model.CacheStrategy, testWorkload workload) (result, error) {
	locationsByCity := make(map[string]location, len(testWorkload.locations))
	locationsByCoordinates := make(map[model.Coordinates]location, len(testWorkload.locations))
	for _, city := range testWorkload.locations {
		locationsByCity[city.name] = city
		locationsByCoordinates[city.coordinates] = city
	}

	var upstreamRequests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests.Add(1)

		city, ok := lookupLocation(r, locationsByCity, locationsByCoordinates)
		if !ok {
			http.Error(w, "unknown benchmark location", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		payload := weatherPayload{
			Coordinates: city.coordinates,
			Conditions:  []condition{{Main: "Clear"}},
			Main: mainWeather{
				Temp:      20,
				FeelsLike: 19,
				TempMin:   18,
				TempMax:   22,
				Humidity:  50,
			},
			System:   system{Country: "XX"},
			Timezone: 0,
			Name:     city.name,
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, "could not encode mock weather", http.StatusInternalServerError)
		}
	}))
	defer upstream.Close()

	originalWeatherDataURL := model.WeatherDataURL
	model.WeatherDataURL = upstream.URL
	defer func() {
		model.WeatherDataURL = originalWeatherDataURL
	}()

	cache := model.NewCacheWithStrategy(strategy)
	defer cache.Close()
	handler := router.NewHandler(&model.ApiContext{Key: "benchmark-only-key"}, &cache)
	endpoint := newEndpoint(handler, testWorkload.kind)

	start := time.Now()
	for _, request := range testWorkload.requests {
		recorder := httptest.NewRecorder()
		endpoint.ServeHTTP(recorder, makeRequest(request, testWorkload.kind))
		if recorder.Code != http.StatusOK {
			return result{}, fmt.Errorf("request returned status %d", recorder.Code)
		}
	}
	elapsed := time.Since(start)

	misses := upstreamRequests.Load()
	cacheContents, err := model.GetCache(&cache)
	if err != nil {
		return result{}, err
	}

	return result{
		strategy: name,
		hits:     int64(len(testWorkload.requests)) - misses,
		misses:   misses,
		entries:  len(cacheContents.Entries),
		elapsed:  elapsed,
	}, nil
}

func lookupLocation(
	request *http.Request,
	locationsByCity map[string]location,
	locationsByCoordinates map[model.Coordinates]location,
) (location, bool) {
	if city := request.URL.Query().Get("q"); city != "" {
		location, ok := locationsByCity[city]
		return location, ok
	}

	lat, err := strconv.ParseFloat(request.URL.Query().Get("lat"), 64)
	if err != nil {
		return location{}, false
	}
	lon, err := strconv.ParseFloat(request.URL.Query().Get("lon"), 64)
	if err != nil {
		return location{}, false
	}

	key := floorCoordinates(model.Coordinates{Lat: lat, Lon: lon})
	location, ok := locationsByCoordinates[key]
	return location, ok
}

func newEndpoint(handler *router.Handler, kind requestKind) http.Handler {
	endpoint := http.NewServeMux()
	switch kind {
	case coordinateRequest:
		endpoint.HandleFunc("GET /api/pos", handler.HandleLatLon)
	default:
		endpoint.HandleFunc("GET /api/weather", handler.HandleWeatherAPI)
	}
	return endpoint
}

func makeRequest(request benchmarkRequest, kind requestKind) *http.Request {
	query := url.Values{}
	path := "/api/weather"
	switch kind {
	case coordinateRequest:
		path = "/api/pos"
		query.Set("lat", strconv.FormatFloat(request.coordinates.Lat, 'f', 3, 64))
		query.Set("lon", strconv.FormatFloat(request.coordinates.Lon, 'f', 3, 64))
	default:
		query.Set("city", request.city)
		query.Set("format", "json")
	}
	return httptest.NewRequest(http.MethodGet, path+"?"+query.Encode(), nil)
}

func floorCoordinates(coordinates model.Coordinates) model.Coordinates {
	return model.Coordinates{
		Lat: math.Floor(coordinates.Lat*100) / 100,
		Lon: math.Floor(coordinates.Lon*100) / 100,
	}
}

func printResult(result result) {
	total := result.hits + result.misses
	hitRate := float64(result.hits) / float64(total) * 100
	fmt.Printf("%-14s %8d %8d %8d %7.2f%% %10d  (%s)\n",
		result.strategy,
		result.hits,
		result.misses,
		total,
		hitRate,
		result.entries,
		result.elapsed.Round(time.Millisecond),
	)
}
