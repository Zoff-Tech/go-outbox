package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"go.opentelemetry.io/otel"
)

func TestInit_Success(t *testing.T) {
	// Mock observability configuration
	cfg := config.Observability{
		ServiceName: "test-service",
		TracingURL:  "localhost:4318", // Mock OTLP endpoint
	}

	// Call Init and ensure no errors occur
	shutdown, err := Init(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// Ensure the global tracer provider is set
	tp := otel.GetTracerProvider()
	assert.NotNil(t, tp)

	// Shutdown telemetry and ensure no errors occur
	shutdown()
}

func TestInit_InvalidTracingURL(t *testing.T) {
	// Mock observability configuration with an invalid TracingURL
	cfg := config.Observability{
		ServiceName: "test-service",
		TracingURL:  "", // Invalid endpoint
	}

	// Call Init and ensure it returns an error
	shutdown, err := Init(cfg)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
}

func TestInit_EmptyServiceName(t *testing.T) {
	// Mock observability configuration with an empty ServiceName
	cfg := config.Observability{
		ServiceName: "",
		TracingURL:  "localhost:4318",
	}

	// Call Init and ensure it returns an error
	shutdown, err := Init(cfg)
	assert.Error(t, err)
	assert.Nil(t, shutdown)
}

func TestInit_Shutdown(t *testing.T) {
	// Mock observability configuration
	cfg := config.Observability{
		ServiceName: "test-service",
		TracingURL:  "localhost:4318",
	}

	// Call Init and ensure no errors occur
	shutdown, err := Init(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	// Call the shutdown function and ensure it completes without errors
	shutdown()
}
