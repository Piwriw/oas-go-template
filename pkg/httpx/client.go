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

// Get issues a GET and decodes the JSON response into *T.
func Get[T any](ctx context.Context, c *Client, url string) (*T, error) {
	return Do[T](ctx, c, http.MethodGet, url, nil)
}

// Post issues a POST with a JSON body and decodes the JSON response into *T.
func Post[T any](ctx context.Context, c *Client, url string, body any) (*T, error) {
	return Do[T](ctx, c, http.MethodPost, url, body)
}

// Put issues a PUT with a JSON body and decodes the JSON response into *T.
func Put[T any](ctx context.Context, c *Client, url string, body any) (*T, error) {
	return Do[T](ctx, c, http.MethodPut, url, body)
}

// Patch issues a PATCH with a JSON body and decodes the JSON response into *T.
func Patch[T any](ctx context.Context, c *Client, url string, body any) (*T, error) {
	return Do[T](ctx, c, http.MethodPatch, url, body)
}

// Delete issues a DELETE and decodes the JSON response into *T.
func Delete[T any](ctx context.Context, c *Client, url string) (*T, error) {
	return Do[T](ctx, c, http.MethodDelete, url, nil)
}

// PostVoid is like Post but does not decode the response body.
func PostVoid(ctx context.Context, c *Client, url string, body any) (*http.Response, error) {
	return DoVoid(ctx, c, http.MethodPost, url, body)
}

// PutVoid is like Put but does not decode the response body.
func PutVoid(ctx context.Context, c *Client, url string, body any) (*http.Response, error) {
	return DoVoid(ctx, c, http.MethodPut, url, body)
}

// PatchVoid is like Patch but does not decode the response body.
func PatchVoid(ctx context.Context, c *Client, url string, body any) (*http.Response, error) {
	return DoVoid(ctx, c, http.MethodPatch, url, body)
}

// DeleteVoid is like Delete but does not decode the response body.
func DeleteVoid(ctx context.Context, c *Client, url string) (*http.Response, error) {
	return DoVoid(ctx, c, http.MethodDelete, url, nil)
}
