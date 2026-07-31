package httpx

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestDefaultRetry verifies the standard transient-failure retry budget and backoff values.
func TestDefaultRetry(t *testing.T) {
	p := DefaultRetry()
	if p.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", p.MaxAttempts)
	}
	if p.Initial != 100*time.Millisecond {
		t.Errorf("Initial = %v, want 100ms", p.Initial)
	}
	if p.Max != 2*time.Second {
		t.Errorf("Max = %v, want 2s", p.Max)
	}
	if p.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", p.Multiplier)
	}
	if p.Jitter != 0.2 {
		t.Errorf("Jitter = %v, want 0.2", p.Jitter)
	}
}

// TestRetryPolicy_Backoff_NoJitter verifies deterministic exponential delays respect their cap.
func TestRetryPolicy_Backoff_NoJitter(t *testing.T) {
	p := DefaultRetry()
	p.Jitter = 0

	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
		{5, 2000 * time.Millisecond}, // capped
		{6, 2000 * time.Millisecond}, // still capped
	}
	for _, c := range cases {
		got := p.backoff(c.attempt)
		if got != c.want {
			t.Errorf("backoff(%d) = %v, want %v", c.attempt, got, c.want)
		}
	}
}

// TestRetryPolicy_Backoff_JitterInRange verifies randomized delays remain within the configured band.
func TestRetryPolicy_Backoff_JitterInRange(t *testing.T) {
	p := DefaultRetry() // Jitter 0.2
	// attempt 2 → base 400ms → jitter ±80ms → range [320ms, 480ms]
	for i := 0; i < 100; i++ {
		got := p.backoff(2)
		if got < 320*time.Millisecond || got > 480*time.Millisecond {
			t.Errorf("backoff(2) = %v, want within [320ms, 480ms]", got)
		}
	}
}

// TestRetryPolicy_Backoff_ZeroPolicy verifies a disabled retry policy adds no delay.
func TestRetryPolicy_Backoff_ZeroPolicy(t *testing.T) {
	// Zero-value policy must not panic and must return 0.
	var p RetryPolicy
	if got := p.backoff(0); got != 0 {
		t.Errorf("zero-value backoff = %v, want 0", got)
	}
}

// TestShouldRetry_ByMethod verifies retries are limited to configured idempotent HTTP methods.
func TestShouldRetry_ByMethod(t *testing.T) {
	p := DefaultRetry()
	cases := []struct {
		method string
		want   bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"PUT", true},
		{"DELETE", true},
		{"POST", false},  // non-idempotent
		{"PATCH", false}, // non-idempotent
		{"BREW", false},  // unknown → safe default: no retry
	}
	for _, c := range cases {
		got := p.shouldRetry(c.method, 503, nil)
		if got != c.want {
			t.Errorf("shouldRetry(method=%s) = %v, want %v", c.method, got, c.want)
		}
	}
}

// TestShouldRetry_ByStatus verifies only designated transient response statuses are retried.
func TestShouldRetry_ByStatus(t *testing.T) {
	p := DefaultRetry()
	// Per spec §4: only 408, 429, 502, 503, 504 retry. NOT 500.
	retryStatuses := []int{408, 429, 502, 503, 504}
	noRetryStatuses := []int{200, 301, 400, 401, 403, 404, 422, 500}
	for _, s := range retryStatuses {
		if !p.shouldRetry("GET", s, nil) {
			t.Errorf("status %d: want retry", s)
		}
	}
	for _, s := range noRetryStatuses {
		if p.shouldRetry("GET", s, nil) {
			t.Errorf("status %d: want no retry", s)
		}
	}
}

// TestShouldRetry_ByError verifies network failures retry while context termination does not.
func TestShouldRetry_ByError(t *testing.T) {
	p := DefaultRetry()

	// Simulated network error (not context-related) → retry.
	netErr := errors.New("connection reset")
	if !p.shouldRetry("GET", 0, netErr) {
		t.Errorf("network error: want retry")
	}

	// context.Canceled → do NOT retry.
	if p.shouldRetry("GET", 0, context.Canceled) {
		t.Errorf("context.Canceled: want no retry")
	}

	// context.DeadlineExceeded → do NOT retry.
	if p.shouldRetry("GET", 0, context.DeadlineExceeded) {
		t.Errorf("context.DeadlineExceeded: want no retry")
	}
}

// TestShouldRetry_ZeroPolicy verifies transient failures do not retry when attempts are disabled.
func TestShouldRetry_ZeroPolicy(t *testing.T) {
	var p RetryPolicy
	if p.shouldRetry("GET", 503, nil) {
		t.Errorf("zero-value policy: want no retry")
	}
}
