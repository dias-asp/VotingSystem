package domain

import "time"

type EventType string

const (
	EventVoteCast      EventType = "VOTE_CAST"
	EventVoteCancelled EventType = "VOTE_CANCELLED"
	EventVoteChanged   EventType = "VOTE_CHANGED"
)

type AuditRecord struct {
	EventID     string
	MdmID       string
	CandidateID string
	Type        EventType
	Timestamp   time.Time
	Hash        []byte
	BlockID     *int64
}
