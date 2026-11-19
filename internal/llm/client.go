package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"writepilot/internal/config"
)

// timeUnit is the length of one configured "second". Tests shrink it.
var timeUnit = time.Second

// maxAttemptsPerEndpoint is how many times one provider is tried before
// moving on to the next fallback.
const maxAttemptsPerEndpoint = 2

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
}

// Failure records one failed attempt against one provider.
type Failure struct {
	Endpoint config.Endpoint
	// Kind is a short, fixed label such as "overloaded (503)" or "timed out".
	Kind string
	// Detail holds the provider's error message or the transport error, for the log.
	Detail string

	retryable  bool
	retryAfter time.Duration
}

// Result is a successful correction plus the failures that preceded it.
type Result struct {
	Text     string
	Used     config.Endpoint
	Failures []Failure
}

// FellBack reports whether the result came from a fallback provider.
func (r Result) FellBack() bool {
	for _, f := range r.Failures {
		if f.Endpoint != r.Used {
			return true
		}
	}
	return false
}

// ChainError is returned when every provider failed.
type ChainError struct {
	Failures []Failure
}

func (e *ChainError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, f := range e.Failures {
		parts[i] = fmt.Sprintf("%s: %s", f.Endpoint.Label(), f.Kind)
		if f.Detail != "" {
			parts[i] += " — " + f.Detail
		}
	}
	return "all providers failed: " + strings.Join(parts, "; ")
}

// Summary is a short, notification-friendly description: the last failure
// per provider, e.g. "gemini: overloaded (503), groq: timed out".
func (e *ChainError) Summary() string {
	var order []string
	last := map[string]string{}
	for _, f := range e.Failures {
		if _, seen := last[f.Endpoint.Provider]; !seen {
			order = append(order, f.Endpoint.Provider)
		}
		last[f.Endpoint.Provider] = f.Kind
	}
	parts := make([]string, len(order))
	for i, p := range order {
		parts[i] = p + ": " + last[p]
	}
	return strings.Join(parts, ", ")
}

// Correct sends text to the configured providers — the primary first, then
// each fallback — and returns the first successful correction.
//
// Each provider gets up to maxAttemptsPerEndpoint attempts on retryable
// errors (overload, rate limit, server error, timeout), except that a
// timeout moves straight to the next provider when one exists. The whole
// call never takes longer than cfg.TimeoutSeconds.
func Correct(cfg *config.Config, text string) (Result, error) {
	body, err := json.Marshal(chatRequest{
		Messages: []message{
			{Role: "system", Content: systemPrompt(cfg)},
			{Role: "user", Content: text},
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}

	deadline := time.Now().Add(time.Duration(cfg.TimeoutSeconds) * timeUnit)
	attemptTimeout := time.Duration(cfg.AttemptTimeoutSeconds) * timeUnit
	endpoints := cfg.Endpoints()
	var failures []Failure

	for i, ep := range endpoints {
		hasNext := i < len(endpoints)-1
		for attempt := 1; attempt <= maxAttemptsPerEndpoint; attempt++ {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return Result{}, budgetExceeded(cfg, failures)
			}

			start := time.Now()
			out, f := callOnce(ep, body, min(attemptTimeout, remaining))
			elapsed := time.Since(start).Round(100 * time.Millisecond)

			if f == nil {
				log.Printf("[WritePilot] %s answered in %v (attempt %d)", ep.Label(), elapsed, attempt)
				return Result{Text: out, Used: ep, Failures: failures}, nil
			}

			failures = append(failures, *f)
			log.Printf("[WritePilot] %s attempt %d failed after %v: %s%s",
				ep.Label(), attempt, elapsed, f.Kind, detailSuffix(f.Detail))

			if !f.retryable || attempt == maxAttemptsPerEndpoint {
				break
			}
			if f.Kind == "timed out" && hasNext {
				break // a hung provider is unlikely to answer faster on a retry
			}
			sleepFor(backoff(f, attempt), deadline)
		}
	}

	return Result{}, &ChainError{Failures: failures}
}

func budgetExceeded(cfg *config.Config, failures []Failure) error {
	log.Printf("[WritePilot] total time budget of %ds used up", cfg.TimeoutSeconds)
	if len(failures) == 0 {
		return fmt.Errorf("no time left to call any provider (timeout_seconds=%d)", cfg.TimeoutSeconds)
	}
	return &ChainError{Failures: failures}
}

// callOnce makes a single request to one endpoint.
func callOnce(ep config.Endpoint, body []byte, timeout time.Duration) (string, *Failure) {
	fail := func(kind, detail string, retryable bool) *Failure {
		return &Failure{Endpoint: ep, Kind: kind, Detail: detail, retryable: retryable}
	}

	// Each endpoint has its own model; patch it into the shared request body.
	var req chatRequest
	_ = json.Unmarshal(body, &req)
	req.Model = ep.Model
	payload, err := json.Marshal(req)
	if err != nil {
		return "", fail("bad request", err.Error(), false)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.APIBaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fail("bad request", err.Error(), false)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if ep.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+ep.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		var ne net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
			return "", fail("timed out", "", true)
		}
		// Connection refused, DNS failure, etc. won't fix themselves in a second.
		return "", fail("unreachable", err.Error(), false)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		var ne net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
			return "", fail("timed out", "while reading the response", true)
		}
		return "", fail("read error", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		f := fail(statusKind(resp.StatusCode), providerMessage(raw), resp.StatusCode == 429 || resp.StatusCode >= 500)
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs > 0 {
			f.retryAfter = time.Duration(secs) * timeUnit
		}
		return "", f
	}

	var chat chatResponse
	if err := json.Unmarshal(raw, &chat); err != nil {
		return "", fail("invalid response", providerMessage(raw), false)
	}
	if len(chat.Choices) == 0 || strings.TrimSpace(chat.Choices[0].Message.Content) == "" {
		return "", fail("empty response", providerMessage(raw), false)
	}
	return chat.Choices[0].Message.Content, nil
}

