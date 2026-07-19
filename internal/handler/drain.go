package handler

import (
	"sync/atomic"
	"time"
)

const defaultDrainTimeout = 5 * time.Second

// DrainState coordinates graceful shutdown between the HTTP lifecycle and the
// readiness handler. It is safe to read and update from concurrent requests.
type DrainState struct {
	draining atomic.Bool
	timeout  time.Duration
}

// NewDrainState creates a readiness state. An omitted timeout defaults to five
// seconds; callers may pass zero to skip the wait or a positive timeout to let
// load balancers observe the 503 readiness response before shutdown.
func NewDrainState(timeout ...time.Duration) *DrainState {
	drainTimeout := defaultDrainTimeout
	if len(timeout) > 0 {
		drainTimeout = timeout[0]
	}
	if drainTimeout < 0 {
		drainTimeout = 0
	}
	return &DrainState{timeout: drainTimeout}
}

// Begin marks the process as draining. Calling it repeatedly is safe.
func (s *DrainState) Begin() {
	if s != nil {
		s.draining.Store(true)
	}
}

// Draining reports whether readiness should fail during shutdown.
func (s *DrainState) Draining() bool {
	return s != nil && s.draining.Load()
}

// Timeout returns the configured endpoint-removal grace period.
func (s *DrainState) Timeout() time.Duration {
	if s == nil {
		return 0
	}
	return s.timeout
}
