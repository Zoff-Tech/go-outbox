package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Settings struct {
	Database        DbSettings     `mapstructure:"database"`
	Broker          BrokerSettings `mapstructure:"broker"`
	PollInterval    time.Duration  `mapstructure:"poll_interval"`
	BatchSize       int            `mapstructure:"batch_size"`
	MaxRetries      int            `mapstructure:"max_retries"`
	RetryBackoff    time.Duration  `mapstructure:"retry_backoff"` // initial backoff duration
	DeadLetterTopic string         `mapstructure:"dead_letter_topic"`
	Observability   Observability  `mapstructure:"observability"` // Observability settings
}

func (c *Settings) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

func LoadFromFile(filePath string) (*Settings, error) {

	env := getEnvWithDefaultLookup("ENVIRONMENT", "development")

	cfg := &Settings{}
	viper.SetConfigType("yaml") // Set the config type to YAML
	viper.SetConfigName("sidecar")
	viper.AddConfigPath(filePath) // path to config
	viper.AddConfigPath(".")      // current directory

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No config file found or read error: %v (will rely on env)", err)
	}

	err := mergeConfig(filePath, "sidecar."+env)
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error merging dev config: %s\n", err)
			os.Exit(1)
		}
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	if err := cfg.LoadFromEnv(); err != nil {
		log.Fatalf("Failed to load from env: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	return cfg, nil
}

func (c *Settings) LoadFromEnv() error {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SIDECAR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // env vars like SIDECAR_DATABASE_TYPE

	// Bind environment variables explicitly to ensure they map correctly
	viper.BindEnv("database.type")
	viper.BindEnv("database.dsn")
	viper.BindEnv("database.uri")
	viper.BindEnv("broker.type")
	viper.BindEnv("broker.url")
	viper.BindEnv("broker.projectID")
	viper.BindEnv("poll_interval")
	viper.BindEnv("batch_size")
	viper.BindEnv("max_retries")
	viper.BindEnv("retry_backoff")
	viper.BindEnv("dead_letter_topic")
	viper.BindEnv("observability.service_name")
	viper.BindEnv("observability.tracing_url")
	viper.BindEnv("observability.metrics_url")

	if err := viper.Unmarshal(&c); err != nil {
		return err
	}
	return nil
}

func mergeConfig(path string, name string) error {
	viper.SetConfigName(name)
	viper.AddConfigPath(path)
	err := viper.MergeInConfig()
	if err != nil {
		return err
	}
	return nil
}

func getEnvWithDefaultLookup(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
