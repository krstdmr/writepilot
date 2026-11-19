package config

import (
	"encoding/json"
	"testing"
)

func TestApplyDefaults_FillsPrimaryAndFallbacks(t *testing.T) {
	raw := `{
		"provider": "gemini",
		"model": "gemini-x",
		"fallbacks": [
			{"provider": "groq", "api_key": "k"},
			{"provider": "ollama", "model": "qwen3"}
		]
	}`
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.applyDefaults()

	eps := cfg.Endpoints()
	if len(eps) != 3 {
		t.Fatalf("got %d endpoints, want 3", len(eps))
	}
	want := []Endpoint{
		{Provider: "gemini", Model: "gemini-x", APIBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai"},
		{Provider: "groq", APIKey: "k", Model: "qwen/qwen3.8-27b", APIBaseURL: "https://api.groq.com/openai/v1"},
		{Provider: "ollama", Model: "qwen3", APIBaseURL: "http://localhost:11434/v1"},
	}
	for i := range want {
		if eps[i] != want[i] {
			t.Errorf("endpoint %d = %+v, want %+v", i, eps[i], want[i])
		}
	}
	if cfg.TimeoutSeconds != 30 || cfg.AttemptTimeoutSeconds != 10 {
		t.Errorf("timeouts = %d/%d, want 30/10", cfg.TimeoutSeconds, cfg.AttemptTimeoutSeconds)
	}
}

func TestApplyDefaults_AttemptTimeoutNeverExceedsTotal(t *testing.T) {
	cfg := Config{TimeoutSeconds: 8}
	cfg.applyDefaults()
	if cfg.AttemptTimeoutSeconds != 8 {
		t.Errorf("attempt timeout = %d, want capped at 8", cfg.AttemptTimeoutSeconds)
	}
}
