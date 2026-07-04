package handler

import (
	"context"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/version"
)

// GetVersion implements api.StrictServerInterface.GetVersion.
func (h *Handler) GetVersion(_ context.Context, _ api.GetVersionRequestObject) (api.GetVersionResponseObject, error) {
	info := version.Info()
	return api.GetVersion200JSONResponse(api.VersionInfo{
		Version:   info.Version,
		GitCommit: info.GitCommit,
		BuildTime: info.BuildTime,
	}), nil
}
