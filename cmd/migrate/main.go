// Package main runs embedded database migrations without starting the HTTP server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/db"
	"github.com/piwriw/oas-go-template/internal/logging"
)

// main parses migration arguments and exits nonzero when the operation fails.
func main() {
	if err := run(); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}
}

// run loads database configuration and runs one up or down migration operation.
func run() error {
	configPath := flag.String("c", "config.yaml", "path to config file")
	flag.Parse()

	if flag.NArg() != 1 {
		return errors.New("usage: migrate [-c config.yaml] up|down")
	}

	var direction db.MigrationDirection
	switch command := flag.Arg(0); command {
	case string(db.MigrationUp):
		direction = db.MigrationUp
	case string(db.MigrationDown):
		direction = db.MigrationDown
	default:
		return fmt.Errorf("unknown migration command %q (want up|down)", command)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	slog.SetDefault(logging.New(cfg.Log))
	if cfg.DB.Disabled() {
		return errors.New("db.driver is empty; configure a database to run migrations")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return db.Migrate(ctx, nil, cfg.DB, direction)
}
