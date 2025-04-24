package store

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const lockExpiration = 5 * time.Minute
const maxRetries = 3

func addDBStatsToSpan(span trace.Span, statement string, eventsCount int, duration time.Duration) {
	span.SetAttributes(
		attribute.Int("eventsCount", eventsCount),
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", statement),
		attribute.Float64("db.execution_time_ms", float64(duration.Milliseconds())),
	)
}
