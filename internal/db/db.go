// Package db initializes a *gorm.DB with OTel tracing, connection pooling, and
// embedded SQL schema migrations.
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
	"net"
	"net/url"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	gormotel "gorm.io/plugin/opentelemetry/tracing"
)

const (
	// DriverPostgres selects the PostgreSQL database dialect.
	DriverPostgres = "postgres"
	// DriverMySQL selects the MySQL database dialect.
	DriverMySQL = "mysql"
	// DriverSQLite selects the SQLite database dialect.
	DriverSQLite = "sqlite"

	databasePingTimeout = 5 * time.Second
)

// Config holds database configuration. Loaded from config.yaml by the
// config package; defaults are filled in by config.Load before this struct
// reaches db.Init.
type Config struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
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

// Init opens the configured database, enables SQL tracing, configures pooling, and applies migrations.
func Init(ctx context.Context, cfg Config) (*gorm.DB, error) {
	if cfg.Disabled() {
		return nil, nil
	}

	dialector, err := dialectorFor(cfg)
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
	pingCtx, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	if err := Migrate(ctx, gdb, cfg, MigrationUp); err != nil {
		return nil, fmt.Errorf("db migrate: %w", err)
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

// connectionString encodes structured connection fields for the configured database driver.
func (c Config) connectionString() string {
	switch c.Driver {
	case DriverPostgres:
		connection := url.URL{
			Scheme: DriverPostgres,
			User:   url.UserPassword(c.User, c.Password),
			Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
			Path:   "/" + c.Database,
		}
		query := url.Values{"sslmode": {c.SSLMode}}
		connection.RawQuery = query.Encode()
		return connection.String()
	case DriverMySQL:
		connection := mysqldriver.NewConfig()
		connection.User = c.User
		connection.Passwd = c.Password
		connection.Net = "tcp"
		connection.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
		connection.DBName = c.Database
		connection.ParseTime = true
		return connection.FormatDSN()
	default:
		return c.Database
	}
}

// dialectorFor selects the Gorm dialect using structured connection settings.
func dialectorFor(cfg Config) (gorm.Dialector, error) {
	dsn := cfg.connectionString()
	switch cfg.Driver {
	case DriverPostgres:
		return postgres.Open(dsn), nil
	case DriverMySQL:
		return mysql.Open(dsn), nil
	case DriverSQLite:
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported db.driver %q (want postgres|mysql|sqlite)", cfg.Driver)
	}
}

// newLogger builds a Gorm logger that always reports warnings while optionally tracing SQL.
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

// LogMode returns a SQL logger configured for the requested Gorm severity.
func (l *sqlToggleLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &sqlToggleLogger{inner: l.inner.LogMode(level), logSQL: l.logSQL}
}

// Info forwards informational database events to the wrapped Gorm logger.
func (l *sqlToggleLogger) Info(ctx context.Context, msg string, args ...any) {
	l.inner.Info(ctx, msg, args...)
}

// Warn forwards database warnings to the wrapped Gorm logger.
func (l *sqlToggleLogger) Warn(ctx context.Context, msg string, args ...any) {
	l.inner.Warn(ctx, msg, args...)
}

// Error forwards database failures to the wrapped Gorm logger.
func (l *sqlToggleLogger) Error(ctx context.Context, msg string, args ...any) {
	l.inner.Error(ctx, msg, args...)
}

// Trace forwards SQL execution details only when SQL logging is enabled or the operation failed.
func (l *sqlToggleLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if !l.logSQL {
		return
	}
	l.inner.Trace(ctx, begin, fc, err)
}
