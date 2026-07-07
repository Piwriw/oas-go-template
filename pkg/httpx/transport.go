package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tracer is package-wide; otel.Tracer returns a no-op tracer when OTel
// isn't initialized (matches the pattern in internal/handler/version.go).
var tracer = otel.Tracer("github.com/piwriw/oas-go-template/pkg/httpx")

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

// traceTransport creates a span per RoundTrip attempt and injects W3C
// Trace Context headers so the downstream service can continue the trace.
type traceTransport struct {
	parent http.RoundTripper
}

func (t traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", req.Method),
		attribute.String("url.full", req.URL.String()),
		attribute.String("server.address", req.URL.Hostname()),
	}
	ctx, span := tracer.Start(ctx, "HTTP "+req.Method, trace.WithAttributes(attrs...))
	defer span.End()

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	req = req.WithContext(ctx)
	resp, err := t.parent.RoundTrip(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	if resp.StatusCode >= 500 {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return resp, nil
}
