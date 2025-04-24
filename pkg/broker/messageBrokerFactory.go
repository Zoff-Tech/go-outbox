package broker

import (
	"context"
	"fmt"

	"github.com/zoff-tech/go-outbox/pkg/config"
)

func NewBroker(ctx context.Context, cfg *config.BrokerSettings) (MessageBroker, error) {
	switch cfg.Type {
	case "rabbitmq":
		// Use NewRabbitMqBroker to create the RabbitMQ broker
		broker, err := NewRabbitMqBroker(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return broker, nil
	case "pubsub":
		broker, err := NewPubSubClient(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return broker, nil
	default:
		return nil, fmt.Errorf("unsupported broker type: %s", cfg.Type)
	}
}
