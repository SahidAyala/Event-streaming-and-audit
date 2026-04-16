package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event is the core domain entity representing an immutable fact.
// Version is set by the store on Append; callers should treat it as
// zero until after a successful Append.
type Event struct {
	ID         uuid.UUID         `json:"id"`
	StreamID   string            `json:"stream_id"`
	Type       string            `json:"type"`
	Source     string            `json:"source"`
	Version    int64             `json:"version"`
	OccurredAt time.Time         `json:"occurred_at"`
	Payload    json.RawMessage   `json:"payload"`
	Metadata   map[string]string `json:"metadata"`
}

// New builds an Event with a fresh ID and timestamp.
// Version is intentionally left as 0; the store assigns it on Append.
func New(streamID, eventType, source string, payload json.RawMessage, metadata map[string]string) *Event {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	return &Event{
		ID:         uuid.New(),
		StreamID:   streamID,
		Type:       eventType,
		Source:     source,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
		Metadata:   metadata,
	}
}
