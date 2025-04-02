package broker

import (
	"context"
	"fmt"

	"github.com/zoff-tech/go-outbox/pkg/config"
)

func NewBroker(ctx context.Context, cfg config.BrokerSettings) (MessageBroker, error) {
	switch cfg.Type {
	case "rabbitmq":
		conn, err := DefaultRabbitMQConnection(cfg.URL)
		if err != nil {
			return nil, err
		}
		return &rabbitMqBroker{connection: conn, exchange: cfg.Exchange}, nil
	case "pubsub":
		client, err := DefaultPubSubClient(ctx, cfg.ProjectID)
		if err != nil {
			return nil, err
		}
		return &pubSubBroker{client: client}, nil
	default:
		return nil, fmt.Errorf("unsupported broker type: %s", cfg.Type)
	}
}
