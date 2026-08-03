package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"anonym_service/internal/domain"
)

type fakeRepo struct {
	mapping map[string]string
	err     error
	calls   int
}

func (f *fakeRepo) GetOrCreate(_ context.Context, userID string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if v, ok := f.mapping[userID]; ok {
		return v, nil
	}
	v := "mdm-" + userID
	f.mapping[userID] = v
	return v, nil
}

func (f *fakeRepo) Get(_ context.Context, userID string) (string, bool, error) {
	v, ok := f.mapping[userID]
	return v, ok, nil
}

type fakePublisher struct {
	last domain.AnonymizedEvent
	err  error
	n    int
}

func (p *fakePublisher) Publish(_ context.Context, event domain.AnonymizedEvent) error {
	p.n++
	p.last = event
	return p.err
}

func TestAnonymizer_Handle_PublishesAnonymizedEvent(t *testing.T) {
	repo := &fakeRepo{mapping: map[string]string{}}
	pub := &fakePublisher{}
	a := NewAnonymizer(repo, pub)

	ts := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	in := domain.VoteEvent{
		EventID:     "evt-1",
		UserID:      "user-1",
		CandidateID: "candA",
		Type:        domain.EventVoteCast,
		Timestamp:   ts,
	}
	if err := a.Handle(context.Background(), in); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if pub.n != 1 {
		t.Fatalf("publish calls = %d, want 1", pub.n)
	}
	got := pub.last
	if got.EventID != in.EventID || got.CandidateID != in.CandidateID ||
		got.Type != in.Type || !got.Timestamp.Equal(ts) {
		t.Fatalf("anonymized event mismatch: %+v", got)
	}
	if got.MdmID == "" || got.MdmID == in.UserID {
		t.Fatalf("MdmID not anonymized: %q", got.MdmID)
	}
}

func TestAnonymizer_Handle_ReusesExistingMapping(t *testing.T) {
	repo := &fakeRepo{mapping: map[string]string{"user-1": "fixed-mdm"}}
	pub := &fakePublisher{}
	a := NewAnonymizer(repo, pub)

	in := domain.VoteEvent{EventID: "e1", UserID: "user-1", CandidateID: "c", Type: domain.EventVoteCast}
	if err := a.Handle(context.Background(), in); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if pub.last.MdmID != "fixed-mdm" {
		t.Fatalf("expected reused mapping, got %q", pub.last.MdmID)
	}

	in2 := domain.VoteEvent{EventID: "e2", UserID: "user-1", CandidateID: "c2", Type: domain.EventVoteChanged}
	if err := a.Handle(context.Background(), in2); err != nil {
		t.Fatalf("Handle 2: %v", err)
	}
	if pub.last.MdmID != "fixed-mdm" {
		t.Fatalf("expected same mdm across events, got %q", pub.last.MdmID)
	}
}

func TestAnonymizer_Handle_RejectsInvalidEvent(t *testing.T) {
	a := NewAnonymizer(&fakeRepo{mapping: map[string]string{}}, &fakePublisher{})
	err := a.Handle(context.Background(), domain.VoteEvent{EventID: "", UserID: "u"})
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("want ErrInvalidEvent, got %v", err)
	}
}

func TestAnonymizer_Handle_PropagatesRepoError(t *testing.T) {
	repo := &fakeRepo{mapping: map[string]string{}, err: errors.New("boom")}
	pub := &fakePublisher{}
	a := NewAnonymizer(repo, pub)
	in := domain.VoteEvent{EventID: "e", UserID: "u", Type: domain.EventVoteCast}
	if err := a.Handle(context.Background(), in); err == nil {
		t.Fatalf("expected error")
	}
	if pub.n != 0 {
		t.Fatalf("publish should not be called on repo error, got %d", pub.n)
	}
}
