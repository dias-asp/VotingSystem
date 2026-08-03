package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"voting_service/internal/domain"
)

var (
	ErrNotImplemented    = errors.New("not implemented")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrPollNotFound      = errors.New("poll not found")
	ErrPollNotActive     = errors.New("poll not active")
	ErrPollExists        = errors.New("poll already exists")
	ErrCandidateNotFound = errors.New("candidate not found")
	ErrCandidateExists   = errors.New("candidate already exists")
	ErrAlreadyVoted      = errors.New("already voted")
	ErrNoActiveVote      = errors.New("no active vote")
	ErrCannotChange      = errors.New("cannot change vote without prior cancel")
	ErrNoVote            = errors.New("no vote")
)

type VoteRepository interface {
	Record(ctx context.Context, vote domain.Vote) error
	LatestByUser(ctx context.Context, pollID, userID string) (domain.Vote, error)
	CandidateExists(ctx context.Context, pollID, candidateID string) (bool, error)
	Tally(ctx context.Context, pollID string) (map[string]int, error)
}

type PollRepository interface {
	CreatePoll(ctx context.Context, poll domain.Poll) error
	GetPoll(ctx context.Context, id string) (domain.Poll, error)
	ListPolls(ctx context.Context) ([]domain.Poll, error)
	UpdatePollStatus(ctx context.Context, id string, status domain.PollStatus) error
	AddCandidate(ctx context.Context, c domain.Candidate) error
	ListCandidates(ctx context.Context, pollID string) ([]domain.Candidate, error)
}

type Clock func() time.Time

type Voting struct {
	repo  VoteRepository
	polls PollRepository
	now   Clock
}

func NewVoting(repo VoteRepository, polls PollRepository) *Voting {
	return &Voting{repo: repo, polls: polls, now: time.Now}
}

func (v *Voting) assertActivePoll(ctx context.Context, pollID string) error {
	if pollID == "" {
		return ErrInvalidRequest
	}
	p, err := v.polls.GetPoll(ctx, pollID)
	if err != nil {
		return err
	}
	if p.Status != domain.PollActive {
		return ErrPollNotActive
	}
	return nil
}

func (v *Voting) Cast(ctx context.Context, req domain.VoteRequest) error {
	if req.UserID == "" || req.CandidateID == "" || req.PollID == "" {
		return ErrInvalidRequest
	}
	if err := v.assertActivePoll(ctx, req.PollID); err != nil {
		return err
	}
	if err := v.assertCandidate(ctx, req.PollID, req.CandidateID); err != nil {
		return err
	}

	latest, err := v.repo.LatestByUser(ctx, req.PollID, req.UserID)
	switch {
	case errors.Is(err, ErrNoVote):
	case err != nil:
		return fmt.Errorf("latest by user: %w", err)
	default:
		if latest.Type == domain.EventVoteCast || latest.Type == domain.EventVoteChanged {
			return ErrAlreadyVoted
		}
		return ErrAlreadyVoted
	}

	return v.record(ctx, req.PollID, req.UserID, req.CandidateID, domain.EventVoteCast)
}

func (v *Voting) Cancel(ctx context.Context, pollID, userID string) error {
	if pollID == "" || userID == "" {
		return ErrInvalidRequest
	}
	if err := v.assertActivePoll(ctx, pollID); err != nil {
		return err
	}

	latest, err := v.repo.LatestByUser(ctx, pollID, userID)
	if errors.Is(err, ErrNoVote) {
		return ErrNoActiveVote
	}
	if err != nil {
		return fmt.Errorf("latest by user: %w", err)
	}
	if latest.Type != domain.EventVoteCast && latest.Type != domain.EventVoteChanged {
		return ErrNoActiveVote
	}

	return v.record(ctx, pollID, userID, latest.CandidateID, domain.EventVoteCancelled)
}

func (v *Voting) Change(ctx context.Context, req domain.VoteRequest) error {
	if req.UserID == "" || req.CandidateID == "" || req.PollID == "" {
		return ErrInvalidRequest
	}
	if err := v.assertActivePoll(ctx, req.PollID); err != nil {
		return err
	}
	if err := v.assertCandidate(ctx, req.PollID, req.CandidateID); err != nil {
		return err
	}

	latest, err := v.repo.LatestByUser(ctx, req.PollID, req.UserID)
	if errors.Is(err, ErrNoVote) {
		return ErrCannotChange
	}
	if err != nil {
		return fmt.Errorf("latest by user: %w", err)
	}
	if latest.Type != domain.EventVoteCancelled {
		return ErrCannotChange
	}

	return v.record(ctx, req.PollID, req.UserID, req.CandidateID, domain.EventVoteChanged)
}

