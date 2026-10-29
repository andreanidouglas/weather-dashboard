package main

import (
	"testing"

	"github.com/andreanidouglas/weather-dashboard/model"
)

func TestRunStrategyHitAndEntryCounts(t *testing.T) {
	sharedCoordinates := model.Coordinates{Lat: 10.25, Lon: 20.5}
	tests := []struct {
		name      string
		strategy  model.CacheStrategy
		workload  workload
		wantHits  int64
		wantMiss  int64
		wantCache int
	}{
		{
			name:     "coordinate strategy hits within a floored bucket",
			strategy: model.CacheByCoordinates,
			workload: workload{
				kind: coordinateRequest,
				locations: []location{{
					name:        "Town",
					coordinates: sharedCoordinates,
				}},
				requests: []benchmarkRequest{
					{coordinates: model.Coordinates{Lat: 10.251, Lon: 20.501}},
					{coordinates: model.Coordinates{Lat: 10.259, Lon: 20.509}},
				},
			},
			wantHits:  1,
			wantMiss:  1,
			wantCache: 1,
		},
		{
			name:     "city strategy misses coordinate lookups",
			strategy: model.CacheByCity,
			workload: workload{
				kind: coordinateRequest,
				locations: []location{{
					name:        "Town",
					coordinates: sharedCoordinates,
				}},
				requests: []benchmarkRequest{
					{coordinates: model.Coordinates{Lat: 10.251, Lon: 20.501}},
					{coordinates: model.Coordinates{Lat: 10.259, Lon: 20.509}},
				},
			},
			wantHits:  0,
			wantMiss:  2,
			wantCache: 1,
		},
		{
			name:      "coordinate strategy deduplicates city aliases",
			strategy:  model.CacheByCoordinates,
			workload:  aliasWorkload(sharedCoordinates),
			wantHits:  1,
			wantMiss:  2,
			wantCache: 1,
		},
		{
			name:      "city strategy keeps separate alias entries",
			strategy:  model.CacheByCity,
			workload:  aliasWorkload(sharedCoordinates),
			wantHits:  1,
			wantMiss:  2,
			wantCache: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runStrategy(tt.name, tt.strategy, tt.workload)
			if err != nil {
				t.Fatalf("run benchmark strategy: %v", err)
			}
			if got.hits != tt.wantHits || got.misses != tt.wantMiss || got.entries != tt.wantCache {
				t.Errorf("result = hits:%d misses:%d entries:%d, want hits:%d misses:%d entries:%d",
					got.hits,
					got.misses,
					got.entries,
					tt.wantHits,
					tt.wantMiss,
					tt.wantCache,
				)
			}
		})
	}
}

func aliasWorkload(coordinates model.Coordinates) workload {
	return workload{
		kind: cityRequest,
		locations: []location{
			{name: "Town North", coordinates: coordinates},
			{name: "North Town", coordinates: coordinates},
		},
		requests: []benchmarkRequest{
			{city: "Town North"},
			{city: "North Town"},
			{city: "Town North"},
		},
	}
}
