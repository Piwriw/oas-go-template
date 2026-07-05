package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestLoad_fullYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `
server:
  http_addr: ":9999"
  gin_mode: release
db:
  driver: postgres
  dsn: "host=localhost dbname=app"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h
log:
  format: json
  level: debug
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
	if cfg.DB.Driver != "postgres" {
		t.Errorf("DBDriver = %q", cfg.DB.Driver)
	}
	if cfg.DB.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %d", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.ConnMaxLifetime != time.Hour {
		t.Errorf("ConnMaxLifetime = %v", cfg.DB.ConnMaxLifetime)
	}
	if cfg.Log.Format != "json" || cfg.Log.Level != "debug" {
		t.Errorf("Log = %+v", cfg.Log)
	}
	if cfg.OTel.Enabled {
		t.Errorf("OTel.Enabled = true, want false")
	}
	if cfg.OTel.ExporterOTLPEndpoint != "http://collector:4318" {
		t.Errorf("OTLPEndpoint = %q", cfg.OTel.ExporterOTLPEndpoint)
	}
}

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
}

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

func TestLoad_dbDriverWithoutDSN(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `
db:
  driver: postgres
  dsn: ""
`)
	if _, err := Load(p); err == nil {
		t.Fatal("expected validation error when driver set but dsn empty")
	}
}

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

// viper.Unmarshal zero-fills fields that exist in the struct but aren't in
// the YAML. This test pins the contract: if a yaml is missing a nested field,
// the default for that field is preserved.
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
	if !cfg.OTel.Enabled {
		t.Errorf("OTel.Enabled default should be true")
	}
}
