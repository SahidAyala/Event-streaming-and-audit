package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

// Command carries the data needed to ingest a new event.
type Command struct {
	StreamID string
	Type     string
	Source   string
	Payload  json.RawMessage
	Metadata map[string]string
}

// Service orchestrates event ingestion: store first, then publish.
type Service struct {
	store     event.Store
	publisher event.Publisher
	log       *slog.Logger
}

func NewService(store event.Store, publisher event.Publisher, log *slog.Logger) *Service {
	return &Service{store: store, publisher: publisher, log: log}
}

// Ingest appends the event to the store and publishes it to Kafka.
// A Kafka publish failure is logged but does not fail the request —
// the event is already durable in PostgreSQL.
func (s *Service) Ingest(ctx context.Context, cmd Command) (*event.Event, error) {
	if cmd.StreamID == "" || cmd.Type == "" || cmd.Source == "" {
		return nil, fmt.Errorf("stream_id, type, and source are required")
	}

	e := event.New(cmd.StreamID, cmd.Type, cmd.Source, cmd.Payload, cmd.Metadata)

	if err := s.store.Append(ctx, e); err != nil {
		return nil, fmt.Errorf("append to store: %w", err)
	}

	if err := s.publisher.Publish(ctx, e); err != nil {
		s.log.Warn("failed to publish event to kafka",
			"event_id", e.ID,
			"stream_id", e.StreamID,
			"error", err,
		)
	}

	s.log.Info("event ingested",
		"event_id", e.ID,
		"stream_id", e.StreamID,
		"type", e.Type,
		"version", e.Version,
	)
	return e, nil
}
