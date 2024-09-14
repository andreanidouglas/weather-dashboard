package model

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
