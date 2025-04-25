package config

type Observability struct {
	ServiceName string `mapstructure:"service_name" validate:"required"`
	TracingURL  string `mapstructure:"tracing_url" validate:"required,url"`
	MetricsURL  string `mapstructure:"metrics_url" validate:"required,url"`
}
