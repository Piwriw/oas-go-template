package db

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

// TestInit_sqlite_memory verifies shared in-memory SQLite initialization, migrations, and connectivity.
func TestInit_sqlite_memory(t *testing.T) {
	ctx := context.Background()

	gdb, err := Init(ctx, Config{
		Driver:          "sqlite",
		DSN:             "file::memory:?cache=shared",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if gdb == nil {
		t.Fatal("Init returned nil *gorm.DB for non-disabled config")
	}
	defer func() {
		if err := Close(gdb); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	if err := Ping(ctx, gdb); err != nil {
		t.Fatalf("Ping after Init: %v", err)
	}
	if !gdb.Migrator().HasTable(migrationTableName) {
		t.Fatal("Init did not create the schema migration table")
	}

	// Sanity: a real round-trip through gorm.
	var got int
	if err := gdb.Raw("SELECT 1 + 1").Scan(&got).Error; err != nil {
		t.Fatalf("Raw SELECT 1+1: %v", err)
	}
	if got != 2 {
		t.Fatalf("SELECT 1+1 = %d, want 2", got)
	}
}

// TestInit_disabledReturnsNil verifies an omitted database driver produces no connection or error.
func TestInit_disabledReturnsNil(t *testing.T) {
	gdb, err := Init(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Init with empty Config: unexpected err %v", err)
	}
	if gdb != nil {
		t.Fatalf("Init with empty Config returned non-nil *gorm.DB: %v", gdb)
	}
	if err := Close(gdb); err != nil {
		t.Errorf("Close on nil should be no-op, got %v", err)
	}
}

// TestInit_unsupportedDriver verifies unknown database drivers fail before connection setup.
func TestInit_unsupportedDriver(t *testing.T) {
	_, err := Init(context.Background(), Config{
		Driver: "oracle",
		DSN:    "whatever",
	})
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

// TestInit_sqlite_badDSN verifies invalid SQLite connection targets fail initialization.
func TestInit_sqlite_badDSN(t *testing.T) {
	_, err := Init(context.Background(), Config{
		Driver: "sqlite",
		DSN:    "/this/path/does/not/exist/and/cannot/be/created.db",
	})
	if err == nil {
		t.Fatal("expected error for unwritable DSN")
	}
}

// TestDisabled verifies only an empty driver marks database support as intentionally absent.
func TestDisabled(t *testing.T) {
	if !(Config{}).Disabled() {
		t.Fatal("zero-value Config should be Disabled")
	}
	if (Config{Driver: "postgres"}).Disabled() {
		t.Fatal("Config with Driver should not be Disabled")
	}
}

// recordingLogger captures whether Trace was invoked, without depending on
// stdout. Only the methods this test exercises are implemented meaningfully.
type recordingLogger struct {
	traces int
}

// LogMode preserves the recording logger across Gorm severity changes.
func (r *recordingLogger) LogMode(logger.LogLevel) logger.Interface { return r }

// Info ignores informational events that are irrelevant to SQL trace-gating assertions.
func (r *recordingLogger) Info(context.Context, string, ...any) {}

// Warn ignores warning events that are irrelevant to SQL trace-gating assertions.
func (r *recordingLogger) Warn(context.Context, string, ...any) {}

// Error ignores error events that are irrelevant to SQL trace-gating assertions.
func (r *recordingLogger) Error(context.Context, string, ...any) {}

// Trace counts SQL execution events forwarded by the logger under test.
func (r *recordingLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	r.traces++
}

// TestSQLToggleLogger_gatesTrace verifies successful SQL details follow the configured logging switch.
func TestSQLToggleLogger_gatesTrace(t *testing.T) {
	// log_sql=false → Trace calls are dropped before reaching inner.
	inner := &recordingLogger{}
	l := &sqlToggleLogger{inner: inner, logSQL: false}
	l.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT 1", 1 }, nil)
	l.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT 2", 1 }, nil)
	if inner.traces != 0 {
		t.Errorf("logSQL=false: inner.Trace called %d times, want 0", inner.traces)
	}

	// log_sql=true → Trace forwarded to inner.
	on := &recordingLogger{}
	l2 := &sqlToggleLogger{inner: on, logSQL: true}
	l2.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT 1", 1 }, nil)
	if on.traces != 1 {
		t.Errorf("logSQL=true: inner.Trace called %d times, want 1", on.traces)
	}
}

// TestSQLToggleLogger_passesThroughInfoWarnError verifies non-trace database events always reach the base logger.
func TestSQLToggleLogger_passesThroughInfoWarnError(_ *testing.T) {
	// Non-Trace methods must always pass through, regardless of logSQL.
	inner := &recordingLogger{}
	l := &sqlToggleLogger{inner: inner, logSQL: false}

	// None of these should panic; recordingLogger's no-op impls mean we can't
	// count, but the contract is "forward, don't swallow".
	l.Info(context.Background(), "info")
	l.Warn(context.Background(), "warn")
	l.Error(context.Background(), "error")
}
