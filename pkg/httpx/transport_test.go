package httpx

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// closeBody silences bodyclose/errcheck on responses whose body we don't
// otherwise read in a test.
func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestLogTransport_2xx_Info(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	defer func() { _ = resp.Body.Close() }()

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	defer func() { _ = resp.Body.Close() }()

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("want WARN level, got: %s", out)
	}
}

func TestLogTransport_5xx_Warn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	defer func() { _ = resp.Body.Close() }()

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
	resp, err := rt.RoundTrip(req)
	closeBody(t, resp)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	defer func() { _ = resp.Body.Close() }()

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
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotHeaders = r.Header.Clone()
		mu.Unlock()
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
	defer func() { _ = resp.Body.Close() }()

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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	defer func() { _ = resp.Body.Close() }()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("want span status Error, got %v", spans[0].Status.Code)
	}
}

func TestRetryTransport_RetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	policy := RetryPolicy{MaxAttempts: 3, Initial: time.Millisecond, Max: 5 * time.Millisecond, Multiplier: 2, Jitter: 0}
	rt := retryTransport{parent: http.DefaultTransport, policy: policy}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestRetryTransport_DoesNotRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	policy := RetryPolicy{MaxAttempts: 3, Initial: time.Millisecond, Max: 5 * time.Millisecond, Multiplier: 2, Jitter: 0}
	rt := retryTransport{parent: http.DefaultTransport, policy: policy}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", calls)
	}
}

func TestRetryTransport_DoesNotRetryOnPOST(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	policy := RetryPolicy{MaxAttempts: 3, Initial: time.Millisecond, Max: 5 * time.Millisecond, Multiplier: 2, Jitter: 0}
	rt := retryTransport{parent: http.DefaultTransport, policy: policy}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retry on POST)", calls)
	}
}

func TestRetryTransport_RespectsMaxAttempts(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	policy := RetryPolicy{MaxAttempts: 3, Initial: time.Millisecond, Max: 5 * time.Millisecond, Multiplier: 2, Jitter: 0}
	rt := retryTransport{parent: http.DefaultTransport, policy: policy}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestRetryTransport_FinalAttemptReturnsImmediatelyWithReadableBody(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls int
	parent := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		header := make(http.Header)
		if calls == 2 {
			header.Set("Retry-After", "3600")
			cancel()
		}
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader("upstream still unavailable")),
			Request:    req,
		}, nil
	})
	policy := RetryPolicy{MaxAttempts: 2}
	rt := retryTransport{parent: parent, policy: policy}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		closeBody(t, resp)
		t.Fatalf("RoundTrip err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatalf("read final response body: %v", readErr)
	}
	if got, want := string(body), "upstream still unavailable"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestRetryTransport_ContextCancelStopsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	policy := RetryPolicy{MaxAttempts: 10, Initial: 50 * time.Millisecond, Max: 50 * time.Millisecond, Multiplier: 1, Jitter: 0}
	rt := retryTransport{parent: http.DefaultTransport, policy: policy}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	resp, err := rt.RoundTrip(req)
	closeBody(t, resp)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want error from canceled request")
	}
	if elapsed > 30*time.Millisecond {
		t.Errorf("cancel didn't short-circuit backoff; elapsed = %v", elapsed)
	}
}

func TestRetryTransport_RetriesOnNetworkError(t *testing.T) {
	policy := RetryPolicy{MaxAttempts: 3, Initial: time.Millisecond, Max: 5 * time.Millisecond, Multiplier: 2, Jitter: 0}
	rt := retryTransport{parent: http.DefaultTransport, policy: policy}

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	closeBody(t, resp)
	if err == nil {
		t.Fatal("want error")
	}
}
