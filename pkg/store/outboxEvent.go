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
	ID           string            `json:"id"`
	Topic        string            `json:"topic"`
	Payload      []byte            `json:"payload"`
	Status       Status            `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	SentAt       time.Time         `json:"sent_at,omitempty"`
	EventHeaders map[string]string `json:"event_headers"`
	RetryCount   int               `json:"retry_count"`
}
