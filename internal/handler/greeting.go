package handler

import (
	"context"
	"errors"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
	"github.com/piwriw/oas-go-template/internal/service"
)

// GenerateGreeting maps the greeting service result to the public API contract.
func (h *Handler) GenerateGreeting(_ context.Context, request api.GenerateGreetingRequestObject) (api.GenerateGreetingResponseObject, error) {
	message, err := h.svc.Greeting(request.Body.Name)
	if errors.Is(err, service.ErrBlankName) {
		return api.GenerateGreeting400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Code:    int32(errcode.InvalidRequest),
				Message: "name must not be blank",
			},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return api.GenerateGreeting200JSONResponse{Message: message}, nil
}
