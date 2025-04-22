package store

import "time"

// Status represents the status of an outbox event.
type Status string

const (
	StatusPending    Status = "pending"
	StatusSent       Status = "sent"
	StatusFailed     Status = "failed"
	StatusCanceled   Status = "canceled"
	StatusProcessing Status = "processing"
)

// OutboxEvent represents an event stored in the outbox table.
type OutboxEvent struct {
	ID         string            `json:"id"`
	Entity     string            `json:"entity"`
	EntityType string            `json:"entity_type"`
	Payload    []byte            `json:"payload"`
	Status     Status            `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	SentAt     time.Time         `json:"sent_at,omitempty"`
	Headers    map[string]string `json:"headers"`
	RetryCount int               `json:"retry_count"`
	RoutingKey string            `json:"routing_key"`
}
