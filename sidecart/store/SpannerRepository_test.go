package store

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/spannertest"
	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/schema"
)

func setupSpannerTestServer(t *testing.T) (*spanner.Client, func()) {
	server, err := spannertest.NewServer("localhost:0")
	assert.NoError(t, err)

	conn, err := spanner.NewClient(context.Background(), server.Addr)
	assert.NoError(t, err)

	return conn, func() {
		conn.Close()
		server.Close()
	}
}

func SpannerTestFetchPending(t *testing.T) {
	client, cleanup := setupSpannerTestServer(t)
	defer cleanup()

	repo := NewSpannerRepositoryFactory(client)

	// Insert mock data into the Spanner test server
	ctx := context.Background()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("outbox", []string{"id", "status", "retry_count", "updated_at"}, []interface{}{"1", "pending", 0, time.Now()}),
	})
	assert.NoError(t, err)

	// Call FetchPending
	events, err := repo.FetchPending(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "1", events[0].ID)
	assert.Equal(t, schema.StatusPending, events[0].Status)
}

func SpannerTestSetStatus(t *testing.T) {
	client, cleanup := setupSpannerTestServer(t)
	defer cleanup()

	repo := NewSpannerRepositoryFactory(client)

	// Insert mock data into the Spanner test server
	ctx := context.Background()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("outbox", []string{"id", "status", "retry_count", "updated_at"}, []interface{}{"1", "pending", 0, time.Now()}),
	})
	assert.NoError(t, err)

	// Call SetStatus
	err = repo.SetStatus(ctx, "1", schema.StatusProcessing)
	assert.NoError(t, err)

	// Verify the status was updated
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT status FROM outbox WHERE id = @id`,
		Params: map[string]interface{}{"id": "1"},
	})
	defer iter.Stop()

	row, err := iter.Next()
	assert.NoError(t, err)

	var status string
	err = row.Columns(&status)
	assert.NoError(t, err)
	assert.Equal(t, schema.StatusProcessing, status)
}

func SpannerTestSetStatusAndIncrementRetry(t *testing.T) {
	client, cleanup := setupSpannerTestServer(t)
	defer cleanup()

	repo := NewSpannerRepositoryFactory(client)

	// Insert mock data into the Spanner test server
	ctx := context.Background()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("outbox", []string{"id", "status", "retry_count", "updated_at"}, []interface{}{"1", "pending", 0, time.Now()}),
	})
	assert.NoError(t, err)

	// Call SetStatusAndIncrementRetry
	err = repo.SetStatusAndIncrementRetry(ctx, "1", schema.StatusProcessing)
	assert.NoError(t, err)

	// Verify the status and retry count were updated
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT status, retry_count FROM outbox WHERE id = @id`,
		Params: map[string]interface{}{"id": "1"},
	})
	defer iter.Stop()

	row, err := iter.Next()
	assert.NoError(t, err)

	var status string
	var retryCount int64
	err = row.Columns(&status, &retryCount)
	assert.NoError(t, err)
	assert.Equal(t, schema.StatusProcessing, status)
	assert.Equal(t, int64(1), retryCount)
}

func SpannerTestIncrementRetryCount(t *testing.T) {
	client, cleanup := setupSpannerTestServer(t)
	defer cleanup()

	repo := NewSpannerRepositoryFactory(client)

	// Insert mock data into the Spanner test server
	ctx := context.Background()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("outbox", []string{"id", "status", "retry_count", "updated_at"}, []interface{}{"1", "pending", 0, time.Now()}),
	})
	assert.NoError(t, err)

	// Call IncrementRetryCount
	err = repo.IncrementRetryCount(ctx, "1")
	assert.NoError(t, err)

	// Verify the retry count was incremented
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT retry_count FROM outbox WHERE id = @id`,
		Params: map[string]interface{}{"id": "1"},
	})
	defer iter.Stop()

	row, err := iter.Next()
	assert.NoError(t, err)

	var retryCount int64
	err = row.Columns(&retryCount)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), retryCount)
}

func SpannerTestMarkProcessed(t *testing.T) {
	client, cleanup := setupSpannerTestServer(t)
	defer cleanup()

	repo := NewSpannerRepositoryFactory(client)

	// Insert mock data into the Spanner test server
	ctx := context.Background()
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("outbox", []string{"id", "status", "retry_count", "updated_at"}, []interface{}{"1", "pending", 0, time.Now()}),
	})
	assert.NoError(t, err)

	// Call MarkProcessed
	err = repo.MarkProcessed(ctx, "1")
	assert.NoError(t, err)

	// Verify the status was updated
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT status FROM outbox WHERE id = @id`,
		Params: map[string]interface{}{"id": "1"},
	})
	defer iter.Stop()

	row, err := iter.Next()
	assert.NoError(t, err)

	var status string
	err = row.Columns(&status)
	assert.NoError(t, err)
	assert.Equal(t, schema.StatusSent, status)
}
