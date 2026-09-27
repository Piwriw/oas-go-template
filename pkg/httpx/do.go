package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	// contentTypeHeader carries the media type of an outbound request body.
	contentTypeHeader = "Content-Type"
	// acceptHeader advertises the response media type the client can decode.
	acceptHeader = "Accept"
	// jsonMediaType is the only media type this client encodes or decodes.
	jsonMediaType = "application/json"
	// errorBodyLimit bounds how many response bytes an httpError retains.
	errorBodyLimit = 1024
)

// ErrNon2xx is the sentinel unwrap target for any non-2xx HTTP response.
// Use errors.Is(err, httpx.ErrNon2xx) to distinguish HTTP error responses
// from transport / context errors.
var ErrNon2xx = errors.New("httpx: non-2xx response")

// httpError represents a non-2xx HTTP response. Fields are unexported
// to discourage callers from type-asserting on this type — read the
// error message instead, or branch via errors.Is(err, ErrNon2xx).
type httpError struct {
	method     string
	url        string
	statusCode int
	body       string
}

// Error formats a non-successful upstream response with request and body context.
func (e *httpError) Error() string {
	return fmt.Sprintf("httpx: %s %s -> %d %s: %s",
		e.method, e.url, e.statusCode, http.StatusText(e.statusCode), e.body)
}

// Unwrap exposes ErrNon2xx for stable upstream response classification.
func (e *httpError) Unwrap() error { return ErrNon2xx }

// do is the template method behind every request helper. It owns the invariant
// skeleton of an outbound call — resolve the URL, marshal and attach the JSON
// body, send through the retry / trace / log transport chain, and classify
// non-2xx responses — while the single variant step, consuming a successful
// response, is supplied by the caller as handle.
//
// The response is returned with its body already consumed: on a non-2xx status
// it holds the truncated error snippet, on success handle has drained it, so
// either way the connection returns to the pool.
func (c *Client) do(ctx context.Context, method, url string, body any, handle func(*http.Response) error) (*http.Response, error) {
	fullURL := joinURL(c.baseURL, url)

	var bodyBytes []byte
	if body != nil {
		var err error
		if bodyBytes, err = json.Marshal(body); err != nil {
			return nil, fmt.Errorf("httpx: marshal request body: %w", err)
		}
	}

	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("httpx: build request: %w", err)
	}
	if bodyBytes != nil {
		req.Header.Set(contentTypeHeader, jsonMediaType)
		req.Header.Set(acceptHeader, jsonMediaType)
		// GetBody lets retryTransport rewind the body for each retry attempt.
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	resp, err := c.base.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpx: send request: %w", err)
	}
	// DoVoid replaces resp.Body after draining it; close the original body.
	defer func(body io.ReadCloser) { _ = body.Close() }(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		return resp, &httpError{
			method:     method,
			url:        fullURL,
			statusCode: resp.StatusCode,
			body:       string(snippet),
		}
	}

	if err := handle(resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Do sends a policy-aware HTTP request and decodes a successful JSON response into *T.
func (c *Client) Do[T any](ctx context.Context, method, url string, body any) (*T, error) {
	var out T
	//nolint:bodyclose // c.do closes the original response body before returning.
	_, err := c.do(ctx, method, url, body, func(resp *http.Response) error {
		if resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0 {
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return fmt.Errorf("httpx: decode response: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DoVoid sends a policy-aware HTTP request and drains its successful response
// without decoding a body, keeping the response usable for header inspection.
func (c *Client) DoVoid(ctx context.Context, method, url string, body any) (*http.Response, error) {
	return c.do(ctx, method, url, body, func(resp *http.Response) error {
		// Drain body so the connection returns to the pool, then swap in an
		// empty closed body to keep resp readable for status and headers.
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return nil
	})
}
