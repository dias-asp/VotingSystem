package domain

import "time"

type EventType string

const (
	EventVoteCast      EventType = "VOTE_CAST"
	EventVoteCancelled EventType = "VOTE_CANCELLED"
	EventVoteChanged   EventType = "VOTE_CHANGED"
)

type PollStatus string

const (
	PollDraft  PollStatus = "draft"
	PollActive PollStatus = "active"
	PollClosed PollStatus = "closed"
)

const DefaultPollID = "00000000-0000-0000-0000-000000000001"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type Poll struct {
	ID          string
	Title       string
	Description string
	Status      PollStatus
	CreatedAt   time.Time
}

type Candidate struct {
	PollID string
	ID     string
	Name   string
}

type Vote struct {
	EventID     string
	PollID      string
	UserID      string
	CandidateID string
	Type        EventType
	Timestamp   time.Time
}

type VoteRequest struct {
	PollID      string
	UserID      string
	CandidateID string
}
