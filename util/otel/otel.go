package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	log = util.NewLogger("otel")
)

// Config holds OpenTelemetry configuration
type Config struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Protocol string `json:"protocol"` // "grpc" or "http"
	Insecure bool   `json:"insecure"`
}

// Init initializes OpenTelemetry tracing
func Init(ctx context.Context, cfg Config) error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.Endpoint == "" {
		return fmt.Errorf("otel endpoint is required when enabled")
	}

	var exporter sdktrace.SpanExporter
	var err error

	// Default to grpc if protocol not specified
	protocol := cfg.Protocol
	switch protocol {
	case "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	default:
		protocol = "grpc"
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	}

	if err != nil {
		return fmt.Errorf("failed to create otel exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("evcc"),
			semconv.ServiceVersion(util.FormattedVersion()),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.INFO.Printf("OpenTelemetry tracing enabled: endpoint=%s, protocol=%s", cfg.Endpoint, protocol)
	return nil
}

// Shutdown gracefully shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	tp := otel.GetTracerProvider()
	if tp == nil {
		return nil
	}

	// Check if it's an SDK tracer provider that supports Shutdown
	sdkTracerProvider, ok := tp.(*sdktrace.TracerProvider)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sdkTracerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown otel tracer provider: %w", err)
	}

	log.INFO.Println("OpenTelemetry tracing shutdown complete")
	return nil
}
