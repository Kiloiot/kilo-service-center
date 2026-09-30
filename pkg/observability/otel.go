// Package observability provides shared OpenTelemetry bootstrap for KiloCenter services.
package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Error prose for observability bootstrap failures.
const (
	errTextTraceExporterCreate = "failed to create OTLP trace exporter"
	errTextResourceCreate      = "failed to create OTel resource"
)

// defaultMetricsPath is the metrics endpoint path used when the configuration
// provides none.
const defaultMetricsPath = "/metrics"

// metricsReadHeaderTimeout bounds how long the metrics server waits for
// request headers.
const metricsReadHeaderTimeout = 5 * time.Second

// TracingConfig mirrors the monitoring config fields relevant to tracing.
type TracingConfig struct {
	Enabled    bool
	Endpoint   string
	SampleRate float64
}

// MetricsConfig mirrors the monitoring config fields relevant to metrics.
type MetricsConfig struct {
	Enabled bool
	Port    int
	Path    string
}

const msgFmtMetricsServerError = "metrics server error on port %d: %v\n"

// InitTracing sets up the OTel trace provider with OTLP gRPC export.
// Returns a shutdown function and any initialization error.
// If tracing is disabled, returns a no-op shutdown function.
// msgFmtMetricsServerError reports a non-fatal metrics listener failure.
func InitTracing(ctx context.Context, cfg TracingConfig, serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if !cfg.Enabled {
		return noop, nil
	}

	opts := []otlptracegrpc.Option{}
	if cfg.Endpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
	}
	opts = append(opts, otlptracegrpc.WithInsecure())

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return noop, fmt.Errorf("%s: %w", errTextTraceExporterCreate, err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return noop, fmt.Errorf("%s: %w", errTextResourceCreate, err)
	}

	sampler := sdktrace.AlwaysSample()
	if cfg.SampleRate > 0 && cfg.SampleRate < 1 {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// InitMetrics starts a Prometheus HTTP metrics server on the given port.
// Returns a shutdown function and any initialization error.
// If metrics are disabled, returns a no-op shutdown function.
func InitMetrics(_ context.Context, cfg MetricsConfig, _ string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if !cfg.Enabled {
		return noop, nil
	}

	path := cfg.Path
	if path == "" {
		path = defaultMetricsPath
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: metricsReadHeaderTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Metrics server failure is non-fatal
			fmt.Printf(msgFmtMetricsServerError, cfg.Port, err)
		}
	}()

	return func(ctx context.Context) error {
		return server.Shutdown(ctx)
	}, nil
}
