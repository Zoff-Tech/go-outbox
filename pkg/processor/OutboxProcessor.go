package processor

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/zoff-tech/go-outbox/pkg/broker"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"github.com/zoff-tech/go-outbox/pkg/store"
)

// OutboxProcessor processes outbox events.
type OutboxProcessor struct {
	repo         store.OutBoxRepository
	broker       broker.MessageBroker
	tracer       trace.Tracer
	maxRetries   int
	retryBackoff time.Duration
}

// NewOutboxProcessor creates a new instance of OutboxProcessor.
func NewOutboxProcessor(repo store.OutBoxRepository, broker broker.MessageBroker, cfg *config.Settings) *OutboxProcessor {
	return &OutboxProcessor{
		repo:         repo,
		broker:       broker,
		tracer:       otel.Tracer("go-outbox"),
		maxRetries:   cfg.MaxRetries,
		retryBackoff: cfg.RetryBackoff,
	}
}

func (p *OutboxProcessor) ProcessEvents(ctx context.Context) {
	for {
		events, err := p.repo.FetchPending(ctx, 10)
		if err != nil {
			log.Printf("Failed to fetch events: %v", err)
			continue
		}

		for _, event := range events {
			ctx, span := p.tracer.Start(ctx, "ProcessOutboxEvent", trace.WithAttributes(
				attribute.String("event.id", event.ID),
				attribute.String("event.topic", event.Topic),
				attribute.String("event.status", string(event.Status)),
				attribute.Int("event.retry_count", event.RetryCount),
				attribute.String("event.created_at", event.CreatedAt.String()),
				attribute.String("event.updated_at", event.UpdatedAt.String()),
			))

			headers := event.EventHeaders

			// Inject the trace context into the message headers
			propagator := otel.GetTextMapPropagator()
			propagator.Inject(ctx, propagation.MapCarrier(headers))

			if err := p.broker.Publish(ctx, event.Topic, event.Payload, headers); err != nil {
				log.Printf("Failed to publish event %s: %v", event.ID, err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())

				// Increment retry count and update status
				if event.RetryCount < p.maxRetries {
					if err := p.repo.SetStatusAndIncrementRetry(ctx, event.ID, store.StatusPending); err != nil {
						log.Printf("Failed to update retry count for event %s: %v", event.ID, err)
					}
				} else {
					if err := p.repo.SetStatus(ctx, event.ID, store.StatusFailed); err != nil {
						log.Printf("Failed to mark event %s as failed: %v", event.ID, err)
					}
				}

				span.End()
				continue
			}

			if err := p.repo.MarkProcessed(ctx, event.ID); err != nil {
				log.Printf("Failed to mark event %s as processed: %v", event.ID, err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			span.End()
		}

		time.Sleep(5 * time.Second)
	}
}
