package config

// BrokerSettings holds configuration for connecting to a message broker.
type BrokerSettings struct {
	Type      string
	URL       string
	ProjectID string // Optional for brokers like GCP Pub/Sub
	PoolSize  int    // Optional for RabbitMQ
}
