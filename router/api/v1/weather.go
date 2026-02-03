package v1

import (
	"encoding/json"
	"net/http"

	"github.com/andreanidouglas/weather-dashboard/model"
)

type WeatherHandler struct {
	API        http.Handler
	apiContext model.ApiContext
}

func NewWeatherHandler(apiContext model.ApiContext) *WeatherHandler {
	api := http.NewServeMux()

	wh := &WeatherHandler{
		API:        api,
		apiContext: apiContext,
	}

	api.HandleFunc("GET /", wh.WeatherByCity())

	return wh
}

func (wh *WeatherHandler) WeatherByCity() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		city := query.Get("city")
		fahrenheit := false

		if city == "" {
			http.Error(w, "Parameter city is mandatory", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		wr := model.WeatherRequest{
			City:       city,
			Fahrenheit: fahrenheit,
		}
		weather, err := model.GetWeather(wr, &wh.apiContext)
		if err != nil {
			http.Error(w, "Error in the API layer", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(200)

		err = json.NewEncoder(w).Encode(weather)
		if err != nil {
			http.Error(w, "Error encoding json", http.StatusInternalServerError)
			return
		}
	}
}
