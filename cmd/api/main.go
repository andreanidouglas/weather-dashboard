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

	//h.mux.HandleFunc("/api/", h.HandleWeather)

	//	h.mux.HandleFunc("GET /api/{id}", h.HandleWeather)

	// if strings.HasPrefix(req.URL.Path, "/api") {
	// 	h.HandleWeather(w, req)
	// } else {
	// 	h.HandleStatic(w, req)
	// }
}

func (h *MyHandler) HandleStatic(w http.ResponseWriter, req *http.Request) {

	if !h.standalone {
		log.Print("Serving static...")
		h.FileServer(http.Dir("./view/src/"), w, req)
	}
}

func (h *MyHandler) HandleWeather(w http.ResponseWriter, req *http.Request) {

	city := "Sao Paulo"
	cityRequest := model.WeatherRequest {
		City: city,
	}
	weather, err := model.GetWeather(cityRequest, h.apiContext)
	if err != nil {

		w.WriteHeader(500)
		w.Write([]byte("Could not get weather request"))
		return
	}

/*
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
*/
	component := template.Weather(*weather)
	component.Render(req.Context(), w)

}

func (h MyHandler) HandlePost(w http.ResponseWriter, req *http.Request) {
	log.Printf("POST %v", req.URL.Path)
}

func (h MyHandler) FileServer(root http.FileSystem, w http.ResponseWriter, req *http.Request) {
	log.Printf("Got %v", root)
	fs := http.FileServer(root)
	fs.ServeHTTP(w, req)
}

// func (h MyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

// 	//h.mux.Handle("GET /", h)
// 	// switch req.Method {
// 	// case http.MethodGet:
// 	// 	h.HandleGet(w, req)
// 	// case http.MethodPost:
// 	// 	h.HandlePost(w, req)
// 	// }
// }

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
		//AllowedHeaders:   []string{"hx-current-url", "hx-request"},
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
		WriteTimeout:   300 * time.Millisecond,
		MaxHeaderBytes: 10 << 10,
	}

	mux.HandleFunc("GET /api/", w.HandleWeather)
	if w.standalone {
		mux.Handle("GET /", http.FileServer(http.Dir("./view/src")))
	}

	s.ErrorLog = log.Default()
	l := s.ErrorLog
	l.Print("Running server at: 8080")
	l.Fatal(s.ListenAndServe())
}
