package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	v1 "github.com/andreanidouglas/weather-dashboard/router/api/v1"
)

func main() {

	standalone_arg := os.Getenv("STANDALONE")
	key := os.Getenv("API_KEY")

	if len(key) == 0 {
		log.Fatalf("Need API_KEY as environment variable")
	}

	// apiContext := model.ApiContext{
	// 	Key: key,
	// }

	standalone := false
	if standalone_arg == "true" {
		standalone = true
	}

	log.Printf("Mode standalone: %v", standalone)

	mux := http.NewServeMux()

	s := &http.Server{
		Addr:           "0.0.0.0:8080",
		Handler:        mux,
		ReadTimeout:    300 * time.Millisecond, // TODO: find better values for these
		WriteTimeout:   900 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}

	ctx := model.ApiContext{
		Key: key,
	}

	weatherHandler := v1.NewWeatherHandler(ctx)
	healthHandler := v1.NewHealthHandler()

	mux.Handle("/api/v1/weather/", http.StripPrefix("/api/v1/weather", weatherHandler.API))
	mux.Handle("/api/v1/health/", http.StripPrefix("/api/v1/health", healthHandler.API))

	s.ErrorLog = log.Default()
	l := s.ErrorLog
	l.Print("Running server at: 0.0.0.0:8080")
	l.Fatal(s.ListenAndServe())
}
