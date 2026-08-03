package postgres

import (
	"context"
	"database/sql"
	"errors"

	"audit_service/internal/domain"
	"audit_service/internal/service"
)

var _ service.BlockRepository = (*BlockRepository)(nil)

type BlockRepository struct {
	db *sql.DB
}

func NewBlockRepository(db *sql.DB) *BlockRepository {
	return &BlockRepository{db: db}
}

func (r *BlockRepository) Append(ctx context.Context, block domain.Block) (int64, error) {
	const q = `
        INSERT INTO blocks (prev_hash, merkle_root, hash, created_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id`
	var id int64
	if err := r.db.QueryRowContext(ctx, q, block.PrevHash, block.MerkleRoot, block.Hash, block.CreatedAt).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *BlockRepository) Latest(ctx context.Context) (domain.Block, bool, error) {
	const q = `
        SELECT id, prev_hash, merkle_root, hash, created_at
        FROM blocks
        ORDER BY id DESC
        LIMIT 1`
	var b domain.Block
	err := r.db.QueryRowContext(ctx, q).Scan(&b.ID, &b.PrevHash, &b.MerkleRoot, &b.Hash, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Block{}, false, nil
	}
	if err != nil {
		return domain.Block{}, false, err
	}
	return b, true, nil
}

func (r *BlockRepository) GetByID(ctx context.Context, id int64) (domain.Block, error) {
	const q = `
        SELECT id, prev_hash, merkle_root, hash, created_at
        FROM blocks
        WHERE id = $1`
	var b domain.Block
	err := r.db.QueryRowContext(ctx, q, id).Scan(&b.ID, &b.PrevHash, &b.MerkleRoot, &b.Hash, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Block{}, service.ErrNotFound
	}
	if err != nil {
		return domain.Block{}, err
	}
	return b, nil
}

func (r *BlockRepository) RecordsForBlock(ctx context.Context, id int64) ([]domain.AuditRecord, error) {
	const q = `
        SELECT event_id, mdm_id, candidate_id, event_type, event_ts, hash, block_id
        FROM records
        WHERE block_id = $1
        ORDER BY created_at, event_id`
	rows, err := r.db.QueryContext(ctx, q, id)
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
			bid := blockID.Int64
			rec.BlockID = &bid
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
