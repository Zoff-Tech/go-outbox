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

type connectionInterface interface {
	Channel() (*amqp.Channel, error)
	Close() error
	IsClosed() bool
}

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
	connection      connectionInterface
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

type channelInterface interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

type pooledChannel struct {
	channel     channelInterface
	notifyClose chan *amqp.Error
}

type NewConnection func(settings *config.BrokerSettings) (*amqp.Connection, error)

var newConnection NewConnection = func(settings *config.BrokerSettings) (*amqp.Connection, error) {
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

	return conn, nil
}

func (r *rabbitMqBroker) connectAndInitialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close existing connection if it exists
	if r.connection != nil && !r.connection.IsClosed() {
		r.connection.Close()
	}

	// Establish a new connection
	connection, err := newConnection(r.settings)

	if err != nil {
		return err
	}
	r.connection = connection

	// Clear the existing channel pool
	close(r.channelPool)
	r.channelPool = make(chan *pooledChannel, r.settings.PoolSize)

	// Declare the exchange
	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// Reinitialize the channel pool
	for i := 0; i < r.settings.PoolSize; i++ {
		channel, err := connection.Channel()
		if err != nil {
			return err
		}
		r.channelPool <- &pooledChannel{
			channel:     channel,
			notifyClose: channel.NotifyClose(make(chan *amqp.Error)),
		}
	}

	log.Println("RabbitMQ connection, exchange, and channel pool initialized")
	return nil
}

func (r *rabbitMqBroker) recoverConnection() {
	for {
		select {
		case <-r.reconnectTicker.C:
			if r.connection == nil || r.connection.IsClosed() {
				log.Println("Attempting to reconnect to RabbitMQ...")
				if err := r.connectAndInitialize(); err != nil {
					log.Printf("Failed to reconnect to RabbitMQ: %v", err)
				} else {
					log.Println("Reconnected to RabbitMQ successfully")
				}
			}
		case <-r.stopReconnect:
			log.Println("Stopping RabbitMQ connection recovery")
			return
		}
	}
}

func (r *rabbitMqBroker) getChannel() (*pooledChannel, error) {
	for {
		select {
		case pooledChan := <-r.channelPool:
			select {
			case err := <-pooledChan.notifyClose:
				// Channel is closed, discard it
				log.Printf("Discarding closed channel: %v", err)
				continue
			default:
				// Channel is valid
				log.Println("Retrieved channel from pool")
				return pooledChan, nil
			}
		default:
			// Create a new channel if none are available
			log.Println("Creating new channel")
			channel, err := r.connection.Channel()
			if err != nil {
				return nil, err
			}
			return &pooledChannel{
				channel:     channel,
				notifyClose: channel.NotifyClose(make(chan *amqp.Error)),
			}, nil
		}
	}
}

func (r *rabbitMqBroker) releaseChannel(pooledChan *pooledChannel) {
	select {
	case err := <-pooledChan.notifyClose:
		// Channel is closed, discard it
		log.Printf("Discarding closed channel: %v", err)
		return
	default:
		// Channel is valid, return it to the pool
		select {
		case r.channelPool <- pooledChan:
			log.Println("Returned channel to pool")
		default:
			// Pool is full, close the channel
			log.Println("Closing channel as pool is full")
			pooledChan.channel.Close()
		}
	}
}
