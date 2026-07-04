package handler

import (
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
