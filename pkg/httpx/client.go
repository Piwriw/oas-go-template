package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Client is the high-level HTTP client. Construct with New.
type Client struct {
	base      *http.Client
	baseURL   string
	timeout   time.Duration
	retry     RetryPolicy
	transport http.RoundTripper
	log       *slog.Logger
}
