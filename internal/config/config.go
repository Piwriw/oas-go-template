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

const bytesPerMB = 1 << 20

// Config holds all runtime configuration for the server.
type Config struct {
	Server ServerConfig      `mapstructure:"server"`
	DB     db.Config         `mapstructure:"db"`
	Log    logging.LogConfig `mapstructure:"log"`
	OTel   otel.Config       `mapstructure:"otel"`
	CORS   CORSConfig        `mapstructure:"cors"`
}

// CORSConfig carries cross-origin request policy. CORS is disabled by default;
// when enabled, at least one explicit origin must be configured.
type CORSConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	AllowOrigins     []string      `mapstructure:"allow_origins"`
	AllowMethods     []string      `mapstructure:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers"`
	ExposeHeaders    []string      `mapstructure:"expose_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age"`
}

// ServerConfig carries HTTP server settings.
type ServerConfig struct {
	HTTPAddr          string        `mapstructure:"http_addr"`
	GinMode           string        `mapstructure:"gin_mode"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderMB       int           `mapstructure:"max_header_mb"`
	MaxBodyMB         int64         `mapstructure:"max_body_mb"`
}

// MaxHeaderBytes converts the configured header limit from MB to bytes for net/http.
func (cfg ServerConfig) MaxHeaderBytes() int {
	return cfg.MaxHeaderMB * bytesPerMB
}

// MaxBodyBytes converts the configured request body limit from MB to bytes for middleware.
func (cfg ServerConfig) MaxBodyBytes() int64 {
	return cfg.MaxBodyMB * bytesPerMB
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
			HTTPAddr:          ":8000",
			GinMode:           "debug",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderMB:       1,
			MaxBodyMB:         1,
		},
		DB: db.Config{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		},
		Log: logging.LogConfig{
			Format: "text",
			Level:  "info",
		},
		OTel: otel.Config{
			Enabled: true,
		},
		CORS: CORSConfig{
			AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
			ExposeHeaders: []string{"X-Request-ID"},
			MaxAge:        12 * time.Hour,
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

	for name, value := range map[string]time.Duration{
		"server.read_header_timeout": cfg.Server.ReadHeaderTimeout,
		"server.read_timeout":        cfg.Server.ReadTimeout,
		"server.write_timeout":       cfg.Server.WriteTimeout,
		"server.idle_timeout":        cfg.Server.IdleTimeout,
	} {
		if value < 0 {
			return fmt.Errorf("%s must be non-negative", name)
		}
	}
	if cfg.Server.MaxHeaderMB < 0 {
		return fmt.Errorf("server.max_header_mb must be non-negative")
	}
	if cfg.Server.MaxHeaderMB > int(^uint(0)>>1)/bytesPerMB {
		return fmt.Errorf("server.max_header_mb is too large")
	}
	if cfg.Server.MaxBodyMB < 0 {
		return fmt.Errorf("server.max_body_mb must be non-negative")
	}
	if cfg.Server.MaxBodyMB > int64(^uint64(0)>>1)/bytesPerMB {
		return fmt.Errorf("server.max_body_mb is too large")
	}
	if err := validateCORS(cfg.CORS); err != nil {
		return err
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

	if cfg.DB.Driver != "" {
		switch strings.ToLower(cfg.DB.Driver) {
		case "postgres", "postgresql", "pg", "mysql", "sqlite", "sqlite3":
			if strings.TrimSpace(cfg.DB.DSN) == "" {
				return fmt.Errorf("db.driver=%q but db.dsn is empty", cfg.DB.Driver)
			}
		default:
			return fmt.Errorf("unsupported db.driver %q (want postgres|mysql|sqlite)", cfg.DB.Driver)
		}
	}
	return nil
}

// validateCORS rejects unsafe or incomplete cross-origin request policies.
func validateCORS(cfg CORSConfig) error {
	if cfg.MaxAge < 0 {
		return fmt.Errorf("cors.max_age must be non-negative")
	}
	for _, origin := range cfg.AllowOrigins {
		if cfg.AllowCredentials && strings.TrimSpace(origin) == "*" {
			return fmt.Errorf("cors.allow_credentials cannot be true when cors.allow_origins contains *")
		}
	}
	if !cfg.Enabled {
		return nil
	}
	if len(cfg.AllowOrigins) == 0 {
		return fmt.Errorf("cors.allow_origins must contain at least one origin when cors.enabled is true")
	}
	for _, origin := range cfg.AllowOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return fmt.Errorf("cors.allow_origins must not contain an empty origin")
		}
		if origin != "*" && !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("invalid cors.allow_origins value %q (want http://, https://, or *)", origin)
		}
	}
	return nil
}
