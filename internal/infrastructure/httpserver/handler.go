package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/chi/v5"

	"github.com/SheykoWk/event-streaming-and-audit/internal/application/ingest"
	"github.com/SheykoWk/event-streaming-and-audit/internal/application/query"
	infraauth "github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/auth"
	authmw "github.com/SheykoWk/event-streaming-and-audit/internal/infrastructure/httpserver/middleware"
)

type handler struct {
	ingestSvc *ingest.Service
	querySvc  *query.Service
	log       *slog.Logger
}

// NewRouter wires up all routes and middleware.
// The authenticator is applied to all /events routes; /health is unprotected.
func NewRouter(ingestSvc *ingest.Service, querySvc *query.Service, authenticator infraauth.Authenticator, log *slog.Logger) http.Handler {
	h := &handler{ingestSvc: ingestSvc, querySvc: querySvc, log: log}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	r.Get("/health", h.health)

	r.Route("/events", func(r chi.Router) {
		r.Use(authmw.Auth(authenticator))
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

	e, err := h.ingestSvc.Ingest(r.Context(), ingest.Command{
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

// getByStream handles GET /events/{streamID}?limit=N&offset=N.
// Results come from Elasticsearch and are eventually consistent with
// the PostgreSQL event store. Response headers communicate this explicitly.
func (h *handler) getByStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		writeError(w, http.StatusBadRequest, "stream_id is required")
		return
	}

	limit := parseIntParam(r, "limit", 20)
	offset := parseIntParam(r, "offset", 0)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	result, err := h.querySvc.QueryByStream(ctx, query.Query{
		StreamID: streamID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		h.log.Error("query by stream failed", "stream_id", streamID, "error", err)
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}

	w.Header().Set("X-Read-Model", "elasticsearch")
	w.Header().Set("X-Data-Consistency", "eventual")
	writeJSON(w, http.StatusOK, result)
}

// parseIntParam reads an integer query parameter with a fallback default.
// Returns the default if the param is absent, non-numeric, or negative.
func parseIntParam(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
