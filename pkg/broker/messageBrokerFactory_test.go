package broker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"github.com/zoff-tech/go-outbox/pkg/store"
	"google.golang.org/api/option"
)

// Mock implementations for RabbitMQ and PubSub brokers
type mockRabbitMqBroker struct{}

func (m *mockRabbitMqBroker) Publish(ctx context.Context, message *store.OutboxEvent) error {
	return nil
}

func (m *mockRabbitMqBroker) Close() error {
	return nil
}

type mockPubSubBroker struct{}

func (m *mockPubSubBroker) Publish(ctx context.Context, message *store.OutboxEvent) error {
	return nil
}

func (m *mockPubSubBroker) Close() error {
	return nil
}

// Factory functions
func NewMockRabbitMqBroker(ctx context.Context, cfg *config.BrokerSettings) (MessageBroker, error) {
	if cfg.Type == "error" {
		return nil, errors.New("failed to create RabbitMQ broker")
	}
	return &mockRabbitMqBroker{}, nil
}

func NewMockPubSubClient(ctx context.Context, cfg *config.BrokerSettings, opts ...option.ClientOption) (MessageBroker, error) {
	if cfg.Type == "error" {
		return nil, errors.New("failed to create PubSub broker")
	}
	return &mockPubSubBroker{}, nil
}

// Tests
func TestNewBroker(t *testing.T) {
	// Save the original implementations
	originalNewRabbitMqBroker := NewRabbitMqBroker
	originalNewPubSubClient := NewPubSubClient

	// Replace the actual implementations with mocks for testing
	NewRabbitMqBroker = NewMockRabbitMqBroker
	NewPubSubClient = NewMockPubSubClient

	// Restore the original implementations after the test
	defer func() {
		NewRabbitMqBroker = originalNewRabbitMqBroker
		NewPubSubClient = originalNewPubSubClient
	}()

	tests := []struct {
		name        string
		cfg         *config.BrokerSettings
		expectedErr string
	}{
		{
			name: "Valid RabbitMQ configuration",
			cfg: &config.BrokerSettings{
				Type:     "rabbitmq",
				URL:      "amqp://guest:guest@localhost:5672/",
				PoolSize: 5,
			},
			expectedErr: "",
		},
		{
			name: "Invalid RabbitMQ configuration",
			cfg: &config.BrokerSettings{
				Type:     "rabbitmq",
				URL:      "invalid-url",
				PoolSize: 5,
			},
			expectedErr: "failed to connect to RabbitMQ",
		},
		{
			name: "Valid Pub/Sub configuration",
			cfg: &config.BrokerSettings{
				Type:      "pubsub",
				ProjectID: "valid-project",
			},
			expectedErr: "",
		},
		{
			name: "Invalid Pub/Sub configuration",
			cfg: &config.BrokerSettings{
				Type:      "pubsub",
				ProjectID: "invalid-project",
			},
			expectedErr: "failed to connect to Pub/Sub",
		},
		{
			name: "Unsupported broker type",
			cfg: &config.BrokerSettings{
				Type: "unsupported",
			},
			expectedErr: "unsupported broker type: unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker, err := NewBroker(context.Background(), tt.cfg)
			if tt.expectedErr != "" {
				assert.Nil(t, broker)
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NotNil(t, broker)
				assert.NoError(t, err)
			}
		})
	}
}
