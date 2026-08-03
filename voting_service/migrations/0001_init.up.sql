CREATE TABLE IF NOT EXISTS candidates (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS votes (
    event_id     TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    candidate_id TEXT NOT NULL REFERENCES candidates(id),
    event_type   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS votes_user_created_idx ON votes (user_id, created_at DESC);
