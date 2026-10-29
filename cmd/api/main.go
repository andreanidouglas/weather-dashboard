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
	cache := model.NewCache()
	defer cache.Close()
	h := router.NewHandler(&model.ApiContext{Key: cfg.APIKey}, &cache)

	s := &http.Server{
		Addr:           config.Addr,
		Handler:        middleware.LogRecover(h.Routes()),
		ReadTimeout:    300 * time.Millisecond, // TODO: find better values for these
		WriteTimeout:   12 * time.Second,
		MaxHeaderBytes: 10 << 10,
		ErrorLog:       log.Default(),
	}

	log.Printf("Running server at: %s", config.Addr)
	if err := s.ListenAndServe(); err != nil {
		cache.Close()
		log.Fatal(err)
	}
}
