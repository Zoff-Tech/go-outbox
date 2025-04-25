package broker

import (
	"context"

	"github.com/zoff-tech/go-outbox/schema"
)

// MessageBroker defines the operations to publish messages to a broker.
type MessageBroker interface {
	// Publish sends the message to the specified topic or exchange with optional headers.
	Publish(ctx context.Context, event *schema.OutboxEvent) error
	// Close cleans up any resources (connections).
	Close() error
}
