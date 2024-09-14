package main

import (
	"log"
	"net/http"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/template"
	"github.com/rs/cors"
)

type MyHandler struct {
}

func (h MyHandler) HandleGet(w http.ResponseWriter, req *http.Request) {

	weatherResponse := model.Weather{

		CurrentTemp: 32.2,
		MaxTemp:     34.6,
		MinTemp:     23.5,
		FeelsLike:   33.0,
		City:        "São Paulo",
		Country:     "Brazil",
		Condition:   "Overcast",
		Humidity:    0.33,
	}

	component := template.Weather(weatherResponse)
	component.Render(req.Context(), w)

	// var weatherRequest model.WeatherRequest
	// weatherDecoder := json.NewDecoder(req.Body)
	// weatherDecoder.Decode(&weatherRequest)

	// weatherResponse := []model.Weather{
	// 	{
	// 		CurrentTemp: 32.2,
	// 		MaxTemp:     34.6,
	// 		MinTemp:     23.5,
	// 		FeelsLike:   33.0,
	// 		City:        "São Paulo",
	// 	},
	// 	{
	// 		CurrentTemp: 12.3,
	// 		MaxTemp:     15.4,
	// 		MinTemp:     7.3,
	// 		FeelsLike:   11,
	// 		City:        "London",
	// 	},
	// 	{
	// 		CurrentTemp: 34.5,
	// 		MaxTemp:     34.9,
	// 		MinTemp:     30.1,
	// 		FeelsLike:   38.3,
	// 		City:        "Canberra",
	// 	},
	// }
	// w.Header().Add("content-type", "application/json")
	// weatherEncoder := json.NewEncoder(w)
	// weatherEncoder.Encode(weatherResponse)

}

func (h MyHandler) HandlePost(w http.ResponseWriter, req *http.Request) {

}

func (h MyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		h.HandleGet(w, req)
	case http.MethodPost:
		h.HandlePost(w, req)
	}
}

func main() {

	w := MyHandler{}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{""},
		AllowCredentials: false,
		Debug:            true,
		AllowedHeaders:   []string{"hx-current-url", "hx-request"},
	})

	handler := c.Handler(w)

	s := &http.Server{
		Addr:           "0.0.0.0:8080",
		Handler:        handler,
		ReadTimeout:    300 * time.Millisecond,
		WriteTimeout:   300 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}
	log.Print("Running server at: 8080")
	log.Fatal(s.ListenAndServe())
}
