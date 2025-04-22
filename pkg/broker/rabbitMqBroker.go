package broker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"maps"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/zoff-tech/go-outbox/pkg/config"
	"github.com/zoff-tech/go-outbox/pkg/store"
)

type RabbitMQBrokerCreator func(ctx context.Context, settings *config.BrokerSettings) (MessageBroker, error)

var NewRabbitMqBroker RabbitMQBrokerCreator = func(ctx context.Context, settings *config.BrokerSettings) (MessageBroker, error) {
	if settings.PoolSize <= 0 {
		return nil, errors.New("poolSize must be greater than 0")
	}

	conn, err := amqp.Dial(settings.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	// Set up a channel to handle connection close notifications
	notifyClose := make(chan *amqp.Error)
	conn.NotifyClose(notifyClose)
	go func() {
		for err := range notifyClose {
			log.Printf("RabbitMQ connection closed: %v", err)
		}
	}()

	if err != nil {
		return nil, err
	}
	// Check if the connection is closed

	broker := &rabbitMqBroker{
		connection:      conn,
		channelPool:     make(chan *pooledChannel, settings.PoolSize),
		settings:        settings,
		reconnectTicker: time.NewTicker(5 * time.Second), // Retry every 5 seconds
		stopReconnect:   make(chan struct{}),
	}

	// Initialize the connection and channel pool
	if err := broker.connectAndInitialize(); err != nil {
		return nil, err
	}

	// Start connection recovery in a separate goroutine
	go broker.recoverConnection()

	return broker, nil
}

type rabbitMqBroker struct {
	connection      *amqp.Connection
	channelPool     chan *pooledChannel
	mu              sync.Mutex
	settings        *config.BrokerSettings
	reconnectTicker *time.Ticker
	stopReconnect   chan struct{}
}

func (r *rabbitMqBroker) Publish(ctx context.Context, event *store.OutboxEvent) error {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, "Publish",
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("rabbitmq"),
			semconv.MessagingDestinationKindKey.String(event.EntityType),
			semconv.MessagingDestinationKey.String(event.Entity),
			semconv.MessagingRabbitmqRoutingKeyKey.String(event.RoutingKey),
		),
	)
	defer span.End()

	// Inject the trace context into the message headers
	propagator := otel.GetTextMapPropagator()
	traceHeaders := make(map[string]string)
	propagator.Inject(ctx, propagation.MapCarrier(traceHeaders))

	// Merge traceHeaders into headers
	maps.Copy(event.Headers, traceHeaders)

	// Convert headers to amqp.Table
	amqpHeaders := make(amqp.Table)
	for k, v := range event.Headers {
		amqpHeaders[k] = v
	}

	// Get a channel from the pool
	pooledChan, err := r.getChannel()
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer r.releaseChannel(pooledChan)

	// ExchangeDeclare is idempotent and has no effect if the exchange is already in place
	err = pooledChan.channel.ExchangeDeclare(
		event.Entity,     // name of the exchange
		event.EntityType, // type of the exchange (e.g., "direct", "fanout", "topic", "headers")
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	err = pooledChan.channel.Publish(
		event.Entity, event.RoutingKey, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        event.Payload,
			Headers:     amqpHeaders,
		},
	)
	if err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.Int("messaging.message_payload_size_bytes", len(event.Payload)),
	)

	return nil
}

func (r *rabbitMqBroker) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop the connection recovery goroutine
	close(r.stopReconnect)
	r.reconnectTicker.Stop()

	// Close all channels in the pool
	close(r.channelPool)
	for pooledChan := range r.channelPool {
		pooledChan.channel.Close()
	}

	// Close the connection
	if r.connection != nil {
		return r.connection.Close()
	}
	return nil
}
