package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SheykoWk/event-streaming-and-audit/internal/application/ingest"
)

type handler struct {
	svc *ingest.Service
	log *slog.Logger
}

// NewRouter wires up all routes and middleware.
func NewRouter(svc *ingest.Service, log *slog.Logger) http.Handler {
	h := &handler{svc: svc, log: log}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.health)

	r.Route("/events", func(r chi.Router) {
		r.Post("/", h.ingest)
		r.Get("/{streamID}", h.getByStream)
	})

	return r
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type ingestRequest struct {
	StreamID string            `json:"stream_id"`
	Type     string            `json:"type"`
	Source   string            `json:"source"`
	Payload  json.RawMessage   `json:"payload"`
	Metadata map[string]string `json:"metadata"`
}

func (h *handler) ingest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	e, err := h.svc.Ingest(r.Context(), ingest.Command{
		StreamID: req.StreamID,
		Type:     req.Type,
		Source:   req.Source,
		Payload:  req.Payload,
		Metadata: req.Metadata,
	})
	if err != nil {
		h.log.Error("ingest failed", "error", err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, e)
}

func (h *handler) getByStream(w http.ResponseWriter, _ *http.Request) {
	// TODO: query service + Elasticsearch read model
	writeError(w, http.StatusNotImplemented, "query layer not yet implemented")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
