// create test for GetWeatherByCoords function in model/weather.go
package model_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/andreanidouglas/weather-dashboard/router"
	"github.com/stretchr/testify/assert"
)

func TestGetWeatherByCoords(t *testing.T) {
	assert := assert.New(t)
	api_key := os.Getenv("API_KEY")

	assert.NotEmpty(api_key, "Should provide an API KEY for testing")

	// rome italy coordinates
	req := httptest.NewRequest(http.MethodGet, "/weather?lat=41.89&lon=12.49&appid="+api_key, nil)
	w := httptest.NewRecorder()

	api_context := &model.ApiContext{
		Key: api_key,
	}
	cache := model.NewCache()
	handler := router.NewHandler(false, api_context, &cache)

	handler.HandleWeatherByCoords(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(http.StatusOK, res.StatusCode, "Should return status OK")

}
