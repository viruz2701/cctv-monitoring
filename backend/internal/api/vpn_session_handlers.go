// ═══════════════════════════════════════════════════════════════════════════
// Package api — VPN Session HTTP Handlers (EDGE-08)
//
// REST API для управления WireGuard VPN сессиями удалённого доступа.
//
// Endpoints:
//   POST   /api/v1/vpn/sessions           — создать сессию (admin/support only)
//   GET    /api/v1/vpn/sessions           — список активных сессий
//   GET    /api/v1/vpn/sessions/{id}      — детали сессии
//   POST   /api/v1/vpn/sessions/{id}/revoke — закрыть сессию
//   GET    /api/v1/vpn/sessions/{id}/config — получить WG config для инженера
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement (RBAC)
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
//   - ISO 27001 A.12.4: Audit trail
//   - OWASP ASVS L3: Input validation, access control, error handling
// ═══════════════════════════════════════════════════════════════════════════

package api

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gb-telemetry-collector/internal/edge"
)

// ═══ DTO ═══════════════════════════════════════════════════════════════

type createSessionRequest struct {
	AgentID    string   `json:"agent_id"`
	EngineerID string   `json:"engineer_id"`
	AllowedIPs []string `json:"allowed_ips"`
	Duration   string   `json:"duration"` // human-readable, e.g. "1h", "30m"
}

type sessionResponse struct {
	ID               string   `json:"id"`
	AgentID          string   `json:"agent_id"`
	EngineerID       string   `json:"engineer_id"`
	StartedAt        string   `json:"started_at"`
	ExpiresAt        string   `json:"expires_at"`
	AllowedIPs       []string `json:"allowed_ips"`
	PublicKey        string   `json:"public_key"`
	Status           string   `json:"status"`
	BytesTransferred int64    `json:"bytes_transferred"`
	CreatedAt        string   `json:"created_at"`
	ClosedAt         *string  `json:"closed_at,omitempty"`
}

// ═══ Handlers ═════════════════════════════════════════════════════════

// handleCreateVPNSession создаёт новую VPN сессию.
//
// RBAC: admin/support only (проверяется middleware).
//
// Compliance:
//   - OWASP ASVS V1.1: Input validation
//   - OWASP ASVS V3.3: Privilege escalation prevention
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
func (s *Server) handleCreateVPNSession(w http.ResponseWriter, r *http.Request) {
	if s.vpnSessionManager == nil {
		RespondError(w, r, NewBadRequestError("VPN sessions are not configured"))
		return
	}

	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewValidationError("invalid request body"))
		return
	}

	// OWASP ASVS V5.1: Input validation
	if req.AgentID == "" {
		RespondError(w, r, NewValidationError("agent_id is required"))
		return
	}
	if req.EngineerID == "" {
		RespondError(w, r, NewValidationError("engineer_id is required"))
		return
	}

	engineerUUID, err := uuid.Parse(req.EngineerID)
	if err != nil {
		RespondError(w, r, NewValidationError("invalid engineer_id format"))
		return
	}

	// Парсим duration
	duration := 1 * time.Hour
	if req.Duration != "" {
		parsed, err := time.ParseDuration(req.Duration)
		if err != nil {
			RespondError(w, r, NewValidationError("invalid duration format, use e.g. 30m, 1h"))
			return
		}
		if parsed <= 0 || parsed > 2*time.Hour {
			RespondError(w, r, NewValidationError("duration must be between 1m and 2h"))
			return
		}
		duration = parsed
	}

	// Парсим allowed IPs (OWASP ASVS V5.1: CIDR validation)
	allowedIPs := make([]net.IPNet, 0, len(req.AllowedIPs))
	for _, ipStr := range req.AllowedIPs {
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			RespondError(w, r, NewValidationError("invalid allowed_ip: "+ipStr))
			return
		}
		allowedIPs = append(allowedIPs, *ipNet)
	}

	if len(allowedIPs) == 0 {
		RespondError(w, r, NewValidationError("at least one allowed_ip is required"))
		return
	}

	createReq := edge.CreateSessionRequest{
		AgentID:    req.AgentID,
		EngineerID: engineerUUID,
		AllowedIPs: allowedIPs,
		Duration: edge.Duration{
			Duration: duration,
		},
	}

	session, err := s.vpnSessionManager.CreateSession(r.Context(), createReq)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to create session", err))
		return
	}

	resp := sessionToResponse(session)
	respondJSON(w, http.StatusCreated, resp)
}

