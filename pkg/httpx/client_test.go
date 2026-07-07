package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestNew_CustomTransport_WrappedByChain(t *testing.T) {
	custom := &countingTransport{}
	c := New(WithTransport(custom))
	if c.base.Transport == custom {
		t.Errorf("transport should be wrapped, not used directly")
	}
	srv := newOKServer(t)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := c.base.Do(req)
	if err != nil {
		t.Fatalf("Do err: %v", err)
	}
	defer resp.Body.Close()
	if custom.calls == 0 {
		t.Errorf("custom transport not invoked")
	}
}

type countingTransport struct{ calls int }

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls++
	return http.DefaultTransport.RoundTrip(req)
}

func newOKServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}