func (v *Voting) Results(ctx context.Context, pollID string) (map[string]int, error) {
	if pollID == "" {
		return nil, ErrInvalidRequest
	}
	if _, err := v.polls.GetPoll(ctx, pollID); err != nil {
		return nil, err
	}
	results, err := v.repo.Tally(ctx, pollID)
	if err != nil {
		return nil, fmt.Errorf("tally: %w", err)
	}
	return results, nil
}

func (v *Voting) MyVote(ctx context.Context, pollID, userID string) (domain.Vote, error) {
	if pollID == "" || userID == "" {
		return domain.Vote{}, ErrInvalidRequest
	}
	return v.repo.LatestByUser(ctx, pollID, userID)
}

func (v *Voting) assertCandidate(ctx context.Context, pollID, candidateID string) error {
	exists, err := v.repo.CandidateExists(ctx, pollID, candidateID)
	if err != nil {
		return fmt.Errorf("candidate exists: %w", err)
	}
	if !exists {
		return ErrCandidateNotFound
	}
	return nil
}

func (v *Voting) record(ctx context.Context, pollID, userID, candidateID string, t domain.EventType) error {
	event := domain.Vote{
		EventID:     newUUID(),
		PollID:      pollID,
		UserID:      userID,
		CandidateID: candidateID,
		Type:        t,
		Timestamp:   v.now().UTC(),
	}
	if err := v.repo.Record(ctx, event); err != nil {
		return fmt.Errorf("record vote: %w", err)
	}
	return nil
}

type Polls struct {
	repo PollRepository
	now  Clock
}

func NewPolls(repo PollRepository) *Polls {
	return &Polls{repo: repo, now: time.Now}
}

type CreatePollInput struct {
	Title       string
	Description string
}

func (p *Polls) Create(ctx context.Context, in CreatePollInput) (domain.Poll, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return domain.Poll{}, ErrInvalidRequest
	}
	poll := domain.Poll{
		ID:          newUUID(),
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Status:      domain.PollDraft,
		CreatedAt:   p.now().UTC(),
	}
	if err := p.repo.CreatePoll(ctx, poll); err != nil {
		return domain.Poll{}, err
	}
	return poll, nil
}

func (p *Polls) List(ctx context.Context) ([]domain.Poll, error) {
	return p.repo.ListPolls(ctx)
}

func (p *Polls) Get(ctx context.Context, id string) (domain.Poll, error) {
	if id == "" {
		return domain.Poll{}, ErrInvalidRequest
	}
	return p.repo.GetPoll(ctx, id)
}

func (p *Polls) Candidates(ctx context.Context, pollID string) ([]domain.Candidate, error) {
	if pollID == "" {
		return nil, ErrInvalidRequest
	}
	if _, err := p.repo.GetPoll(ctx, pollID); err != nil {
		return nil, err
	}
	return p.repo.ListCandidates(ctx, pollID)
}

type AddCandidateInput struct {
	PollID string
	ID     string
	Name   string
}

func (p *Polls) AddCandidate(ctx context.Context, in AddCandidateInput) (domain.Candidate, error) {
	id := strings.TrimSpace(in.ID)
	name := strings.TrimSpace(in.Name)
	if in.PollID == "" || name == "" {
		return domain.Candidate{}, ErrInvalidRequest
	}
	poll, err := p.repo.GetPoll(ctx, in.PollID)
	if err != nil {
		return domain.Candidate{}, err
	}
	if poll.Status == domain.PollClosed {
		return domain.Candidate{}, ErrPollNotActive
	}
	if id == "" {
		id = newUUID()
	}
	c := domain.Candidate{PollID: in.PollID, ID: id, Name: name}
	if err := p.repo.AddCandidate(ctx, c); err != nil {
		return domain.Candidate{}, err
	}
	return c, nil
}

func (p *Polls) SetStatus(ctx context.Context, id string, status domain.PollStatus) error {
	if id == "" {
		return ErrInvalidRequest
	}
	switch status {
	case domain.PollDraft, domain.PollActive, domain.PollClosed:
	default:
		return ErrInvalidRequest
	}
	if _, err := p.repo.GetPoll(ctx, id); err != nil {
		return err
	}
	return p.repo.UpdatePollStatus(ctx, id, status)
}
