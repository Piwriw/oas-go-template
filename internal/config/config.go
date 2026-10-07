// Package config loads runtime configuration from a YAML file.
//
// Loading order:
//
//  1. Built-in defaults applied to the Config struct
//  2. config.yaml (path from -c flag, default "config.yaml")
//
// When the file is absent, defaults take over so the server still boots in
// dev/test contexts. Secrets stay in the (gitignored) config.yaml; commit only
// config.example.yaml.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/piwriw/oas-go-template/internal/db"
	"github.com/piwriw/oas-go-template/internal/logging"
	"github.com/piwriw/oas-go-template/internal/otel"
)

const (
	defaultPostgresPort    = 5432
	defaultMySQLPort       = 3306
	maxDatabasePort        = 65535
	defaultDatabaseSSLMode = "prefer"
)

// Config holds all runtime configuration for the server.
type Config struct {
	Server ServerConfig      `mapstructure:"server"`
	DB     db.Config         `mapstructure:"db"`
	Log    logging.LogConfig `mapstructure:"log"`
	OTel   otel.Config       `mapstructure:"otel"`
}

// ServerConfig carries HTTP server settings.
type ServerConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
	GinMode  string `mapstructure:"gin_mode"`
}

// Load merges a YAML file over server defaults and validates the resulting runtime configuration.
func Load(path string) (*Config, error) {
	cfg := defaults()

	switch _, statErr := os.Stat(path); {
	case statErr == nil:
		v := viper.New()
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config %q: %w", path, err)
		}
		if err := v.Unmarshal(&cfg); err != nil {
			return nil, fmt.Errorf("decode config: %w", err)
		}
	case errors.Is(statErr, os.ErrNotExist):
		// fall through to defaults
	default:
		return nil, fmt.Errorf("stat config %q: %w", path, statErr)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// defaults returns the Config used when nothing else is configured.
func defaults() Config {
	return Config{
		Server: ServerConfig{
			HTTPAddr: ":8000",
			GinMode:  "debug",
		},
		DB: db.Config{
			Host:            "localhost",
			SSLMode:         defaultDatabaseSSLMode,
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		},
		Log: logging.LogConfig{
			Format: "text",
			Level:  "info",
			Caller: true,
		},
		OTel: otel.Config{
			Enabled: true,
		},
	}
}

// validate enforces invariants after YAML has been merged into defaults.
func validate(cfg *Config) error {
	switch cfg.Server.GinMode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("invalid server.gin_mode %q (want debug|release|test)", cfg.Server.GinMode)
	}

	switch strings.ToLower(cfg.Log.Format) {
	case "text", "json":
	default:
		return fmt.Errorf("invalid log.format %q (want text|json)", cfg.Log.Format)
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Log.Level)) {
	case "debug", "info", "warn", "warning", "error", "":
	default:
		return fmt.Errorf("invalid log.level %q", cfg.Log.Level)
	}

	return validateDB(&cfg.DB)
}

// validateDB validates enabled database connection fields and resolves driver-specific defaults.
func validateDB(cfg *db.Config) error {
	cfg.Driver = strings.ToLower(cfg.Driver)
	switch cfg.Driver {
	case "":
		return nil
	case db.DriverPostgres, "postgresql", "pg":
		cfg.Driver = db.DriverPostgres
		if cfg.Port == 0 {
			cfg.Port = defaultPostgresPort
		}
		cfg.SSLMode = strings.ToLower(cfg.SSLMode)
		switch cfg.SSLMode {
		case "disable", "allow", defaultDatabaseSSLMode, "require", "verify-ca", "verify-full":
		default:
			return fmt.Errorf("invalid db.ssl_mode %q", cfg.SSLMode)
		}
	case db.DriverMySQL:
		if cfg.Port == 0 {
			cfg.Port = defaultMySQLPort
		}
	case db.DriverSQLite, "sqlite3":
		cfg.Driver = db.DriverSQLite
	default:
		return fmt.Errorf("unsupported db.driver %q (want postgres|mysql|sqlite)", cfg.Driver)
	}

	if strings.TrimSpace(cfg.Database) == "" {
		return fmt.Errorf("db.driver=%q but db.database is empty", cfg.Driver)
	}
	if cfg.Driver == db.DriverSQLite {
		return nil
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("db.driver=%q but db.host is empty", cfg.Driver)
	}
	if strings.TrimSpace(cfg.User) == "" {
		return fmt.Errorf("db.driver=%q but db.user is empty", cfg.Driver)
	}
	if cfg.Port < 1 || cfg.Port > maxDatabasePort {
		return fmt.Errorf("db.port must be between 1 and %d", maxDatabasePort)
	}
	return nil
}
