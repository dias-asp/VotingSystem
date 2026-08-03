package service

import (
	"context"
	"testing"
	"time"

	"audit_service/internal/domain"
)

type proverBlockRepo struct {
	blocks  map[int64]domain.Block
	records map[int64][]domain.AuditRecord
}

func (p *proverBlockRepo) Append(context.Context, domain.Block) (int64, error) {
	return 0, ErrNotImplemented
}
func (p *proverBlockRepo) Latest(context.Context) (domain.Block, bool, error) {
	return domain.Block{}, false, nil
}
func (p *proverBlockRepo) GetByID(_ context.Context, id int64) (domain.Block, error) {
	if b, ok := p.blocks[id]; ok {
		return b, nil
	}
	return domain.Block{}, ErrNotFound
}
func (p *proverBlockRepo) RecordsForBlock(_ context.Context, id int64) ([]domain.AuditRecord, error) {
	return p.records[id], nil
}

func TestProver_Proof_RoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	records := []domain.AuditRecord{mkRecord("e1", now), mkRecord("e2", now.Add(time.Second)), mkRecord("e3", now.Add(2*time.Second))}
	blockID := int64(1)
	for i := range records {
		bid := blockID
		records[i].BlockID = &bid
	}
	root := MerkleRoot([][]byte{records[0].Hash, records[1].Hash, records[2].Hash})
	block := domain.Block{ID: blockID, MerkleRoot: root, Hash: []byte("h"), CreatedAt: now}

	rrepo := newFakeRecordRepo()
	for _, r := range records {
		rrepo.byID[r.EventID] = r
	}
	brepo := &proverBlockRepo{
		blocks:  map[int64]domain.Block{blockID: block},
		records: map[int64][]domain.AuditRecord{blockID: records},
	}

	p := NewProver(brepo, rrepo)
	proof, err := p.Proof(context.Background(), "e2")
	if err != nil {
		t.Fatalf("Proof: %v", err)
	}
	if proof.BlockID != blockID {
		t.Fatalf("block id mismatch")
	}
	if !p.Verify(proof) {
		t.Fatalf("proof verification failed")
	}
}

func TestProver_Proof_PendingRecord_NotFound(t *testing.T) {
	rrepo := newFakeRecordRepo()
	rrepo.byID["pending"] = domain.AuditRecord{EventID: "pending"}
	brepo := &proverBlockRepo{}

	p := NewProver(brepo, rrepo)
	if _, err := p.Proof(context.Background(), "pending"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
