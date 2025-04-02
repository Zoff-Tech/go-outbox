package broker

import (
	"context"

	"cloud.google.com/go/pubsub"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

type pubSubBroker struct {
	client *pubsub.Client
}

func (p *pubSubBroker) Publish(ctx context.Context, topic string, data []byte, headers map[string]string) error {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, "Publish",
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("pubsub"),
			semconv.MessagingDestinationKindKey.String("topic"),
			semconv.MessagingDestinationKey.String(topic),
		),
	)
	defer span.End()

	// Inject the trace context into the message attributes
	propagator := otel.GetTextMapPropagator()
	attributes := make(map[string]string)
	propagator.Inject(ctx, propagation.MapCarrier(attributes))

	// Merge headers into attributes
	for key, value := range headers {
		attributes[key] = value
	}

	message := &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	}

	res := p.client.Topic(topic).Publish(ctx, message)
	_, err := res.Get(ctx) // wait for server ack
	if err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.Int("messaging.message_payload_size_bytes", len(data)),
	)

	return nil
}

func (p *pubSubBroker) Close() error {
	return p.client.Close()
}
