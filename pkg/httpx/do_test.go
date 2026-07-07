package httpx

import (
	"errors"
	"strings"
	"testing"
)

func TestHttpError_Error_Format(t *testing.T) {
	e := &httpError{
		method:     "GET",
		url:        "https://example.com/foo",
		statusCode: 503,
		body:       "service unavailable",
	}
	msg := e.Error()
	if !strings.Contains(msg, "GET") {
		t.Errorf("missing method: %q", msg)
	}
	if !strings.Contains(msg, "https://example.com/foo") {
		t.Errorf("missing url: %q", msg)
	}
	if !strings.Contains(msg, "503") {
		t.Errorf("missing status: %q", msg)
	}
	if !strings.Contains(msg, "service unavailable") {
		t.Errorf("missing body: %q", msg)
	}
}

func TestHttpError_Unwrap_ErrNon2xx(t *testing.T) {
	e := &httpError{statusCode: 500}
	if !errors.Is(e, ErrNon2xx) {
		t.Errorf("errors.Is(httpError, ErrNon2xx) = false, want true")
	}
}

func TestErrNon2xx_DoesNotMatchOtherErrors(t *testing.T) {
	if errors.Is(errors.New("other"), ErrNon2xx) {
		t.Errorf("plain error should not match ErrNon2xx")
	}
}
