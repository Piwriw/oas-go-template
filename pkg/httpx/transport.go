package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// logTransport logs every RoundTrip attempt with method, URL, status, elapsed,
// and (when available) trace_id / span_id so logs can be joined to traces.
type logTransport struct {
	parent http.RoundTripper
	log    *slog.Logger
}

func (t logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.parent.RoundTrip(req)
	elapsed := time.Since(start)

	fields := []any{
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Duration("elapsed", elapsed),
	}
	if sctx := trace.SpanContextFromContext(req.Context()); sctx.HasTraceID() {
		fields = append(fields, slog.String("trace_id", sctx.TraceID().String()))
		if sctx.HasSpanID() {
			fields = append(fields, slog.String("span_id", sctx.SpanID().String()))
		}
	}

	if err != nil {
		t.log.With(fields...).ErrorContext(req.Context(), "http request failed", "err", err)
		return nil, err
	}

	fields = append(fields, slog.Int("status", resp.StatusCode))
	if resp.StatusCode >= 400 {
		t.log.With(fields...).WarnContext(req.Context(), "http request non-2xx")
	} else {
		t.log.With(fields...).InfoContext(req.Context(), "http request ok")
	}
	return resp, nil
}
