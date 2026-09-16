package service

import "errors"

// Sentinel failures returned by business operations. Handlers map them to
// public error codes at the API boundary; causes stay wrapped so logs keep
// the original detail.
var (
	// ErrDBHandle indicates the configured database did not yield a usable handle.
	ErrDBHandle = errors.New("readiness database handle failed")
	// ErrDBPing indicates the configured database did not answer a readiness ping.
	ErrDBPing = errors.New("readiness database ping failed")
)
