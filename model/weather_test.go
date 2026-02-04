package model_test

import (
	"os"
	"testing"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/stretchr/testify/assert"
)

func TestWeatherRequestLatLon(t *testing.T) {

	key := os.Getenv("API_KEY")

	// API_KEY is needed for accessing openweathermap API
	assert.Greater(t, len(key), 0, "Test expect ENVIRONMENT variable for API_KEY")
	ctx := model.ApiContext{
		Key: key,
	}

	req := model.WeatherRequestLatLon{
		Lat:        44.34,
		Lon:        10.99,
		Fahrenheit: false,
	}

	weather, err := model.GetByLatLon(req, &ctx)

	// For a valid lat and lon, no error should be returned
	assert.NoError(t, err, "API should not return error for valid Lat and Lon")
	// For a valid lat and lon, Weather should be valid
	assert.NotEmpty(t, weather, "Object for Weather should not be empty")

}
