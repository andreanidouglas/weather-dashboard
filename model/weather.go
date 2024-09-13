package model

type Weather struct {
	City        string
	CurrentTemp float32
	MaxTemp     float32
	MinTemp     float32
	FeelsLike   float32
}

type WeatherRequest struct {
	City string `json:"city"`
}
