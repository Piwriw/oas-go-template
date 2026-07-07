package httpx

import (
	"errors"
	"fmt"
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
