package httpx

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), &buf
}

func TestLogTransport_2xx_Info(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("want INFO level, got: %s", out)
	}
	if !strings.Contains(out, "http request ok") {
		t.Errorf("want 'http request ok' message, got: %s", out)
	}
	if !strings.Contains(out, "status=200") {
		t.Errorf("want status=200, got: %s", out)
	}
}

func TestLogTransport_4xx_Warn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("want WARN level, got: %s", out)
	}
}

func TestLogTransport_5xx_Warn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("want WARN level for 5xx, got: %s", out)
	}
	if strings.Contains(out, "level=ERROR") {
		t.Errorf("5xx must NOT be ERROR (per spec §5.2), got: %s", out)
	}
}

func TestLogTransport_NetworkError_Error(t *testing.T) {
	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	// Port 1 dials → connection refused → transport error.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("want error, got nil")
	}

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Errorf("want ERROR for network failure, got: %s", out)
	}
	if !strings.Contains(out, "http request failed") {
		t.Errorf("want 'http request failed', got: %s", out)
	}
}

func TestLogTransport_ElapsedAndMethodRecorded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log, buf := newTestLogger()
	rt := logTransport{parent: http.DefaultTransport, log: log}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "method=POST") {
		t.Errorf("want method=POST, got: %s", out)
	}
	if !strings.Contains(out, "elapsed=") {
		t.Errorf("want elapsed=, got: %s", out)
	}
}

func newTracerProviderWithExporter(exp *tracetest.InMemoryExporter) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
}

// withTestPropagator installs the W3C TraceContext propagator for the
// duration of the test, restoring whatever was previously registered.
// Production code expects internal/otel.Init to do this once at startup.
func withTestPropagator(t *testing.T) {
	t.Helper()
	orig := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(orig) })
}

func TestTraceTransport_CreatesSpanAndInjectsHeaders(t *testing.T) {
	withTestPropagator(t)
	exp := tracetest.NewInMemoryExporter()
	tp := newTracerProviderWithExporter(exp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	var gotHeaders http.Header
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	origTracer := tracer
	tracer = tp.Tracer("test")
	t.Cleanup(func() { tracer = origTracer })

	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	rt := traceTransport{parent: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	if !strings.HasPrefix(spans[0].Name, "HTTP ") {
		t.Errorf("span name = %q, want 'HTTP ...' prefix", spans[0].Name)
	}
	if h := gotHeaders.Get("Traceparent"); h == "" {
		t.Errorf("want traceparent header injected, got none")
	}
}

func TestTraceTransport_5xxMarksError(t *testing.T) {
	withTestPropagator(t)
	exp := tracetest.NewInMemoryExporter()
	tp := newTracerProviderWithExporter(exp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	origTracer := tracer
	tracer = tp.Tracer("test")
	t.Cleanup(func() { tracer = origTracer })

	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	rt := traceTransport{parent: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer resp.Body.Close()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("want span status Error, got %v", spans[0].Status.Code)
	}
}
