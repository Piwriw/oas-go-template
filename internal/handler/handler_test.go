package handler

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

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

func TestGetReadySanitizesDatabaseError(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("gdb.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close: %v", err)
	}

	response, err := New(gdb).GetReady(context.Background(), api.GetReadyRequestObject{})
	if err != nil {
		t.Fatalf("GetReady() error = %v", err)
	}
	ready, ok := response.(api.GetReady503JSONResponse)
	if !ok {
		t.Fatalf("GetReady() response type = %T, want api.GetReady503JSONResponse", response)
	}
	if ready.Message != "database unavailable" {
		t.Errorf("GetReady() message = %q, want stable message", ready.Message)
	}
	if strings.Contains(ready.Message, "closed") {
		t.Errorf("database detail leaked: %q", ready.Message)
	}
}
