package httpx

import (
	"testing"
	"time"
)

func TestDefaultRetry(t *testing.T) {
	p := DefaultRetry()
	if p.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", p.MaxAttempts)
	}
	if p.Initial != 100*time.Millisecond {
		t.Errorf("Initial = %v, want 100ms", p.Initial)
	}
	if p.Max != 2*time.Second {
		t.Errorf("Max = %v, want 2s", p.Max)
	}
	if p.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", p.Multiplier)
	}
	if p.Jitter != 0.2 {
		t.Errorf("Jitter = %v, want 0.2", p.Jitter)
	}
}
