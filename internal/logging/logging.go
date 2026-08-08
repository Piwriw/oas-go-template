// Package logging configures slog with OpenTelemetry trace context and per-request
// metadata injection.
//
// Each log record carries trace_id / span_id when the calling context has an active
// OTel span (e.g. inside an otelgin-instrumented HTTP request), and request_id when
// emitted through the gin middleware.
//
// LogConfig is loaded from config.yaml by internal/config:
//
//	format = text | json   (default: text)
//	level  = debug | info | warn | error   (default: info)
//	caller = true | false   (default: true)
package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

const (
	healthPath   = "/healthz"
	readyPath    = "/readyz"
	metricsPath  = "/metrics"
	loggerKey    = "slog.logger"
	requestIDKey = "request_id"
)

// LogConfig drives slog setup. Loaded by internal/config.
type LogConfig struct {
	Format string `mapstructure:"format"`
	Level  string `mapstructure:"level"`
	Caller bool   `mapstructure:"caller"`
}

// New builds the process logger from the validated output format and severity configuration.
func New(cfg LogConfig) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     parseLevel(cfg.Level),
		AddSource: cfg.Caller,
	}
	var inner slog.Handler
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "json") {
		inner = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		inner = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(&otelHandler{inner: inner})
}

// Middleware attaches request and trace identifiers and emits rate-limited structured access logs.
func Middleware() gin.HandlerFunc {
	base := slog.Default()
	var loggedOperationalPaths sync.Map
	return func(c *gin.Context) {
		start := time.Now()

		reqID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if reqID == "" {
			reqID = randomID()
		}
		c.Header("X-Request-ID", reqID)
		c.Set(requestIDKey, reqID)

		// Attach request_id to the logger so subsequent InfoContext calls in handlers
		// pick it up automatically.
		logger := base.With(slog.String("request_id", reqID))
		c.Set(loggerKey, logger)

		c.Next()

		status := c.Writer.Status()
		if skipAccessLog(c.Request.URL.Path, status, &loggedOperationalPaths) {
			return
		}

		logger.LogAttrs(c.Request.Context(), accessLogLevel(status), "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", status),
			slog.Int("bytes", c.Writer.Size()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}

// skipAccessLog suppresses repeated successful logs for high-frequency operational endpoints.
func skipAccessLog(path string, status int, logged *sync.Map) bool {
	if status >= http.StatusBadRequest {
		return false
	}
	switch path {
	case healthPath, readyPath, metricsPath:
		_, alreadyLogged := logged.LoadOrStore(path, struct{}{})
		return alreadyLogged
	default:
		return false
	}
}

// accessLogLevel maps an HTTP response status to its operational log severity.
func accessLogLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// From returns the per-request logger from the gin context, or slog.Default() as fallback.
func From(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(loggerKey); ok {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

// RequestID returns the per-request ID stored by Middleware, or "" if not set.
func RequestID(c *gin.Context) string {
	if v, ok := c.Get(requestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// otelHandler wraps an inner slog.Handler so that any record emitted via the
// *Context family (InfoContext, ErrorContext, ...) carries the active span's
// trace_id and span_id when ctx has a valid OTel SpanContext.
type otelHandler struct {
	inner slog.Handler
}

// Enabled delegates severity filtering to the wrapped structured log handler.
func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle enriches log records with active OpenTelemetry trace identifiers before writing them.
func (h *otelHandler) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record)
}

// WithAttrs preserves trace enrichment while adding persistent structured fields.
func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup preserves trace enrichment while nesting subsequent structured fields.
func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{inner: h.inner.WithGroup(name)}
}

// parseLevel converts configured log-level aliases to their slog severity.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// randomID creates a request correlation identifier without external state.
func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// rand.Read on Linux/macOS crypto/rand never errors in practice; fall back to
		// a timestamp-derived value so we still produce something unique.
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(b)
}
