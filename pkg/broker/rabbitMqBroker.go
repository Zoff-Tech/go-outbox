package broker

import (
	"context"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

type rabbitMqBroker struct {
	channel  *amqp.Channel
	exchange string
}

func (r *rabbitMqBroker) Publish(ctx context.Context, routingKey string, data []byte, headers map[string]string) error {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, "Publish",
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("rabbitmq"),
			semconv.MessagingDestinationKindKey.String("exchange"),
			semconv.MessagingDestinationKey.String(r.exchange),
			semconv.MessagingRabbitmqRoutingKeyKey.String(routingKey),
		),
	)
	defer span.End()

	// Inject the trace context into the message headers
	propagator := otel.GetTextMapPropagator()
	traceHeaders := make(map[string]string)
	propagator.Inject(ctx, propagation.MapCarrier(traceHeaders))

	// Merge traceHeaders into headers
	for k, v := range traceHeaders {
		headers[k] = v
	}

	// Convert headers to amqp.Table
	amqpHeaders := make(amqp.Table)
	for k, v := range headers {
		amqpHeaders[k] = v
	}

	err := r.channel.Publish(
		r.exchange, routingKey, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
			Headers:     amqpHeaders,
		},
	)
	if err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.Int("messaging.message_payload_size_bytes", len(data)),
	)

	return nil
}

func (r *rabbitMqBroker) Close() error {
	return r.channel.Close()
}
