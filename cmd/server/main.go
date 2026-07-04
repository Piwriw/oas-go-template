// Package main is the server entrypoint: load config, init logging, init otel,
// wire gin + handler, serve HTTP.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
	"github.com/piwriw/oas-go-template/internal/otel"
	"github.com/piwriw/oas-go-template/internal/version"
)

const serviceName = "oas-go-template"

func main() {
	cfg, err := config.NewFromEnv()
	if err != nil {
		// Use stdlib default logger here because ours isn't initialized yet.
		slog.Error("config load failed", "err", err)
		panic(err)
	}

	logger := logging.New()
	slog.SetDefault(logger)

	gin.SetMode(cfg.GinMode)

	// OTel init before signal setup so a failure exits cleanly without orphan defers.
	otelCtx := context.Background()
	otelShutdown, err := otel.Init(otelCtx, serviceName, version.Version)
	if err != nil {
		slog.Error("otel init failed", "err", err)
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	h := handler.New()
	strictHandler := api.NewStrictHandler(h, nil)

	r := gin.New()
	// otelgin must run before logging so logging.Middleware can read the active span
	// from c.Request.Context() and inject trace_id into the log line.
	r.Use(gin.Recovery(), otelgin.Middleware(serviceName), logging.Middleware())
	api.RegisterHandlers(r, strictHandler)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr, "mode", cfg.GinMode, "version", version.Version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server crashed", "err", err)
			panic(err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, draining...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "err", err)
	}
	if otelShutdown != nil {
		if err := otelShutdown(shutdownCtx); err != nil {
			slog.Error("otel shutdown error", "err", err)
		}
	}
}
