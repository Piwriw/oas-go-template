// Package main is the server entrypoint: load config, init logging, init otel,
// wire gin + handler, serve HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"

	internalapi "github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/db"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/logging"
	"github.com/piwriw/oas-go-template/internal/middleware"
	oascontract "github.com/piwriw/oas-go-template/internal/oas"
	"github.com/piwriw/oas-go-template/internal/otel"
	"github.com/piwriw/oas-go-template/internal/version"
	specapi "github.com/piwriw/oas-go-template/pkg/api"
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

	drainState := handler.NewDrainState(cfg.Server.DrainTimeout)
	srv := newHTTPServer(cfg, gdb, drainState)
	return serveAndWait(ctx, srv, drainState)
}

// newHTTPServer wires the gin router (recovery + otelgin + logging + optional
// CORS + request body limit), validates API requests against the embedded OAS
// document, registers the strict API handler, mounts the Prometheus /metrics
// route, and applies HTTP timeouts and header limits to defuse common resource
// attacks. The optional DrainState makes readiness fail before shutdown.
//
// The OTel Prometheus exporter (when OTel is enabled in cfg.OTel.Enabled)
// and client_golang's built-in Go/process collectors both feed
// prometheus.DefaultRegisterer — /metrics reads from there via
// promhttp.Handler. No explicit collector registration needed.
func newHTTPServer(cfg *config.Config, gdb *gorm.DB, drainStates ...*handler.DrainState) *http.Server {
	drainState := handler.NewDrainState(cfg.Server.DrainTimeout)
	if len(drainStates) > 0 && drainStates[0] != nil {
		drainState = drainStates[0]
	}
	h := handler.New(gdb, drainState)
	strictHandler := internalapi.NewStrictHandlerWithOptions(h, nil, handler.StrictServerOptions())
	swaggerSpec := openAPISpec()

	r := gin.New()
	r.HandleMethodNotAllowed = true
	middleware.Use(r, middleware.Options{
		ServiceName:  serviceName,
		MaxBodyBytes: cfg.Server.MaxBodyBytes,
		CORS:         cfg.CORS,
		OpenAPISpec:  swaggerSpec,
	})
	r.NoRoute(handler.NoRoute)
	r.NoMethod(handler.NoMethod)

	// /metrics is intentionally NOT in spec/openapi.yaml and not configurable —
	// it's an ops endpoint, not part of the API contract, and there's no good
	// reason to disable it. The client SDK doesn't carry a useless
	// GetMetricsWithResponses method.
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Keep the operational /metrics route outside the OAS validator because it
	// is intentionally not part of the public API contract.
	apiRoutes := r.Group("", openAPIValidator(swaggerSpec))
	internalapi.RegisterHandlers(apiRoutes, strictHandler)

	return &http.Server{
		Addr:              cfg.Server.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}
}

// openAPISpec loads the generated embedded contract once per server build.
// Server URLs in the document describe deployment endpoints; they should not
// make local host validation reject otherwise valid requests.
func openAPISpec() (swaggerSpec *openapi3.T) {
	swaggerSpec, err := specapi.GetSpec()
	if err != nil {
		panic(fmt.Sprintf("load embedded OpenAPI spec: %v", err))
	}
	if err := oascontract.Validate(swaggerSpec); err != nil {
		panic(fmt.Sprintf("validate embedded OpenAPI contract: %v", err))
	}
	swaggerSpec.Servers = nil
	return swaggerSpec
}

func openAPIValidator(swaggerSpec *openapi3.T) gin.HandlerFunc {
	return ginmiddleware.OapiRequestValidatorWithOptions(swaggerSpec, &ginmiddleware.Options{
		ErrorHandler:          handler.OAPIValidationError,
		SilenceServersWarning: true,
	})
}

// serveAndWait opens the listener before entering the signal wait so bind and
// startup failures are returned directly instead of racing with cancellation.
func serveAndWait(ctx context.Context, srv *http.Server, drainStates ...*handler.DrainState) error {
	return serveAndWaitWithListener(ctx, srv, net.Listen, drainStates...)
}

// serveAndWaitWithListener starts srv and blocks until either ctx is canceled
// or Serve exits. On a signal it marks readiness as draining, waits for the
// configured endpoint-removal window, and then runs a 10s-bounded Shutdown.
// listen is injectable so startup failures can be tested without port races.
func serveAndWaitWithListener(
	ctx context.Context,
	srv *http.Server,
	listen func(network, address string) (net.Listener, error),
	drainStates ...*handler.DrainState,
) error {
	var drainState *handler.DrainState
	if len(drainStates) > 0 {
		drainState = drainStates[0]
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	listener, err := listen("tcp", addr)
	if err != nil {
		slog.Error("server listen failed", "addr", addr, "err", err)
		return err
	}

	serverErr := make(chan error, 1)
	slog.Info("server listening", "addr", listener.Addr().String(), "version", version.Version)
	go func() {
		serverErr <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining...")
		if drainState != nil {
			drainState.Begin()
			if timeout := drainState.Timeout(); timeout > 0 {
				time.Sleep(timeout)
			}
		}
	case serveErr := <-serverErr:
		if serveErr == nil || errors.Is(serveErr, http.ErrServerClosed) {
			return nil
		}
		slog.Error("server crashed", "err", serveErr)
		return serveErr
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "err", err)
	}
	serveErr := <-serverErr
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		slog.Error("server stopped with error during shutdown", "err", serveErr)
		return serveErr
	}

	return nil
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
