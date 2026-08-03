package domain

import "time"

type EventType string

const (
	EventVoteCast      EventType = "VOTE_CAST"
	EventVoteCancelled EventType = "VOTE_CANCELLED"
	EventVoteChanged   EventType = "VOTE_CHANGED"
)

type VoteEvent struct {
	EventID     string
	PollID      string
	UserID      string
	CandidateID string
	Type        EventType
	Timestamp   time.Time
}

type AnonymizedEvent struct {
	EventID     string
	PollID      string
	MdmID       string
	CandidateID string
	Type        EventType
	Timestamp   time.Time
}

type Mapping struct {
	UserID string
	MdmID  string
}
