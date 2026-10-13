package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/gin-gonic/gin"
)

// TestMiddlewareLogsRequestParameters verifies parameter capture preserves bodies, repeated query values, and read errors.
func TestMiddlewareLogsRequestParameters(t *testing.T) {
	readErr := errors.New("request read failed")
	tests := []struct {
		name string
		body string
		err  error
	}{
		{name: "JSON", body: `{"name":"Ada"}`},
		{name: "form", body: "name=Ada&tag=one&tag=two"},
		{name: "malformed JSON", body: `{"name":`},
		{name: "empty body"},
		{name: "body at log limit", body: strings.Repeat("a", maxLoggedBodyBytes)},
		{name: "body exceeds log limit", body: strings.Repeat("a", maxLoggedBodyBytes+100)},
		{name: "body read failure", body: `{"name":`, err: readErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			setDefaultLogger(t, slog.New(slog.NewJSONHandler(&output, nil)))
			r := newTestRouter()
			r.POST("/users/:id", func(c *gin.Context) {
				body, err := io.ReadAll(c.Request.Body)
				if string(body) != tt.body || !errors.Is(err, tt.err) {
					t.Errorf("handler read body=%q err=%v, want body=%q err=%v", body, err, tt.body, tt.err)
				}
				c.Status(http.StatusNoContent)
			})
			var reader io.Reader = strings.NewReader(tt.body)
			if tt.err != nil {
				reader = io.MultiReader(reader, iotest.ErrReader(tt.err))
			}
			req := httptest.NewRequest(http.MethodPost, "/users/42?tag=one&tag=two&name=Ada+Lovelace", reader)
			r.ServeHTTP(httptest.NewRecorder(), req)

			var record struct {
				Query     map[string][]string `json:"query"`
				Params    map[string]string   `json:"path_params"`
				Body      string              `json:"request_body"`
				Truncated bool                `json:"request_body_truncated"`
			}
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatalf("decode access log %q: %v", output.String(), err)
			}
			wantQuery := map[string][]string{"tag": {"one", "two"}, "name": {"Ada Lovelace"}}
			if !reflect.DeepEqual(record.Query, wantQuery) {
				t.Errorf("query=%v, want %v", record.Query, wantQuery)
			}
			if record.Params["id"] != "42" {
				t.Errorf("path_params=%v, want id=42", record.Params)
			}
			wantBody := tt.body[:min(len(tt.body), maxLoggedBodyBytes)]
			if record.Body != wantBody || record.Truncated != (len(tt.body) > maxLoggedBodyBytes) {
				t.Errorf("logged body length=%d truncated=%v, want length=%d truncated=%v",
					len(record.Body), record.Truncated, len(wantBody), len(tt.body) > maxLoggedBodyBytes)
			}
		})
	}
}

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
