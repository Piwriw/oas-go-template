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

func TestDo_Get_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestDo_Get_204_NoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestDo_Post_SetsContentType(t *testing.T) {
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
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

func TestDo_Non2xx_ReturnsErrNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestDo_NetworkError(t *testing.T) {
	c := New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	_, err := Do[echoResp](context.Background(), c, http.MethodGet, srv.URL, nil)
	if err == nil {
		t.Fatal("want err, got nil")
	}
	if errors.Is(err, ErrNon2xx) {
		t.Errorf("network err should NOT match ErrNon2xx: %v", err)
	}
}

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

func TestDo_BodyTruncatedInError(t *testing.T) {
	big := strings.Repeat("x", 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
