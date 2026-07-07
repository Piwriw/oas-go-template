// Package httpx is an opinionated HTTP client wrapper providing retry,
// OpenTelemetry trace propagation, and structured request logging.
package httpx

import "time"

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

// DefaultRetry returns the package's default retry policy:
// up to 3 attempts, 100ms initial backoff, 2s cap, 2× growth, ±20% jitter.
func DefaultRetry() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Initial:     100 * time.Millisecond,
		Max:         2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.2,
	}
}
