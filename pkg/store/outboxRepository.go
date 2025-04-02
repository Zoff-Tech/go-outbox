package store

import (
	"context"
)

// OutBoxRepository defines the database operations for outbox events.
type OutBoxRepository interface {
	// FetchPending retrieves unprocessed outbox events (e.g., status = "pending").
	FetchPending(ctx context.Context, batchSize int) ([]OutboxEvent, error)
	// MarkProcessed marks an outbox event as processed (sent) to avoid reprocessing.
	MarkProcessed(ctx context.Context, eventID string) error
	// SetStatus sets the status of an outbox event.
	SetStatus(ctx context.Context, eventID string, status Status) error
	// SetStatusAndIncrementRetry sets the status of an outbox event and increments the retry count.
	SetStatusAndIncrementRetry(ctx context.Context, eventID string, status Status) error
	// IncrementRetryCount increments the retry count of an outbox event.
	IncrementRetryCount(ctx context.Context, eventID string) error
}
