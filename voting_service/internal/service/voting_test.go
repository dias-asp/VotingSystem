package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"voting_service/internal/domain"
)

const testPoll = "00000000-0000-0000-0000-000000000001"

type fakeRepo struct {
	candidates map[string]bool // key = pollID|candidateID
	saved      []domain.Vote
	saveErr    error
	tallyErr   error
}

func newFakeRepo(candidates ...string) *fakeRepo {
	r := &fakeRepo{candidates: map[string]bool{}}
	for _, c := range candidates {
		r.candidates[testPoll+"|"+c] = true
	}
	return r
}

func (r *fakeRepo) Record(_ context.Context, v domain.Vote) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, v)
	return nil
}

func (r *fakeRepo) LatestByUser(_ context.Context, pollID, userID string) (domain.Vote, error) {
	for i := len(r.saved) - 1; i >= 0; i-- {
		if r.saved[i].UserID == userID && r.saved[i].PollID == pollID {
			return r.saved[i], nil
		}
	}
	return domain.Vote{}, ErrNoVote
}

func (r *fakeRepo) CandidateExists(_ context.Context, pollID, id string) (bool, error) {
	return r.candidates[pollID+"|"+id], nil
}

func (r *fakeRepo) Tally(_ context.Context, pollID string) (map[string]int, error) {
	if r.tallyErr != nil {
		return nil, r.tallyErr
	}
	latest := map[string]domain.Vote{}
	for _, v := range r.saved {
		if v.PollID != pollID {
			continue
		}
		latest[v.UserID] = v
	}
	out := map[string]int{}
	for _, v := range latest {
		if v.Type == domain.EventVoteCancelled {
			continue
		}
		out[v.CandidateID]++
	}
	return out, nil
}

type fakePolls struct {
	polls map[string]domain.Poll
}

func newFakePolls() *fakePolls {
	return &fakePolls{polls: map[string]domain.Poll{
		testPoll: {ID: testPoll, Title: "Demo", Status: domain.PollActive},
	}}
}

func (p *fakePolls) CreatePoll(_ context.Context, poll domain.Poll) error {
	if _, ok := p.polls[poll.ID]; ok {
		return ErrPollExists
	}
	p.polls[poll.ID] = poll
	return nil
}

func (p *fakePolls) GetPoll(_ context.Context, id string) (domain.Poll, error) {
	v, ok := p.polls[id]
	if !ok {
		return domain.Poll{}, ErrPollNotFound
	}
	return v, nil
}

func (p *fakePolls) ListPolls(_ context.Context) ([]domain.Poll, error) {
	out := make([]domain.Poll, 0, len(p.polls))
	for _, v := range p.polls {
		out = append(out, v)
	}
	return out, nil
}

func (p *fakePolls) UpdatePollStatus(_ context.Context, id string, st domain.PollStatus) error {
	v, ok := p.polls[id]
	if !ok {
		return ErrPollNotFound
	}
	v.Status = st
	p.polls[id] = v
	return nil
}

func (p *fakePolls) AddCandidate(_ context.Context, c domain.Candidate) error { return nil }
func (p *fakePolls) ListCandidates(_ context.Context, _ string) ([]domain.Candidate, error) {
	return nil, nil
}

func newVoting(repo *fakeRepo) *Voting {
	v := NewVoting(repo, newFakePolls())
	v.now = func() time.Time { return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC) }
	return v
}

func req(user, cand string) domain.VoteRequest {
	return domain.VoteRequest{PollID: testPoll, UserID: user, CandidateID: cand}
}

func TestVoting_Cast_HappyPath(t *testing.T) {
	repo := newFakeRepo("candA")
	v := newVoting(repo)

	if err := v.Cast(context.Background(), req("u1", "candA")); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("expected 1 saved, got %d", len(repo.saved))
	}
	got := repo.saved[0]
	if got.Type != domain.EventVoteCast || got.CandidateID != "candA" || got.UserID != "u1" || got.PollID != testPoll {
		t.Fatalf("unexpected event: %+v", got)
	}
	if got.EventID == "" {
		t.Fatalf("EventID should be generated")
	}
}

func TestVoting_Cast_RejectsUnknownCandidate(t *testing.T) {
	v := newVoting(newFakeRepo("candA"))
	err := v.Cast(context.Background(), req("u1", "ghost"))
	if !errors.Is(err, ErrCandidateNotFound) {
		t.Fatalf("want ErrCandidateNotFound, got %v", err)
	}
}

func TestVoting_Cast_BlocksSecondCast(t *testing.T) {
	repo := newFakeRepo("candA")
	v := newVoting(repo)
	_ = v.Cast(context.Background(), req("u1", "candA"))

	err := v.Cast(context.Background(), req("u1", "candA"))
	if !errors.Is(err, ErrAlreadyVoted) {
		t.Fatalf("want ErrAlreadyVoted, got %v", err)
	}
}

func TestVoting_Cancel_RequiresActiveVote(t *testing.T) {
	v := newVoting(newFakeRepo("candA"))
	if err := v.Cancel(context.Background(), testPoll, "ghost"); !errors.Is(err, ErrNoActiveVote) {
		t.Fatalf("want ErrNoActiveVote on fresh user, got %v", err)
	}
}

func TestVoting_FullFlow_CastCancelChange(t *testing.T) {
	repo := newFakeRepo("candA", "candB")
	v := newVoting(repo)
	ctx := context.Background()

	if err := v.Cast(ctx, req("u1", "candA")); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	if err := v.Cancel(ctx, testPoll, "u1"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := v.Cancel(ctx, testPoll, "u1"); !errors.Is(err, ErrNoActiveVote) {
		t.Fatalf("double cancel should fail, got %v", err)
	}
	if err := v.Change(ctx, req("u1", "candB")); err != nil {
		t.Fatalf("Change: %v", err)
	}

	results, err := v.Results(ctx, testPoll)
	if err != nil {
		t.Fatalf("Results: %v", err)
	}
	if results["candA"] != 0 || results["candB"] != 1 {
		t.Fatalf("expected candA=0 candB=1, got %v", results)
	}

	types := []domain.EventType{}
	for _, e := range repo.saved {
		types = append(types, e.Type)
	}
	want := []domain.EventType{domain.EventVoteCast, domain.EventVoteCancelled, domain.EventVoteChanged}
	if len(types) != 3 || types[0] != want[0] || types[1] != want[1] || types[2] != want[2] {
		t.Fatalf("event sequence wrong: %v", types)
	}
}

func TestVoting_Change_RequiresPriorCancel(t *testing.T) {
	repo := newFakeRepo("candA", "candB")
	v := newVoting(repo)
	ctx := context.Background()

	_ = v.Cast(ctx, req("u1", "candA"))
	err := v.Change(ctx, req("u1", "candB"))
	if !errors.Is(err, ErrCannotChange) {
		t.Fatalf("Change without cancel should fail, got %v", err)
	}
}

func TestVoting_RecordFailureBubblesUp(t *testing.T) {
	repo := newFakeRepo("candA")
	repo.saveErr = errors.New("db down")
	v := newVoting(repo)

	err := v.Cast(context.Background(), req("u1", "candA"))
	if err == nil {
		t.Fatalf("expected error when Record fails")
	}
}

func TestVoting_RejectsEmptyFields(t *testing.T) {
	v := newVoting(newFakeRepo("candA"))
	if err := v.Cast(context.Background(), req("", "candA")); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest for empty user, got %v", err)
	}
	if err := v.Cancel(context.Background(), testPoll, ""); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest for empty cancel, got %v", err)
	}
}
