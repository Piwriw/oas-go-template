package handler

import (
	"context"

	"github.com/piwriw/oas-go-template/internal/api"
)

// GetHealth implements api.StrictServerInterface.GetHealth.
func (h *Handler) GetHealth(_ context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	version := "0.1.0"
	return api.GetHealth200JSONResponse(api.Health{
		Status:  "ok",
		Version: &version,
	}), nil
}
