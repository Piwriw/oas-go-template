// Package httpx is an opinionated HTTP client wrapper providing retry,
// OpenTelemetry trace propagation, and structured request logging.
package httpx

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"time"
)

// RetryPolicy controls retry behavior for failed HTTP requests.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts including the first
	// request. Default 3 means at most 2 retries.
	MaxAttempts int

	// Initial is the backoff duration after the first failed attempt.
	Initial time.Duration

	// Max caps the backoff between attempts.
	Max time.Duration

	// Multiplier is the exponential growth factor between consecutive
	// backoffs. Default 2.0 doubles the wait each time.
	Multiplier float64

	// Jitter is the relative randomness applied to each backoff, in [0,1].
	// 0.2 means ±20% of the computed value. Prevents thundering-herd
	// retries against the same upstream.
	Jitter float64
}

// DefaultRetry returns bounded exponential backoff suitable for transient upstream failures.
func DefaultRetry() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Initial:     100 * time.Millisecond,
		Max:         2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.2,
	}
}

// backoff computes the capped jittered delay after the specified failed request attempt.
func (p RetryPolicy) backoff(attempt int) time.Duration {
	if p.Initial <= 0 || p.Multiplier <= 0 {
		return 0
	}
	d := float64(p.Initial)
	for i := 0; i < attempt; i++ {
		d *= p.Multiplier
	}
	if p.Max > 0 && d > float64(p.Max) {
		d = float64(p.Max)
	}
	if p.Jitter > 0 {
		// ±Jitter fraction
		delta := d * p.Jitter
		d = d - delta + 2*delta*rand.Float64() // #nosec G404 -- jitter only spreads retry timing; it is not security-sensitive.
	}
	return time.Duration(d)
}

// shouldRetry permits transient failures only for configured idempotent requests with active contexts.
func (p RetryPolicy) shouldRetry(method string, status int, err error) bool {
	if p.MaxAttempts <= 0 {
		return false
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		// idempotent — ok
	default:
		return false
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	switch status {
	case 408, 429, 502, 503, 504:
		return true
	}
	return false
}
