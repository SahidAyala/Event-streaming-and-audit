package consume

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

// Subscriber is satisfied by any message source that can deliver a stream of events.
// kafka.Consumer implements this interface; define it here so the application layer
// has no import dependency on the kafka infrastructure package.
type Subscriber interface {
	Run(ctx context.Context, handle func(context.Context, *event.Event) error) error
}

// Service orchestrates the consume loop: receive from Subscriber → index via Indexer.
type Service struct {
	sub     Subscriber
	indexer event.Indexer
	log     *slog.Logger
}

func NewService(sub Subscriber, indexer event.Indexer, log *slog.Logger) *Service {
	return &Service{sub: sub, indexer: indexer, log: log}
}

// Run blocks until ctx is cancelled or a non-recoverable error occurs.
func (s *Service) Run(ctx context.Context) error {
	return s.sub.Run(ctx, s.handle)
}

func (s *Service) handle(ctx context.Context, e *event.Event) error {
	if err := s.indexer.Index(ctx, e); err != nil {
		return fmt.Errorf("index event: %w", err)
	}
	s.log.Info("event indexed",
		"event_id", e.ID,
		"stream_id", e.StreamID,
		"type", e.Type,
		"version", e.Version,
	)
	return nil
}
