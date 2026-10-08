package router_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/router"
)

const testAPIKey = "test-api-key"

var (
	lastWeatherQuery url.Values
	mockWeatherBody  = `{"name":"São Paulo","sys":{"country":"BR"},"main":{"temp":22.5,"feels_like":23.0,"temp_min":20.0,"temp_max":25.0,"humidity":65},"weather":[{"main":"Clouds"}],"timezone":-10800}`
	mockGeocodeBody  = `[{"name":"São Paulo","lat":-23.5,"lon":-46.6,"country":"BR","state":"São Paulo"},{"name":"Santos","lat":-23.9,"lon":-46.3,"country":"BR","state":"São Paulo"}]`
)

// setup creates a fake OpenWeatherMap server and a router handler pointing at it.
func setup(t *testing.T) (*router.Handler, *httptest.Server, func()) {
	t.Helper()

	owm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/data/2.5/weather"):
			lastWeatherQuery = r.URL.Query()
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, mockWeatherBody)
		case strings.HasPrefix(r.URL.Path, "/geo/1.0/direct"):
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, mockGeocodeBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	origWeatherURL := model.WeatherDataURL
	origGeocodeURL := model.GeocodeURL
	model.WeatherDataURL = owm.URL + "/data/2.5/weather"
	model.GeocodeURL = owm.URL + "/geo/1.0/direct"

	ctx := &model.ApiContext{Key: testAPIKey}
	cache := model.NewCache()
	h := router.NewHandler(false, ctx, &cache)

	cleanup := func() {
		owm.Close()
		model.WeatherDataURL = origWeatherURL
		model.GeocodeURL = origGeocodeURL
	}

	return h, owm, cleanup
}

func newTestRouter(h *router.Handler) http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.HandleFunc("GET /api/text/{city}", h.HandleTextWeather)
	r.HandleFunc("GET /api/pos", h.HandleLatLon)
	r.HandleFunc("GET /api/suggest", h.HandleSuggest)
	r.HandleFunc("GET /api/cache", h.HandleCache)
	r.HandleFunc("GET /api/{city}", h.HandleWeather)
	return r
}

func TestHealth(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHandleWeather(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/S%C3%A3o%20Paulo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	if !strings.Contains(string(body), "São Paulo") {
		t.Errorf("expected response to contain city name, got: %s", string(body))
	}
	if !strings.Contains(string(body), "22.5") {
		t.Errorf("expected response to contain temperature, got: %s", string(body))
	}
	if !strings.Contains(string(body), "Clouds") {
		t.Errorf("expected response to contain condition, got: %s", string(body))
	}
}

func TestHandleWeatherMissingCity(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/", nil)
	req.SetPathValue("city", "")

	h.HandleWeather(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "Need city parameter") {
		t.Errorf("expected missing city error, got: %s", string(body))
	}
}

func TestHandleTextWeather(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/text/S%C3%A3o%20Paulo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected text/plain content-type, got: %s", contentType)
	}

	if !strings.Contains(string(body), "Weather for São Paulo, BR") {
		t.Errorf("expected header line, got: %s", string(body))
	}
	if !strings.Contains(string(body), "Current: 22.5°C") {
		t.Errorf("expected metric current temp, got: %s", string(body))
	}
}

func TestHandleTextWeatherFahrenheit(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	// First request is metric; it populates both metric and imperial cache entries.
	first, err := http.Get(server.URL + "/api/text/S%C3%A3o%20Paulo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	first.Body.Close()

	// Second request asks for Fahrenheit; it should be served from cache.
	resp, err := http.Get(server.URL + "/api/text/S%C3%A3o%20Paulo?fahrenheit=yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "°F") {
		t.Errorf("expected Fahrenheit unit, got: %s", string(body))
	}
	if !strings.Contains(string(body), "72.5") {
		t.Errorf("expected converted temperature 72.5, got: %s", string(body))
	}
}

func TestHandleSuggest(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/suggest?q=sao")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	if !strings.Contains(string(body), `<option value="São Paulo, São Paulo, BR" data-lat="-23.5" data-lon="-46.6">`) {
		t.Errorf("expected São Paulo option with coordinates, got: %s", string(body))
	}
}

func TestHandleSuggestSameNameCities(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	orig := mockGeocodeBody
	mockGeocodeBody = `[{"name":"Rome","lat":34.25,"lon":-85.16,"country":"US","state":"Georgia"},{"name":"Rome","lat":43.21,"lon":-75.45,"country":"US","state":"New York"},{"name":"Rome","lat":41.89,"lon":12.48,"country":"IT","state":"Lazio"}]`
	defer func() { mockGeocodeBody = orig }()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/suggest?q=rome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	for _, want := range []string{
		`<option value="Rome, Georgia, US" data-lat="34.25" data-lon="-85.16">`,
		`<option value="Rome, New York, US" data-lat="43.21" data-lon="-75.45">`,
		`<option value="Rome, Lazio, IT" data-lat="41.89" data-lon="12.48">`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("expected %s, got: %s", want, string(body))
		}
	}
}

func TestHandleSuggestShortQuery(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/suggest?q=a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}
	if strings.TrimSpace(string(body)) != "" {
		t.Errorf("expected empty body for short query, got: %s", string(body))
	}
}

func TestHandleCache(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	// Initially the cache is empty.
	resp, err := http.Get(server.URL + "/api/cache")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var empty model.Cache
	if err := json.Unmarshal(body, &empty); err != nil {
		t.Fatalf("failed to unmarshal empty cache: %v", err)
	}
	if len(empty.Entries) != 0 {
		t.Fatalf("expected empty cache, got %d entries", len(empty.Entries))
	}

	// Requesting weather should populate the cache.
	_, err = http.Get(server.URL + "/api/S%C3%A3o%20Paulo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err = http.Get(server.URL + "/api/cache")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	var cache model.Cache
	if err := json.Unmarshal(body, &cache); err != nil {
		t.Fatalf("failed to unmarshal cache: %v", err)
	}
	if len(cache.Entries) != 1 {
		t.Fatalf("expected 1 cache entry, got %d", len(cache.Entries))
	}
}

func TestHandleLatLon(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/pos?lat=-23.5&lon=-46.6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	if !strings.Contains(string(body), "São Paulo") {
		t.Errorf("expected response to contain city name, got: %s", string(body))
	}
	if !strings.Contains(string(body), "22.5") {
		t.Errorf("expected response to contain temperature, got: %s", string(body))
	}
}

func TestHandleLatLonMissingParam(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/pos")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleLatLonFahrenheit(t *testing.T) {
	h, _, cleanup := setup(t)
	defer cleanup()

	server := httptest.NewServer(newTestRouter(h))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/pos?lat=41.89&lon=12.48&fahrenheit=yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if got := lastWeatherQuery.Get("units"); got != "imperial" {
		t.Errorf("expected units=imperial, got %q", got)
	}
	if lastWeatherQuery.Get("lat") != "41.890000" || lastWeatherQuery.Get("lon") != "12.480000" {
		t.Errorf("expected lat/lon forwarded, got %v", lastWeatherQuery)
	}
}
