package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"audit_service/internal/domain"
)

type fakeRecordRepo struct {
	appended  []domain.AuditRecord
	pending   []domain.AuditRecord
	byID      map[string]domain.AuditRecord
	assigned  map[int64][]string
	appendErr error
}

func newFakeRecordRepo() *fakeRecordRepo {
	return &fakeRecordRepo{byID: map[string]domain.AuditRecord{}, assigned: map[int64][]string{}}
}

func (f *fakeRecordRepo) Append(_ context.Context, r domain.AuditRecord) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	f.appended = append(f.appended, r)
	f.byID[r.EventID] = r
	return nil
}

func (f *fakeRecordRepo) PendingBatch(_ context.Context, limit int) ([]domain.AuditRecord, error) {
	if limit > len(f.pending) {
		limit = len(f.pending)
	}
	out := f.pending[:limit]
	f.pending = f.pending[limit:]
	return out, nil
}

func (f *fakeRecordRepo) AssignBlock(_ context.Context, ids []string, blockID int64) error {
	f.assigned[blockID] = append(f.assigned[blockID], ids...)
	for _, id := range ids {
		r := f.byID[id]
		bid := blockID
		r.BlockID = &bid
		f.byID[id] = r
	}
	return nil
}

func (f *fakeRecordRepo) GetByEventID(_ context.Context, id string) (domain.AuditRecord, error) {
	if r, ok := f.byID[id]; ok {
		return r, nil
	}
	return domain.AuditRecord{}, ErrNotFound
}

func TestAuditor_Handle_AppendsWithComputedHash(t *testing.T) {
	repo := newFakeRecordRepo()
	a := NewAuditor(repo)
	ts := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	rec := domain.AuditRecord{
		EventID: "evt-1", MdmID: "mdm-1", CandidateID: "candA",
		Type: domain.EventVoteCast, Timestamp: ts,
	}
	if err := a.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(repo.appended) != 1 {
		t.Fatalf("appended = %d", len(repo.appended))
	}
	got := repo.appended[0]
	if len(got.Hash) == 0 {
		t.Fatalf("expected hash to be set")
	}
	expect := HashLeaf(CanonicalRecordBytes(rec))
	if string(got.Hash) != string(expect) {
		t.Fatalf("hash mismatch")
	}
}

func TestAuditor_Handle_RejectsInvalid(t *testing.T) {
	a := NewAuditor(newFakeRecordRepo())
	cases := []domain.AuditRecord{
		{EventID: "", MdmID: "m", Type: domain.EventVoteCast},
		{EventID: "e", MdmID: "", Type: domain.EventVoteCast},
		{EventID: "e", MdmID: "m", Type: domain.EventType("BOGUS")},
	}
	for i, c := range cases {
		if err := a.Handle(context.Background(), c); !errors.Is(err, ErrInvalidEvent) {
			t.Fatalf("case %d: want ErrInvalidEvent, got %v", i, err)
		}
	}
}

func TestAuditor_Handle_PropagatesAppendError(t *testing.T) {
	repo := newFakeRecordRepo()
	repo.appendErr = errors.New("boom")
	a := NewAuditor(repo)
	rec := domain.AuditRecord{EventID: "e", MdmID: "m", Type: domain.EventVoteCast, Timestamp: time.Now()}
	if err := a.Handle(context.Background(), rec); err == nil {
		t.Fatal("expected error")
	}
}
