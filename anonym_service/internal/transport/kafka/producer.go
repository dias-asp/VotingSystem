package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"

	"anonym_service/internal/domain"
	"anonym_service/internal/service"
)

var _ service.EventPublisher = (*Producer)(nil)

type anonymizedEventDTO struct {
	EventID     string    `json:"event_id"`
	PollID      string    `json:"poll_id"`
	MdmID       string    `json:"mdm_id"`
	CandidateID string    `json:"candidate_id"`
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			Async:        false,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, event domain.AnonymizedEvent) error {
	payload, err := json.Marshal(anonymizedEventDTO{
		EventID:     event.EventID,
		PollID:      event.PollID,
		MdmID:       event.MdmID,
		CandidateID: event.CandidateID,
		Type:        string(event.Type),
		Timestamp:   event.Timestamp,
	})
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.MdmID),
		Value: payload,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
