-- Idempotency table: stores event IDs that have already been processed
-- to prevent duplicate notifications.
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     VARCHAR(100) PRIMARY KEY,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);
