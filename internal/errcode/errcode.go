// Package errcode defines the stable numeric codes returned in
// api.Error.Code. Codes are part of the public API contract even though
// they don't appear in spec/openapi.yaml — clients branch on them.
//
// Use the typed constants below; never construct ad-hoc Code values.
// Convert at the API boundary with int(c) when assigning to api.Error.Code
// (which is int32 per the OAS schema).
//
// Numbering convention:
//
//   - 5-digit codes leave room for ~100 categories × ~100 codes each.
//   - Ranges are reserved per subsystem; allocate new ranges here in the
//     doc comment before adding constants.
//   - Never recycle a retired code's number for a new meaning — clients
//     in the wild may still be branching on it. Retire by comment, not
//     by reuse.
//
// Range allocation:
//
//	10xxx  request / validation
//	20xxx  auth / authorization
//	30xxx  not found
//	50xxx  database / infrastructure
//	99xxx  internal / unknown
package errcode

// Code is a stable int32 identifier for an error category returned in
// api.Error.Code. Compile-time type safety prevents callers from passing
// arbitrary numbers where a known code is expected.
type Code int32

// Request and validation errors (10xxx).
const (
	InvalidRequest      Code = 10001
	RequestBodyTooLarge Code = 10002
)

const (
	// Forbidden indicates the request is not allowed by the server policy.
	Forbidden Code = 20001
)

// Resource and routing errors (30xxx).
const (
	NotFound         Code = 30001
	MethodNotAllowed Code = 30002
)

// Internal errors (99xxx).
const Internal Code = 99001

// Database-related codes (50xxx) — returned when a configured DB is
// misbehaving or unreachable. See internal/handler/health.go:GetReady.
const (
	// Deprecated: an intentionally disabled database is not a readiness error.
	// Keep this value reserved because public error codes must never be reused.
	DBUnavailable   Code = 50001
	DBHandle        Code = 50002 // (*gorm.DB).DB() returned a non-nil error
	DBPing          Code = 50003 // (*sql.DB).PingContext failed
	ServiceDraining Code = 50004 // readiness is failing during graceful shutdown
)
