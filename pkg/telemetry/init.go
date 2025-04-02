package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/zoff-tech/go-outbox/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

// Init initializes telemetry (tracing and metrics) and returns a shutdown function.
func Init(cfg config.Observability) (func(), error) {
	// Validate configuration
	if cfg.ServiceName == "" {
		return nil, errors.New("service name cannot be empty")
	}
	if cfg.TracingURL == "" {
		return nil, errors.New("tracing URL cannot be empty")
	}

	// Create an OTLP trace exporter
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.TracingURL),
		otlptracehttp.WithInsecure(),
	)
	traceExporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create a resource to describe the service
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create a TracerProvider with the exporter and resource
	tp := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Return a shutdown function to clean up resources
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}
