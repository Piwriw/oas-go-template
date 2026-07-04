package handler

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/version"
)

// tracer is project-wide; otel.Tracer returns a no-op when Init wasn't called.
var tracer = otel.Tracer("github.com/piwriw/oas-go-template/internal/handler")

// GetVersion implements api.StrictServerInterface.GetVersion.
// Demonstrates manual span creation on top of the otelgin middleware.
func (h *Handler) GetVersion(ctx context.Context, _ api.GetVersionRequestObject) (api.GetVersionResponseObject, error) {
	_, span := tracer.Start(ctx, "Handler.GetVersion")
	defer span.End()

	info := version.Info()
	// Degrade to "dev" rather than 500 — running via `go run` (no ldflags) is
	// common during development and /version should still work.
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.GitCommit == "" {
		info.GitCommit = "unknown"
	}
	if info.BuildTime == "" {
		info.BuildTime = "unknown"
	}

	span.SetAttributes(
		attribute.String("version.info.version", info.Version),
		attribute.String("version.info.git_commit", info.GitCommit),
		attribute.String("version.info.build_time", info.BuildTime),
	)
	span.SetStatus(codes.Ok, "")

	return api.GetVersion200JSONResponse(api.VersionInfo{
		Version:   info.Version,
		GitCommit: info.GitCommit,
		BuildTime: info.BuildTime,
	}), nil
}
