package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/zoff-tech/go-outbox/schema"
	"go.opentelemetry.io/otel"
)

type PostgresRepository struct {
	Db *sql.DB // using database/sql
}

func (p *PostgresRepository) FetchPending(ctx context.Context, batchSize int) ([]schema.OutboxEvent, error) {
	return p.withTransaction(ctx, "FetchPending", func(ctx context.Context, tx *sql.Tx) ([]schema.OutboxEvent, error) {
		rows, err := tx.QueryContext(ctx,
			`SELECT id, entity, entity_type, payload, retry_count, headers, routing_key FROM outbox_events
             WHERE (status='pending' OR (status='processing' AND updated_at < $1)) 
             FOR UPDATE SKIP LOCKED LIMIT $2`, time.Now().Add(-lockExpiration), batchSize)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var events []schema.OutboxEvent
		var rawHeaders string // Use a temporary variable to store the raw headers

		for rows.Next() {
			var event schema.OutboxEvent
			if err := rows.Scan(&event.ID,
				&event.Entity,
				&event.EntityType,
				&event.Payload,
				&event.RetryCount,
				&rawHeaders,
				&event.RoutingKey); err != nil {
				return nil, err
			}

			// Convert rawHeaders to a map[string]string
			if rawHeaders != "" {
				if err := json.Unmarshal([]byte(rawHeaders), &event.Headers); err != nil {
					return nil, errors.New("failed to parse headers: " + err.Error())
				}
			}

			events = append(events, event)
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}

		// Update status and updated_at for fetched events
		for _, event := range events {
			if event.RetryCount >= maxRetries {
				if err := p.SetStatus(ctx, event.ID, schema.StatusFailed); err != nil {
					return nil, err
				}
			} else {
				if err := p.SetStatusAndIncrementRetry(ctx, event.ID, schema.StatusProcessing); err != nil {
					return nil, err
				}
			}
		}

		return events, nil
	})
}

func (p *PostgresRepository) MarkProcessed(ctx context.Context, eventID string) error {
	_, err := p.withTransaction(ctx, "MarkProcessed", func(ctx context.Context, tx *sql.Tx) ([]schema.OutboxEvent, error) {
		if err := p.SetStatus(ctx, eventID, schema.StatusSent); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (p *PostgresRepository) SetStatus(ctx context.Context, eventID string, status schema.Status) error {
	_, err := p.withTransaction(ctx, "SetStatus", func(ctx context.Context, tx *sql.Tx) ([]schema.OutboxEvent, error) {
		_, err := tx.ExecContext(ctx,
			`UPDATE outbox_events SET status=$1, updated_at=$2 WHERE id=$3`,
			status, time.Now(), eventID)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (p *PostgresRepository) SetStatusAndIncrementRetry(ctx context.Context, eventID string, status schema.Status) error {
	_, err := p.withTransaction(ctx, "SetStatusAndIncrementRetry", func(ctx context.Context, tx *sql.Tx) ([]schema.OutboxEvent, error) {
		_, err := tx.ExecContext(ctx,
			`UPDATE outbox_events SET status=$1, retry_count = retry_count + 1, updated_at=$2 WHERE id=$3`,
			status, time.Now(), eventID)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (p *PostgresRepository) IncrementRetryCount(ctx context.Context, eventID string) error {
	_, err := p.withTransaction(ctx, "IncrementRetryCount", func(ctx context.Context, tx *sql.Tx) ([]schema.OutboxEvent, error) {
		_, err := tx.ExecContext(ctx,
			`UPDATE outbox_events SET retry_count = retry_count + 1, updated_at=$1 WHERE id=$2`,
			time.Now(), eventID)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

func (p *PostgresRepository) withTransaction(ctx context.Context, spanName string, fn func(ctx context.Context, tx *sql.Tx) ([]schema.OutboxEvent, error)) ([]schema.OutboxEvent, error) {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()

	tx, ok := ctx.Value("tx").(*sql.Tx)
	if !ok {
		var err error
		tx, err = p.Db.BeginTx(ctx, nil)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}()
		ctx = context.WithValue(ctx, "tx", tx)
	}

	events, err := fn(ctx, tx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	addDBStatsToSpan(span, spanName, len(events), time.Since(time.Now()))

	return events, nil
}
