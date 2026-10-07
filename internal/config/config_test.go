package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/piwriw/oas-go-template/internal/db"
)

// writeFile creates a temporary YAML configuration fixture and returns its path.
func writeFile(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// TestLoad_fullYAML verifies every supported YAML section overrides its runtime default.
func TestLoad_fullYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `
server:
  http_addr: ":9999"
  gin_mode: release
db:
  driver: postgres
  host: "::1"
  port: 6543
  user: "app user"
  password: "p@ss:/?#&%= space"
  database: "app/db?name"
  ssl_mode: verify-full
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h
log:
  format: json
  level: debug
  caller: false
otel:
  enabled: false
  exporter_otlp_endpoint: "http://collector:4318"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr = %q, want :9999", cfg.Server.HTTPAddr)
	}
	if cfg.Server.GinMode != "release" {
		t.Errorf("GinMode = %q", cfg.Server.GinMode)
	}
	wantDB := db.Config{
		Driver:          db.DriverPostgres,
		Host:            "::1",
		Port:            6543,
		User:            "app user",
		Password:        "p@ss:/?#&%= space",
		Database:        "app/db?name",
		SSLMode:         "verify-full",
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
	}
	if cfg.DB != wantDB {
		t.Error("database fields did not preserve explicit YAML values")
	}
	if cfg.Log.Format != "json" || cfg.Log.Level != "debug" || cfg.Log.Caller {
		t.Errorf("Log = %+v", cfg.Log)
	}
	if cfg.OTel.Enabled {
		t.Errorf("OTel.Enabled = true, want false")
	}
	if cfg.OTel.ExporterOTLPEndpoint != "http://collector:4318" {
		t.Errorf("OTLPEndpoint = %q", cfg.OTel.ExporterOTLPEndpoint)
	}
}

// TestLoad_missingFileFallsBackToDefaults verifies absent explicit configuration retains operational defaults.
func TestLoad_missingFileFallsBackToDefaults(t *testing.T) {
	// Any path that doesn't exist → no error, defaults returned so dev/test
	// workflows don't need to author a config file.
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("missing file should fall back to defaults, got: %v", err)
	}
	if cfg.Server.HTTPAddr != ":8000" {
		t.Errorf("default HTTPAddr = %q, want :8000", cfg.Server.HTTPAddr)
	}
	if !cfg.OTel.Enabled {
		t.Errorf("default OTel.Enabled should be true")
	}
	if !cfg.DB.Disabled() || cfg.DB.Host != "localhost" || cfg.DB.Port != 0 || cfg.DB.SSLMode != "prefer" {
		t.Error("missing configuration should preserve disabled DB and connection defaults")
	}
}

// TestLoad_defaultPathMissingOK verifies a missing conventional config path does not prevent local startup.
func TestLoad_defaultPathMissingOK(t *testing.T) {
	// Switch into a temp dir so the default "config.yaml" doesn't exist.
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load("config.yaml")
	if err != nil {
		t.Fatalf("Load default config.yaml in empty dir: %v", err)
	}
	if cfg.Server.HTTPAddr != ":8000" {
		t.Errorf("default HTTPAddr = %q, want :8000", cfg.Server.HTTPAddr)
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("default GinMode = %q", cfg.Server.GinMode)
	}
	if !cfg.OTel.Enabled {
		t.Errorf("default OTel.Enabled should be true")
	}
}

// TestLoad_invalidGinMode verifies unsupported Gin modes fail configuration validation.
func TestLoad_invalidGinMode(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `
server:
  gin_mode: bogus
`)
	if _, err := Load(p); err == nil {
		t.Fatal("expected validation error for invalid gin_mode")
	}
}

// TestLoad_databaseDefaults verifies each database dialect resolves omitted connection fields correctly.
func TestLoad_databaseDefaults(t *testing.T) {
	for _, tt := range []struct {
		name   string
		yaml   string
		driver string
		port   int
	}{
		{"postgres", "db:\n  driver: postgres\n  user: app\n  database: app\n", "postgres", 5432},
		{"mysql", "db:\n  driver: mysql\n  user: app\n  database: app\n", "mysql", 3306},
		{"sqlite", "db:\n  driver: sqlite\n  database: ':memory:'\n", "sqlite", 0},
		{"postgres_alias", "db:\n  driver: PG\n  user: app\n  database: app\n", "postgres", 5432},
		{"sqlite_alias", "db:\n  driver: sqlite3\n  database: app.db\n", "sqlite", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(writeFile(t, t.TempDir(), tt.yaml))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.DB.Driver != tt.driver || cfg.DB.Host != "localhost" || cfg.DB.Port != tt.port || cfg.DB.SSLMode != "prefer" {
				t.Error("unexpected driver-specific connection defaults")
			}
			if cfg.DB.Password != "" {
				t.Error("an omitted password should remain empty")
			}
		})
	}
}

// TestLoad_invalidDatabaseSettings rejects enabled databases with incomplete or invalid connection fields.
func TestLoad_invalidDatabaseSettings(t *testing.T) {
	for _, tt := range []struct {
		name string
		yaml string
		want string
	}{
		{"missing_database", "db:\n  driver: postgres\n  user: app\n", "db.database"},
		{"missing_sqlite_path", "db:\n  driver: sqlite\n", "db.database"},
		{"missing_user", "db:\n  driver: mysql\n  database: app\n", "db.user"},
		{"empty_host", "db:\n  driver: postgres\n  host: ''\n  user: app\n  database: app\n", "db.host"},
		{"negative_port", "db:\n  driver: mysql\n  port: -1\n  user: app\n  database: app\n", "db.port"},
		{"port_too_large", "db:\n  driver: postgres\n  port: 65536\n  user: app\n  database: app\n", "db.port"},
		{"invalid_ssl_mode", "db:\n  driver: postgres\n  user: app\n  database: app\n  ssl_mode: invalid\n", "db.ssl_mode"},
		{"unsupported_driver", "db:\n  driver: oracle\n  user: app\n  database: app\n", "db.driver"},
		{"legacy_dsn", "db:\n  driver: postgres\n  dsn: 'host=localhost user=app dbname=app'\n", "db.database"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeFile(t, t.TempDir(), tt.yaml))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load error = %v, want an error about %s", err, tt.want)
			}
		})
	}
}

// TestLoad_invalidLogFormat verifies unsupported structured log encodings are rejected.
func TestLoad_invalidLogFormat(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `
log:
  format: xml
`)
	if _, err := Load(p); err == nil {
		t.Fatal("expected validation error for invalid log.format")
	}
}

// TestLoad_partialYAMLPreservesDefaults verifies omitted YAML fields retain built-in runtime values.
func TestLoad_partialYAMLPreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	// Only set server.http_addr; everything else relies on built-in defaults.
	p := writeFile(t, dir, `
server:
  http_addr: ":9090"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want :9090", cfg.Server.HTTPAddr)
	}
	if cfg.Server.GinMode != "debug" {
		t.Errorf("GinMode default dropped: got %q", cfg.Server.GinMode)
	}
	if cfg.DB.MaxOpenConns != 25 {
		t.Errorf("DB.MaxOpenConns default dropped: got %d", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 5 {
		t.Errorf("DB.MaxIdleConns default dropped: got %d", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("DB.ConnMaxLifetime default dropped: got %v", cfg.DB.ConnMaxLifetime)
	}
	if cfg.Log.Format != "text" || cfg.Log.Level != "info" {
		t.Errorf("Log defaults dropped: got %+v", cfg.Log)
	}
	if !cfg.Log.Caller {
		t.Errorf("Log.Caller default should be true")
	}
	if !cfg.OTel.Enabled {
		t.Errorf("OTel.Enabled default should be true")
	}
}
