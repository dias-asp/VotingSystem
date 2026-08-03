DROP INDEX IF EXISTS votes_poll_user_created_idx;
ALTER TABLE votes DROP CONSTRAINT IF EXISTS votes_candidate_fkey;
ALTER TABLE votes DROP COLUMN IF EXISTS poll_id;

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_poll_fkey;
ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_pkey;
ALTER TABLE candidates DROP COLUMN IF EXISTS poll_id;
ALTER TABLE candidates ADD PRIMARY KEY (id);

DROP TABLE IF EXISTS polls;
