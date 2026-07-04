package db

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

// SQLite :memory: with cache=shared keeps a single in-memory DB across the
// connection pool. Combined with MaxOpenConns=1, this avoids the "first query
// creates the schema, second connection sees an empty DB" trap.
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

	// Sanity: a real round-trip through gorm.
	var got int
	if err := gdb.Raw("SELECT 1 + 1").Scan(&got).Error; err != nil {
		t.Fatalf("Raw SELECT 1+1: %v", err)
	}
	if got != 2 {
		t.Fatalf("SELECT 1+1 = %d, want 2", got)
	}
}

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

func TestInit_unsupportedDriver(t *testing.T) {
	_, err := Init(context.Background(), Config{
		Driver: "oracle",
		DSN:    "whatever",
	})
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestInit_sqlite_badDSN(t *testing.T) {
	_, err := Init(context.Background(), Config{
		Driver: "sqlite",
		DSN:    "/this/path/does/not/exist/and/cannot/be/created.db",
	})
	if err == nil {
		t.Fatal("expected error for unwritable DSN")
	}
}

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

func (r *recordingLogger) LogMode(logger.LogLevel) logger.Interface { return r }
func (r *recordingLogger) Info(context.Context, string, ...any)     {}
func (r *recordingLogger) Warn(context.Context, string, ...any)     {}
func (r *recordingLogger) Error(context.Context, string, ...any)    {}
func (r *recordingLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	r.traces++
}

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
