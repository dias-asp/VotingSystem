package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"audit_service/internal/domain"
	"audit_service/internal/service"
)

var _ service.RecordRepository = (*RecordRepository)(nil)

type RecordRepository struct {
	db *sql.DB
}

func NewRecordRepository(db *sql.DB) *RecordRepository {
	return &RecordRepository{db: db}
}

func (r *RecordRepository) Append(ctx context.Context, record domain.AuditRecord) error {
	const q = `
        INSERT INTO records (event_id, mdm_id, candidate_id, event_type, event_ts, hash)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (event_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, q,
		record.EventID,
		record.MdmID,
		record.CandidateID,
		string(record.Type),
		record.Timestamp,
		record.Hash,
	)
	return err
}

func (r *RecordRepository) PendingBatch(ctx context.Context, limit int) ([]domain.AuditRecord, error) {
	const q = `
        SELECT event_id, mdm_id, candidate_id, event_type, event_ts, hash, block_id
        FROM records
        WHERE block_id IS NULL
        ORDER BY created_at, event_id
        LIMIT $1`
	rows, err := r.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AuditRecord
	for rows.Next() {
		var (
			rec     domain.AuditRecord
			etype   string
			blockID sql.NullInt64
		)
		if err := rows.Scan(&rec.EventID, &rec.MdmID, &rec.CandidateID, &etype, &rec.Timestamp, &rec.Hash, &blockID); err != nil {
			return nil, err
		}
		rec.Type = domain.EventType(etype)
		if blockID.Valid {
			id := blockID.Int64
			rec.BlockID = &id
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *RecordRepository) AssignBlock(ctx context.Context, eventIDs []string, blockID int64) error {
	if len(eventIDs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const q = `UPDATE records SET block_id = $1 WHERE event_id = $2 AND block_id IS NULL`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range eventIDs {
		res, err := stmt.ExecContext(ctx, blockID, id)
		if err != nil {
			return fmt.Errorf("assign %s: %w", id, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("assign %s: record missing or already in a block", id)
		}
	}
	return tx.Commit()
}

func (r *RecordRepository) GetByEventID(ctx context.Context, eventID string) (domain.AuditRecord, error) {
	const q = `
        SELECT event_id, mdm_id, candidate_id, event_type, event_ts, hash, block_id
        FROM records
        WHERE event_id = $1`
	var (
		rec     domain.AuditRecord
		etype   string
		blockID sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, q, eventID).Scan(
		&rec.EventID, &rec.MdmID, &rec.CandidateID, &etype, &rec.Timestamp, &rec.Hash, &blockID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuditRecord{}, service.ErrNotFound
	}
	if err != nil {
		return domain.AuditRecord{}, err
	}
	rec.Type = domain.EventType(etype)
	if blockID.Valid {
		id := blockID.Int64
		rec.BlockID = &id
	}
	return rec, nil
}
