package broker

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"github.com/zoff-tech/go-outbox/pkg/store"
	"google.golang.org/api/option"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

// PubSubBrokerCreator defines a function type for creating Pub/Sub clients.
type PubSubBrokerCreator func(ctx context.Context, settings *config.BrokerSettings, opts ...option.ClientOption) (MessageBroker, error)

// NewPubSubClient is the default implementation of PubSubClientCreator.
var NewPubSubClient PubSubBrokerCreator = func(ctx context.Context, settings *config.BrokerSettings, opts ...option.ClientOption) (MessageBroker, error) {
	client, err := pubsub.NewClient(ctx, settings.ProjectID, opts...)
	if err != nil {
		return nil, err
	}
	return &pubSubBroker{client: client}, nil
}

type pubSubBroker struct {
	client *pubsub.Client
}

func (p *pubSubBroker) Publish(ctx context.Context, event *store.OutboxEvent) error {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, "Publish",
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("pubsub"),
			semconv.MessagingDestinationKindKey.String("topic"),
			semconv.MessagingDestinationKey.String(event.Entity),
		),
	)
	defer span.End()

	// Inject the trace context into the message attributes
	propagator := otel.GetTextMapPropagator()
	attributes := make(map[string]string)
	propagator.Inject(ctx, propagation.MapCarrier(attributes))

	// Merge headers into attributes
	for key, value := range event.Headers {
		attributes[key] = value
	}

	message := &pubsub.Message{
		Data:       event.Payload,
		Attributes: attributes,
	}

	message.OrderingKey = event.RoutingKey

	res := p.client.Topic(event.Entity).Publish(ctx, message)
	_, err := res.Get(ctx) // wait for server ack
	if err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.Int("messaging.message_payload_size_bytes", len(event.Payload)),
	)

	return nil
}

func (p *pubSubBroker) Close() error {
	return p.client.Close()
}
