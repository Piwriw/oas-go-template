// Package handler implements the StrictServerInterface generated from OAS.
package handler

import "gorm.io/gorm"

// Handler implements internal/api.StrictServerInterface.
//
// db may be nil when the server boots without a configured database. Because
// the dependency is intentionally disabled, /readyz still reports ready.
type Handler struct {
	db *gorm.DB
}

// New returns a Handler wired to the given dependencies. Pass nil for any
// dependency that isn't available; affected endpoints degrade gracefully.
func New(gdb *gorm.DB) *Handler {
	return &Handler{db: gdb}
}
