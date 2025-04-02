package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/spannertest"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/zoff-tech/go-outbox/pkg/config"
)

var sqlOpen = sql.Open

func TestNewRepository_Postgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Mock sql.Open
	originalOpen := sqlOpen
	sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
		return db, nil
	}
	defer func() { sqlOpen = originalOpen }()

	cfg := config.DbSettings{
		Type: "postgres",
		DSN:  "postgres://user:password@localhost:5432/dbname",
	}

	ctx := context.Background()
	repo, err := NewRepository(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.IsType(t, &PostgresRepository{}, repo)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// func TestNewRepository_Mongo(t *testing.T) {
// 	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
// 	defer mt.Client.Disconnect(context.Background())

// 	cfg := config.DbSettings{
// 		Type:   "mongo",
// 		URI:    "mongodb://localhost:27017",
// 		DBName: "testdb",
// 	}

// 	ctx := context.Background()
// 	mt.Run("create mongo repository", func(mt *mtest.T) {
// 		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
// 		assert.NoError(t, err)
// 		defer client.Disconnect(ctx)

// 		repo, err := NewRepository(ctx, cfg)
// 		assert.NoError(t, err)
// 		assert.NotNil(t, repo)
// 		assert.IsType(t, &MongoRepository{}, repo)
// 	})
// }

func TestNewRepository_Unsupported(t *testing.T) {
	cfg := config.DbSettings{
		Type: "unsupported",
	}

	ctx := context.Background()
	repo, err := NewRepository(ctx, cfg)
	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Equal(t, "unsupported DB type: unsupported", err.Error())
}
func TestNewRepository_Spanner(t *testing.T) {
	// Set up a Spanner test server
	server, err := spannertest.NewServer("localhost:0")
	assert.NoError(t, err)
	defer server.Close()

	// Use a valid mock Spanner connection string
	mockURI := "projects/test-project/instances/test-instance/databases/test-database"

	// Mock configuration for Spanner
	cfg := config.DbSettings{
		Type: "spanner",
		URI:  mockURI,
	}

	// Create a Spanner client using the spannertest server's address
	ctx := context.Background()

	os.Setenv("SPANNER_EMULATOR_HOST", server.Addr)

	client, err := spanner.NewClient(ctx, mockURI)

	assert.NoError(t, err)
	defer client.Close()

	// Override the NewSpannerRepositoryFactory function to use the mock client
	originalFactory := NewSpannerRepositoryFactory
	NewSpannerRepositoryFactory = func(client *spanner.Client) OutBoxRepository {
		return &SpannerRepository{client: client}
	}
	defer func() { NewSpannerRepositoryFactory = originalFactory }()

	// Call NewRepository
	repo, err := NewRepository(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.IsType(t, &SpannerRepository{}, repo)

}
