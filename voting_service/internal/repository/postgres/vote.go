package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"voting_service/internal/domain"
	"voting_service/internal/service"
	"voting_service/internal/wire"
)

var _ service.VoteRepository = (*VoteRepository)(nil)

type VoteRepository struct {
	db *sql.DB
}

func NewVoteRepository(db *sql.DB) *VoteRepository {
	return &VoteRepository{db: db}
}

// Record persists the vote and an outbox row in a single transaction so the
// downstream Kafka publish can never diverge from the local event log.
func (r *VoteRepository) Record(ctx context.Context, vote domain.Vote) error {
	payload, err := wire.MarshalVoteEvent(vote)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
        INSERT INTO votes (event_id, poll_id, user_id, candidate_id, event_type, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)`,
		vote.EventID, vote.PollID, vote.UserID, vote.CandidateID, string(vote.Type), vote.Timestamp,
	); err != nil {
		return fmt.Errorf("insert vote: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
        INSERT INTO outbox (event_id, aggregate_key, payload)
        VALUES ($1, $2, $3)`,
		vote.EventID, vote.UserID, payload,
	); err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *VoteRepository) LatestByUser(ctx context.Context, pollID, userID string) (domain.Vote, error) {
	const q = `
        SELECT event_id, poll_id, user_id, candidate_id, event_type, created_at
        FROM votes
        WHERE poll_id = $1 AND user_id = $2
        ORDER BY created_at DESC
        LIMIT 1`
	var (
		v  domain.Vote
		et string
	)
	err := r.db.QueryRowContext(ctx, q, pollID, userID).Scan(&v.EventID, &v.PollID, &v.UserID, &v.CandidateID, &et, &v.Timestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Vote{}, service.ErrNoVote
	}
	if err != nil {
		return domain.Vote{}, err
	}
	v.Type = domain.EventType(et)
	return v, nil
}

func (r *VoteRepository) CandidateExists(ctx context.Context, pollID, candidateID string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM candidates WHERE poll_id = $1 AND id = $2)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, pollID, candidateID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *VoteRepository) Tally(ctx context.Context, pollID string) (map[string]int, error) {
	const q = `
        WITH latest AS (
            SELECT DISTINCT ON (user_id)
                   user_id, candidate_id, event_type
            FROM votes
            WHERE poll_id = $1
            ORDER BY user_id, created_at DESC
        )
        SELECT candidate_id, COUNT(*)::bigint
        FROM latest
        WHERE event_type <> 'VOTE_CANCELLED'
        GROUP BY candidate_id`

	rows, err := r.db.QueryContext(ctx, q, pollID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var (
			candidate string
			count     int64
		)
		if err := rows.Scan(&candidate, &count); err != nil {
			return nil, err
		}
		out[candidate] = int(count)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
