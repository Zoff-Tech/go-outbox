package store

import (
	"context"
	"time"

	"github.com/zoff-tech/go-outbox/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
)

type MongoRepository struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewMongoRepository(client *mongo.Client, database, collection string) *MongoRepository {
	return &MongoRepository{
		client:     client,
		database:   database,
		collection: collection,
	}
}

func (m *MongoRepository) FetchPending(ctx context.Context, batchSize int) ([]schema.OutboxEvent, error) {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, "FetchPending")
	defer span.End()

	startTime := time.Now()

	collection := m.client.Database(m.database).Collection(m.collection)
	filter := bson.M{
		"$or": []bson.M{
			{"status": schema.StatusPending},
			{"status": schema.StatusProcessing, "updated_at": bson.M{"$lt": time.Now().Add(-lockExpiration)}},
		},
	}
	opts := options.Find().SetLimit(int64(batchSize)).SetSort(bson.D{{Key: "updated_at", Value: 1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []schema.OutboxEvent
	for cursor.Next(ctx) {
		var event schema.OutboxEvent
		if err := cursor.Decode(&event); err != nil {
			span.RecordError(err)
			return nil, err
		}
		events = append(events, event)
	}

	if err := cursor.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Update status and updated_at for fetched events
	for _, event := range events {
		if event.RetryCount >= maxRetries {
			if err := m.SetStatus(ctx, event.ID, schema.StatusFailed); err != nil {
				return nil, err
			}
		} else {
			if err := m.SetStatusAndIncrementRetry(ctx, event.ID, schema.StatusProcessing); err != nil {
				return nil, err
			}
		}
	}

	addDBStatsToSpan(span, "FetchPending", len(events), time.Since(startTime))

	return events, nil
}

func (m *MongoRepository) MarkProcessed(ctx context.Context, eventID string) error {
	tracer := otel.Tracer("go-outbox")
	ctx, span := tracer.Start(ctx, "MarkProcessed")
	defer span.End()

	startTime := time.Now()
	if err := m.SetStatus(ctx, eventID, schema.StatusSent); err != nil {
		span.RecordError(err)
		return err
	}

	addDBStatsToSpan(span, "MarkProcessed", 1, time.Since(startTime))

	return nil
}

func (m *MongoRepository) SetStatus(ctx context.Context, eventID string, status schema.Status) error {
	collection := m.client.Database(m.database).Collection(m.collection)
	filter := bson.M{"id": eventID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (m *MongoRepository) SetStatusAndIncrementRetry(ctx context.Context, eventID string, status schema.Status) error {
	collection := m.client.Database(m.database).Collection(m.collection)
	filter := bson.M{"id": eventID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
		"$inc": bson.M{"retry_count": 1},
	}
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (m *MongoRepository) IncrementRetryCount(ctx context.Context, eventID string) error {
	collection := m.client.Database(m.database).Collection(m.collection)
	filter := bson.M{"id": eventID}
	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
		"$inc": bson.M{"retry_count": 1},
	}
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}
