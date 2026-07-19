package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"

	internalapi "github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/errcode"
	"github.com/piwriw/oas-go-template/internal/handler"
)

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			HTTPAddr:          ":0",
			GinMode:           "test",
			ReadHeaderTimeout: 2 * time.Second,
			ReadTimeout:       3 * time.Second,
			WriteTimeout:      4 * time.Second,
			IdleTimeout:       5 * time.Second,
			MaxHeaderBytes:    2048,
			MaxBodyBytes:      1024,
		},
	}
}

func TestServeAndWaitMarksReadinessDrainingBeforeShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	drainState := handler.NewDrainState(0)
	srv := &http.Server{Addr: "127.0.0.1:0"}

	if err := serveAndWait(ctx, srv, func() {}, drainState); err != nil {
		t.Fatalf("serveAndWait() error = %v", err)
	}
	if !drainState.Draining() {
		t.Fatal("serveAndWait() did not mark readiness as draining")
	}
}

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

func TestHealthEndpointPassesOASValidation(t *testing.T) {
	srv := newHTTPServer(testConfig(), nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReadinessReturns503WhileDraining(t *testing.T) {
	drainState := handler.NewDrainState(0)
	srv := newHTTPServer(testConfig(), nil, drainState)
	drainState.Begin()

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body internalapi.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode readiness error: %v", err)
	}
	if body.Code != int32(errcode.ServiceDraining) || body.Message != "service is shutting down" {
		t.Errorf("body=%+v", body)
	}
}

func TestCORSIsDisabledByDefault(t *testing.T) {
	srv := newHTTPServer(testConfig(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://app.example.com")
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("CORS unexpectedly enabled: Access-Control-Allow-Origin=%q", got)
	}
}

func TestCORSAllowedOriginAndPreflight(t *testing.T) {
	cfg := testConfig()
	cfg.CORS = config.CORSConfig{
		Enabled:          true,
		AllowOrigins:     []string{"https://app.example.com"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           time.Hour,
	}
	srv := newHTTPServer(cfg, nil)

	t.Run("normal request", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", "https://app.example.com")
		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Errorf("allow origin=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("allow credentials=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
			t.Errorf("expose headers=%q", got)
		}
	})

	t.Run("preflight", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Request-ID")
		srv.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Errorf("allow origin=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET,OPTIONS" {
			t.Errorf("allow methods=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type,X-Request-Id" {
			t.Errorf("allow headers=%q", got)
		}
		if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
			t.Errorf("max age=%q", got)
		}
	})
}

func TestCORSRejectsDisallowedOrigin(t *testing.T) {
	cfg := testConfig()
	cfg.CORS = config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"https://app.example.com"},
		AllowMethods: []string{"GET", "OPTIONS"},
	}
	srv := newHTTPServer(cfg, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://attacker.example")
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body internalapi.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode forbidden response: %v; body=%s", err, rec.Body.String())
	}
	if body.Code != int32(errcode.Forbidden) || body.Message != "forbidden" {
		t.Errorf("body=%+v", body)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin got CORS header=%q", got)
	}
}

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

func TestRequestBodyLimitUsesAPIError(t *testing.T) {
	cfg := testConfig()
	cfg.Server.MaxBodyBytes = 4
	srv := newHTTPServer(cfg, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/healthz", strings.NewReader("12345"))
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

func TestHTTPServerProtectionConfig(t *testing.T) {
	cfg := testConfig()
	srv := newHTTPServer(cfg, nil)

	if srv.ReadHeaderTimeout != cfg.Server.ReadHeaderTimeout || srv.ReadTimeout != cfg.Server.ReadTimeout {
		t.Errorf("read timeouts = %v/%v", srv.ReadHeaderTimeout, srv.ReadTimeout)
	}
	if srv.WriteTimeout != cfg.Server.WriteTimeout || srv.IdleTimeout != cfg.Server.IdleTimeout {
		t.Errorf("write/idle timeouts = %v/%v", srv.WriteTimeout, srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != cfg.Server.MaxHeaderBytes {
		t.Errorf("MaxHeaderBytes = %d", srv.MaxHeaderBytes)
	}
}
