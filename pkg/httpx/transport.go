package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
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

// retryTransport wraps an inner transport and retries failed attempts
// per the configured RetryPolicy. It is the outermost transport in the
// chain so each retry gets its own trace span and log entry.
type retryTransport struct {
	parent http.RoundTripper
	policy RetryPolicy
}

func (t retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Zero-value policy → no retry, just one attempt through the parent.
	if t.policy.MaxAttempts <= 0 {
		return t.parent.RoundTrip(req)
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt < t.policy.MaxAttempts; attempt++ {
		// For retries we must send a fresh body. GetBody is set by Do[T]/DoVoid.
		if attempt > 0 {
			if req.GetBody != nil {
				newReq, err := req.GetBody()
				if err != nil {
					lastErr = err
					break
				}
				req.Body = newReq
			}
		}

		resp, err := t.parent.RoundTrip(req)
		lastResp, lastErr = resp, err

		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		if !t.policy.shouldRetry(req.Method, status, err) {
			return resp, err
		}

		// Drain body so the connection can be reused.
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		wait := t.policy.backoff(attempt)
		// Honor Retry-After on 429 / 503 if present (overrides backoff).
		if resp != nil {
			if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
				wait = ra
			}
		}

		select {
		case <-req.Context().Done():
			lastErr = req.Context().Err()
			return nil, lastErr
		case <-time.After(wait):
		}
	}
	return lastResp, lastErr
}

// parseRetryAfter parses the Retry-After header, which can be either
// delta-seconds or an HTTP-date. Returns 0 if unparseable.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return time.Until(t)
	}
	return 0
}
