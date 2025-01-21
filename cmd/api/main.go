package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/template"
	"github.com/rs/cors"
)

type MyHandler struct {
	standalone bool
	apiContext model.ApiContext
}

func (h *MyHandler) HandleGet(w http.ResponseWriter, req *http.Request) {

	log.Printf("GET %v", req.URL.Path)
}

func (h *MyHandler) HandleStatic(w http.ResponseWriter, req *http.Request) {

	if !h.standalone {
		log.Print("Serving static...")
		h.FileServer(http.Dir("./view/src/"), w, req)
	}
}

func (h *MyHandler) HandleWeather(w http.ResponseWriter, req *http.Request) {

	log.Printf("Handle weather for: %s", req.PathValue("city"))
	city := req.PathValue("city")
	if city == "" {
		w.WriteHeader(400)
		w.Write([]byte("Need city parameter for API"))
		return
	}
	cityRequest := model.WeatherRequest{
		City: city,
	}
	
	weather, err := model.GetWeather(cityRequest, h.apiContext)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Could not get weather request"))
		return
	}

	component := template.Weather(*weather)
	err = component.Render(req.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error rendering template %v", err)
		return
	}
}

func (h MyHandler) HandlePost(w http.ResponseWriter, req *http.Request) {
	log.Printf("POST %v", req.URL.Path)
}

func (h MyHandler) FileServer(root http.FileSystem, w http.ResponseWriter, req *http.Request) {
	log.Printf("Got %v", root)
	fs := http.FileServer(root)
	fs.ServeHTTP(w, req)
}

func main() {

	standalone_arg := os.Getenv("STANDALONE")
	key := os.Getenv("API_KEY")

	apiContext := model.ApiContext{
		Key: key,
	}

	standalone := false
	if standalone_arg == "true" {
		standalone = true
	}

	log.Printf("Mode: %v", standalone)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{""},
		AllowCredentials: false,
		Debug:            false,
	})

	mux := http.NewServeMux()

	w := MyHandler{
		standalone: standalone,
		apiContext: apiContext,
	}
	handler := c.Handler(mux)

	s := &http.Server{
		Addr:           "0.0.0.0:8080",
		Handler:        handler,
		ReadTimeout:    300 * time.Millisecond,
		WriteTimeout:   900 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}

	mux.HandleFunc("GET /api/{city}", w.HandleWeather)
	if w.standalone {
		mux.Handle("GET /", http.FileServer(http.Dir("./view/src")))
	}

	s.ErrorLog = log.Default()
	l := s.ErrorLog
	l.Print("Running server at: 8080")
	l.Fatal(s.ListenAndServe())
}
