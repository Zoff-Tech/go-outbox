INSERT INTO outbox_events (
    id,
    topic,
    payload,
    status,
    created_at,
    updated_at,
    sent_at,
    event_headers,
    retry_count
)
VALUES (
    '123e4567-e89b-12d3-a456-426614174000', -- Example UUID for id
    'example-topic',                       -- Example topic
    '\x48656c6c6f20576f726c64',            -- Example payload in hexadecimal (Hello World)
    'pending',                             -- Example status
    '2025-04-06 00:00:00',                 -- Example created_at timestamp
    '2025-04-06 00:00:00',                 -- Example updated_at timestamp
    NULL,                                  -- Example sent_at (NULL if not sent yet)
    '{"key1": "value1", "key2": "value2"}',-- Example event_headers in JSONB format
    0                                      -- Example retry_count
);