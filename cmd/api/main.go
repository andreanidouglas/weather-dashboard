package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andreanidouglas/weather-dashboard/config"
	"github.com/andreanidouglas/weather-dashboard/middleware"
	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/router"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Mode standalone: %v", cfg.Standalone)

	cache := model.NewCache()
	h := router.NewHandler(cfg.Standalone, &model.ApiContext{Key: cfg.APIKey}, &cache)

	s := &http.Server{
		Addr:           config.Addr,
		Handler:        middleware.LogRecover(h.Routes()),
		ReadTimeout:    300 * time.Millisecond, // TODO: find better values for these
		WriteTimeout:   900 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
		ErrorLog:       log.Default(),
	}

	log.Printf("Running server at: %s", config.Addr)
	log.Fatal(s.ListenAndServe())
}
