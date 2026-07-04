// Package main is the server entrypoint: load config, init logging, init otel,
// wire gin + handler, serve HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/db"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
	"github.com/piwriw/oas-go-template/internal/otel"
	"github.com/piwriw/oas-go-template/internal/version"
)

const serviceName = "oas-go-template"

func main() {
	configPath := flag.String("c", "config.yaml", "path to config file")
	flag.Parse()

	// run() returns nil on graceful shutdown, non-nil on any init or runtime
	// failure. Going through os.Exit (not panic) keeps deferred shutdowns
	// running on the way out.
	if err := run(*configPath); err != nil {
		slog.Error("server exiting", "err", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		// Logger isn't initialized yet — stdlib default slog will have to do.
		slog.Error("config load failed", "path", configPath, "err", err)
		return err
	}

	logger := logging.New(cfg.Log)
	slog.SetDefault(logger)

	gin.SetMode(cfg.Server.GinMode)

	// OTel init before signal setup so a failure exits cleanly without orphan defers.
	otelShutdown, err := otel.Init(context.Background(), cfg.OTel, serviceName, version.Version)
	if err != nil {
		slog.Error("otel init failed", "err", err)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	gdb, err := db.Init(ctx, cfg.DB)
	if err != nil {
		slog.Error("db init failed", "err", err)
		shutdownOTel(otelShutdown)
		return err
	}

	defer func() {
		if err := db.Close(gdb); err != nil {
			slog.Error("db shutdown error", "err", err)
		}
		shutdownOTel(otelShutdown)
	}()

	srv := newHTTPServer(cfg, gdb)
	return serveAndWait(ctx, srv, stop)
}

// newHTTPServer wires the gin router (recovery + otelgin + logging middleware),
// registers the strict API handler, and wraps it in an *http.Server with a
// read-header timeout to defuse slowloris-style attacks.
func newHTTPServer(cfg *config.Config, gdb *gorm.DB) *http.Server {
	h := handler.New(gdb)
	strictHandler := api.NewStrictHandler(h, nil)

	r := gin.New()
	// otelgin must run before logging so logging.Middleware can read the active span
	// from c.Request.Context() and inject trace_id into the log line.
	r.Use(gin.Recovery(), otelgin.Middleware(serviceName), logging.Middleware())
	api.RegisterHandlers(r, strictHandler)

	return &http.Server{
		Addr:              cfg.Server.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// serveAndWait starts srv in a goroutine and blocks until either ctx is
// canceled (signal) or ListenAndServe returns a non-ErrServerClosed error
// (crash). Either way it then runs a 10s-bounded Shutdown. Returns the serve
// error (nil on graceful signal-driven shutdown).
func serveAndWait(ctx context.Context, srv *http.Server, stop context.CancelFunc) error {
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr, "version", version.Version)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Trigger ctx cancel so the main goroutine runs Shutdown / Close in
			// order instead of panicking past defers.
			serverErr <- err
			stop()
		}
	}()

	var serveErr error
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining...")
	case serveErr = <-serverErr:
		slog.Error("server crashed", "err", serveErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "err", err)
	}

	return serveErr
}

// shutdownOTel runs the otel shutdown func with a fresh 10s timeout when one
// was installed. Safe to call with nil.
func shutdownOTel(shutdown func(context.Context) error) {
	if shutdown == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		slog.Error("otel shutdown error", "err", err)
	}
}
