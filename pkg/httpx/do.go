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

func (e *httpError) Error() string {
	return fmt.Sprintf("httpx: %s %s -> %d %s: %s",
		e.method, e.url, e.statusCode, http.StatusText(e.statusCode), e.body)
}

func (e *httpError) Unwrap() error { return ErrNon2xx }

// Do sends one HTTP request (with retries per the client's RetryPolicy)
// and decodes the JSON response into *T. A non-2xx response is returned
// as an error wrapping ErrNon2xx.
func Do[T any](ctx context.Context, c *Client, method, url string, body any) (*T, error) {
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
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		// GetBody lets retryTransport rewind the body for each retry attempt.
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	resp, err := c.base.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpx: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &httpError{
			method:     method,
			url:        fullURL,
			statusCode: resp.StatusCode,
			body:       string(snippet),
		}
	}

	var out T
	if resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0 {
		return &out, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("httpx: decode response: %w", err)
	}
	return &out, nil
}
