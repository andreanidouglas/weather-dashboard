package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/template"
	"github.com/rs/cors"
)

type MyHandler struct {
	standalone bool
}

func (h *MyHandler) HandleGet(w http.ResponseWriter, req *http.Request) {

	log.Printf("%v", req.URL.Path)

	//switch req.

	if strings.HasPrefix(req.URL.Path, "/api") {
		h.HandleWeather(w, req)
	} else {
		h.HandleStatic(w, req)
	}
}

func (h *MyHandler) HandleStatic(w http.ResponseWriter, req *http.Request) {

	if !h.standalone {
		log.Print("Serving static...")
		h.FileServer(http.Dir("./view/src/"), w, req)
	}
}

func (h *MyHandler) HandleWeather(w http.ResponseWriter, req *http.Request) {
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

}

func (h MyHandler) HandlePost(w http.ResponseWriter, req *http.Request) {

}

func (h MyHandler) FileServer(root http.FileSystem, w http.ResponseWriter, req *http.Request) {
	log.Printf("Got %v", root)
	fs := http.FileServer(root)
	fs.ServeHTTP(w, req)
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

	standalone_arg := os.Getenv("STANDALONE")

	standalone := false
	if standalone_arg == "true" {
		standalone = true
	}
	w := MyHandler{
		standalone: standalone,
	}

	log.Printf("Mode: %v", standalone)

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

	s.ErrorLog = log.Default()
	l := s.ErrorLog
	l.Print("Running server at: 8080")
	l.Fatal(s.ListenAndServe())
}
