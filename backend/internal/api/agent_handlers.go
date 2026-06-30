// Package api — Agent Management HTTP handlers.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-EDGE Block 6 — API-03: Agent Management Endpoints
//
// Endpoints:
//
//	GET    /api/v1/agents         — список всех агентов
//	GET    /api/v1/agents/{id}    — детали агента
//	POST   /api/v1/agents/{id}/command — отправить команду агенту
//	DELETE /api/v1/agents/{id}    — удалить агента
//
// Таблица agents:
//
//	CREATE TABLE agents (
//	    id VARCHAR(100) PRIMARY KEY,
//	    name VARCHAR(255),
//	    site_id UUID REFERENCES sites(id),
//	    status VARCHAR(50),        -- online, offline, error
//	    last_seen TIMESTAMPTZ,
//	    version VARCHAR(50),
//	    config JSONB,
//	    created_at TIMESTAMPTZ DEFAULT NOW()
//	);
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend), SR 1.1 (Defense in depth)
//   - OWASP ASVS V3.3: RBAC (admin for mutations)
//   - OWASP ASVS V5.1: Input validation (whitelist)
//   - OWASP ASVS V7.1: Error handling (no information leakage)
//   - ISO 27001 A.9.2.3: Privileged access management
//   - ISO 27001 A.12.4.1: Event logging (audit trail)
//   - Приказ ОАЦ №66 п. 7.18: mTLS для agent communication
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/respond"
)

// ────────────────────────────────────────────────────────────────────────────
// AgentStore — интерфейс для работы с агентами в БД.
// ────────────────────────────────────────────────────────────────────────────

// Agent представляет запись агента из таблицы agents.
type Agent struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	SiteID    *string                `json:"site_id,omitempty"`
	Status    string                 `json:"status"`
	LastSeen  *time.Time             `json:"last_seen,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// AgentStore определяет контракт для доступа к данным агентов.
type AgentStore interface {
	// ListAgents возвращает список всех агентов.
	ListAgents() ([]Agent, error)

	// GetAgent возвращает агента по ID.
	GetAgent(id string) (*Agent, error)

	// DeleteAgent удаляет агента по ID.
	DeleteAgent(id string) error

	// SendCommand отправляет команду агенту.
	SendCommand(id string, command string, params map[string]interface{}) error
}

// ────────────────────────────────────────────────────────────────────────────
// Agent Command Types and Validation
// ────────────────────────────────────────────────────────────────────────────

// validAgentCommands — whitelist команд для агентов (OWASP ASVS V5.1).
var validAgentCommands = map[string]bool{
	"restart":     true,
	"update":      true,
	"reboot":      true,
	"diagnose":    true,
	"sync_config": true,
	"ping":        true,
}

// agentCommandRequest — тело запроса для POST /api/v1/agents/{id}/command.
type agentCommandRequest struct {
	Command string                 `json:"command" validate:"required"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// agentCommandResponse — ответ на команду.
