CREATE TABLE IF NOT EXISTS polls (
    id          UUID PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO polls (id, title, description, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Poll', 'Default seed poll', 'active')
ON CONFLICT (id) DO NOTHING;

ALTER TABLE votes DROP CONSTRAINT IF EXISTS votes_candidate_id_fkey;

ALTER TABLE candidates ADD COLUMN IF NOT EXISTS poll_id UUID;
UPDATE candidates SET poll_id = '00000000-0000-0000-0000-000000000001' WHERE poll_id IS NULL;
ALTER TABLE candidates ALTER COLUMN poll_id SET NOT NULL;
ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_pkey;
ALTER TABLE candidates ADD PRIMARY KEY (poll_id, id);
ALTER TABLE candidates
    ADD CONSTRAINT candidates_poll_fkey FOREIGN KEY (poll_id) REFERENCES polls(id) ON DELETE CASCADE;

ALTER TABLE votes ADD COLUMN IF NOT EXISTS poll_id UUID;
UPDATE votes SET poll_id = '00000000-0000-0000-0000-000000000001' WHERE poll_id IS NULL;
ALTER TABLE votes ALTER COLUMN poll_id SET NOT NULL;
ALTER TABLE votes
    ADD CONSTRAINT votes_candidate_fkey FOREIGN KEY (poll_id, candidate_id) REFERENCES candidates(poll_id, id);

DROP INDEX IF EXISTS votes_user_created_idx;
CREATE INDEX IF NOT EXISTS votes_poll_user_created_idx ON votes (poll_id, user_id, created_at DESC);
