// Package handler implements the StrictServerInterface generated from OAS.
package handler

import "gorm.io/gorm"

// Handler implements internal/api.StrictServerInterface.
//
// db may be nil when the server boots without a configured database. Because
// the dependency is intentionally disabled, /readyz still reports ready.
type Handler struct {
	db         *gorm.DB
	drainState *DrainState
}

// New returns a Handler wired to the given dependencies. Pass nil for any
// dependency that isn't available; affected endpoints degrade gracefully.
// An optional DrainState makes readiness fail before graceful shutdown.
func New(gdb *gorm.DB, drainStates ...*DrainState) *Handler {
	drainState := NewDrainState()
	if len(drainStates) > 0 && drainStates[0] != nil {
		drainState = drainStates[0]
	}
	return &Handler{db: gdb, drainState: drainState}
}
