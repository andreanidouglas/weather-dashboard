package model_test

import (
	"github.com/andreanidouglas/weather-dashboard/model"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGeocode(t *testing.T) {

	key := os.Getenv("API_KEY")
	assert.Greater(t, len(key), 0, "Requires API KEY to be valid")

	ctx := model.ApiContext{
		Key: key,
	}
	locations, err := model.GetLocations("London", 3, &ctx)
	assert.NoError(t, err, "For a valid location, no error should be returned")

	assert.Greater(t, len(locations), 0, "For a valid location, at least one result must be returned")

}
