package router

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/template"
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

// HandleTextWeather serves GET /api/text/<city> as plain text.
func (h *Handler) HandleTextWeather(w http.ResponseWriter, req *http.Request) {

	city := req.PathValue("city")

	fahrenheit := req.FormValue("fahrenheit")
	fahrenheit_select := true
	if len(fahrenheit) == 0 {
		fahrenheit_select = false
	}

	if city == "" {
		w.WriteHeader(400)
		w.Write([]byte("Need city parameter for API"))
		return
	}

	cityRequest := model.WeatherRequest{
		City:       city,
		Fahrenheit: fahrenheit_select,
	}

	weather, err := h.cachedWeather(cityRequest)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Could not get weather request"))
		return
	}

	writeTextWeather(w, weather, cityRequest.Fahrenheit)
}

// weatherResponse is the JSON/XML body of GET /api/weather.
type weatherResponse struct {
	XMLName     xml.Name `json:"-" xml:"weather"`
	City        string   `json:"city" xml:"city"`
	Country     string   `json:"country" xml:"country"`
	Units       string   `json:"units" xml:"units"`
	CurrentTemp float64  `json:"current_temp" xml:"current_temp"`
	FeelsLike   float64  `json:"feels_like" xml:"feels_like"`
	MaxTemp     float64  `json:"max_temp" xml:"max_temp"`
	MinTemp     float64  `json:"min_temp" xml:"min_temp"`
	Condition   string   `json:"condition" xml:"condition"`
	Humidity    float64  `json:"humidity" xml:"humidity"`
	Timezone    int      `json:"timezone" xml:"timezone"`
}

// HandleWeatherAPI serves GET /api/weather?city=London&format=text|json|xml
// for terminal use (eg curl). format defaults to text; set fahrenheit=<any>
// for imperial units.
func (h *Handler) HandleWeatherAPI(w http.ResponseWriter, req *http.Request) {
	city := strings.TrimSpace(req.URL.Query().Get("city"))
	format := strings.ToLower(req.URL.Query().Get("format"))
	if format == "" {
		format = "text"
	}

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	if city == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Missing city parameter. Usage: /api/weather?city=London&format=text|json|xml")
		return
	}
	if format != "text" && format != "json" && format != "xml" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid format %q. Use text, json or xml\n", format)
		return
	}

	cityRequest := model.WeatherRequest{
		City:       city,
		Fahrenheit: req.URL.Query().Get("fahrenheit") != "",
	}

	weather, err := h.cachedWeather(cityRequest)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Could not get weather for %s\n", city)
		return
	}

	if format == "text" {
		writeTextWeather(w, weather, cityRequest.Fahrenheit)
		return
	}

	units := "metric"
	if cityRequest.Fahrenheit {
		units = "imperial"
	}
	res := weatherResponse{
		City:        weather.City,
		Country:     weather.Country,
		Units:       units,
		CurrentTemp: round2(weather.CurrentTemp),
		FeelsLike:   round2(weather.FeelsLike),
		MaxTemp:     round2(weather.MaxTemp),
		MinTemp:     round2(weather.MinTemp),
		Condition:   weather.Condition,
		Humidity:    weather.Humidity,
		Timezone:    weather.Timezone,
	}

	if format == "json" {
		w.Header().Set("content-type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(res)
		return
	}

	w.Header().Set("content-type", "application/xml; charset=utf-8")
	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(res)
	fmt.Fprintln(w)
}

// cachedWeather returns the weather for a city from the cache, fetching and
// caching it on a miss.
func (h *Handler) cachedWeather(cityRequest model.WeatherRequest) (*model.Weather, error) {
	ok, weather := h.cache.GetWeather(cityRequest.City, cityRequest.Fahrenheit)
	if ok {
		log.Printf("Cache hit for %s", cityRequest.City)
		return weather, nil
	}

	log.Printf("Cache miss for %s", cityRequest.City)
	weather, err := model.GetWeather(cityRequest, h.apiContext)
	if err != nil {
		return nil, err
	}
	h.cache.SetWeather(cityRequest.City, *weather, cityRequest.Fahrenheit)
	return weather, nil
}

