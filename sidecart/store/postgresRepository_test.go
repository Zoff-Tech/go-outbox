package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/schema"
)

func TestFetchPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{Db: db}

	// Mock rows for the SELECT query
	rows := sqlmock.NewRows([]string{"id", "entity", "entity_type", "payload", "retry_count", "headers", "routing_key"}).
		AddRow("1", "entity1", "type1", []byte("payload1"), 0, []byte(`{"header1":"value1"}`), "key1").
		AddRow("2", "entity2", "type2", []byte("payload2"), 3, []byte(`{"header2":"value2"}`), "key2")

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, entity, entity_type, payload, retry_count, headers, routing_key FROM outbox_events WHERE \(status='pending' OR \(status='processing' AND updated_at < \$1\)\) FOR UPDATE SKIP LOCKED LIMIT \$2`).
		WithArgs(sqlmock.AnyArg(), 10).
		WillReturnRows(rows)
		// Accept both possible update queries
	// Accept both possible update queries and status values
	mock.ExpectExec(`UPDATE outbox_events SET status=\$1, retry_count = retry_count \+ 1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE outbox_events SET status=\$1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "2").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	events, err := repo.FetchPending(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	assert.Equal(t, "1", events[0].ID)
	assert.Equal(t, "entity1", events[0].Entity)
	assert.Equal(t, "type1", events[0].EntityType)
	assert.Equal(t, []byte("payload1"), events[0].Payload)
	assert.Equal(t, 0, events[0].RetryCount)
	assert.Equal(t, "key1", events[0].RoutingKey)

	assert.Equal(t, "2", events[1].ID)
	assert.Equal(t, "entity2", events[1].Entity)
	assert.Equal(t, "type2", events[1].EntityType)
	assert.Equal(t, []byte("payload2"), events[1].Payload)
	assert.Equal(t, 3, events[1].RetryCount)
	assert.Equal(t, "key2", events[1].RoutingKey)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkProcessed(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{Db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox_events SET status=\$1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(schema.StatusSent, sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.MarkProcessed(ctx, "1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{Db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox_events SET status=\$1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(schema.StatusProcessing, sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.SetStatus(ctx, "1", schema.StatusProcessing)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatusAndIncrementRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{Db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox_events SET status=\$1, retry_count = retry_count \+ 1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(schema.StatusProcessing, sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.SetStatusAndIncrementRetry(ctx, "1", schema.StatusProcessing)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementRetryCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{Db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox_events SET retry_count = retry_count \+ 1, updated_at=\$1 WHERE id=\$2`).
		WithArgs(sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.IncrementRetryCount(ctx, "1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
