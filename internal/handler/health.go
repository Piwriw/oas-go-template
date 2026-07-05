package handler

import (
	"context"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
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
//
// Errors are intentionally converted to typed 503 responses and paired with
// a nil return error — StrictServerInterface convention. Returning the raw
// err would route to gin's generic 500 path and discard the structured body.
func (h *Handler) GetReady(ctx context.Context, _ api.GetReadyRequestObject) (api.GetReadyResponseObject, error) {
	if h.db == nil {
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(errcode.DBUnavailable),
			Message: "db not configured",
		}), nil
	}
	sqlDB, err := h.db.DB()
	if err != nil {
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(errcode.DBHandle),
			Message: err.Error(),
		}), nil
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(errcode.DBPing),
			Message: err.Error(),
		}), nil
	}
	return api.GetReady200JSONResponse(api.Health{
		Status: "ok",
	}), nil
}
