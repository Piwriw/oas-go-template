package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMiddlewareAccessLogLevels verifies response classes map to the expected access-log severity.
func TestMiddlewareAccessLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantLevel string
	}{
		{name: "success", status: http.StatusNoContent, wantLevel: "INFO"},
		{name: "redirect", status: http.StatusTemporaryRedirect, wantLevel: "INFO"},
		{name: "client error", status: http.StatusBadRequest, wantLevel: "WARN"},
		{name: "server error", status: http.StatusInternalServerError, wantLevel: "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			setDefaultLogger(t, slog.New(slog.NewJSONHandler(&output, nil)))

			r := newTestRouter()
			r.GET("/test", func(c *gin.Context) {
				c.Status(tt.status)
			})

			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))

			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatalf("decode access log %q: %v", output.String(), err)
			}
			if got := record["level"]; got != tt.wantLevel {
				t.Errorf("level=%v, want %s", got, tt.wantLevel)
			}
			if got := int(record["status"].(float64)); got != tt.status {
				t.Errorf("status=%d, want %d", got, tt.status)
			}
		})
	}
}

// TestMiddlewareLogsOperationalAccessOnceAndOnError verifies probe success deduplication and failure logging.
func TestMiddlewareLogsOperationalAccessOnceAndOnError(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		t.Run(path, func(t *testing.T) {
			var output bytes.Buffer
			setDefaultLogger(t, slog.New(slog.NewJSONHandler(&output, nil)))

			status := http.StatusOK
			r := newTestRouter()
			r.GET(path, func(c *gin.Context) {
				c.Status(status)
			})

			request := func() *httptest.ResponseRecorder {
				recorder := httptest.NewRecorder()
				r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
				return recorder
			}

			if recorder := request(); recorder.Header().Get("X-Request-ID") == "" {
				t.Error("X-Request-ID header is missing")
			}
			if output.Len() == 0 {
				t.Fatal("first request was not logged")
			}

			output.Reset()
			request()
			if output.Len() != 0 {
				t.Errorf("repeated successful request was logged: %s", output.String())
			}

			status = http.StatusServiceUnavailable
			request()
			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatalf("decode error access log %q: %v", output.String(), err)
			}
			if got := int(record["status"].(float64)); got != status {
				t.Errorf("status=%d, want %d", got, status)
			}
		})
	}
}

// newTestRouter builds a minimal Gin router with request logging enabled.
func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())
	return r
}

// setDefaultLogger installs a test logger and restores the process logger during cleanup.
func setDefaultLogger(t *testing.T, logger *slog.Logger) {
	t.Helper()
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})
}
