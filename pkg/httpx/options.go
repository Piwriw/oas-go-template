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

// WithTimeout sets the overall deadline shared by every attempt in one client request.
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// WithRetry replaces the default retry behavior, with a zero policy disabling retries.
func WithRetry(p RetryPolicy) Option { return func(c *Client) { c.retry = p } }

// WithTransport sets the base round tripper wrapped by retry, tracing, and logging.
func WithTransport(rt http.RoundTripper) Option { return func(c *Client) { c.transport = rt } }

// WithLogger sets the structured logger used for outbound request events.
func WithLogger(l *slog.Logger) Option { return func(c *Client) { c.log = l } }
