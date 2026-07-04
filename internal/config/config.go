// Package config loads runtime configuration from environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the server.
type Config struct {
	HTTPAddr string
	GinMode  string
}

// NewFromEnv reads configuration from environment variables.
func NewFromEnv() (*Config, error) {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = "debug"
	}

	if err := validateMode(mode); err != nil {
		return nil, err
	}

	return &Config{HTTPAddr: addr, GinMode: mode}, nil
}

func validateMode(mode string) error {
	switch mode {
	case "debug", "release", "test":
		return nil
	default:
		return fmt.Errorf("invalid GIN_MODE %q (want debug|release|test)", mode)
	}
}

// EnvBool helper for downstream code (used by middleware, etc.).
func EnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
