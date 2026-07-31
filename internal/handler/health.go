package handler

import (
	"context"
	"log/slog"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
	"github.com/piwriw/oas-go-template/internal/version"
)

// GetHealth reports process liveness without probing optional dependencies.
func (h *Handler) GetHealth(_ context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	v := version.Info().Version
	return api.GetHealth200JSONResponse(api.Health{
		Status:  "ok",
		Version: &v,
	}), nil
}

// GetReady reports whether the server is accepting traffic and configured dependencies are reachable.
func (h *Handler) GetReady(ctx context.Context, _ api.GetReadyRequestObject) (api.GetReadyResponseObject, error) {
	if h.drainState != nil && h.drainState.Draining() {
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(errcode.ServiceDraining),
			Message: "service is shutting down",
		}), nil
	}
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
