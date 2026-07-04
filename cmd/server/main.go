// Package main is the server entrypoint: load config, init otel, wire gin + handler, serve HTTP.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/middleware"
	"github.com/piwriw/oas-go-template/internal/otel"
	"github.com/piwriw/oas-go-template/internal/version"
)

const serviceName = "oas-go-template"

func main() {
	cfg, err := config.NewFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	gin.SetMode(cfg.GinMode)

	// OTel init before signal setup so a failure exits cleanly without orphan defers.
	otelCtx := context.Background()
	otelShutdown, err := otel.Init(otelCtx, serviceName, version.Version)
	if err != nil {
		log.Fatalf("otel init: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	h := handler.New()
	strictHandler := api.NewStrictHandler(h, nil)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger())
	r.Use(otelgin.Middleware(serviceName))
	api.RegisterHandlers(r, strictHandler)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s (mode=%s)", cfg.HTTPAddr, cfg.GinMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutdown signal received, draining...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	if otelShutdown != nil {
		if err := otelShutdown(shutdownCtx); err != nil {
			log.Printf("otel shutdown: %v", err)
		}
	}
}
