package service

import (
	"context"
	"errors"
	"fmt"

	"audit_service/internal/domain"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrInvalidEvent   = errors.New("invalid event")
	ErrNotFound       = errors.New("not found")
)

type RecordRepository interface {
	Append(ctx context.Context, record domain.AuditRecord) error
	PendingBatch(ctx context.Context, limit int) ([]domain.AuditRecord, error)
	AssignBlock(ctx context.Context, eventIDs []string, blockID int64) error
	GetByEventID(ctx context.Context, eventID string) (domain.AuditRecord, error)
}

type Auditor struct {
	records RecordRepository
}

func NewAuditor(records RecordRepository) *Auditor {
	return &Auditor{records: records}
}

func (a *Auditor) Handle(ctx context.Context, event domain.AuditRecord) error {
	if event.EventID == "" || event.MdmID == "" {
		return ErrInvalidEvent
	}
	switch event.Type {
	case domain.EventVoteCast, domain.EventVoteCancelled, domain.EventVoteChanged:
	default:
		return ErrInvalidEvent
	}

	event.Hash = HashLeaf(CanonicalRecordBytes(event))
	if err := a.records.Append(ctx, event); err != nil {
		return fmt.Errorf("append record: %w", err)
	}
	return nil
}

// CanonicalRecordBytes is the deterministic serialization used for the leaf
// hash. Any party with the same record fields can recompute it; that's what
// makes the Merkle proof verifiable off-line.
func CanonicalRecordBytes(r domain.AuditRecord) []byte {
	return []byte(fmt.Sprintf(
		"%s|%s|%s|%s|%s",
		r.EventID,
		r.MdmID,
		r.CandidateID,
		r.Type,
		r.Timestamp.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"),
	))
}
