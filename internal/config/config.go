package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all WritePilot settings loaded from config.json.
type Config struct {
	// Language to correct toward, e.g. "Norwegian", "English".
	// If nil, the LLM will auto-detect the language from the text.
	Language *string `json:"language"`

	// Mode controls LLM output format:
	//   "correct"  – return only the corrected text (default)
	//   "suggest"  – return corrected text plus an explanation of each change
	Mode string `json:"mode"`

	// Hotkey trigger string, e.g. "Ctrl+Shift+Space".
	// Supported modifiers: Ctrl, Shift, Alt, Win
	// Supported keys: Space, Return, F1-F12, A-Z, 0-9
	Hotkey string `json:"hotkey"`

	// Endpoint is the primary LLM provider. Its fields (provider, api_key,
	// api_base_url, model) sit at the top level of config.json.
	Endpoint

	// Fallbacks are tried in order when the primary provider is overloaded,
	// rate-limited, times out or rejects the request.
	Fallbacks []Endpoint `json:"fallbacks"`

	// AutoPaste controls whether WritePilot automatically pastes the corrected
	// text after correction. Default: false (manual paste with Ctrl+V).
	AutoPaste bool `json:"auto_paste"`

	// TimeoutSeconds is the total time budget for one correction, across all
	// retries and fallbacks. Default: 30.
	TimeoutSeconds int `json:"timeout_seconds"`

	// AttemptTimeoutSeconds limits a single HTTP request before WritePilot
	// gives up on it and retries or moves to the next provider. Default: 10
	// (capped at TimeoutSeconds). Raise it for slow local models.
	AttemptTimeoutSeconds int `json:"attempt_timeout_seconds"`
}

// Endpoint describes one LLM provider to call.
type Endpoint struct {
	// Provider selects the LLM backend:
	//   "groq"   – Groq API (default, fast, generous free tier)
	//   "openai" – OpenAI API
	//   "gemini" – Google Gemini via OpenAI-compatible endpoint
	//   "ollama" – Local Ollama instance
	Provider string `json:"provider"`

	// APIKey for the chosen provider. Leave empty for Ollama.
	APIKey string `json:"api_key"`

	// APIBaseURL overrides the provider's default base URL.
	// Useful for self-hosted or proxy setups.
	APIBaseURL string `json:"api_base_url"`

	// Model name to use. Defaults to a sensible choice per provider.
	Model string `json:"model"`
}

// Label identifies the endpoint in logs and notifications, e.g. "groq/qwen/qwen3.8-27b".
func (e Endpoint) Label() string {
	return e.Provider + "/" + e.Model
}

// Endpoints returns the primary endpoint followed by the fallbacks, in the
// order they should be tried.
func (c *Config) Endpoints() []Endpoint {
	return append([]Endpoint{c.Endpoint}, c.Fallbacks...)
}

// Load reads config.json from the directory of the running executable.
// Falls back to the current working directory (useful during development).
func Load() (*Config, error) {
	paths := configSearchPaths()

	var (
		f   *os.File
		err error
	)
	for _, p := range paths {
		f, err = os.Open(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf(
			"config.json not found; searched: %v — copy config.example.json to config.json next to the executable and fill in your settings",
			paths,
		)
	}
	defer f.Close()

	var cfg Config
	if decErr := json.NewDecoder(f).Decode(&cfg); decErr != nil {
		return nil, fmt.Errorf("parse config.json: %w", decErr)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func configSearchPaths() []string {
	var paths []string

	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config.json"))
	}

	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "config.json"))
	}

	return paths
}

func (c *Config) applyDefaults() {
	if c.Mode == "" {
		c.Mode = "correct"
	}
	if c.Hotkey == "" {
		c.Hotkey = "Ctrl+Shift+Space"
	}
	c.Endpoint.applyDefaults()
	for i := range c.Fallbacks {
		c.Fallbacks[i].applyDefaults()
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 30
	}
	if c.AttemptTimeoutSeconds <= 0 {
		c.AttemptTimeoutSeconds = 10
	}
	if c.AttemptTimeoutSeconds > c.TimeoutSeconds {
		c.AttemptTimeoutSeconds = c.TimeoutSeconds
	}
}

func (c *Endpoint) applyDefaults() {
	if c.Provider == "" {
		c.Provider = "groq"
	}
	if c.APIBaseURL == "" {
		switch c.Provider {
		case "openai":
			c.APIBaseURL = "https://api.openai.com/v1"
		case "gemini":
			c.APIBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		case "ollama":
			c.APIBaseURL = "http://localhost:11434/v1"
		default:
			c.APIBaseURL = "https://api.groq.com/openai/v1"
		}
	}
	if c.Model == "" {
		switch c.Provider {
		case "openai":
			c.Model = "gpt-4o-mini"
		case "gemini":
			c.Model = "gemini-3.1-flash-lite"
		case "ollama":
			c.Model = "llama3.1"
		default:
			c.Model = "qwen/qwen3.8-27b"
		}
	}
}
