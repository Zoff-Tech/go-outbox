package broker

import (
	"context"
	"testing"

	"cloud.google.com/go/pubsub"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/pkg/config"
	"google.golang.org/api/option"
)

func TestNewBroker_RabbitMQ(t *testing.T) {
	t.Run("should create a RabbitMQ broker successfully", func(t *testing.T) {
		// Arrange
		mockURL := "amqp://guest:guest@localhost:5672/"
		mockExchange := "test-exchange"
		cfg := config.BrokerSettings{
			Type:     "rabbitmq",
			URL:      mockURL,
			Exchange: mockExchange,
		}
		ctx := context.Background()

		// Mock DefaultRabbitMQConnection to avoid real RabbitMQ calls
		originalConnection := DefaultRabbitMQConnection
		DefaultRabbitMQConnection = func(url string) (*amqp.Connection, error) {
			assert.Equal(t, mockURL, url)
			return &amqp.Connection{}, nil
		}
		defer func() { DefaultRabbitMQConnection = originalConnection }()

		// Act
		broker, err := NewBroker(ctx, cfg)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, broker)
		assert.IsType(t, &rabbitMqBroker{}, broker)

		// Additional validation
		rabbitBroker, ok := broker.(*rabbitMqBroker)
		assert.True(t, ok)
		assert.Equal(t, mockExchange, rabbitBroker.exchange)
	})
}

func TestNewBroker_GCPPubSub(t *testing.T) {
	t.Run("should create a GCP Pub/Sub broker successfully", func(t *testing.T) {
		// Arrange
		mockProjectID := "test-project"
		cfg := config.BrokerSettings{
			Type:      "pubsub",
			ProjectID: mockProjectID,
		}
		ctx := context.Background()

		// Mock DefaultPubSubClient to avoid real GCP calls
		originalClient := DefaultPubSubClient
		DefaultPubSubClient = func(ctx context.Context, projectID string, opts ...option.ClientOption) (*pubsub.Client, error) {
			assert.Equal(t, mockProjectID, projectID)
			return &pubsub.Client{}, nil
		}
		defer func() { DefaultPubSubClient = originalClient }()

		// Act
		broker, err := NewBroker(ctx, cfg)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, broker)
		assert.IsType(t, &pubSubBroker{}, broker)

		// Additional validation
		pubSubBroker, ok := broker.(*pubSubBroker)
		assert.True(t, ok)
		assert.NotNil(t, pubSubBroker.client)
	})
}

func TestNewBroker_Unsupported(t *testing.T) {
	t.Run("should return an error for unsupported broker type", func(t *testing.T) {
		// Arrange
		cfg := config.BrokerSettings{
			Type: "unsupported",
		}
		ctx := context.Background()

		// Act
		broker, err := NewBroker(ctx, cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, broker)
		assert.Equal(t, "unsupported broker type: unsupported", err.Error())
	})
}
