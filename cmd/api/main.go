package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/router"
	"github.com/go-chi/httprate"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// logRecover logs each request and turns handler panics into a 500 response.
func logRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if err := recover(); err != nil {
				if err == http.ErrAbortHandler {
					panic(err)
				}
				log.Printf("panic: %v", err)
				rec.WriteHeader(http.StatusInternalServerError)
			}
			log.Printf("%s %s from %s - %d in %v", r.Method, r.URL, r.RemoteAddr, rec.status, time.Since(start))
		}()
		next.ServeHTTP(rec, r)
	})
}

func main() {

	standalone_arg := os.Getenv("STANDALONE")
	key := os.Getenv("API_KEY")

	if len(key) == 0 {
		log.Fatalf("Need API_KEY as environment variable")
	}

	apiContext := model.ApiContext{
		Key: key,
	}

	standalone := false
	if standalone_arg == "true" {
		standalone = true
	}

	log.Printf("Mode standalone: %v", standalone)

	mux := http.NewServeMux()

	cache := model.NewCache()

	w := router.NewHandler(standalone, &apiContext, &cache)

	s := &http.Server{
		Addr:           "0.0.0.0:8080",
		Handler:        logRecover(mux),
		ReadTimeout:    300 * time.Millisecond, // TODO: find better values for these
		WriteTimeout:   900 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}

	// serve GET requests. eg: GET /api/Sao%20Paulo?param=foo
	limit := httprate.Limit(
		30,
		10*time.Second,
		httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
	)
	mux.Handle("GET /api/text/{city}", limit(http.HandlerFunc(w.HandleTextWeather)))
	mux.Handle("GET /api/{city}", limit(http.HandlerFunc(w.HandleWeather)))
	mux.Handle("GET /api/suggest", limit(http.HandlerFunc(w.HandleSuggest)))
	mux.Handle("GET /api/cache", limit(http.HandlerFunc(w.HandleCache)))
	mux.Handle("GET /api/pos", limit(http.HandlerFunc(w.HandleLatLon)))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// if standalone env varilable is set, then serve static files from ./src/view
	if w.Standalone {
		w.FileServer(mux, "/", http.Dir(filepath.Join(".", "view/src")))
	}

	s.ErrorLog = log.Default()
	l := s.ErrorLog
	l.Print("Running server at: 0.0.0.0:8080")
	l.Fatal(s.ListenAndServe())
}
