package wire

import (
	"encoding/json"
	"time"

	"voting_service/internal/domain"
)

type VoteEventDTO struct {
	EventID     string    `json:"event_id"`
	PollID      string    `json:"poll_id"`
	UserID      string    `json:"user_id"`
	CandidateID string    `json:"candidate_id"`
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
}

func MarshalVoteEvent(v domain.Vote) ([]byte, error) {
	return json.Marshal(VoteEventDTO{
		EventID:     v.EventID,
		PollID:      v.PollID,
		UserID:      v.UserID,
		CandidateID: v.CandidateID,
		Type:        string(v.Type),
		Timestamp:   v.Timestamp,
	})
}
