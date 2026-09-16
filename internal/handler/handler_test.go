package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/errcode"
	"github.com/piwriw/oas-go-template/internal/service"
)

// Compile-time assertion that Handler implements StrictServerInterface.
var _ api.StrictServerInterface = (*Handler)(nil)

// TestNewReturnsHandler verifies dependency construction produces a usable strict API handler.
func TestNewReturnsHandler(t *testing.T) {
	if New(service.New(nil)) == nil {
		t.Fatal("New(service.New(nil)) returned nil")
	}
}

// TestGetReadyWithoutDB verifies readiness succeeds when database support is intentionally disabled.
func TestGetReadyWithoutDB(t *testing.T) {
	response, err := New(service.New(nil)).GetReady(context.Background(), api.GetReadyRequestObject{})
	if err != nil {
		t.Fatalf("GetReady() error = %v", err)
	}

	ready, ok := response.(api.GetReady200JSONResponse)
	if !ok {
		t.Fatalf("GetReady() response type = %T, want api.GetReady200JSONResponse", response)
	}
	if ready.Status != statusOK {
		t.Errorf("GetReady() status = %q, want ok", ready.Status)
	}
}

// TestGetReadyMapsDatabasePingToStableError verifies dependency failures return stable readiness errors.
func TestGetReadyMapsDatabasePingToStableError(t *testing.T) {
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

	response, err := New(service.New(gdb)).GetReady(context.Background(), api.GetReadyRequestObject{})
	if err != nil {
		t.Fatalf("GetReady() error = %v", err)
	}
	ready, ok := response.(api.GetReady503JSONResponse)
	if !ok {
		t.Fatalf("GetReady() response type = %T, want api.GetReady503JSONResponse", response)
	}
	if ready.Code != int32(errcode.DBPing) {
		t.Errorf("GetReady() code = %d, want %d", ready.Code, int32(errcode.DBPing))
	}
	if ready.Message != msgDatabaseUnavailable {
		t.Errorf("GetReady() message = %q, want stable message", ready.Message)
	}
	if strings.Contains(ready.Message, "closed") {
		t.Errorf("database detail leaked: %q", ready.Message)
	}
}

// TestReadinessCodeMapsServiceSentinels verifies every service sentinel has a stable public code.
func TestReadinessCodeMapsServiceSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want errcode.Code
	}{
		{"db handle", fmt.Errorf("%w: %w", service.ErrDBHandle, errors.New("boom")), errcode.DBHandle},
		{"db ping", fmt.Errorf("%w: %w", service.ErrDBPing, errors.New("boom")), errcode.DBPing},
		{"unknown", errors.New("boom"), errcode.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := readinessCode(tc.err); got != tc.want {
				t.Errorf("readinessCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
