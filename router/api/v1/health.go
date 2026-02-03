package v1

import (
	"net/http"
)

type HealthHandler struct {
	API http.Handler
}

func NewHealthHandler() *WeatherHandler {
	api := http.NewServeMux()

	api.HandleFunc("GET /", GetHealth())

	return &WeatherHandler{
		API: api,
	}
}

func GetHealth() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}
}
