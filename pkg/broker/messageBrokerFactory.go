package broker

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/streadway/amqp"
	"github.com/zoff-tech/go-outbox/pkg/config"
)

func NewBroker(ctx context.Context, cfg config.BrokerSettings) (MessageBroker, error) {
	switch cfg.Type {
	case "rabbitmq":
		conn, err := amqp.Dial(cfg.URL)
		if err != nil {
			return nil, err
		}
		ch, err := conn.Channel()
		if err != nil {
			return nil, err
		}
		err = ch.ExchangeDeclare(
			cfg.Exchange, // name
			"topic",      // type
			true,         // durable
			false,        // auto-deleted
			false,        // internal
			false,        // no-wait
			nil,          // arguments
		)
		if err != nil {
			return nil, err
		}
		return &rabbitMqBroker{channel: ch, exchange: cfg.Exchange}, nil
	case "gcp-pubsub":
		client, err := pubsub.NewClient(ctx, cfg.ProjectID)
		if err != nil {
			return nil, err
		}
		return &pubSubBroker{client: client}, nil
	default:
		return nil, fmt.Errorf("unsupported broker type: %s", cfg.Type)
	}
}
