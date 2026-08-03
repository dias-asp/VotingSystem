package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"audit_service/internal/domain"
)

type BlockRepository interface {
	Append(ctx context.Context, block domain.Block) (int64, error)
	Latest(ctx context.Context) (domain.Block, bool, error)
	GetByID(ctx context.Context, id int64) (domain.Block, error)
	RecordsForBlock(ctx context.Context, id int64) ([]domain.AuditRecord, error)
}

type BlockBuilder struct {
	records   RecordRepository
	blocks    BlockRepository
	batchSize int
	now       func() time.Time
}

func NewBlockBuilder(records RecordRepository, blocks BlockRepository, batchSize int) *BlockBuilder {
	return &BlockBuilder{records: records, blocks: blocks, batchSize: batchSize, now: time.Now}
}

func (b *BlockBuilder) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := b.buildOnce(ctx); err != nil {
				slog.Error("build block", "err", err)
			}
		}
	}
}

func (b *BlockBuilder) buildOnce(ctx context.Context) error {
	pending, err := b.records.PendingBatch(ctx, b.batchSize)
	if err != nil {
		return fmt.Errorf("pending batch: %w", err)
	}
	if len(pending) == 0 {
		return nil
	}

	leaves := make([][]byte, len(pending))
	ids := make([]string, len(pending))
	for i, r := range pending {
		leaves[i] = r.Hash
		ids[i] = r.EventID
	}
	root := MerkleRoot(leaves)

	prev, ok, err := b.blocks.Latest(ctx)
	if err != nil {
		return fmt.Errorf("latest block: %w", err)
	}
	var prevHash []byte
	if ok {
		prevHash = prev.Hash
	}

	// Postgres TIMESTAMPTZ rounds to microsecond, so truncate before
	// hashing — otherwise the round-tripped timestamp recomputes a
	// different block hash and ChainVerifier reports a mismatch.
	createdAt := b.now().UTC().Truncate(time.Microsecond)
	block := domain.Block{
		PrevHash:   prevHash,
		MerkleRoot: root,
		Hash:       blockHash(prevHash, root, createdAt),
		CreatedAt:  createdAt,
		RecordIDs:  ids,
	}

	id, err := b.blocks.Append(ctx, block)
	if err != nil {
		return fmt.Errorf("append block: %w", err)
	}
	if err := b.records.AssignBlock(ctx, ids, id); err != nil {
		return fmt.Errorf("assign block: %w", err)
	}
	slog.Info("block built", "block_id", id, "records", len(ids))
	return nil
}

// blockHash binds the previous hash, merkle root, and creation time so that
// (a) blocks with identical record sets across rebuilds remain distinct and
// (b) verifiers can walk the chain by recomputing each step.
func blockHash(prev, root []byte, createdAt time.Time) []byte {
	h := sha256.New()
	h.Write([]byte{0x02})
	h.Write(prev)
	h.Write(root)
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(createdAt.UnixNano()))
	h.Write(ts[:])
	return h.Sum(nil)
}
