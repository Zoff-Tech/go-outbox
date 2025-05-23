CREATE TABLE outbox_events (
    id TEXT PRIMARY KEY,                     -- Matches the ID field (string)
    entity TEXT NOT NULL,                    -- Matches the Topic field (string)
    entity_type TEXT NOT NULL,               -- Matches the EntityType field (string)
    payload BYTEA NOT NULL,                  -- Matches the Payload field ([]byte)
    status TEXT NOT NULL DEFAULT 'pending',  -- Matches the Status field (Status enum)
    created_at TIMESTAMP NOT NULL,           -- Matches the CreatedAt field (time.Time)
    updated_at TIMESTAMP NOT NULL,           -- Matches the UpdatedAt field (time.Time)
    sent_at TIMESTAMP,                       -- Matches the SentAt field (time.Time, nullable)
    headers JSONB,                           -- Matches the EventHeaders field (map[string]string)
    retry_count INT NOT NULL DEFAULT 0,      -- Matches the RetryCount field (int)
    routing_key TEXT NOT NULL DEFAULT ''     -- Matches the RoutingKey field (string)
);