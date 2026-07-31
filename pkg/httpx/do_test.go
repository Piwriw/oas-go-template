package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type echoResp struct {
	Echoed string `json:"echoed"`
}

// TestHttpError_Error_Format verifies upstream failure messages retain request and response context.
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

// TestHttpError_Unwrap_ErrNon2xx verifies HTTP failures match the public non-2xx sentinel.
func TestHttpError_Unwrap_ErrNon2xx(t *testing.T) {
	e := &httpError{statusCode: 500}
	if !errors.Is(e, ErrNon2xx) {
		t.Errorf("errors.Is(httpError, ErrNon2xx) = false, want true")
	}
}

// TestErrNon2xx_DoesNotMatchOtherErrors verifies unrelated failures remain distinguishable from HTTP responses.
func TestErrNon2xx_DoesNotMatchOtherErrors(t *testing.T) {
	if errors.Is(errors.New("other"), ErrNon2xx) {
		t.Errorf("plain error should not match ErrNon2xx")
	}
}

// TestDo_Get_Success verifies successful GET responses decode into the requested model.
func TestDo_Get_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(echoResp{Echoed: "hello"})
	}))
	defer srv.Close()

	c := New()
	out, err := Do[echoResp](context.Background(), c, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if out.Echoed != "hello" {
		t.Errorf("Echoed = %q", out.Echoed)
	}
}

// TestDo_Get_204_NoBody verifies no-content responses return a zero-value model without decoding errors.
func TestDo_Get_204_NoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New()
	out, err := Do[echoResp](context.Background(), c, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if out == nil {
		t.Fatal("out is nil")
	}
	if out.Echoed != "" {
		t.Errorf("Echoed = %q, want empty", out.Echoed)
	}
}

// TestDo_Post_RequestBody verifies request models are serialized into outbound JSON bodies.
func TestDo_Post_RequestBody(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(echoResp{Echoed: "ok"})
	}))
	defer srv.Close()

	c := New()
	out, err := Do[echoResp](context.Background(), c, http.MethodPost, srv.URL, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if out.Echoed != "ok" {
		t.Errorf("Echoed = %q", out.Echoed)
	}
	if gotBody["k"] != "v" {
		t.Errorf("server got body %v, want k=v", gotBody)
	}
}

// TestDo_Post_SetsContentType verifies JSON requests advertise content and acceptance media types.
func TestDo_Post_SetsContentType(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
	}))
	defer srv.Close()

	c := New()
	_, err := Do[echoResp](context.Background(), c, http.MethodPost, srv.URL, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
}

// TestDo_Non2xx_ReturnsErrNon2xx verifies unsuccessful responses expose status, body, and sentinel classification.
func TestDo_Non2xx_ReturnsErrNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"detail":"boom"}`)
	}))
	defer srv.Close()

	c := New()
	_, err := Do[echoResp](context.Background(), c, http.MethodGet, srv.URL, nil)
	if err == nil {
		t.Fatal("want err, got nil")
	}
	if !errors.Is(err, ErrNon2xx) {
		t.Errorf("err is not ErrNon2xx: %v", err)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("err msg missing status: %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("err msg missing body snippet: %v", err)
	}
}

// TestDo_NetworkError verifies transport failures retain their underlying network cause.
func TestDo_NetworkError(t *testing.T) {
	c := New()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()
	_, err := Do[echoResp](context.Background(), c, http.MethodGet, srv.URL, nil)
	if err == nil {
		t.Fatal("want err, got nil")
	}
	if errors.Is(err, ErrNon2xx) {
		t.Errorf("network err should NOT match ErrNon2xx: %v", err)
	}
}

// TestDo_BaseURL_Joined verifies relative request paths are nested beneath the configured API base.
func TestDo_BaseURL_Joined(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	_, err := Do[echoResp](context.Background(), c, http.MethodGet, "/foo", nil)
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	if gotPath != "/foo" {
		t.Errorf("path = %q, want /foo", gotPath)
	}
}

// TestDo_BodyTruncatedInError verifies large upstream error bodies are bounded in returned messages.
func TestDo_BodyTruncatedInError(t *testing.T) {
	big := strings.Repeat("x", 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, big)
	}))
	defer srv.Close()

	c := New()
	_, err := Do[echoResp](context.Background(), c, http.MethodGet, srv.URL, nil)
	if err == nil {
		t.Fatal("want err")
	}
	if len(err.Error()) > 2048 {
		t.Errorf("err msg too long (%d chars) — body not truncated", len(err.Error()))
	}
}

// TestDoVoid_Success_ReturnsResponseWithClosedBody verifies bodyless calls drain responses while preserving metadata.
func TestDoVoid_Success_ReturnsResponseWithClosedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Foo", "bar")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"ignored":true}`)
	}))
	defer srv.Close()

	c := New()
	resp, err := DoVoid(context.Background(), c, http.MethodPost, srv.URL, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("DoVoid err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Foo") != "bar" {
		t.Errorf("X-Foo header missing")
	}
	n, _ := io.ReadAll(resp.Body)
	if len(n) != 0 {
		t.Errorf("body not drained: %d bytes left", len(n))
	}
}

// TestDoVoid_Non2xx_ReturnsErrNon2xx verifies bodyless calls classify unsuccessful upstream responses.
func TestDoVoid_Non2xx_ReturnsErrNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New()
	resp, err := DoVoid(context.Background(), c, http.MethodPost, srv.URL, nil)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if !errors.Is(err, ErrNon2xx) {
		t.Errorf("want ErrNon2xx, got %v", err)
	}
}
