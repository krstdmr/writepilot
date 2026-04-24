package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"writepilot/internal/config"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Correct sends text to the configured LLM provider and returns the
// corrected (and optionally annotated) text.
func Correct(cfg *config.Config, text string) (string, error) {
	reqBody := chatRequest{
		Model: cfg.Model,
		Messages: []message{
			{Role: "system", Content: systemPrompt(cfg)},
			{Role: "user", Content: text},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.TimeoutSeconds)*time.Second,
	)
	defer cancel()

	url := cfg.APIBaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request to %s: %w", cfg.Provider, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to surface the provider's error message.
		var errResp chatResponse
		if json.Unmarshal(raw, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("provider error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return "", fmt.Errorf("parse response JSON: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// systemPrompt builds the LLM system prompt based on the user's config.
func systemPrompt(cfg *config.Config) string {
	lang := cfg.Language

	if cfg.Mode == "suggest" {
		return fmt.Sprintf(`You are a language assistant helping a non-native speaker improve their %s writing.

The user will provide a piece of text they wrote. Your tasks:
1. Correct ALL grammar, spelling, punctuation, and style mistakes.
2. Preserve the original meaning and tone exactly — do not paraphrase unnecessarily.
3. After the corrected text, add a section titled "--- Corrections ---" that lists every change you made. For each change, show the original phrase, the corrected phrase, and a brief explanation of why it was wrong. This helps the user learn from their mistakes.
4. Write the corrections section in %s as well.

Format:
<corrected text>

--- Corrections ---
• Original: "..." → Corrected: "..." — <explanation>
• ...`, lang, lang)
	}

	// Default: "correct" mode — return only the fixed text.
	return fmt.Sprintf(`You are a language assistant helping a non-native speaker improve their %s writing.

The user will provide a piece of text they wrote. Your tasks:
1. Correct ALL grammar, spelling, and punctuation mistakes.
2. Preserve the original meaning and tone exactly — do not paraphrase unnecessarily.
3. Return ONLY the corrected text, nothing else. No explanations, no comments, no labels.
4. If the text contains no mistakes, return it unchanged.`, lang)
}
