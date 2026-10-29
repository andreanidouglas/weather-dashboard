package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
)

var WeatherDataURL = "https://api.openweathermap.org/data/2.5/weather"

const maxWeatherResponseSize = 1 << 20

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type TemperatureUnit string

const (
	UnitCelsius    TemperatureUnit = "celsius"
	UnitFahrenheit TemperatureUnit = "fahrenheit"
)

type Temperature struct {
	Unit      TemperatureUnit `json:"unit"`
	Current   float64         `json:"current"`
	Max       float64         `json:"max"`
	Min       float64         `json:"min"`
	FeelsLike float64         `json:"feels_like"`
}

func (t Temperature) ConvertTo(unit TemperatureUnit) Temperature {
	if t.Unit == unit {
		return t
	}

	converted := t
	converted.Unit = unit
	if t.Unit == UnitCelsius && unit == UnitFahrenheit {
		converted.Current = t.Current*9.0/5.0 + 32
		converted.FeelsLike = t.FeelsLike*9.0/5.0 + 32
		converted.Max = t.Max*9.0/5.0 + 32
		converted.Min = t.Min*9.0/5.0 + 32
		return converted
	}

	if t.Unit != UnitFahrenheit || unit != UnitCelsius {
		return t
	}
	converted.Current = 5.0 / 9.0 * (t.Current - 32.0)
	converted.FeelsLike = 5.0 / 9.0 * (t.FeelsLike - 32.0)
	converted.Max = 5.0 / 9.0 * (t.Max - 32.0)
	converted.Min = 5.0 / 9.0 * (t.Min - 32.0)
	return converted
}

func floorCoordinates(lat, lon float64) Coordinates {
	return Coordinates{
		Lat: math.Floor(lat*100) / 100,
		Lon: math.Floor(lon*100) / 100,
	}
}

type Weather struct {
	Coordinates
	City       string      `json:"city"`
	Country    string      `json:"country"`
	Celsius    Temperature `json:"celsius"`
	Fahrenheit Temperature `json:"fahrenheit"`
	Condition  string      `json:"condition"`
	Humidity   float64     `json:"humidity"`
	Timezone   int         `json:"timezone"`
}

func (w Weather) TemperatureFor(fahrenheit bool) Temperature {
	if fahrenheit {
		return w.Fahrenheit
	}
	return w.Celsius
}

type WeatherRequest struct {
	City       string `json:"city"`
	Fahrenheit bool   `json:"fahrenheit"`
}

type ApiContext struct {
	Key string
}

type LatLon struct {
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Fahrenheit bool    `json:"fahrenheit"`
}

// GetWeather uses openweathermap api to fetch weather for a given city
func GetWeather(req WeatherRequest, apiContext *ApiContext) (*Weather, error) {
	log.Printf("Requested weather for: %q", req.City)
	units := "metric"
	unit := UnitCelsius
	if req.Fahrenheit {
		units = "imperial"
		unit = UnitFahrenheit
	}

	endpoint := fmt.Sprintf("%s?q=%s&units=%s&appid=%s", WeatherDataURL, url.QueryEscape(req.City), units, apiContext.Key)

	res, err := http.Get(endpoint)
	if err != nil {
		return nil, errors.New("weather request failed")
	}
	return parseWeatherResponse(res, unit)
}

func GetWeatherByLatLon(req LatLon, apiContext ApiContext) (*Weather, error) {
	log.Printf("Requested weather for: %v", req)
	units := "metric"
	unit := UnitCelsius
	if req.Fahrenheit {
		units = "imperial"
		unit = UnitFahrenheit
	}

	endpoint := fmt.Sprintf("%s?lat=%f&lon=%f&units=%s&appid=%s", WeatherDataURL, req.Lat, req.Lon, units, apiContext.Key)

	res, err := http.Get(endpoint)
	if err != nil {
		return nil, errors.New("weather request failed")
	}

	return parseWeatherResponse(res, unit)
}

func parseWeatherResponse(res *http.Response, unit TemperatureUnit) (*Weather, error) {
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", res.StatusCode)
	}

	var response OpenWeather
	decoder := json.NewDecoder(io.LimitReader(res.Body, maxWeatherResponseSize))
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("decode weather response: %w", err)
	}
	if len(response.Weather) == 0 {
		return nil, errors.New("weather response has no conditions")
	}

	temperature := Temperature{
		Unit:      unit,
		Current:   response.Main.Temp,
		Max:       response.Main.TempMax,
		Min:       response.Main.TempMin,
		FeelsLike: response.Main.FeelsLike,
	}
	weather := &Weather{
		Coordinates: floorCoordinates(response.Coord.Lat, response.Coord.Lon),
		City:        response.Name,
		Country:     response.Sys.Country,
		Condition:   response.Weather[0].Main,
		Humidity:    float64(response.Main.Humidity),
		Timezone:    response.Timezone,
	}
	if unit == UnitFahrenheit {
		weather.Fahrenheit = temperature
		weather.Celsius = temperature.ConvertTo(UnitCelsius)
		return weather, nil
	}
	weather.Celsius = temperature
	weather.Fahrenheit = temperature.ConvertTo(UnitFahrenheit)
	return weather, nil
}
