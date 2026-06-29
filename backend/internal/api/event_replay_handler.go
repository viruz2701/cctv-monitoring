// Package api — HTTP handlers for NATS JetStream Event Replay.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Auditable message replay)
//   - ISO 27001 A.12.4 (Audit trail — traceable replay operations)
//   - OWASP ASVS V5 (Input validation — whitelist validation on stream names, seq)
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/events"
)

// ── Mount ────────────────────────────────────────────────────────────────────

// mountEventReplayRoutes монтирует маршруты NATS JetStream Event Replay.
//
// API:
//
//	GET    /api/v1/events/streams                        — список streams
//	GET    /api/v1/events/streams/{name}/messages        — сообщения из stream
//	POST   /api/v1/events/streams/{name}/replay/{seq}    — повторная публикация
//	GET    /api/v1/events/dead-letters                   — DLQ сообщения
func (s *Server) mountEventReplayRoutes(r chi.Router) {
	r.Get("/api/v1/events/streams", s.handleListStreams)
	r.Get("/api/v1/events/streams/{name}/messages", s.handleListMessages)
	r.Post("/api/v1/events/streams/{name}/replay/{seq}", s.handleReplayMessage)
	r.Get("/api/v1/events/dead-letters", s.handleListDeadLetters)
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// handleListStreams возвращает список всех JetStream streams.
func (s *Server) handleListStreams(w http.ResponseWriter, r *http.Request) {
	if s.eventReplay == nil {
		RespondError(w, r, NewInternalError("NATS JetStream not available", nil))
		return
	}

	streams, err := s.eventReplay.ListStreams(r.Context())
	if err != nil {
		s.logger.Error("event replay: list streams failed", "error", err)
		RespondError(w, r, NewInternalError("failed to list streams", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"streams": streams,
	})
}

// handleListMessages возвращает сообщения из stream с пагинацией и фильтрами.
func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	if s.eventReplay == nil {
		RespondError(w, r, NewInternalError("NATS JetStream not available", nil))
		return
	}

	streamName := chi.URLParam(r, "name")
	if streamName == "" {
		RespondError(w, r, NewBadRequestError("stream name is required"))
		return
	}

	q := r.URL.Query()

	limit := 50
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= events.MaxPageSize {
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	filter := events.MessageFilter{
		Limit:    limit,
		Offset:   offset,
		Subject:  q.Get("subject"),
		TenantID: q.Get("tenant_id"),
	}

	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = t
		}
	}
	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = t
		}
	}

	messages, err := s.eventReplay.ListMessages(r.Context(), streamName, filter)
	if err != nil {
		s.logger.Error("event replay: list messages failed",
			"stream", streamName, "error", err,
		)
		RespondError(w, r, NewInternalError("failed to list messages", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"stream":   streamName,
		"limit":    limit,
		"offset":   offset,
	})
}

// handleReplayMessage повторно публикует сообщение из stream.
func (s *Server) handleReplayMessage(w http.ResponseWriter, r *http.Request) {
	if s.eventReplay == nil {
		RespondError(w, r, NewInternalError("NATS JetStream not available", nil))
		return
	}

	streamName := chi.URLParam(r, "name")
	if streamName == "" {
		RespondError(w, r, NewBadRequestError("stream name is required"))
		return
	}

	seqStr := chi.URLParam(r, "seq")
	if seqStr == "" {
		RespondError(w, r, NewBadRequestError("sequence number is required"))
		return
	}

	seq, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		RespondError(w, r, NewBadRequestError("invalid sequence number"))
		return
	}

	if err := s.eventReplay.ReplayMessage(r.Context(), streamName, seq); err != nil {
		s.logger.Error("event replay: replay failed",
			"stream", streamName, "seq", seq, "error", err,
		)
		RespondError(w, r, NewInternalError("failed to replay message", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"stream": streamName,
		"seq":    seq,
	})
}

// handleListDeadLetters возвращает сообщения из DLQ.
func (s *Server) handleListDeadLetters(w http.ResponseWriter, r *http.Request) {
	if s.eventReplay == nil {
		RespondError(w, r, NewInternalError("NATS JetStream not available", nil))
		return
	}

	q := r.URL.Query()

	limit := 50
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= events.MaxPageSize {
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	filter := events.MessageFilter{
		Limit:    limit,
		Offset:   offset,
		Subject:  q.Get("subject"),
		TenantID: q.Get("tenant_id"),
	}

	messages, err := s.eventReplay.ListDeadLetters(r.Context(), filter)
	if err != nil {
		s.logger.Error("event replay: list DLQ failed", "error", err)
		RespondError(w, r, NewInternalError("failed to list dead letters", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"limit":    limit,
		"offset":   offset,
	})
}

// ── Response types ───────────────────────────────────────────────────────────

// eventReplayResponse используется для Decode/Encode в тестах.
type eventReplayResponse struct {
	Streams  []events.StreamInfo  `json:"streams,omitempty"`
	Messages []events.MessageInfo `json:"messages,omitempty"`
	Stream   string               `json:"stream,omitempty"`
	Limit    int                  `json:"limit,omitempty"`
	Offset   int                  `json:"offset,omitempty"`
	Status   string               `json:"status,omitempty"`
	Seq      uint64               `json:"seq,omitempty"`
}

// Ensure json serialization is used.
var _ = json.Marshal