// round2 trims float noise from °C/°F conversions, eg 55.364000000000004.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func writeTextWeather(w http.ResponseWriter, weather *model.Weather, fahrenheit bool) {
	unit := "°C"
	if fahrenheit {
		unit = "°F"
	}

	w.Header().Set("content-type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Weather for %s, %s\n", weather.City, weather.Country)
	fmt.Fprintf(w, "Current: %.1f%s\n", weather.CurrentTemp, unit)
	fmt.Fprintf(w, "Feels like: %.1f%s\n", weather.FeelsLike, unit)
	fmt.Fprintf(w, "Condition: %s\n", weather.Condition)
	fmt.Fprintf(w, "High: %.1f%s / Low: %.1f%s\n", weather.MaxTemp, unit, weather.MinTemp, unit)
	fmt.Fprintf(w, "Humidity: %.1f%%\n", weather.Humidity)
}

func (h *Handler) HandleWeather(w http.ResponseWriter, req *http.Request) {

	city := req.PathValue("city")

	// if url has param fahrenheit set, then serve the weather with imperial metric
	fahrenheit := req.FormValue("fahrenheit")
	fahrenheit_select := true
	if len(fahrenheit) == 0 {
		fahrenheit_select = false
	}
	log.Printf("Handle weather for: %s with fahrenheit: %v", req.PathValue("city"), fahrenheit)

	if city == "" {
		w.WriteHeader(400)
		w.Write([]byte("Need city parameter for API"))
		return
	}
	cityRequest := model.WeatherRequest{
		City:       city,
		Fahrenheit: fahrenheit_select,
	}

	weather, err := h.cachedWeather(cityRequest)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Could not get weather request"))
		return
	}

	// uses templ to render the template as HTML
	component := template.Weather(*weather, cityRequest)
	err = component.Render(req.Context(), w)
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

func (h *Handler) HandleCache(w http.ResponseWriter, req *http.Request) {

	cache, err := model.GetCache(h.cache)
	if err != nil {
		log.Printf("could not get cache: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("{\"error\": \"cannot get cache\"}"))
		return
	}

	cacheJson, err := json.Marshal(cache)
	if err != nil {
		log.Printf("could not get cache: %v", err)
		w.WriteHeader(500)
		w.Write([]byte("{\"error\": \"cannot marshall cache\"}"))
		return
	}

	w.WriteHeader(200)
	w.Header().Add("content-type", "application/json")
	w.Write(cacheJson)
}

func (h *Handler) HandleLatLon(w http.ResponseWriter, req *http.Request) {

	lat := req.URL.Query().Get("lat")
	lon := req.URL.Query().Get("lon")

	if lat == "" || lon == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\": \"missing lat or lon parameters\"}"))
		return
	}

	lat_value, lat_err := strconv.ParseFloat(lat, 64)
	lon_value, lon_err := strconv.ParseFloat(lon, 64)

	if lat_err != nil || lon_err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"error\": \"lat or lon parameters with invalid value. cannot parse float\"}"))
		return

	}

	fahrenheit := req.FormValue("fahrenheit")
	fahrenheit_select := true
	if len(fahrenheit) == 0 {
		fahrenheit_select = false
	}

	lat_lon := model.LatLon{
		Lat:        lat_value,
		Lon:        lon_value,
		Fahrenheit: fahrenheit_select,
	}

	weather_req, err := model.GetWeatherByLatLon(lat_lon, *h.apiContext)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		err_message := fmt.Sprintf("{\"error\": \"cannot get value for lat: %f / lon: %f: %v\"}", lat_value, lon_value, err)
		w.Write([]byte(err_message))
		return
	}

	cityRequest := model.WeatherRequest{
		City:       weather_req.City,
		Fahrenheit: fahrenheit_select,
	}

	component := template.Weather(*weather_req, cityRequest)
	err = component.Render(req.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		log.Printf("Error rendering template %v", err)
		return
	}

}
