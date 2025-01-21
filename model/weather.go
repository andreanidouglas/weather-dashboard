package model

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type Weather struct {
	City        string
	Country     string
	CurrentTemp float64
	MaxTemp     float64
	MinTemp     float64
	FeelsLike   float64
	Condition   string
	Humidity    float64
}

type WeatherRequest struct {
	City string `json:"city"`
}

type ApiContext struct {
	Key string
}

func GetWeather(req WeatherRequest, apiContext ApiContext) (*Weather, error) {

	//https://api.openweathermap.org/data/2.5/weather?q=Sao%20Paulo&units=metric&appid=<apiid>

	log.Printf("Requested weather for: %s", req.City)
	var weather = &OpenWeather{}
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&units=metric&appid=%s", url.QueryEscape(req.City), apiContext.Key)

	res, err := http.Get(url)
	if err != nil {
		log.Printf("HTTP GET error: %s\n%v", url, err)
		return nil, err
	}

	if res.StatusCode != 200 {
		log.Printf("HTTP GET error %s, status code: %d", url, res.StatusCode)
		return nil, fmt.Errorf("HTTP GET status code is: %s", res.StatusCode)
	}

	decoder := json.NewDecoder(res.Body)
	decoder.Decode(weather)

	resWeather := Weather{
		City:        weather.Name,
		Country:     "",
		CurrentTemp: weather.Main.Temp,
		MaxTemp:     weather.Main.TempMax,
		MinTemp:     weather.Main.TempMin,
		FeelsLike:   weather.Main.FeelsLike,
		Condition:   weather.Weather[0].Main,
		Humidity:    float64(weather.Main.Humidity),
	}

	return &resWeather, nil
}
