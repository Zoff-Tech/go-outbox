package schema

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

// NewEvent creates a new OutboxEvent with required fields and sensible defaults.
func NewEvent(
	id, entity, entityType string,
	payload []byte,
	headers map[string]string,
	routingKey string,
) *OutboxEvent {
	now := time.Now()
	return &OutboxEvent{
		ID:         id,
		Entity:     entity,
		EntityType: entityType,
		Payload:    payload,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		Headers:    headers,
		RetryCount: 0,
		RoutingKey: routingKey,
	}
}
