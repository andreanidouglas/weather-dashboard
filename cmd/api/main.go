package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/router"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

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

	mux := chi.NewRouter()

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RealIP)

	cache := model.NewCache()

	w := router.NewHandler(standalone, &apiContext, &cache)

	s := &http.Server{
		Addr:           "0.0.0.0:8080",
		Handler:        mux,
		ReadTimeout:    300 * time.Millisecond, // TODO: find better values for these
		WriteTimeout:   900 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}

	// serve GET requests. eg: GET /api/Sao%20Paulo?param=foo
	mux.Route("/api", func(r chi.Router) {
		r.Use(httprate.Limit(
			30,
			10*time.Second,
			httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		))
		r.Get("/{city}", w.HandleWeather)
		r.Get("/suggest", w.HandleSuggest)
		r.Get("/weather", w.HandleWeatherByCoords)
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