type agentCommandResponse struct {
	AgentID   string `json:"agent_id"`
	Command   string `json:"command"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// validateAgentCommand проверяет запрос команды агенту.
func validateAgentCommand(req *agentCommandRequest) error {
	v := NewValidator()

	v.Required("command", req.Command)

	if !validAgentCommands[req.Command] {
		v.OneOf("command", req.Command, []string{"restart", "update", "reboot", "diagnose", "sync_config", "ping"})
	}

	if !v.Valid() {
		return v.ToValidationErrors()
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────────────

// handleListAgents возвращает список всех агентов (GET).
//
// Compliance:
//   - OWASP ASVS V7.1: Стандартизированный response
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if s.agentStore == nil {
		respond.RespondError(w, r, respond.NewInternalError("agent store not available", nil))
		return
	}

	agents, err := s.agentStore.ListAgents()
	if err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to list agents", err))
		return
	}

	// Маскируем sensitive поля из конфига (OWASP ASVS V8 — Data protection)
	sanitized := make([]Agent, len(agents))
	for i, a := range agents {
		sanitized[i] = sanitizeAgent(a)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"agents": sanitized,
		"total":  len(sanitized),
	})
}

// handleGetAgent возвращает детали агента (GET).
//
// Compliance:
//   - OWASP ASVS V7.1: Error handling (not found)
func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("agent_id is required"))
		return
	}

	if s.agentStore == nil {
		respond.RespondError(w, r, respond.NewInternalError("agent store not available", nil))
		return
	}

	agent, err := s.agentStore.GetAgent(agentID)
	if err != nil {
		respond.RespondError(w, r, respond.NewNotFoundError("agent not found"))
		return
	}

	jsonResponse(w, http.StatusOK, sanitizeAgent(*agent))
}

// handleSendAgentCommand отправляет команду агенту (POST).
//
// Compliance:
//   - OWASP ASVS V3.3: RBAC (admin only)
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.12.4.1: Audit logging
//   - IEC 62443-3-3 SR 3.1: Command execution control
func (s *Server) handleSendAgentCommand(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("agent_id is required"))
		return
	}

	// RBAC check (OWASP ASVS V3.3)
	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can send commands to agents"))
		return
	}

	var req agentCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewBadRequestError("invalid request body"))
		return
	}

	// Input validation (OWASP ASVS V5.1)
	if err := validateAgentCommand(&req); err != nil {
		// Convert validation error
		var ve *ValidationErrors
		if errors.As(err, &ve) {
			respondValidationError(w, r, ve)
		} else {
			respond.RespondError(w, r, respond.NewValidationError(err.Error()))
		}
		return
	}

	if s.agentStore == nil {
		respond.RespondError(w, r, respond.NewInternalError("agent store not available", nil))
		return
	}

	// Проверяем, что агент существует
	if _, err := s.agentStore.GetAgent(agentID); err != nil {
		respond.RespondError(w, r, respond.NewNotFoundError("agent not found"))
		return
	}

	if err := s.agentStore.SendCommand(agentID, req.Command, req.Params); err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to send command to agent", err))
		return
	}

	// Audit trail (ISO 27001 A.12.4.1)
	s.logAudit(getClaimsRole(r), "AGENT_COMMAND_SEND", "agent", agentID, nil, map[string]interface{}{
		"agent_id": agentID,
		"command":  req.Command,
		"params":   req.Params,
	})

	jsonResponse(w, http.StatusOK, agentCommandResponse{
		AgentID:   agentID,
		Command:   req.Command,
		Status:    "sent",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleDeleteAgent удаляет агента (DELETE).
//
// Compliance:
//   - OWASP ASVS V3.3: RBAC (admin only)
//   - ISO 27001 A.12.4.1: Audit logging
//   - ISO 27001 A.8.1.2: Asset disposal
func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("agent_id is required"))
		return
	}

	// RBAC check (OWASP ASVS V3.3)
	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can delete agents"))
		return
	}

	if s.agentStore == nil {
		respond.RespondError(w, r, respond.NewInternalError("agent store not available", nil))
		return
	}

	// Проверяем, что агент существует
	if _, err := s.agentStore.GetAgent(agentID); err != nil {
		respond.RespondError(w, r, respond.NewNotFoundError("agent not found"))
		return
	}

	if err := s.agentStore.DeleteAgent(agentID); err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to delete agent", err))
		return
	}

	// Audit trail (ISO 27001 A.12.4.1)
	s.logAudit(getClaimsRole(r), "AGENT_DELETE", "agent", agentID, nil, map[string]string{
		"agent_id": agentID,
	})

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":   "deleted",
		"agent_id": agentID,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// sanitizeAgent маскирует sensitive поля из конфига агента (OWASP ASVS V8).
func sanitizeAgent(a Agent) Agent {
	// Маскируем потенциально sensitive поля в config
	if a.Config != nil {
		sensitiveKeys := []string{"password", "token", "secret", "key", "api_key"}
		config := make(map[string]interface{})
		for k, v := range a.Config {
			config[k] = v
		}
		for _, key := range sensitiveKeys {
			if _, ok := config[key]; ok {
				config[key] = "****"
			}
		}
		a.Config = config
	}
	return a
}
