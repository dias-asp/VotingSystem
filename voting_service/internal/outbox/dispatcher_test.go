package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeStore struct {
	mu         sync.Mutex
	pending    []Record
	dispatched []string
	failed     map[string]string
}

func (s *fakeStore) Pending(_ context.Context, limit int) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) == 0 {
		return nil, nil
	}
	n := len(s.pending)
	if n > limit {
		n = limit
	}
	out := make([]Record, n)
	copy(out, s.pending[:n])
	s.pending = s.pending[n:]
	return out, nil
}

func (s *fakeStore) MarkDispatched(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatched = append(s.dispatched, id)
	return nil
}

func (s *fakeStore) MarkFailed(_ context.Context, id, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed == nil {
		s.failed = map[string]string{}
	}
	s.failed[id] = msg
	return nil
}

type fakePublisher struct {
	mu       sync.Mutex
	sent     []Record
	failNext bool
}

func (p *fakePublisher) Publish(_ context.Context, key string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failNext {
		p.failNext = false
		return errors.New("kafka down")
	}
	p.sent = append(p.sent, Record{AggregateKey: key, Payload: payload})
	return nil
}

func TestDispatcher_TickPublishesAndMarks(t *testing.T) {
	store := &fakeStore{pending: []Record{
		{EventID: "e1", AggregateKey: "u1", Payload: []byte(`{"a":1}`)},
		{EventID: "e2", AggregateKey: "u2", Payload: []byte(`{"a":2}`)},
	}}
	pub := &fakePublisher{}
	d := NewDispatcher(store, pub, 10, time.Millisecond)

	d.tick(context.Background())

	if len(pub.sent) != 2 {
		t.Fatalf("want 2 published, got %d", len(pub.sent))
	}
	if len(store.dispatched) != 2 {
		t.Fatalf("want 2 dispatched, got %d", len(store.dispatched))
	}
}

func TestDispatcher_FailedPublishMarksFailed(t *testing.T) {
	store := &fakeStore{pending: []Record{
		{EventID: "e1", AggregateKey: "u1", Payload: []byte(`x`)},
	}}
	pub := &fakePublisher{failNext: true}
	d := NewDispatcher(store, pub, 10, time.Millisecond)

	d.tick(context.Background())

	if len(store.dispatched) != 0 {
		t.Fatalf("nothing should be marked dispatched on publish failure")
	}
	if msg, ok := store.failed["e1"]; !ok || msg == "" {
		t.Fatalf("e1 should be marked failed with reason, got %q", msg)
	}
}

func TestDispatcher_RunStopsOnContext(t *testing.T) {
	store := &fakeStore{}
	pub := &fakePublisher{}
	d := NewDispatcher(store, pub, 10, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("Run did not stop on cancel")
	}
}
