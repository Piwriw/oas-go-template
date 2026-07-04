package handler

import (
	"context"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/version"
)

// GetHealth implements api.StrictServerInterface.GetHealth — liveness only.
// Returns 200 as long as the process is up; dependency checks live in GetReady.
func (h *Handler) GetHealth(_ context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	v := version.Info().Version
	return api.GetHealth200JSONResponse(api.Health{
		Status:  "ok",
		Version: &v,
	}), nil
}

// GetReady implements api.StrictServerInterface.GetReady — readiness probe.
// Returns 200 when configured dependencies are reachable; 503 when a
// dependency hasn't been wired (e.g. db.driver empty) or is failing.
func (h *Handler) GetReady(ctx context.Context, _ api.GetReadyRequestObject) (api.GetReadyResponseObject, error) {
	if h.db == nil {
		return api.GetReady503JSONResponse(api.Error{
			Code:    "db_unavailable",
			Message: "db not configured",
		}), nil
	}
	sqlDB, err := h.db.DB()
	if err != nil {
		return api.GetReady503JSONResponse(api.Error{
			Code:    "db_handle",
			Message: err.Error(),
		}), nil
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return api.GetReady503JSONResponse(api.Error{
			Code:    "db_ping",
			Message: err.Error(),
		}), nil
	}
	return api.GetReady200JSONResponse(api.Health{
		Status: "ok",
	}), nil
}
