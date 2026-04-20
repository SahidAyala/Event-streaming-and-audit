package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "github.com/SheykoWk/event-streaming-and-audit/gen/proto"
	"github.com/SheykoWk/event-streaming-and-audit/internal/application/replay"
	"github.com/SheykoWk/event-streaming-and-audit/internal/domain/event"
)

// Handler implements the EventServiceServer gRPC interface.
// Only Replay is implemented here; Ingest is exposed via the HTTP API.
type Handler struct {
	eventsv1.UnimplementedEventServiceServer
	replaySvc *replay.Service
}

func NewHandler(replaySvc *replay.Service) *Handler {
	return &Handler{replaySvc: replaySvc}
}

// Replay reads events from the PostgreSQL event store (source of truth)
// and returns them ordered by version ASC.
// Returns codes.InvalidArgument for bad input.
// Returns codes.DataLoss if a version gap is detected (data integrity violation).
func (h *Handler) Replay(ctx context.Context, req *eventsv1.ReplayRequest) (*eventsv1.EventStream, error) {
	if req.StreamId == "" {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}

	events, err := h.replaySvc.Replay(ctx, replay.Command{
		StreamID:    req.StreamId,
		FromVersion: req.FromVersion,
	})
	if err != nil {
		return nil, status.Errorf(codes.DataLoss, "replay failed: %v", err)
	}

	protoEvents := make([]*eventsv1.Event, 0, len(events))
	for _, e := range events {
		pe, err := domainToProto(e)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "marshal event %s: %v", e.ID, err)
		}
		protoEvents = append(protoEvents, pe)
	}

	return &eventsv1.EventStream{
		StreamId: req.StreamId,
		Events:   protoEvents,
	}, nil
}

// domainToProto converts a domain Event to its protobuf representation.
// Payload is stored as json.RawMessage in the domain and must be converted
// to structpb.Struct for the proto wire format.
func domainToProto(e *event.Event) (*eventsv1.Event, error) {
	var payload *structpb.Struct
	if len(e.Payload) > 0 && string(e.Payload) != "null" {
		var m map[string]any
		if err := json.Unmarshal(e.Payload, &m); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}
		var err error
		payload, err = structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("convert payload to structpb: %w", err)
		}
	}

	return &eventsv1.Event{
		Id:         e.ID.String(),
		StreamId:   e.StreamID,
		Type:       e.Type,
		Source:     e.Source,
		Version:    e.Version,
		OccurredAt: timestamppb.New(e.OccurredAt),
		Payload:    payload,
		Metadata:   e.Metadata,
	}, nil
}
