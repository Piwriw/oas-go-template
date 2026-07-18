package handler

import (
	"context"
	"log/slog"

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
// Returns 200 when every configured dependency is reachable. A nil DB means
// the database is intentionally disabled, so there is nothing to check.
//
// Errors are intentionally converted to typed 503 responses and paired with
// a nil return error — StrictServerInterface convention. Returning the raw
// err would route to gin's generic 500 path and discard the structured body.
func (h *Handler) GetReady(ctx context.Context, _ api.GetReadyRequestObject) (api.GetReadyResponseObject, error) {
	if h.db == nil {
		return api.GetReady200JSONResponse(api.Health{
			Status: "ok",
		}), nil
	}
	sqlDB, err := h.db.DB()
	if err != nil {
		slog.ErrorContext(ctx, "readiness database handle failed", "err", err)
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(errcode.DBHandle),
			Message: "database unavailable",
		}), nil
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		slog.ErrorContext(ctx, "readiness database ping failed", "err", err)
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(errcode.DBPing),
			Message: "database unavailable",
		}), nil
	}
	return api.GetReady200JSONResponse(api.Health{
		Status: "ok",
	}), nil
}
