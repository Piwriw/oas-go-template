// Package handler implements the StrictServerInterface generated from OAS by
// delegating business work to internal/service and mapping its results and
// sentinel errors to typed response objects.
package handler

import "github.com/piwriw/oas-go-template/internal/service"

// Handler implements internal/api.StrictServerInterface. It owns no
// dependencies of its own — dependencies live on service.Service, whose db
// may be nil when the server boots without a configured database.
type Handler struct {
	svc *service.Service
}

// New wires API handlers to the business service.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}
