package store

import "errors"

// ErrNotFound indicates that a requested persistent record does not exist.
var ErrNotFound = errors.New("store: record not found")
