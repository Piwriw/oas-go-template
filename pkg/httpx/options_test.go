package httpx

import (
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestOptions_ApplyAll(t *testing.T) {
	log := slog.Default()
	rt := &http.Transport{}

	c := &Client{}
	for _, opt := range []Option{
		WithBaseURL("https://api.example.com"),
		WithTimeout(5 * time.Second),
		WithRetry(RetryPolicy{MaxAttempts: 7}),
		WithTransport(rt),
		WithLogger(log),
	} {
		opt(c)
	}

	if c.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
	if c.retry.MaxAttempts != 7 {
		t.Errorf("retry.MaxAttempts = %d", c.retry.MaxAttempts)
	}
	if c.log != log {
		t.Errorf("log not set")
	}
	if c.transport != rt {
		t.Errorf("transport not set")
	}
	if c.timeout != 5*time.Second {
		t.Errorf("timeout not set")
	}
}
