// Package db initializes a *gorm.DB with OTel tracing and connection pooling.
//
// Driver is selected via Config.Driver (postgres|mysql|sqlite). When Driver is
// empty, Init returns (nil, nil) so the server can boot without a database —
// useful for tests or services that don't need DB yet.
//
// Config is loaded from config.yaml by the internal/config package; defaults
// are filled in by config.Load before this struct reaches db.Init:
//
//	max_open_conns:    25
//	max_idle_conns:    5
//	conn_max_lifetime: 30m
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	gormotel "gorm.io/plugin/opentelemetry/tracing"
)

// Config holds database configuration. Loaded from config.yaml by the
// config package; defaults are filled in by config.Load before this struct
// reaches db.Init.
type Config struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	// LogSQL routes every SQL statement to the gorm logger's Trace method
	// (visible in dev). Default false keeps production stdout clean while
	// still emitting errors and slow-query warnings.
	LogSQL bool `mapstructure:"log_sql"`
}

// Disabled reports whether DB is intentionally absent (Driver empty).
func (c Config) Disabled() bool { return c.Driver == "" }

// Init opens a *gorm.DB for cfg.Driver, registers the OTel tracing plugin so
// every SQL operation becomes a child span, and configures the connection pool.
// Returns (nil, nil) when cfg.Disabled().
func Init(ctx context.Context, cfg Config) (*gorm.DB, error) {
	if cfg.Disabled() {
		return nil, nil
	}

	dialector, err := dialectorFor(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, err
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger(cfg.LogSQL),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open %s: %w", cfg.Driver, err)
	}

	// On any error past this point, close the *gorm.DB so we don't leak
	// connections. gdb is guaranteed non-nil here.
	success := false
	defer func() {
		if !success {
			_ = Close(gdb)
		}
	}()

	// OTel tracing plugin — uses the global TracerProvider, so it picks up
	// whatever internal/otel.Init installed. Safe under noop provider too.
	if err := gdb.Use(gormotel.NewPlugin()); err != nil {
		return nil, fmt.Errorf("gorm otel plugin: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("get *sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Use a fresh timeout for the ping so a long-lived caller ctx (e.g. the
	// server's signal-aware ctx) doesn't make startup hang on an unreachable DB.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	slog.Info("db connected",
		"driver", cfg.Driver,
		"max_open", cfg.MaxOpenConns,
		"max_idle", cfg.MaxIdleConns,
		"conn_max_lifetime", cfg.ConnMaxLifetime,
	)
	success = true
	return gdb, nil
}

// Close closes the underlying *sql.DB. Safe on a nil *gorm.DB.
func Close(gdb *gorm.DB) error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("get *sql.DB for close: %w", err)
	}
	return sqlDB.Close()
}

// Ping wraps *sql.DB.PingContext for health checks.
func Ping(ctx context.Context, gdb *gorm.DB) error {
	if gdb == nil {
		return errors.New("db not initialized")
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func dialectorFor(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "postgres", "postgresql", "pg":
		return postgres.Open(dsn), nil
	case "mysql":
		return mysql.Open(dsn), nil
	case "sqlite", "sqlite3":
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported db.driver %q (want postgres|mysql|sqlite)", driver)
	}
}

// newLogger builds the gorm logger pipeline. The default gorm logger runs at
// Warn level (so errors and slow-query notices always surface); a wrapper
// gates SQL Trace output behind cfg.LogSQL.
func newLogger(logSQL bool) gormlogger.Interface {
	return &sqlToggleLogger{
		inner:  gormlogger.Default.LogMode(gormlogger.Warn),
		logSQL: logSQL,
	}
}

// sqlToggleLogger wraps a gorm logger and gates Trace (per-SQL-statement
// output) behind a bool. Info/Warn/Error pass through unchanged so non-SQL
// diagnostics aren't coupled to the SQL verbosity knob.
type sqlToggleLogger struct {
	inner  gormlogger.Interface
	logSQL bool
}

func (l *sqlToggleLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &sqlToggleLogger{inner: l.inner.LogMode(level), logSQL: l.logSQL}
}

func (l *sqlToggleLogger) Info(ctx context.Context, msg string, args ...any) {
	l.inner.Info(ctx, msg, args...)
}

func (l *sqlToggleLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.inner.Warn(ctx, msg, args...)
}

func (l *sqlToggleLogger) Error(ctx context.Context, msg string, args ...any) {
	l.inner.Error(ctx, msg, args...)
}

func (l *sqlToggleLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !l.logSQL {
		return
	}
	l.inner.Trace(ctx, begin, fc, err)
}
