package store

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type SpannerRepository struct {
	client *spanner.Client
}

func (s *SpannerRepository) FetchPending(ctx context.Context, batchSize int) ([]OutboxEvent, error) {
	stmt := spanner.Statement{
		SQL: `SELECT id, entity, entity_type, payload, retry_count, headers, routing_key FROM outbox
              WHERE (status = @statusPending OR (status = @statusProcessing AND updated_at < @lockExpiration))
              LIMIT @batchSize`,
		Params: map[string]interface{}{
			"statusPending":    StatusPending,
			"statusProcessing": StatusProcessing,
			"lockExpiration":   time.Now().Add(-lockExpiration),
			"batchSize":        batchSize,
		},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var events []OutboxEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var event OutboxEvent
		if err := row.Columns(
			&event.ID,
			&event.Entity,
			&event.EntityType,
			&event.Payload,
			&event.RetryCount,
			&event.Headers,
			&event.RoutingKey); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	// Update status and updated_at for fetched events
	for _, event := range events {
		if event.RetryCount >= maxRetries {
			if err := s.SetStatus(ctx, event.ID, StatusFailed); err != nil {
				return nil, err
			}
		} else {
			if err := s.SetStatusAndIncrementRetry(ctx, event.ID, StatusProcessing); err != nil {
				return nil, err
			}
		}
	}

	return events, nil
}

func (s *SpannerRepository) SetStatus(ctx context.Context, eventID string, status Status) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE outbox SET status = @status, updated_at = CURRENT_TIMESTAMP() WHERE id = @id`,
			Params: map[string]interface{}{
				"status": status,
				"id":     eventID,
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

func (s *SpannerRepository) SetStatusAndIncrementRetry(ctx context.Context, eventID string, status Status) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE outbox SET status = @status, retry_count = retry_count + 1, updated_at = CURRENT_TIMESTAMP() WHERE id = @id`,
			Params: map[string]interface{}{
				"status": status,
				"id":     eventID,
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

func (s *SpannerRepository) IncrementRetryCount(ctx context.Context, eventID string) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `UPDATE outbox SET retry_count = retry_count + 1, updated_at = CURRENT_TIMESTAMP() WHERE id = @id`,
			Params: map[string]interface{}{
				"id": eventID,
			},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	return err
}

func (s *SpannerRepository) MarkProcessed(ctx context.Context, eventID string) error {
	return s.SetStatus(ctx, eventID, StatusSent)
}