func statusKind(code int) string {
	switch {
	case code == 429:
		return "rate limited (429)"
	case code == 503:
		return "overloaded (503)"
	case code >= 500:
		return fmt.Sprintf("server error (%d)", code)
	default:
		return fmt.Sprintf("HTTP %d", code)
	}
}

// providerMessage extracts a readable error message from a response body.
// It understands {"error":{"message":…}} (OpenAI style) and
// [{"error":{"message":…}}] (Gemini style), and otherwise returns the body
// itself with whitespace collapsed, cut to 300 characters.
func providerMessage(raw []byte) string {
	type apiErr struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	var obj apiErr
	if json.Unmarshal(raw, &obj) == nil && obj.Error != nil && obj.Error.Message != "" {
		return obj.Error.Message
	}
	var arr []apiErr
	if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 && arr[0].Error != nil && arr[0].Error.Message != "" {
		return arr[0].Error.Message
	}

	s := strings.Join(strings.Fields(string(raw)), " ")
	if r := []rune(s); len(r) > 300 {
		s = string(r[:300]) + "…"
	}
	return s
}

func detailSuffix(d string) string {
	if d == "" {
		return ""
	}
	return " — " + d
}

// backoff picks the pause before retrying the same provider: the provider's
// Retry-After when it is short, otherwise a jittered 0.5–1s per attempt.
func backoff(f *Failure, attempt int) time.Duration {
	if f.retryAfter > 0 && f.retryAfter <= 5*timeUnit {
		return f.retryAfter
	}
	base := time.Duration(attempt) * timeUnit / 2
	return base + rand.N(base+1)
}

// sleepFor waits for d, but never past the deadline.
func sleepFor(d time.Duration, deadline time.Time) {
	if left := time.Until(deadline); d > left {
		d = left
	}
	if d > 0 {
		time.Sleep(d)
	}
}

// systemPrompt builds the LLM system prompt based on the user's config.
func systemPrompt(cfg *config.Config) string {
	var langSpec string
	if cfg.Language != nil {
		langSpec = *cfg.Language
	}

	if cfg.Mode == "suggest" {
		if langSpec != "" {
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
• ...`, langSpec, langSpec)
		}
		// Auto-detect language mode for "suggest"
		return `You are a language assistant helping a non-native speaker improve their writing.

The user will provide a piece of text they wrote. Your tasks:
1. Detect the language of the text automatically.
2. Correct ALL grammar, spelling, punctuation, and style mistakes in that language.
3. Preserve the original meaning and tone exactly — do not paraphrase unnecessarily.
4. After the corrected text, add a section titled "--- Corrections ---" that lists every change you made. For each change, show the original phrase, the corrected phrase, and a brief explanation of why it was wrong. This helps the user learn from their mistakes.
5. Write the corrections section in the same language as the original text.

Format:
<corrected text>

--- Corrections ---
• Original: "..." → Corrected: "..." — <explanation>
• ...`
	}

	// Default: "correct" mode — return only the fixed text.
	if langSpec != "" {
		return fmt.Sprintf(`You are a language assistant helping a non-native speaker improve their %s writing.

The user will provide a piece of text they wrote. Your tasks:
1. Correct ALL grammar, spelling, and punctuation mistakes.
2. Preserve the original meaning and tone exactly — do not paraphrase unnecessarily.
3. Return ONLY the corrected text, nothing else. No explanations, no comments, no labels.
4. If the text contains no mistakes, return it unchanged.`, langSpec)
	}
	// Auto-detect language mode for "correct"
	return `You are a language assistant helping a non-native speaker improve their writing.

The user will provide a piece of text they wrote. Your tasks:
1. Detect the language of the text automatically.
2. Correct ALL grammar, spelling, and punctuation mistakes in that language.
3. Preserve the original meaning and tone exactly — do not paraphrase unnecessarily.
4. Return ONLY the corrected text, nothing else. No explanations, no comments, no labels.
5. If the text contains no mistakes, return it unchanged.`
}
