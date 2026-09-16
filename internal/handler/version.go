package handler

import (
	"context"

	"github.com/piwriw/oas-go-template/internal/api"
)

// GetVersion returns build metadata from the service, which owns the
// degradation rules and the child span.
func (h *Handler) GetVersion(ctx context.Context, _ api.GetVersionRequestObject) (api.GetVersionResponseObject, error) {
	info := h.svc.VersionInfo(ctx)
	return api.GetVersion200JSONResponse(api.VersionInfo{
		Version:   info.Version,
		GitCommit: info.GitCommit,
		BuildTime: info.BuildTime,
	}), nil
}
