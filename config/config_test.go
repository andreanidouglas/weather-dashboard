package config

import "testing"

func TestLoad(t *testing.T) {
	env := map[string]string{"API_KEY": "k", "STANDALONE": "true"}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil || cfg.APIKey != "k" || !cfg.Standalone {
		t.Fatalf("got %+v, %v", cfg, err)
	}

	if _, err := Load(func(string) string { return "" }); err == nil {
		t.Fatal("expected error without API_KEY")
	}
}
