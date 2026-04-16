package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/SheykoWk/event-streaming-and-audit/internal/config"
	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

// Producer publishes events to a Kafka topic, partitioned by StreamID.
type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafkago.Hash{}, // consistent partition per StreamID
			RequiredAcks: kafkago.RequireOne,
			WriteTimeout: 10 * time.Second,
			// Async=false ensures WriteMessages blocks until the broker acks.
			// Flip to true for higher throughput at the cost of at-most-once delivery.
			Async: false,
		},
	}
}

// Publish serialises the event to JSON and writes it to Kafka.
// The message key is StreamID so all events for a stream land on the same partition.
func (p *Producer) Publish(ctx context.Context, e *event.Event) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafkago.Message{
		Key:   []byte(e.StreamID),
		Value: payload,
		Time:  e.OccurredAt,
		Headers: []kafkago.Header{
			{Key: "event_id", Value: []byte(e.ID.String())},
			{Key: "event_type", Value: []byte(e.Type)},
			{Key: "source", Value: []byte(e.Source)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write to kafka: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
