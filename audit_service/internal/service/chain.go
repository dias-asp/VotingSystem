package service

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ChainStatus is the result of walking the block chain end-to-end.
type ChainStatus struct {
	Valid       bool
	BlockCount  int
	LastBlockID int64
	LastHash    []byte
	// Error holds a human-readable description of the first integrity
	// violation found (empty if Valid).
	Error string
}

// ChainVerifier walks every block in id order, recomputing each block hash
// from prev_hash + merkle_root + created_at and asserting that prev_hash
// matches the previous block's hash.
type ChainVerifier struct {
	db *sql.DB
}

func NewChainVerifier(db *sql.DB) *ChainVerifier {
	return &ChainVerifier{db: db}
}

func (c *ChainVerifier) Verify(ctx context.Context) (ChainStatus, error) {
	const q = `SELECT id, prev_hash, merkle_root, hash, created_at FROM blocks ORDER BY id`
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return ChainStatus{}, err
	}
	defer rows.Close()

	var (
		status   ChainStatus
		expected []byte
	)
	for rows.Next() {
		var (
			id        int64
			prev      []byte
			root      []byte
			hash      []byte
			createdAt time.Time
		)
		if err := rows.Scan(&id, &prev, &root, &hash, &createdAt); err != nil {
			return ChainStatus{}, err
		}

		if !bytes.Equal(prev, expected) {
			status.Error = fmt.Sprintf("block %d prev_hash mismatch", id)
			return status, nil
		}
		recomputed := blockHash(prev, root, createdAt.UTC())
		if !bytes.Equal(recomputed, hash) {
			status.Error = fmt.Sprintf("block %d hash mismatch", id)
			return status, nil
		}

		expected = hash
		status.BlockCount++
		status.LastBlockID = id
		status.LastHash = hash
	}
	if err := rows.Err(); err != nil {
		return ChainStatus{}, err
	}
	status.Valid = true
	return status, nil
}
