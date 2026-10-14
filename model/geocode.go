package model

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// Location represents a city result from OpenWeatherMap geocoding direct API
// https://openweathermap.org/api/geocoding-api
// Example response element: {"name":"London","lat":51.5085,"lon":-0.1257,"country":"GB","state":"England"}
type Location struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
	State   string  `json:"state"`
}

var GeocodeURL = "https://api.openweathermap.org/geo/1.0/direct"

// Label returns a human readable, unique-enough name, eg "Rome, Lazio, IT".
func (l Location) Label() string {
	parts := []string{l.Name}
	if l.State != "" {
		parts = append(parts, l.State)
	}
	if l.Country != "" {
		parts = append(parts, l.Country)
	}
	return strings.Join(parts, ", ")
}

// GetLocations queries the OpenWeatherMap geocoding API for city name suggestions.
// It returns up to limit matching locations. Query shorter than 2 chars returns empty slice.
func GetLocations(query string, limit int, apiContext *ApiContext) ([]Location, error) {
	trimmed := strings.TrimSpace(query)
	if len(trimmed) < 2 {
		return []Location{}, nil
	}
	if limit <= 0 || limit > 50 { // API limit safeguard
		limit = 5
	}

	endpoint := fmt.Sprintf("%s?q=%s&limit=%d&appid=%s", GeocodeURL, url.QueryEscape(trimmed), limit, apiContext.Key)
	log.Printf("Geocode suggestions for: %s", trimmed)

	res, err := http.Get(endpoint)
	if err != nil {
		log.Printf("HTTP GET error: %s => %v", endpoint, err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("HTTP GET geocode status code %d for %s", res.StatusCode, endpoint)
		return nil, fmt.Errorf("geocode HTTP status: %d", res.StatusCode)
	}

	var locations []Location
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&locations); err != nil {
		return nil, err
	}

	// Deduplicate by label (Name+State+Country), so eg Rome, Georgia and
	// Rome, New York are both kept.
	seen := make(map[string]struct{})
	unique := make([]Location, 0, len(locations))
	for _, l := range locations {
		key := strings.ToLower(l.Label())
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, l)
	}

	return unique, nil
}
