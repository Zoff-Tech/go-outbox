package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestFetchPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{db: db}

	// Mock rows for the SELECT query
	rows := sqlmock.NewRows([]string{"id", "topic", "payload", "retry_count"}).
		AddRow("1", "topic1", []byte("payload1"), 0).
		AddRow("2", "topic2", []byte("payload2"), 2)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, topic, payload, retry_count FROM outbox WHERE \(status='pending' OR \(status='processing' AND updated_at < \$1\)\) FOR UPDATE SKIP LOCKED LIMIT \$2`).
		WithArgs(sqlmock.AnyArg(), 10).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE outbox SET status=\$1, retry_count = retry_count \+ 1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(StatusProcessing, sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE outbox SET status=\$1, retry_count = retry_count \+ 1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(StatusProcessing, sqlmock.AnyArg(), "2").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	events, err := repo.FetchPending(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "1", events[0].ID)
	assert.Equal(t, "topic1", events[0].Topic)
	assert.Equal(t, []byte("payload1"), events[0].Payload)
	assert.Equal(t, 0, events[0].RetryCount)
	assert.Equal(t, "2", events[1].ID)
	assert.Equal(t, "topic2", events[1].Topic)
	assert.Equal(t, []byte("payload2"), events[1].Payload)
	assert.Equal(t, 2, events[1].RetryCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkProcessed(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox SET status=\$1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(StatusSent, sqlmock.AnyArg(), "1").
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

	repo := &PostgresRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox SET status=\$1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(StatusProcessing, sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.SetStatus(ctx, "1", StatusProcessing)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatusAndIncrementRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox SET status=\$1, retry_count = retry_count \+ 1, updated_at=\$2 WHERE id=\$3`).
		WithArgs(StatusProcessing, sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.SetStatusAndIncrementRetry(ctx, "1", StatusProcessing)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementRetryCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &PostgresRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE outbox SET retry_count = retry_count \+ 1, updated_at=\$1 WHERE id=\$2`).
		WithArgs(sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = repo.IncrementRetryCount(ctx, "1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
