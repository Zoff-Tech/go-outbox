package main

import (
	"context"
	"log"

	"github.com/zoff-tech/go-outbox/pkg/broker"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"github.com/zoff-tech/go-outbox/pkg/processor"
	"github.com/zoff-tech/go-outbox/pkg/store"
	"github.com/zoff-tech/go-outbox/pkg/telemetry"
)

func main() {
	ctx := context.Background()

	// Load configuration from file or environment
	cfg, err := config.LoadFromFile("./cmd/outbox-sidecar")
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	// Validate the configuration
	err = cfg.Validate()
	if err != nil {
		log.Fatal("Invalid configuration: ", err)
	}

	// Initialize telemetry (tracing and metrics)
	shutdownTelemetry, err := telemetry.Init(cfg.Observability)
	if err != nil {
		log.Fatal("Failed to initialize telemetry: ", err)
	}
	defer shutdownTelemetry() // Ensure telemetry is properly shut down on exit

	// Initialize the repository
	repo, err := store.NewRepository(ctx, cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize repository: ", err)
	}

	// Initialize the message broker
	broker, err := broker.NewBroker(ctx, &cfg.Broker)
	if err != nil {
		log.Fatal("Failed to initialize broker: ", err)
	}

	// Create the outbox processor
	processor := processor.NewOutboxProcessor(repo, broker, cfg)

	// Run the processor (blocks until context is canceled or an error occurs)
	processor.ProcessEvents(ctx)
}
