package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type Record struct {
	EventID      string
	AggregateKey string
	Payload      []byte
}

type Store interface {
	Pending(ctx context.Context, limit int) ([]Record, error)
	MarkDispatched(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID, msg string) error
}

type Publisher interface {
	Publish(ctx context.Context, key string, payload []byte) error
}

type Dispatcher struct {
	store     Store
	publisher Publisher
	batchSize int
	interval  time.Duration
}

func NewDispatcher(store Store, publisher Publisher, batchSize int, interval time.Duration) *Dispatcher {
	if batchSize <= 0 {
		batchSize = 100
	}
	if interval <= 0 {
		interval = 1 * time.Second
	}
	return &Dispatcher{
		store:     store,
		publisher: publisher,
		batchSize: batchSize,
		interval:  interval,
	}
}

func (d *Dispatcher) Run(ctx context.Context) error {
	t := time.NewTicker(d.interval)
	defer t.Stop()

	// Eagerly run once on start so an initial backlog drains without waiting a tick.
	d.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			d.tick(ctx)
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) {
	records, err := d.store.Pending(ctx, d.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("outbox: fetch pending", "err", err)
		}
		return
	}
	for _, rec := range records {
		if ctx.Err() != nil {
			return
		}
		if err := d.publisher.Publish(ctx, rec.AggregateKey, rec.Payload); err != nil {
			slog.Error("outbox: publish failed", "event_id", rec.EventID, "err", err)
			if mErr := d.store.MarkFailed(ctx, rec.EventID, err.Error()); mErr != nil {
				slog.Error("outbox: mark failed", "event_id", rec.EventID, "err", mErr)
			}
			continue
		}
		if err := d.store.MarkDispatched(ctx, rec.EventID); err != nil {
			slog.Error("outbox: mark dispatched", "event_id", rec.EventID, "err", err)
		}
	}
}
