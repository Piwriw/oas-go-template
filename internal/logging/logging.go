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
package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// LogConfig drives slog setup. Loaded by internal/config.
type LogConfig struct {
	Format string `mapstructure:"format"`
	Level  string `mapstructure:"level"`
}

// New returns the base *slog.Logger writing to stderr. Pass the LogConfig from
// the central config so defaults have already been applied.
func New(cfg LogConfig) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}
	var inner slog.Handler
	if strings.EqualFold(strings.TrimSpace(cfg.Format), "json") {
		inner = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		inner = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(&otelHandler{inner: inner})
}

// Middleware generates (or accepts) a request_id per request, stores it in the gin
// context, mirrors it back via X-Request-ID response header, and writes one structured
// log line per request. The base logger is the project-wide slog.Default().
//
// Note: slog.Default() is captured once per middleware build so request hot path
// does one With() rather than two Default() lookups (Default() walks an atomic
// each call).
func Middleware() gin.HandlerFunc {
	base := slog.Default()
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

		logger.InfoContext(c.Request.Context(), "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int("bytes", c.Writer.Size()),
			slog.Duration("latency", time.Since(start)),
		)
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

func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *otelHandler) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record)
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{inner: h.inner.WithGroup(name)}
}

const (
	loggerKey    = "slog.logger"
	requestIDKey = "request_id"
)

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

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// rand.Read on Linux/macOS crypto/rand never errors in practice; fall back to
		// a timestamp-derived value so we still produce something unique.
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(b)
}
