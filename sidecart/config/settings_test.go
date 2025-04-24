package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestValidate_ValidSettings(t *testing.T) {
	cfg := Settings{
		Database: DbSettings{
			Type: "postgres",
			DSN:  "postgres://user:password@localhost:5432/dbname",
		},
		Broker: BrokerSettings{
			Type: "rabbitmq",
			URL:  "amqp://guest:guest@localhost:5672/",
		},
		PollInterval:    10 * time.Second,
		BatchSize:       100,
		MaxRetries:      5,
		RetryBackoff:    2 * time.Second,
		DeadLetterTopic: "dead-letter-topic",
		Observability: Observability{
			ServiceName: "test-service",
			TracingURL:  "http://localhost:4318",
			MetricsURL:  "http://localhost:9090",
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidate_InvalidSettings(t *testing.T) {
	cfg := Settings{
		Database: DbSettings{
			Type: "invalid-db-type",
		},
		Broker: BrokerSettings{
			Type: "invalid-broker-type",
		},
		Observability: Observability{
			ServiceName: "",
			TracingURL:  "invalid-url",
			MetricsURL:  "invalid-url",
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestLoadFromFile(t *testing.T) {
	viper.Reset()
	viper.SetConfigType("yaml")

	// Mock configuration file
	configFile := `
database:
  type: postgres
  dsn: postgres://user:password@localhost:5432/dbname
broker:
  type: rabbitmq
  url: amqp://guest:guest@localhost:5672/
poll_interval: 10s
batch_size: 100
max_retries: 5
retry_backoff: 2s
dead_letter_topic: dead-letter-topic
observability:
  service_name: test-service
  tracing_url: http://localhost:4318
  metrics_url: http://localhost:9090
`
	viper.ReadConfig(strings.NewReader(configFile))

	cfg, err := LoadFromFile(".")
	assert.NoError(t, err)

	assert.Equal(t, "postgres", cfg.Database.Type)
	assert.Equal(t, "postgres://user:password@localhost:5432/dbname", cfg.Database.DSN)
	assert.Equal(t, "rabbitmq", cfg.Broker.Type)
	assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.Broker.URL)
	assert.Equal(t, 10*time.Second, cfg.PollInterval)
	assert.Equal(t, 100, cfg.BatchSize)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.RetryBackoff)
	assert.Equal(t, "dead-letter-topic", cfg.DeadLetterTopic)
	assert.Equal(t, "test-service", cfg.Observability.ServiceName)
	assert.Equal(t, "http://localhost:4318", cfg.Observability.TracingURL)
	assert.Equal(t, "http://localhost:9090", cfg.Observability.MetricsURL)
}

func TestLoadFromEnv(t *testing.T) {
	viper.Reset()

	// Mock environment variables
	os.Setenv("SIDECAR_DATABASE_TYPE", "mongo")
	os.Setenv("SIDECAR_DATABASE_URI", "mongodb://localhost:27017")
	os.Setenv("SIDECAR_BROKER_TYPE", "gcp-pubsub")
	os.Setenv("SIDECAR_BROKER_PROJECTID", "test-project")
	os.Setenv("SIDECAR_POLL_INTERVAL", "15s")
	os.Setenv("SIDECAR_BATCH_SIZE", "50")
	os.Setenv("SIDECAR_MAX_RETRIES", "3")
	os.Setenv("SIDECAR_RETRY_BACKOFF", "1s")
	os.Setenv("SIDECAR_DEAD_LETTER_TOPIC", "dead-letter-topic")
	os.Setenv("SIDECAR_OBSERVABILITY_SERVICE_NAME", "test-service")
	os.Setenv("SIDECAR_OBSERVABILITY_TRACING_URL", "http://localhost:4318")
	os.Setenv("SIDECAR_OBSERVABILITY_METRICS_URL", "http://localhost:9090")

	cfg := Settings{}
	err := cfg.LoadFromEnv()
	assert.NoError(t, err)

	assert.Equal(t, "mongo", cfg.Database.Type)
	assert.Equal(t, "mongodb://localhost:27017", cfg.Database.URI)
	assert.Equal(t, "gcp-pubsub", cfg.Broker.Type)
	assert.Equal(t, "test-project", cfg.Broker.ProjectID)
	assert.Equal(t, 15*time.Second, cfg.PollInterval)
	assert.Equal(t, 50, cfg.BatchSize)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 1*time.Second, cfg.RetryBackoff)
	assert.Equal(t, "dead-letter-topic", cfg.DeadLetterTopic)
	assert.Equal(t, "test-service", cfg.Observability.ServiceName)
	assert.Equal(t, "http://localhost:4318", cfg.Observability.TracingURL)
	assert.Equal(t, "http://localhost:9090", cfg.Observability.MetricsURL)
}
