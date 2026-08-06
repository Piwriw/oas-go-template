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

// New wires API handlers to the optional database dependency.
func New(gdb *gorm.DB) *Handler {
	return &Handler{db: gdb}
}
