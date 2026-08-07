package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

// TestLoad_corsYAML verifies cross-origin policy fields decode from YAML.
func TestLoad_corsYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, `
cors:
  enabled: true
  allow_origins: ["https://app.example.com"]
  allow_methods: [GET, OPTIONS]
  allow_headers: [Content-Type, X-Request-ID]
  expose_headers: [X-Request-ID]
  allow_credentials: true
  max_age: 1h
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CORS.Enabled || len(cfg.CORS.AllowOrigins) != 1 || cfg.CORS.AllowOrigins[0] != "https://app.example.com" {
		t.Errorf("CORS origins = %+v", cfg.CORS.AllowOrigins)
	}
	if cfg.CORS.MaxAge != time.Hour || !cfg.CORS.AllowCredentials {
		t.Errorf("CORS = %+v", cfg.CORS)
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
	if cfg.CORS.Enabled {
		t.Errorf("default CORS.Enabled should be false")
	}
	if cfg.CORS.MaxAge != 12*time.Hour {
		t.Errorf("default CORS.MaxAge = %v, want 12h", cfg.CORS.MaxAge)
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

// TestLoad_dbDriverWithoutDSN verifies enabled database configurations require connection details.
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

// TestLoad_rejectsInvalidCORSConfig verifies unsafe or incomplete cross-origin policies are rejected.
func TestLoad_rejectsInvalidCORSConfig(t *testing.T) {
	tests := map[string]string{
		"enabled without origins": `cors:
  enabled: true
`,
		"credentials with wildcard": `cors:
  enabled: true
  allow_origins: ["*"]
  allow_credentials: true
`,
		"invalid origin": `cors:
  enabled: true
  allow_origins: ["app.example.com"]
`,
		"negative max age": `cors:
  max_age: -1s
`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if _, err := Load(writeFile(t, dir, body)); err == nil {
				t.Fatal("expected CORS validation error")
			}
		})
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
	if !cfg.OTel.Enabled {
		t.Errorf("OTel.Enabled default should be true")
	}
}
