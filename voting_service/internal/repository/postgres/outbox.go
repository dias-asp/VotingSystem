package postgres

import (
	"context"
	"database/sql"

	"voting_service/internal/outbox"
)

var _ outbox.Store = (*OutboxRepository)(nil)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Pending(ctx context.Context, limit int) ([]outbox.Record, error) {
	const q = `
        SELECT event_id, aggregate_key, payload
        FROM outbox
        WHERE dispatched_at IS NULL
        ORDER BY created_at
        LIMIT $1`
	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []outbox.Record
	for rows.Next() {
		var rec outbox.Record
		if err := rows.Scan(&rec.EventID, &rec.AggregateKey, &rec.Payload); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *OutboxRepository) MarkDispatched(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE outbox
        SET dispatched_at = now(), last_error = NULL
        WHERE event_id = $1`, eventID)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID, msg string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE outbox
        SET attempts = attempts + 1, last_error = $2
        WHERE event_id = $1`, eventID, msg)
	return err
}
