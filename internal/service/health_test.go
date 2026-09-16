package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestReadyWithoutDB verifies readiness succeeds when database support is intentionally disabled.
func TestReadyWithoutDB(t *testing.T) {
	if err := New(nil).Ready(context.Background()); err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
}

// TestReadyDetectsClosedDatabase verifies a configured but unreachable database
// wraps the ping sentinel while keeping the private cause for logging.
func TestReadyDetectsClosedDatabase(t *testing.T) {
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

	err = New(gdb).Ready(context.Background())
	if !errors.Is(err, ErrDBPing) {
		t.Fatalf("Ready() error = %v, want ErrDBPing", err)
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("Ready() error %q lost the private cause", err)
	}
}
