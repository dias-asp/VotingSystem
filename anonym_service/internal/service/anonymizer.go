package service

import (
	"context"
	"errors"
	"fmt"

	"anonym_service/internal/domain"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrInvalidEvent   = errors.New("invalid event")
)

type EventPublisher interface {
	Publish(ctx context.Context, event domain.AnonymizedEvent) error
}

type MappingRepository interface {
	GetOrCreate(ctx context.Context, userID string) (string, error)
	Get(ctx context.Context, userID string) (string, bool, error)
}

type Anonymizer struct {
	repo      MappingRepository
	publisher EventPublisher
}

func NewAnonymizer(repo MappingRepository, publisher EventPublisher) *Anonymizer {
	return &Anonymizer{repo: repo, publisher: publisher}
}

func (a *Anonymizer) Handle(ctx context.Context, event domain.VoteEvent) error {
	if event.UserID == "" || event.EventID == "" {
		return ErrInvalidEvent
	}

	mdmID, err := a.repo.GetOrCreate(ctx, event.UserID)
	if err != nil {
		return fmt.Errorf("map user: %w", err)
	}

	out := domain.AnonymizedEvent{
		EventID:     event.EventID,
		PollID:      event.PollID,
		MdmID:       mdmID,
		CandidateID: event.CandidateID,
		Type:        event.Type,
		Timestamp:   event.Timestamp,
	}
	if err := a.publisher.Publish(ctx, out); err != nil {
		return fmt.Errorf("publish anonymized: %w", err)
	}
	return nil
}
