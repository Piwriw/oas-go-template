package httpx

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), &buf
}

func TestLogTransport_2xx_Info(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("want INFO level, got: %s", out)
	}
	if !strings.Contains(out, "http request ok") {
		t.Errorf("want 'http request ok' message, got: %s", out)
	}
	if !strings.Contains(out, "status=200") {
		t.Errorf("want status=200, got: %s", out)
	}
}

func TestLogTransport_4xx_Warn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("want WARN level, got: %s", out)
	}
}

func TestLogTransport_5xx_Warn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("want WARN level for 5xx, got: %s", out)
	}
	if strings.Contains(out, "level=ERROR") {
		t.Errorf("5xx must NOT be ERROR (per spec §5.2), got: %s", out)
	}
}

func TestLogTransport_NetworkError_Error(t *testing.T) {
	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	// Port 1 dials → connection refused → transport error.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("want error, got nil")
	}

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("want ERROR for network failure, got: %s", out)
	}
	if !strings.Contains(out, "http request failed") {
		t.Errorf("want 'http request failed', got: %s", out)
	}
}

func TestLogTransport_ElapsedAndMethodRecorded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "method=POST") {
		t.Errorf("want method=POST, got: %s", out)
	}
	if !strings.Contains(out, "elapsed=") {
		t.Errorf("want elapsed=, got: %s", out)
	}
}
