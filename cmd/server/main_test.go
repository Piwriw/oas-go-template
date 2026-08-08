package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"

	internalapi "github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/errcode"
)

// testConfig returns isolated server defaults with telemetry disabled.
func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			HTTPAddr: ":0",
			GinMode:  "test",
		},
	}
}

// TestServeAndWaitShutsDownOnContextCancellation verifies cancellation gracefully stops the HTTP service.
func TestServeAndWaitShutsDownOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := &http.Server{Addr: "127.0.0.1:0"}

	if err := serveAndWait(ctx, srv); err != nil {
		t.Fatalf("serveAndWait() error = %v", err)
	}
}

// TestServeAndWaitListenErrorTakesPriorityOverCanceledContext preserves bind failures during cancellation.
func TestServeAndWaitListenErrorTakesPriorityOverCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	srv := &http.Server{Addr: "127.0.0.1:not-a-port"}
	if err := serveAndWait(ctx, srv); err == nil {
		t.Fatal("serveAndWait() error = nil, want listen error")
	}
}

// TestMetricsEndpointServesGoRuntimeMetrics verifies the operational metrics route exposes process collectors.
func TestMetricsEndpointServesGoRuntimeMetrics(t *testing.T) {
	cfg := testConfig()
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

// TestHealthEndpointPassesOASValidation verifies the liveness route conforms to its embedded contract.
func TestHealthEndpointPassesOASValidation(t *testing.T) {
	srv := newHTTPServer(testConfig(), nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// TestCORSAllowsAllOriginsAndPreflight verifies the built-in browser policy accepts every origin.
func TestCORSAllowsAllOriginsAndPreflight(t *testing.T) {
	srv := newHTTPServer(testConfig(), nil)

	t.Run("normal request", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", "https://app.example.com")
		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("allow origin=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("allow credentials=%q, want empty", got)
		}
		if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
			t.Errorf("expose headers=%q", got)
		}
	})

	t.Run("preflight", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
		req.Header.Set("Origin", "https://another.example")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Request-ID")
		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("allow origin=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET,POST,PUT,PATCH,DELETE,OPTIONS" {
			t.Errorf("allow methods=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Origin,Content-Type,Accept,Authorization,X-Request-Id" {
			t.Errorf("allow headers=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Max-Age"); got != "43200" {
			t.Errorf("max age=%q", got)
		}
	})
}

// TestOASValidatorRejectsMissingRequiredQuery verifies contract-required inputs are enforced before handlers run.
func TestOASValidatorRejectsMissingRequiredQuery(t *testing.T) {
	spec := openAPISpec()
	typeValue := openapi3.Types{openapi3.TypeString}
	spec.Paths.Find("/healthz").Get.Parameters = append(spec.Paths.Find("/healthz").Get.Parameters, &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:     "required",
			In:       "query",
			Required: true,
			Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &typeValue}},
		},
	})

	r := gin.New()
	r.Use(openAPIValidator(spec))
	called := false
	r.GET("/healthz", func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v body=%s", rec.Code, called, rec.Body.String())
	}
	var body internalapi.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != int32(errcode.InvalidRequest) || body.Message != "invalid request" {
		t.Errorf("body=%+v", body)
	}
}

// TestRoutingErrorsUseAPIError verifies unknown routes and methods use the shared public error schema.
func TestRoutingErrorsUseAPIError(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   errcode.Code
	}{
		{name: "not found", method: http.MethodGet, path: "/missing", wantStatus: http.StatusNotFound, wantCode: errcode.NotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/healthz", wantStatus: http.StatusMethodNotAllowed, wantCode: errcode.MethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := newHTTPServer(testConfig(), nil)
			rec := httptest.NewRecorder()
			srv.Handler.ServeHTTP(rec, httptest.NewRequest(test.method, test.path, nil))

			if rec.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var body internalapi.Error
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != int32(test.wantCode) || body.Message == "" {
				t.Errorf("body=%+v", body)
			}
		})
	}
}

// TestRequestBodyLimitUsesAPIError verifies oversized payloads return the shared public error schema.
func TestRequestBodyLimitUsesAPIError(t *testing.T) {
	cfg := testConfig()
	srv := newHTTPServer(cfg, nil)
	rec := httptest.NewRecorder()
	payload := strings.Repeat("x", int(maxRequestBodyBytes)+1)
	req := httptest.NewRequest(http.MethodPost, "/healthz", strings.NewReader(payload))
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body internalapi.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != int32(errcode.RequestBodyTooLarge) {
		t.Errorf("code=%d", body.Code)
	}
}

// TestHTTPServerProtections verifies fixed timeout and header protections reach the HTTP server.
func TestHTTPServerProtections(t *testing.T) {
	cfg := testConfig()
	srv := newHTTPServer(cfg, nil)

	if srv.ReadHeaderTimeout != serverReadHeaderTimeout || srv.ReadTimeout != serverReadTimeout {
		t.Errorf("read timeouts = %v/%v", srv.ReadHeaderTimeout, srv.ReadTimeout)
	}
	if srv.WriteTimeout != serverWriteTimeout || srv.IdleTimeout != serverIdleTimeout {
		t.Errorf("write/idle timeouts = %v/%v", srv.WriteTimeout, srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != maxRequestHeaderBytes {
		t.Errorf("MaxHeaderBytes = %d", srv.MaxHeaderBytes)
	}
}
