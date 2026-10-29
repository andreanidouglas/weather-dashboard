package router

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// Routes returns a mux with every API route registered.
func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// serve GET requests. eg: GET /api/Sao%20Paulo?param=foo
	limit := httprate.Limit(
		30,
		10*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
	)
	mux.Handle("GET /api/text/{city}", limit(http.HandlerFunc(h.HandleTextWeather)))
	mux.Handle("GET /api/weather", limit(http.HandlerFunc(h.HandleWeatherAPI))) // eg: /api/weather?city=London&format=json
	mux.Handle("GET /api/forecast", limit(http.HandlerFunc(h.HandleForecast)))
	mux.Handle("GET /api/{city}", limit(http.HandlerFunc(h.HandleWeather)))
	mux.Handle("GET /api/suggest", limit(http.HandlerFunc(h.HandleSuggest)))
	mux.Handle("GET /api/cache", limit(http.HandlerFunc(h.HandleCache)))
	mux.Handle("GET /api/pos", limit(http.HandlerFunc(h.HandleLatLon)))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
