package llm

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"writepilot/internal/config"
)

// Shrink "seconds" so timeout tests run in milliseconds.
func init() {
	timeUnit = 20 * time.Millisecond
}

type fakeProvider struct {
	srv  *httptest.Server
	hits atomic.Int32
}

// newFake serves the given handler and counts requests.
func newFake(t *testing.T, h func(w http.ResponseWriter, r *http.Request, hit int32)) *fakeProvider {
	t.Helper()
	f := &fakeProvider{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(w, r, f.hits.Add(1))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func ok(text string) func(http.ResponseWriter, *http.Request, int32) {
	return func(w http.ResponseWriter, _ *http.Request, _ int32) {
		fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":%q}}]}`, text)
	}
}

func status(code int, body string) func(http.ResponseWriter, *http.Request, int32) {
	return func(w http.ResponseWriter, _ *http.Request, _ int32) {
		w.WriteHeader(code)
		fmt.Fprint(w, body)
	}
}

func hang(d time.Duration) func(http.ResponseWriter, *http.Request, int32) {
	return func(w http.ResponseWriter, r *http.Request, _ int32) {
		select {
		case <-time.After(d):
		case <-r.Context().Done():
		}
	}
}

func ep(name string, f *fakeProvider) config.Endpoint {
	return config.Endpoint{Provider: name, APIBaseURL: f.srv.URL, Model: name + "-model"}
}

func newCfg(primary config.Endpoint, fallbacks ...config.Endpoint) *config.Config {
	return &config.Config{
		Endpoint:              primary,
		Fallbacks:             fallbacks,
		TimeoutSeconds:        30,
		AttemptTimeoutSeconds: 5,
	}
}

func TestCorrect_PrimarySucceeds(t *testing.T) {
	p := newFake(t, ok("fixed"))

	res, err := Correct(newCfg(ep("gemini", p)), "fxied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Text != "fixed" || res.Used.Provider != "gemini" || res.FellBack() {
		t.Fatalf("got %+v", res)
	}
}

func TestCorrect_RetriesOverloadOnceThenFallsBack(t *testing.T) {
	p := newFake(t, status(503, `[{"error":{"code":503,"message":"The model is overloaded.","status":"UNAVAILABLE"}}]`))
	f := newFake(t, ok("fixed by groq"))

	res, err := Correct(newCfg(ep("gemini", p), ep("groq", f)), "txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := p.hits.Load(); got != 2 {
		t.Errorf("primary hits = %d, want 2 (one retry)", got)
	}
	if res.Text != "fixed by groq" || res.Used.Provider != "groq" || !res.FellBack() {
		t.Fatalf("got %+v", res)
	}
	if len(res.Failures) != 2 || res.Failures[0].Kind != "overloaded (503)" {
		t.Fatalf("failures = %+v", res.Failures)
	}
	if !strings.Contains(res.Failures[0].Detail, "The model is overloaded.") {
		t.Errorf("provider message not surfaced: %q", res.Failures[0].Detail)
	}
}

func TestCorrect_RecoversOnRetry(t *testing.T) {
	p := newFake(t, func(w http.ResponseWriter, r *http.Request, hit int32) {
		if hit == 1 {
			status(503, `{"error":{"message":"busy"}}`)(w, r, hit)
			return
		}
		ok("fixed")(w, r, hit)
	})

	res, err := Correct(newCfg(ep("gemini", p)), "txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Text != "fixed" || res.FellBack() {
		t.Fatalf("got %+v", res)
	}
}

func TestCorrect_TimeoutSkipsRetryWhenFallbackExists(t *testing.T) {
	p := newFake(t, hang(300*time.Millisecond))
	f := newFake(t, ok("fixed"))
	cfg := newCfg(ep("gemini", p), ep("groq", f))
	cfg.AttemptTimeoutSeconds = 2 // 40ms in tests

	res, err := Correct(cfg, "txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := p.hits.Load(); got != 1 {
		t.Errorf("primary hits = %d, want 1 (no retry after timeout)", got)
	}
	if res.Used.Provider != "groq" || res.Failures[0].Kind != "timed out" {
		t.Fatalf("got %+v", res)
	}
}

func TestCorrect_NonRetryableGoesStraightToFallback(t *testing.T) {
	p := newFake(t, status(401, `{"error":{"message":"API key not valid"}}`))
	f := newFake(t, ok("fixed"))

	res, err := Correct(newCfg(ep("gemini", p), ep("groq", f)), "txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := p.hits.Load(); got != 1 {
		t.Errorf("primary hits = %d, want 1", got)
	}
	if res.Failures[0].Kind != "HTTP 401" || !strings.Contains(res.Failures[0].Detail, "API key not valid") {
		t.Fatalf("failures = %+v", res.Failures)
	}
}

func TestCorrect_AllFailReturnsChainError(t *testing.T) {
	p := newFake(t, status(503, `oops`))
	f := newFake(t, status(429, `{"error":{"message":"slow down"}}`))

	_, err := Correct(newCfg(ep("gemini", p), ep("groq", f)), "txt")
	var ce *ChainError
	if !errors.As(err, &ce) {
		t.Fatalf("want *ChainError, got %T: %v", err, err)
	}
	sum := ce.Summary()
	if !strings.Contains(sum, "gemini: overloaded (503)") || !strings.Contains(sum, "groq: rate limited (429)") {
		t.Errorf("summary = %q", sum)
	}
}

func TestCorrect_RespectsTotalBudget(t *testing.T) {
	p := newFake(t, hang(600*time.Millisecond))
	cfg := newCfg(ep("gemini", p))
	cfg.TimeoutSeconds = 5        // 100ms in tests
	cfg.AttemptTimeoutSeconds = 4 // 80ms in tests

	start := time.Now()
	_, err := Correct(cfg, "txt")
	if err == nil {
		t.Fatal("expected an error")
	}
	if el := time.Since(start); el > 400*time.Millisecond {
		t.Errorf("took %v, want about the 100ms budget", el)
	}
}

func TestProviderMessage(t *testing.T) {
	cases := map[string]string{
		`{"error":{"message":"obj msg"}}`:              "obj msg",
		`[{"error":{"code":503,"message":"arr msg"}}]`: "arr msg",
		"plain\n  text   body":                         "plain text body",
		strings.Repeat("x", 500):                       strings.Repeat("x", 300) + "…",
	}
	for in, want := range cases {
		if got := providerMessage([]byte(in)); got != want {
			t.Errorf("providerMessage(%.20q) = %q, want %q", in, got, want)
		}
	}
}
