// Package service implements the business operations behind the API handlers.
//
// The package is transport-agnostic: operations return domain values and
// sentinel errors, never HTTP responses or public error codes. Mapping to
// api.ResponseObject and api.Error happens only in internal/handler because
// errcode values are part of the public API contract.
package service

import "gorm.io/gorm"

// Service executes business operations over the optional database dependency.
//
// db may be nil when the server boots without a configured database. Because
// the dependency is intentionally disabled, readiness still reports ready.
type Service struct {
	db *gorm.DB
}

// New wires business operations to the optional database dependency.
func New(gdb *gorm.DB) *Service {
	return &Service{db: gdb}
}
