CREATE TABLE IF NOT EXISTS blocks (
    id          BIGSERIAL PRIMARY KEY,
    prev_hash   BYTEA,
    merkle_root BYTEA NOT NULL,
    hash        BYTEA NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS records (
    event_id     TEXT PRIMARY KEY,
    mdm_id       TEXT NOT NULL,
    candidate_id TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    event_ts     TIMESTAMPTZ NOT NULL,
    hash         BYTEA NOT NULL,
    block_id     BIGINT REFERENCES blocks(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS records_pending_idx ON records (created_at) WHERE block_id IS NULL;
CREATE INDEX IF NOT EXISTS records_block_idx ON records (block_id);

CREATE TABLE IF NOT EXISTS merkle_proofs (
    event_id   TEXT PRIMARY KEY REFERENCES records(event_id),
    block_id   BIGINT NOT NULL REFERENCES blocks(id),
    proof      JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
