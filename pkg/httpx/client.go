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

// New returns a Client configured by opts. The transport chain
// (retry → trace → log → base) is wired here once; callers cannot
// bypass it accidentally.
func New(opts ...Option) *Client {
	c := &Client{
		base:      &http.Client{},
		retry:     DefaultRetry(),
		transport: http.DefaultTransport,
		log:       slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	base := c.transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.base.Transport = retryTransport{
		policy: c.retry,
		parent: traceTransport{
			parent: logTransport{
				parent: base,
				log:    c.log,
			},
		},
	}
	if c.timeout > 0 {
		c.base.Timeout = c.timeout
	}
	return c
}
