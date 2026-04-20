package query

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Query carries the parameters for retrieving events from the read model.
type Query struct {
	StreamID string
	Limit    int
	Offset   int
}

// Result is the paginated response from QueryByStream.
// ReadModel signals the data source and its consistency guarantee.
type Result struct {
	StreamID  string         `json:"stream_id"`
	Events    []*event.Event `json:"events"`
	Total     int64          `json:"total"`
	Limit     int            `json:"limit"`
	Offset    int            `json:"offset"`
	ReadModel string         `json:"read_model"`
}

// Service handles event query use cases against the read model.
type Service struct {
	searcher event.Searcher
	log      *slog.Logger
}

func NewService(searcher event.Searcher, log *slog.Logger) *Service {
	return &Service{searcher: searcher, log: log}
}

// QueryByStream retrieves a paginated, ordered page of events for a stream
// from the Elasticsearch read model. Results are eventually consistent with
// the PostgreSQL event store.
func (s *Service) QueryByStream(ctx context.Context, q Query) (*Result, error) {
	if q.StreamID == "" {
		return nil, fmt.Errorf("stream_id is required")
	}
	if q.Limit <= 0 {
		q.Limit = defaultLimit
	}
	if q.Limit > maxLimit {
		q.Limit = maxLimit
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	events, total, err := s.searcher.Search(ctx, event.SearchQuery{
		StreamID: q.StreamID,
		Limit:    q.Limit,
		Offset:   q.Offset,
	})
	if err != nil {
		s.log.Error("read model search failed",
			"stream_id", q.StreamID,
			"limit", q.Limit,
			"offset", q.Offset,
			"error", err,
		)
		return nil, fmt.Errorf("search events: %w", err)
	}

	if events == nil {
		events = []*event.Event{}
	}

	return &Result{
		StreamID:  q.StreamID,
		Events:    events,
		Total:     total,
		Limit:     q.Limit,
		Offset:    q.Offset,
		ReadModel: "elasticsearch",
	}, nil
}
