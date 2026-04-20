package consume

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

// ---------------------------------------------------------------------------
// mockSubscriber — synchronous delivery of a predefined event list.
//
// Real Kafka blocks indefinitely and requires a running broker. This mock
// delivers all events in order and returns nil, simulating a Kafka run loop
// that naturally terminates after processing its batch.
//
// This design lets Run() be called synchronously in tests without goroutines,
// channels, or timeouts.
// ---------------------------------------------------------------------------

type mockSubscriber struct {
	events []*event.Event
}

func (m *mockSubscriber) Run(ctx context.Context, handle func(context.Context, *event.Event) error) error {
	for _, e := range m.events {
		if err := handle(ctx, e); err != nil {
			// Mirror real Kafka behaviour: if the handler errors,
			// the subscriber surfaces it (and would not commit the offset).
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockIndexer — stateful in-memory event.Indexer.
//
// Stores events keyed by UUID, mirroring Elasticsearch's idempotent
// document upsert (doc_id = event.ID). Indexing the same event twice
// results in one entry — the second write overwrites the first.
// ---------------------------------------------------------------------------

type mockIndexer struct {
	mu      sync.Mutex
	indexed map[uuid.UUID]*event.Event
	calls   int
	failFor uuid.UUID // if set, Index returns error for this specific event ID
}

func newMockIndexer() *mockIndexer {
	return &mockIndexer{indexed: make(map[uuid.UUID]*event.Event)}
}

func (m *mockIndexer) Index(_ context.Context, e *event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if e.ID == m.failFor {
		return errors.New("elasticsearch shard unavailable")
	}
	cp := *e
	m.indexed[e.ID] = &cp
	return nil
}

func (m *mockIndexer) Close() error { return nil }

func (m *mockIndexer) get(id uuid.UUID) (*event.Event, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.indexed[id]
	return e, ok
}

func (m *mockIndexer) size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.indexed)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func makeEvent(streamID string, version int64) *event.Event {
	return &event.Event{
		ID:         uuid.New(),
		StreamID:   streamID,
		Type:       "test.event",
		Source:     "test-svc",
		Version:    version,
		OccurredAt: time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// consume.Service — behavior tests
// ---------------------------------------------------------------------------

func TestConsume_EventReachesIndexer(t *testing.T) {
	e := makeEvent("order:1", 1)
	sub := &mockSubscriber{events: []*event.Event{e}}
	idx := newMockIndexer()
	svc := NewService(sub, idx, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	indexed, ok := idx.get(e.ID)
	if !ok {
		t.Fatal("event must be present in the indexer after Run")
	}
	// The indexed event must be the same event — no mutation in transit.
	if indexed.StreamID != e.StreamID || indexed.Version != e.Version {
		t.Errorf("indexed event does not match original: got stream=%s v=%d, want stream=%s v=%d",
			indexed.StreamID, indexed.Version, e.StreamID, e.Version)
	}
}

func TestConsume_AllEventsFromSubscriberAreIndexed(t *testing.T) {
	events := []*event.Event{
		makeEvent("order:1", 1),
		makeEvent("order:1", 2),
		makeEvent("order:1", 3),
	}
	sub := &mockSubscriber{events: events}
	idx := newMockIndexer()
	svc := NewService(sub, idx, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.size() != 3 {
		t.Errorf("expected 3 indexed events, got %d", idx.size())
	}
	for _, e := range events {
		if _, ok := idx.get(e.ID); !ok {
			t.Errorf("event %s (v%d) not found in indexer", e.ID, e.Version)
		}
	}
}

func TestConsume_IdempotentIndexing(t *testing.T) {
	// Kafka at-least-once delivery means the same event can arrive twice.
	// The service must pass it to the indexer both times — deduplication
	// is the indexer's responsibility (ES upsert by doc_id).
	// After two deliveries, the index must contain exactly one entry.
	e := makeEvent("order:1", 1)
	sub := &mockSubscriber{events: []*event.Event{e, e}} // same event twice
	idx := newMockIndexer()
	svc := NewService(sub, idx, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.calls != 2 {
		t.Errorf("indexer must be called once per delivery: expected 2 calls, got %d", idx.calls)
	}
	if idx.size() != 1 {
		t.Errorf("idempotent indexer must hold exactly 1 entry after 2 identical deliveries, got %d", idx.size())
	}
}

func TestConsume_IndexerErrorPropagates(t *testing.T) {
	// If the indexer returns an error, Run must surface it.
	// In production this prevents the Kafka offset from being committed,
	// ensuring the event will be redelivered. Swallowing the error here
	// would silently lose events from the index.
	e := makeEvent("order:1", 1)
	sub := &mockSubscriber{events: []*event.Event{e}}
	idx := newMockIndexer()
	idx.failFor = e.ID
	svc := NewService(sub, idx, discardLogger())

	err := svc.Run(context.Background())

	if err == nil {
		t.Fatal("indexer error must be propagated by Run — not swallowed")
	}
}

func TestConsume_IndexerErrorStopsProcessing(t *testing.T) {
	// When the indexer fails for event[0], event[1] must NOT be processed.
	// The subscriber returns the first error and stops — mirroring how
	// Kafka would stop consuming further messages until the error is resolved.
	first := makeEvent("order:1", 1)
	second := makeEvent("order:1", 2)
	sub := &mockSubscriber{events: []*event.Event{first, second}}
	idx := newMockIndexer()
	idx.failFor = first.ID
	svc := NewService(sub, idx, discardLogger())

	svc.Run(context.Background()) //nolint:errcheck — error presence tested above

	if _, ok := idx.get(second.ID); ok {
		t.Error("second event must not be indexed after first event fails")
	}
}
