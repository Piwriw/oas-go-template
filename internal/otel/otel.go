// Package otel initializes the OpenTelemetry SDK with OTLP HTTP exporters.
//
// Config is loaded from config.yaml by internal/config. When Enabled is false,
// Init returns (nil, nil) and the caller may skip shutdown.
package otel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// Config drives OTel SDK setup. Loaded by internal/config.
type Config struct {
	// Enabled=false skips SDK init entirely (no exporters, no providers).
	Enabled bool `mapstructure:"enabled"`

	// ExporterOTLPEndpoint is the OTLP HTTP endpoint (e.g. http://localhost:4318).
	// Parsed into host+scheme and forwarded to the OTLP HTTP exporters.
	ExporterOTLPEndpoint string `mapstructure:"exporter_otlp_endpoint"`

	// ServiceName defaults to the value passed by the caller (usually main);
	// set this only if you want to override at the config layer.
	ServiceName string `mapstructure:"service_name"`

	// ServiceVersion defaults to the value passed by the caller.
	ServiceVersion string `mapstructure:"service_version"`
}

// Init configures the global TracerProvider and MeterProvider with OTLP HTTP
// exporters. name and version fall back to cfg.ServiceName / cfg.ServiceVersion
// when set, otherwise the caller-provided values. Returns a shutdown func;
// (nil, nil) means OTel is disabled.
func Init(ctx context.Context, cfg Config, defaultName, defaultVersion string) (func(context.Context) error, error) {
	if !cfg.Enabled {
		slog.Info("otel: disabled via config, skipping init")
		return nil, nil
	}

	name := cfg.ServiceName
	if strings.TrimSpace(name) == "" {
		name = defaultName
	}
	version := cfg.ServiceVersion
	if strings.TrimSpace(version) == "" {
		version = defaultVersion
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithTelemetrySDK(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(name),
			semconv.ServiceVersionKey.String(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	traceOpts, metricOpts, err := exporterOptions(cfg.ExporterOTLPEndpoint)
	if err != nil {
		return nil, fmt.Errorf("otel exporter endpoint: %w", err)
	}

	traceExp, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("otel trace exporter: %w", err)
	}
	metricExp, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		// traceExp holds HTTP connections; shut it down so it doesn't leak.
		if shutErr := traceExp.Shutdown(ctx); shutErr != nil {
			return nil, fmt.Errorf("otel metric exporter: %w (also trace shutdown: %v)", err, shutErr)
		}
		return nil, fmt.Errorf("otel metric exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExp),
		trace.WithResource(res),
	)

	// Prometheus exporter exposes OTel-collected metrics on
	// prometheus.DefaultRegisterer, which /metrics serves via
	// promhttp.Handler(). Sits alongside the OTLP periodic reader — push
	// to a collector for high-fidelity tracing, pull from Prometheus for
	// ops dashboards.
	promExp, err := otelprom.New()
	if err != nil {
		if shutErr := traceExp.Shutdown(ctx); shutErr != nil {
			return nil, fmt.Errorf("otel prometheus exporter: %w (also trace shutdown: %v)", err, shutErr)
		}
		return nil, fmt.Errorf("otel prometheus exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExp)),
		metric.WithReader(promExp),
		metric.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
		return errors.Join(errs...)
	}, nil
}

// endpointConfig holds the parsed result of an OTLP HTTP URL.
type endpointConfig struct {
	host     string
	insecure bool
}

// exporterOptions turns an OTLP HTTP URL like "http://localhost:4318" into the
// trace and metric exporter option slices. Empty input returns no options,
// letting the SDK fall back to its built-in default (https://localhost:4318).
func exporterOptions(endpoint string) ([]otlptracehttp.Option, []otlpmetrichttp.Option, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, nil, nil
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %q: %w", endpoint, err)
	}
	if u.Host == "" {
		return nil, nil, fmt.Errorf("endpoint %q missing host", endpoint)
	}

	ec := endpointConfig{host: u.Host}
	switch strings.ToLower(u.Scheme) {
	case "http":
		ec.insecure = true
	case "https", "":
		// keep TLS
	default:
		return nil, nil, fmt.Errorf("unsupported scheme %q in endpoint %q", u.Scheme, endpoint)
	}

	var traceOpts []otlptracehttp.Option
	var metricOpts []otlpmetrichttp.Option
	traceOpts = append(traceOpts, otlptracehttp.WithEndpoint(ec.host))
	metricOpts = append(metricOpts, otlpmetrichttp.WithEndpoint(ec.host))
	if ec.insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}
	return traceOpts, metricOpts, nil
}
