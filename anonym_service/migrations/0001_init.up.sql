CREATE TABLE IF NOT EXISTS user_mdm_mapping (
    user_id    TEXT PRIMARY KEY,
    mdm_id     TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
