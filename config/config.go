// Package config loads the server configuration from environment variables.
package config

import "errors"

// Addr is the address the server listens on.
const Addr = "0.0.0.0:8080"

type Config struct {
	// APIKey is the OpenWeatherMap API key (API_KEY).
	APIKey string
	// Standalone also serves the static frontend from ./view/src (STANDALONE=true).
	Standalone bool
}

// Load reads the configuration using getenv, eg os.Getenv.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		APIKey:     getenv("API_KEY"),
		Standalone: getenv("STANDALONE") == "true",
	}
	if cfg.APIKey == "" {
		return Config{}, errors.New("need API_KEY as environment variable")
	}
	return cfg, nil
}
