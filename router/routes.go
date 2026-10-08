package router

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

// Routes returns a mux with every API route registered. In standalone mode it
// also serves the static frontend from ./view/src.
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
	mux.Handle("GET /api/{city}", limit(http.HandlerFunc(h.HandleWeather)))
	mux.Handle("GET /api/suggest", limit(http.HandlerFunc(h.HandleSuggest)))
	mux.Handle("GET /api/cache", limit(http.HandlerFunc(h.HandleCache)))
	mux.Handle("GET /api/pos", limit(http.HandlerFunc(h.HandleLatLon)))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if h.Standalone {
		fileServer(mux, "/", http.Dir(filepath.Join(".", "view/src")))
	}

	return mux
}

// fileServer serves static files from root under the given path prefix.
// ServeMux redirects requests for path without the trailing slash.
func fileServer(mux *http.ServeMux, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}") {
		panic("FileServer does not permit any URL parameters.")
	}

	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	mux.Handle("GET "+path, http.StripPrefix(strings.TrimSuffix(path, "/"), http.FileServer(root)))
}
