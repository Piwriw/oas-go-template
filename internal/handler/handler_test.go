package handler

import (
	"context"
	"testing"

	"github.com/piwriw/oas-go-template/internal/api"
)

// Compile-time assertion that Handler implements StrictServerInterface.
var _ api.StrictServerInterface = (*Handler)(nil)

func TestNewReturnsHandler(t *testing.T) {
	if New(nil) == nil {
		t.Fatal("New(nil) returned nil")
	}
}

func TestGetReadyWithoutDB(t *testing.T) {
	response, err := New(nil).GetReady(context.Background(), api.GetReadyRequestObject{})
	if err != nil {
		t.Fatalf("GetReady() error = %v", err)
	}

	ready, ok := response.(api.GetReady200JSONResponse)
	if !ok {
		t.Fatalf("GetReady() response type = %T, want api.GetReady200JSONResponse", response)
	}
	if ready.Status != "ok" {
		t.Errorf("GetReady() status = %q, want ok", ready.Status)
	}
}
