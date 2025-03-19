package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/template"
)

type MyHandler struct {
	standalone bool
	apiContext model.ApiContext
	cache model.WeatherCache
}

// HandleWeather will response HTTP requests to GET /api/<city>?params=foo 
// to any http request with a valid HTML data
func (h *MyHandler) HandleWeather(w http.ResponseWriter, req *http.Request) {

	city := req.PathValue("city")

	// if url has param fahrenheit set, then serve the weather with imperial metric
	fahrenheit := req.FormValue("fahrenheit")
	fahrenheit_select := true
	if len(fahrenheit) == 0 {
		fahrenheit_select = false
	}
	log.Printf("Handle weather for: %s with fahreinheit: %v", req.PathValue("city"), fahrenheit)
	
	if city == "" {
		w.WriteHeader(400)
		w.Write([]byte("Need city parameter for API"))
		return
	}
	cityRequest := model.WeatherRequest{
		City: city,
		Fahrenheit: fahrenheit_select,
	}

	ok, weather := h.cache.GetWeather(cityRequest.City, cityRequest.Fahrenheit)
	if !ok {
		log.Printf("Cache miss for %s", cityRequest.City)
		weather_req, err := model.GetWeather(cityRequest, h.apiContext)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Could not get weather request"))
			return
		}

		weather = weather_req
		go h.cache.SetWeather(*weather_req, cityRequest.Fahrenheit)
	} else {
		log.Printf("Cache hit for %s", cityRequest.City)
	}
	
	// uses templ to render the template as HTML
	component := template.Weather(*weather, cityRequest)
	err := component.Render(req.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error rendering template %v", err)
		return
	}
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

	w := MyHandler{
		standalone: standalone,
		apiContext: apiContext,
		cache: cache,
	}

	s := &http.Server{
		Addr:           "0.0.0.0:8080",
		Handler:        mux,
		ReadTimeout:    300 * time.Millisecond, // TODO: find better values for these
		WriteTimeout:   900 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}

	// serve GET requests. eg: GET /api/Sao%20Paulo?param=foo
	mux.HandleFunc("GET /api/{city}", w.HandleWeather)

	// if standalone env varilable is set, then serve static files from ./view
	if w.standalone {
		mux.Handle("GET /", http.FileServer(http.Dir("./view/src")))
	}

	s.ErrorLog = log.Default()
	l := s.ErrorLog
	l.Print("Running server at: 0.0.0.0:8080")
	l.Fatal(s.ListenAndServe())
}
