// Package api — Protocol Sync API для Edge-агентов.
//
// ═══════════════════════════════════════════════════════════════════════════
// PROTO-04: Protocol Sync API (для Edge-агентов)
//
// Edge-агент отправляет список вендоров, для которых нужны дескрипторы.
// Backend возвращает JSON-дескрипторы для запрошенных вендоров.
//
// Endpoint:
//   POST /api/v1/edge/protocols/sync  — синхронизация дескрипторов
//
// Аутентификация: mTLS (client certificate) для Edge-агентов.
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1: Unique identification (mTLS)
//   - OWASP ASVS V3.3: RBAC
//   - ISO 27001 A.12.4.1: Audit logging
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/respond"
)

// ────────────────────────────────────────────────────────────────────────────
// DTOs
// ────────────────────────────────────────────────────────────────────────────

type protocolSyncRequest struct {
	AgentID string   `json:"agent_id"`
	Vendors []string `json:"vendors"`
}

type protocolSyncResponse struct {
	Descriptors []descriptorSummary `json:"descriptors"`
	SyncedAt    string              `json:"synced_at"`
}

type descriptorSummary struct {
	Vendor  string          `json:"vendor"`
	Version string          `json:"version"`
	RawJSON json.RawMessage `json:"descriptor"`
}

// ────────────────────────────────────────────────────────────────────────────
// Handler
// ────────────────────────────────────────────────────────────────────────────

// handleProtocolSync обрабатывает запрос агента на синхронизацию дескрипторов.
func (s *Server) handleProtocolSync(w http.ResponseWriter, r *http.Request) {
	if s.descriptorRegistry == nil {
		respond.RespondError(w, r, respond.NewInternalError("protocol registry not available", nil))
		return
	}

	var req protocolSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewBadRequestError("invalid request body"))
		return
	}

	if len(req.Vendors) == 0 {
		respond.RespondError(w, r, respond.NewValidationError("at least one vendor is required"))
		return
	}

	var descriptors []descriptorSummary
	for _, vendor := range req.Vendors {
		descriptor, err := s.descriptorRegistry.GetDescriptor(r.Context(), vendor)
		if err != nil {
			s.logger.Warn("descriptor not found", "vendor", vendor)
			continue
		}

		descriptors = append(descriptors, descriptorSummary{
			Vendor:  descriptor.Vendor,
			Version: descriptor.Version,
			RawJSON: descriptor.RawJSON,
		})
	}

	s.logger.Info("protocol sync",
		"agent_id", req.AgentID,
		"vendors", req.Vendors,
		"returned", len(descriptors),
	)

	jsonResponse(w, http.StatusOK, protocolSyncResponse{
		Descriptors: descriptors,
		SyncedAt:    time.Now().UTC().Format(time.RFC3339),
	})
}

// mountProtocolSyncRoutes монтирует маршруты синхронизации протоколов.
func (s *Server) mountProtocolSyncRoutes(r chi.Router) {
	r.Post("/api/v1/edge/protocols/sync", s.handleProtocolSync)
}
