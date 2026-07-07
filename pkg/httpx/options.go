package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Option mutates a Client during construction. Use the WithXxx helpers.
type Option func(*Client)

// WithBaseURL sets the URL prefix prepended to relative request URLs.
func WithBaseURL(url string) Option { return func(c *Client) { c.baseURL = url } }

// WithTimeout sets the *http.Client.Timeout (overall request deadline,
// including all retries — the underlying *http.Client has no awareness of
// retry, so its clock covers the full attempt chain).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// WithRetry overrides the default RetryPolicy. Pass a zero-value
// RetryPolicy to disable retries.
func WithRetry(p RetryPolicy) Option { return func(c *Client) { c.retry = p } }

// WithTransport overrides the base http.RoundTripper (default:
// http.DefaultTransport). The provided transport will be wrapped by the
// retry / trace / log decorators.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.transport = rt } }

// WithLogger overrides the slog.Logger used for request logging
// (default: slog.Default()).
func WithLogger(l *slog.Logger) Option { return func(c *Client) { c.log = l } }
