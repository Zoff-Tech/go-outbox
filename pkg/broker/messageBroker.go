package broker

import "context"

// MessageBroker defines the operations to publish messages to a broker.
type MessageBroker interface {
	// Publish sends the message to the specified topic or exchange with optional headers.
	Publish(ctx context.Context, entity string, data []byte, headers map[string]string) error
	// Close cleans up any resources (connections).
	Close() error
}
