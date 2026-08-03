package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"anonym_service/internal/domain"
)

type EventHandler interface {
	Handle(ctx context.Context, event domain.VoteEvent) error
}

type voteEventDTO struct {
	EventID     string    `json:"event_id"`
	PollID      string    `json:"poll_id"`
	UserID      string    `json:"user_id"`
	CandidateID string    `json:"candidate_id"`
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
}

type Consumer struct {
	reader  *kafka.Reader
	handler EventHandler
}

func NewConsumer(brokers []string, topic, groupID string, handler EventHandler) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,
		MaxBytes:    10 << 20,
		StartOffset: kafka.FirstOffset,
	})
	return &Consumer{reader: r, handler: handler}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		var dto voteEventDTO
		if err := json.Unmarshal(m.Value, &dto); err != nil {
			slog.Error("decode vote event", "err", err, "offset", m.Offset)
			if cerr := c.reader.CommitMessages(ctx, m); cerr != nil {
				slog.Error("commit after decode error", "err", cerr)
			}
			continue
		}

		evt := domain.VoteEvent{
			EventID:     dto.EventID,
			PollID:      dto.PollID,
			UserID:      dto.UserID,
			CandidateID: dto.CandidateID,
			Type:        domain.EventType(dto.Type),
			Timestamp:   dto.Timestamp,
		}
		if err := c.handler.Handle(ctx, evt); err != nil {
			slog.Error("handle vote event", "err", err, "event_id", evt.EventID)
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			slog.Error("commit kafka offset", "err", err, "offset", m.Offset)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