// handleListVPNSessions возвращает список VPN сессий.
func (s *Server) handleListVPNSessions(w http.ResponseWriter, r *http.Request) {
	if s.vpnSessionManager == nil {
		RespondError(w, r, NewBadRequestError("VPN sessions are not configured"))
		return
	}

	filter := edge.SessionFilter{
		Status: r.URL.Query().Get("status"),
		Limit:  100,
	}

	if agentID := r.URL.Query().Get("agent_id"); agentID != "" {
		filter.AgentID = agentID
	}

	sessions, err := s.vpnSessionManager.GetSessions(r.Context(), filter)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to list sessions", err))
		return
	}

	resp := make([]sessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, sessionToResponse(&s))
	}

	respondJSON(w, http.StatusOK, resp)
}

// handleGetVPNSession возвращает детали VPN сессии.
func (s *Server) handleGetVPNSession(w http.ResponseWriter, r *http.Request) {
	if s.vpnSessionManager == nil {
		RespondError(w, r, NewBadRequestError("VPN sessions are not configured"))
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, NewValidationError("invalid session id"))
		return
	}

	session, err := s.vpnSessionManager.GetSession(r.Context(), sessionID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("session not found"))
		return
	}

	resp := sessionToResponse(session)
	respondJSON(w, http.StatusOK, resp)
}

// handleRevokeVPNSession закрывает VPN сессию досрочно.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.2: Отзыв доступа
//   - ISO 27001 A.12.4: Audit trail
func (s *Server) handleRevokeVPNSession(w http.ResponseWriter, r *http.Request) {
	if s.vpnSessionManager == nil {
		RespondError(w, r, NewBadRequestError("VPN sessions are not configured"))
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, NewValidationError("invalid session id"))
		return
	}

	if err := s.vpnSessionManager.RevokeSession(r.Context(), sessionID); err != nil {
		RespondError(w, r, NewInternalError("failed to revoke session", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// handleGetVPNSessionConfig возвращает WireGuard конфигурацию для клиента.
//
// Этот endpoint доступен инженеру для скачивания WG конфига.
func (s *Server) handleGetVPNSessionConfig(w http.ResponseWriter, r *http.Request) {
	if s.vpnSessionManager == nil {
		RespondError(w, r, NewBadRequestError("VPN sessions are not configured"))
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, NewValidationError("invalid session id"))
		return
	}

	config, err := s.vpnSessionManager.GetSessionConfig(r.Context(), sessionID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("session config not available"))
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// ═══ Helpers ═══════════════════════════════════════════════════════════

// sessionToResponse конвертирует модель в response DTO.
func sessionToResponse(s *edge.VPNSession) sessionResponse {
	resp := sessionResponse{
		ID:               s.ID.String(),
		AgentID:          s.AgentID,
		EngineerID:       s.EngineerID.String(),
		StartedAt:        s.StartedAt.Format(time.RFC3339),
		ExpiresAt:        s.ExpiresAt.Format(time.RFC3339),
		AllowedIPs:       ipNetsToStrings(s.AllowedIPs),
		PublicKey:        s.PublicKey,
		Status:           s.Status,
		BytesTransferred: s.BytesTransferred,
		CreatedAt:        s.CreatedAt.Format(time.RFC3339),
	}

	if s.ClosedAt != nil {
		closed := s.ClosedAt.Format(time.RFC3339)
		resp.ClosedAt = &closed
	}

	return resp
}

// ipNetsToStrings конвертирует []net.IPNet в []string (CIDR).
func ipNetsToStrings(ipNets []net.IPNet) []string {
	result := make([]string, 0, len(ipNets))
	for _, ipNet := range ipNets {
		result = append(result, ipNet.String())
	}
	return result
}
