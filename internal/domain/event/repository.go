package event

import (
	"context"

	"github.com/google/uuid"
)

// Store is the outbound port for the append-only event store.
// Append must set e.Version to the DB-assigned value before returning.
type Store interface {
	Append(ctx context.Context, e *Event) error
	GetByStreamID(ctx context.Context, streamID string) ([]*Event, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Event, error)
}

// Publisher is the outbound port for the event bus.
// Implementations should treat Publish as best-effort: the event is
// already durable in the store before Publish is called.
type Publisher interface {
	Publish(ctx context.Context, e *Event) error
	Close() error
}

// Indexer is the outbound port for the search index.
// Index must be idempotent: calling it twice with the same event must
// produce the same result (use event.ID as the document key).
type Indexer interface {
	Index(ctx context.Context, e *Event) error
	Close() error
}

// SearchQuery carries parameters for reading from the read model.
type SearchQuery struct {
	StreamID string
	Limit    int
	Offset   int
}

// Searcher is the outbound port for the Elasticsearch read model.
// Results are eventually consistent with the PostgreSQL event store —
// recently ingested events may not yet be visible.
type Searcher interface {
	Search(ctx context.Context, q SearchQuery) ([]*Event, int64, error)
}
