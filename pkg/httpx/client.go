package httpx

import (
	"context"
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

// New builds an HTTP client with configured retry, tracing, logging, and base transport behavior.
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

// Get issues a GET and decodes the JSON response into *T.
func (c *Client) Get[T any](ctx context.Context, url string) (*T, error) {
	return c.Do[T](ctx, http.MethodGet, url, nil)
}

// Post issues a POST with a JSON body and decodes the JSON response into *T.
func (c *Client) Post[T any](ctx context.Context, url string, body any) (*T, error) {
	return c.Do[T](ctx, http.MethodPost, url, body)
}

// Put issues a PUT with a JSON body and decodes the JSON response into *T.
func (c *Client) Put[T any](ctx context.Context, url string, body any) (*T, error) {
	return c.Do[T](ctx, http.MethodPut, url, body)
}

// Patch issues a PATCH with a JSON body and decodes the JSON response into *T.
func (c *Client) Patch[T any](ctx context.Context, url string, body any) (*T, error) {
	return c.Do[T](ctx, http.MethodPatch, url, body)
}

// Delete issues a DELETE and decodes the JSON response into *T.
func (c *Client) Delete[T any](ctx context.Context, url string) (*T, error) {
	return c.Do[T](ctx, http.MethodDelete, url, nil)
}

// PostVoid is like Post but does not decode the response body.
func (c *Client) PostVoid(ctx context.Context, url string, body any) (*http.Response, error) {
	return c.DoVoid(ctx, http.MethodPost, url, body)
}

// PutVoid is like Put but does not decode the response body.
func (c *Client) PutVoid(ctx context.Context, url string, body any) (*http.Response, error) {
	return c.DoVoid(ctx, http.MethodPut, url, body)
}

// PatchVoid is like Patch but does not decode the response body.
func (c *Client) PatchVoid(ctx context.Context, url string, body any) (*http.Response, error) {
	return c.DoVoid(ctx, http.MethodPatch, url, body)
}

// DeleteVoid is like Delete but does not decode the response body.
func (c *Client) DeleteVoid(ctx context.Context, url string) (*http.Response, error) {
	return c.DoVoid(ctx, http.MethodDelete, url, nil)
}
