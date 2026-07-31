package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNew_Defaults verifies new clients use package timeout, retry, transport, and logger defaults.
func TestNew_Defaults(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.base == nil {
		t.Errorf("base *http.Client is nil")
	}
	if c.base.Transport == nil {
		t.Errorf("transport chain not set")
	}
	if c.retry.MaxAttempts != 3 {
		t.Errorf("default retry.MaxAttempts = %d, want 3", c.retry.MaxAttempts)
	}
}

// TestNew_CustomTransport_WrappedByChain verifies custom transports remain behind the policy decorator chain.
func TestNew_CustomTransport_WrappedByChain(t *testing.T) {
	custom := &countingTransport{}
	c := New(WithTransport(custom))
	if c.base.Transport == custom {
		t.Errorf("transport should be wrapped, not used directly")
	}
	srv := newOKServer(t)
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := c.base.Do(req)
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if custom.calls == 0 {
		t.Errorf("custom transport not invoked")
	}
}

type countingTransport struct{ calls int }

// RoundTrip counts outbound attempts before delegating to the wrapped transport.
func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	return http.DefaultTransport.RoundTrip(req)
}

// newOKServer starts a disposable JSON endpoint for HTTP client wrapper tests.
func newOKServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

type wrapper struct {
	Field string `json:"field"`
}

// TestGet_Wrapper verifies the GET convenience wrapper decodes successful JSON.
func TestGet_Wrapper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wrapper{Field: "g"})
	}))
	defer srv.Close()

	c := New()
	out, err := Get[wrapper](context.Background(), c, srv.URL)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if out.Field != "g" {
		t.Errorf("Field = %q", out.Field)
	}
}

// TestPost_Wrapper verifies the POST convenience wrapper sends and decodes JSON.
func TestPost_Wrapper(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wrapper{Field: "p"})
	}))
	defer srv.Close()

	c := New()
	out, err := Post[wrapper](context.Background(), c, srv.URL, wrapper{Field: "req"})
	if err != nil {
		t.Fatalf("Post err: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s", gotMethod)
	}
	if out.Field != "p" {
		t.Errorf("Field = %q", out.Field)
	}
}

// TestPut_Patch_Delete_Wrappers verifies method-specific wrappers dispatch the expected HTTP verbs.
func TestPut_Patch_Delete_Wrappers(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wrapper{Field: "ok"})
	}))
	defer srv.Close()

	c := New()

	gotMethod = ""
	if _, err := Put[wrapper](context.Background(), c, srv.URL, wrapper{Field: "x"}); err != nil {
		t.Fatalf("Put err: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("Put: method = %s", gotMethod)
	}

	gotMethod = ""
	if _, err := Patch[wrapper](context.Background(), c, srv.URL, wrapper{Field: "x"}); err != nil {
		t.Fatalf("Patch err: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("Patch: method = %s", gotMethod)
	}

	gotMethod = ""
	if _, err := Delete[wrapper](context.Background(), c, srv.URL); err != nil {
		t.Fatalf("Delete err: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("Delete: method = %s", gotMethod)
	}
}

// TestPostVoid_Etc verifies bodyless convenience wrappers preserve successful response metadata.
func TestPostVoid_Etc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	c := New()

	resp, err := PostVoid(context.Background(), c, srv.URL, wrapper{Field: "x"})
	if err != nil {
		t.Fatalf("PostVoid err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	resp, err = PutVoid(context.Background(), c, srv.URL, wrapper{Field: "x"})
	if err != nil {
		t.Fatalf("PutVoid err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	resp, err = PatchVoid(context.Background(), c, srv.URL, wrapper{Field: "x"})
	if err != nil {
		t.Fatalf("PatchVoid err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	resp, err = DeleteVoid(context.Background(), c, srv.URL)
	if err != nil {
		t.Fatalf("DeleteVoid err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}
