package service

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/piwriw/oas-go-template/internal/version"
)

// tracer is project-wide; otel.Tracer returns a no-op when Init wasn't called.
var tracer = otel.Tracer("github.com/piwriw/oas-go-template/internal/service")

// Placeholder build metadata reported when ldflags left a field empty.
const (
	degradedVersion   = "dev"
	degradedBuildMeta = "unknown"
)

// Version returns the raw build version for liveness reporting.
func (s *Service) Version() string {
	return version.Info().Version
}

// VersionInfo returns build metadata degraded to stable placeholders so
// /version keeps working for binaries built without ldflags, recording the
// lookup in the active request trace.
func (s *Service) VersionInfo(ctx context.Context) version.InfoT {
	_, span := tracer.Start(ctx, "Service.VersionInfo")
	defer span.End()

	info := version.Info()
	// Degrade rather than 500 — running via `go run` (no ldflags) is common
	// during development and /version should still work.
	if info.Version == "" {
		info.Version = degradedVersion
	}
	if info.GitCommit == "" {
		info.GitCommit = degradedBuildMeta
	}
	if info.BuildTime == "" {
		info.BuildTime = degradedBuildMeta
	}

	span.SetAttributes(
		attribute.String("version.info.version", info.Version),
		attribute.String("version.info.git_commit", info.GitCommit),
		attribute.String("version.info.build_time", info.BuildTime),
	)
	span.SetStatus(codes.Ok, "")

	return info
}
