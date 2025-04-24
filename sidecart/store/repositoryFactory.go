package store

import (
	"context"
	"database/sql"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/zoff-tech/go-outbox/config"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var NewSpannerRepositoryFactory = func(client *spanner.Client) OutBoxRepository {
	return &SpannerRepository{client: client}
}

func NewRepository(ctx context.Context, cfg config.DbSettings) (OutBoxRepository, error) {
	switch cfg.Type {
	case "postgres":
		db, err := sql.Open("postgres", cfg.DSN)
		if err != nil {
			return nil, err
		}
		return &PostgresRepository{Db: db}, nil
	case "spanner":
		client, err := spanner.NewClient(ctx, cfg.URI)
		if err != nil {
			return nil, err
		}
		return NewSpannerRepositoryFactory(client), nil
	default:
		return nil, fmt.Errorf("unsupported DB type: %s", cfg.Type)
	}
}
