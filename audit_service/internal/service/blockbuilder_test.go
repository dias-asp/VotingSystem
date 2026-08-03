package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"audit_service/internal/domain"
)

type fakeBlockRepo struct {
	blocks []domain.Block
	nextID int64
}

func newFakeBlockRepo() *fakeBlockRepo { return &fakeBlockRepo{nextID: 1} }

func (f *fakeBlockRepo) Append(_ context.Context, b domain.Block) (int64, error) {
	b.ID = f.nextID
	f.nextID++
	f.blocks = append(f.blocks, b)
	return b.ID, nil
}

func (f *fakeBlockRepo) Latest(_ context.Context) (domain.Block, bool, error) {
	if len(f.blocks) == 0 {
		return domain.Block{}, false, nil
	}
	return f.blocks[len(f.blocks)-1], true, nil
}

func (f *fakeBlockRepo) GetByID(_ context.Context, id int64) (domain.Block, error) {
	for _, b := range f.blocks {
		if b.ID == id {
			return b, nil
		}
	}
	return domain.Block{}, ErrNotFound
}

func (f *fakeBlockRepo) RecordsForBlock(_ context.Context, _ int64) ([]domain.AuditRecord, error) {
	return nil, nil
}

func mkRecord(eventID string, ts time.Time) domain.AuditRecord {
	r := domain.AuditRecord{
		EventID: eventID, MdmID: "mdm-" + eventID, CandidateID: "candA",
		Type: domain.EventVoteCast, Timestamp: ts,
	}
	r.Hash = HashLeaf(CanonicalRecordBytes(r))
	return r
}

func TestBlockBuilder_NoPending_NoBlock(t *testing.T) {
	records := newFakeRecordRepo()
	blocks := newFakeBlockRepo()
	bb := NewBlockBuilder(records, blocks, 10)
	if err := bb.buildOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(blocks.blocks) != 0 {
		t.Fatalf("expected no blocks, got %d", len(blocks.blocks))
	}
}

func TestBlockBuilder_PersistsBlockAndAssigns(t *testing.T) {
	records := newFakeRecordRepo()
	blocks := newFakeBlockRepo()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	r1 := mkRecord("e1", now)
	r2 := mkRecord("e2", now.Add(time.Second))
	records.pending = []domain.AuditRecord{r1, r2}
	records.byID[r1.EventID] = r1
	records.byID[r2.EventID] = r2

	bb := NewBlockBuilder(records, blocks, 10)
	bb.now = func() time.Time { return now }

	if err := bb.buildOnce(context.Background()); err != nil {
		t.Fatalf("buildOnce: %v", err)
	}
	if len(blocks.blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks.blocks))
	}
	b := blocks.blocks[0]
	wantRoot := MerkleRoot([][]byte{r1.Hash, r2.Hash})
	if !bytes.Equal(b.MerkleRoot, wantRoot) {
		t.Fatal("merkle root mismatch")
	}
	if b.PrevHash != nil {
		t.Fatalf("first block prev_hash should be nil, got %x", b.PrevHash)
	}
	if got := records.assigned[b.ID]; len(got) != 2 || got[0] != "e1" || got[1] != "e2" {
		t.Fatalf("assignment mismatch: %v", got)
	}
}

func TestBlockBuilder_ChainsPrevHash(t *testing.T) {
	records := newFakeRecordRepo()
	blocks := newFakeBlockRepo()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	r1 := mkRecord("e1", now)
	records.pending = []domain.AuditRecord{r1}
	records.byID[r1.EventID] = r1

	bb := NewBlockBuilder(records, blocks, 10)
	bb.now = func() time.Time { return now }
	if err := bb.buildOnce(context.Background()); err != nil {
		t.Fatalf("buildOnce: %v", err)
	}

	r2 := mkRecord("e2", now.Add(time.Second))
	records.pending = []domain.AuditRecord{r2}
	records.byID[r2.EventID] = r2
	bb.now = func() time.Time { return now.Add(time.Second) }
	if err := bb.buildOnce(context.Background()); err != nil {
		t.Fatalf("buildOnce 2: %v", err)
	}

	if len(blocks.blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks.blocks))
	}
	if !bytes.Equal(blocks.blocks[1].PrevHash, blocks.blocks[0].Hash) {
		t.Fatalf("prev_hash chain broken")
	}
}
