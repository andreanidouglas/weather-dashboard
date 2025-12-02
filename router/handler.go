package router

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/template"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Standalone bool
	apiContext *model.ApiContext
	cache      *model.WeatherCache
}

func NewHandler(standalone bool, apiCtx *model.ApiContext, cache *model.WeatherCache) *Handler {
	return &Handler{
		Standalone: standalone,
		apiContext: apiCtx,
		cache:      cache,
	}
}

func (h *Handler) FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

// HandleWeather will response HTTP requests to GET /api/<city>?params=foo
// to any http request with a valid HTML data
func (h *Handler) HandleWeather(w http.ResponseWriter, req *http.Request) {

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
		City:       city,
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
		// TODO: if request city is changed by the API (eg Sao Paulo -> São Paulo)
		//       the cache will always miss
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

// HandleSuggest serves /api/suggest?q=<partial> returning <option> list for datalist
func (h *Handler) HandleSuggest(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query().Get("q")
	if len(strings.TrimSpace(q)) < 2 { // require at least 2 chars
		w.WriteHeader(200)
		// empty response is fine for short queries
		return
	}

	locations, err := model.GetLocations(q, 8, h.apiContext)
	if err != nil {
		log.Printf("suggest error: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("Could not fetch suggestions"))
		return
	}

	component := template.CitySuggestions(locations)
	if err := component.Render(req.Context(), w); err != nil {
		log.Printf("render suggest error: %v", err)
		w.WriteHeader(500)
		return
	}
}

func (h *Handler) HandleWeatherByCoords(w http.ResponseWriter, req *http.Request) {
	lat := req.URL.Query().Get("lat")
	lon := req.URL.Query().Get("lon")
	if len(strings.TrimSpace(lat)) == 0 || len(strings.TrimSpace(lon)) == 0 {
		w.WriteHeader(400)
		w.Write([]byte("Need lat and lon parameters for API"))
		return
	}
	fahrenheit := req.FormValue("fahrenheit")
	fahrenheit_select := true
	if len(fahrenheit) == 0 {
		fahrenheit_select = false
	}

	lat_float, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Invalid lat parameter"))
		return
	}
	lon_float, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Invalid lon parameter"))
		return
	}

	log.Printf("Handle weather for coords: %s, %s", lat, lon)
	cityRequest := model.RequestWeatherByCoords{
		Latitude:   lat_float,
		Longitude:  lon_float,
		Fahrenheit: fahrenheit_select,
	}
	weather_req, err := model.GetWeatherByCoords(cityRequest, h.apiContext)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Could not get weather request"))
		return
	}
	weather := weather_req
	new_req := model.WeatherRequest{
		City:       weather.City,
		Fahrenheit: fahrenheit_select,
	}
	//add weather to cache
	go h.cache.SetWeather(*weather_req, cityRequest.Fahrenheit)

	// uses templ to render the template as HTML
	component := template.Weather(*weather, new_req)
	err = component.Render(req.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error rendering template %v", err)
		return
	}
}
