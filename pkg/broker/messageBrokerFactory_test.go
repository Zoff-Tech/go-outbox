package broker

import (
	"context"
	"testing"

	"cloud.google.com/go/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"google.golang.org/api/option"
)

func TestNewBroker_RabbitMQ(t *testing.T) {
	// Mock RabbitMQ connection
	mockURL := "amqp://guest:guest@localhost:5672/"
	mockExchange := "test-exchange"

	cfg := config.BrokerSettings{
		Type:     "rabbitmq",
		URL:      mockURL,
		Exchange: mockExchange,
	}

	ctx := context.Background()

	// Call NewBroker
	broker, err := NewBroker(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, broker)
	assert.IsType(t, &rabbitMqBroker{}, broker)

	// Additional validation
	rabbitBroker, ok := broker.(*rabbitMqBroker)
	assert.True(t, ok)
	assert.Equal(t, mockExchange, rabbitBroker.exchange)
}

func TestNewBroker_GCPPubSub(t *testing.T) {
	// Mock GCP Pub/Sub configuration
	mockProjectID := "test-project"

	cfg := config.BrokerSettings{
		Type:      "gcp-pubsub",
		ProjectID: mockProjectID,
	}

	ctx := context.Background()

	// Mock pubsub.NewClient to avoid real GCP calls
	originalNewClient := pubsub.NewClient
	pubsub.NewClient = func(ctx context.Context, projectID string, opts ...option.ClientOption) (*pubsub.Client, error) {
		assert.Equal(t, mockProjectID, projectID)
		return &pubsub.Client{}, nil
	}
	defer func() { pubsub.NewClient = originalNewClient }()

	// Call NewBroker
	broker, err := NewBroker(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, broker)
	assert.IsType(t, &pubSubBroker{}, broker)

	// Additional validation
	pubSubBroker, ok := broker.(*pubSubBroker)
	assert.True(t, ok)
	assert.NotNil(t, pubSubBroker.client)
}

func TestNewBroker_Unsupported(t *testing.T) {
	// Mock unsupported broker configuration
	cfg := config.BrokerSettings{
		Type: "unsupported",
	}

	ctx := context.Background()

	// Call NewBroker
	broker, err := NewBroker(ctx, cfg)
	assert.Error(t, err)
	assert.Nil(t, broker)
	assert.Equal(t, "unsupported broker type: unsupported", err.Error())
}
