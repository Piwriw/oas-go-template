package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/piwriw/oas-go-template/internal/config"
)

func TestMetricsEndpointServesGoRuntimeMetrics(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{HTTPAddr: ":0", GinMode: "test"},
	}
	srv := newHTTPServer(cfg, nil)
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	// Go runtime collector emits go_goroutines regardless of OTel state.
	if !strings.Contains(string(body), "go_goroutines") {
		t.Errorf("body missing go_goroutines; got %d bytes:\n%s", len(body), body)
	}
}
