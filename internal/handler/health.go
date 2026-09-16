package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
	"github.com/piwriw/oas-go-template/internal/service"
)

// Values shared by the liveness and readiness response bodies.
const (
	statusOK               = "ok"
	msgDatabaseUnavailable = "database unavailable"
)

// GetHealth reports process liveness without probing optional dependencies.
func (h *Handler) GetHealth(_ context.Context, _ api.GetHealthRequestObject) (api.GetHealthResponseObject, error) {
	v := h.svc.Version()
	return api.GetHealth200JSONResponse(api.Health{
		Status:  statusOK,
		Version: &v,
	}), nil
}

// GetReady reports whether configured dependencies are reachable, mapping service failures to stable public codes.
func (h *Handler) GetReady(ctx context.Context, _ api.GetReadyRequestObject) (api.GetReadyResponseObject, error) {
	if err := h.svc.Ready(ctx); err != nil {
		slog.ErrorContext(ctx, "readiness probe failed", "err", err)
		return api.GetReady503JSONResponse(api.Error{
			Code:    int32(readinessCode(err)),
			Message: msgDatabaseUnavailable,
		}), nil
	}
	return api.GetReady200JSONResponse(api.Health{
		Status: statusOK,
	}), nil
}

// readinessCode maps a readiness failure to its public error code.
func readinessCode(err error) errcode.Code {
	switch {
	case errors.Is(err, service.ErrDBHandle):
		return errcode.DBHandle
	case errors.Is(err, service.ErrDBPing):
		return errcode.DBPing
	default:
		return errcode.Internal
	}
}
