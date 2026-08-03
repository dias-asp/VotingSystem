CREATE TABLE IF NOT EXISTS outbox (
    event_id       TEXT PRIMARY KEY REFERENCES votes(event_id) ON DELETE CASCADE,
    aggregate_key  TEXT NOT NULL,
    payload        BYTEA NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    dispatched_at  TIMESTAMPTZ,
    attempts       INT NOT NULL DEFAULT 0,
    last_error     TEXT
);

CREATE INDEX IF NOT EXISTS outbox_pending_idx
    ON outbox (created_at)
    WHERE dispatched_at IS NULL;
