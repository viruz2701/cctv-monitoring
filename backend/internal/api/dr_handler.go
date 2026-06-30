// Package api — HTTP handlers для Disaster Recovery модуля.
//
// ═══════════════════════════════════════════════════════════════════════════════
// P3-DR: DR API Endpoints
//
// API:
//
//	GET    /api/v1/dr/health              — статус health checks
//	POST   /api/v1/dr/failover            — запустить failover (admin)
//	POST   /api/v1/dr/failover/{id}/approve  — подтвердить failover (admin)
//	POST   /api/v1/dr/failover/{id}/reject   — отклонить failover (admin)
//	GET    /api/v1/dr/history             — история failover
//	POST   /api/v1/dr/drill               — запустить drill
//	GET    /api/v1/dr/drill/active        — текущий активный drill
//
// Compliance:
//   - ISO 27001 A.17.1 (BCM dashboard)
//   - IEC 62443-3-3 SR 7.1 (Resource availability monitoring)
//   - OWASP ASVS V3 (Session management), V4 (Access control)
//
// ═══════════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/dr"
	"gb-telemetry-collector/internal/respond"
)

// ──────────────────────────────────────────────────────────────────────────────
// DR Handler
// ──────────────────────────────────────────────────────────────────────────────

// DRHandler — HTTP handler для DR API.
type DRHandler struct {
	healthMonitor   *dr.HealthMonitor
	failoverManager *dr.FailoverManager
	drillRunner     *dr.DrillRunner
	logger          *slog.Logger
}

// NewDRHandler создаёт новый DRHandler.
func NewDRHandler(
	hm *dr.HealthMonitor,
	fm *dr.FailoverManager,
	drr *dr.DrillRunner,
	logger *slog.Logger,
) *DRHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DRHandler{
		healthMonitor:   hm,
		failoverManager: fm,
		drillRunner:     drr,
		logger:          logger.With("component", "api.dr-handler"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Route Registration
// ──────────────────────────────────────────────────────────────────────────────

// mountDRRoutes монтирует DR маршруты на chi роутер.
// Все маршруты защищены JWT (вызывается внутри защищённой группы).
func (s *Server) mountDRRoutes(r chi.Router) {
	if s.drHealthMonitor == nil {
		s.logger.Warn("DR health monitor not initialized, DR routes disabled")
		return
	}

	drHandler := NewDRHandler(
		s.drHealthMonitor,
		s.drFailoverManager,
		s.drDrillRunner,
		s.logger,
	)

	r.Route("/api/v1/dr", func(r chi.Router) {
		// GET /api/v1/dr/health — статус health checks
		r.Get("/health", drHandler.handleGetHealth)

		// POST /api/v1/dr/failover — инициировать failover
		r.Post("/failover", drHandler.handleInitiateFailover)

		// POST /api/v1/dr/failover/{id}/approve — подтвердить failover (admin)
		r.Post("/failover/{id}/approve", drHandler.handleApproveFailover)

		// POST /api/v1/dr/failover/{id}/reject — отклонить failover (admin)
		r.Post("/failover/{id}/reject", drHandler.handleRejectFailover)

		// GET /api/v1/dr/history — история failover
		r.Get("/history", drHandler.handleGetHistory)

		// POST /api/v1/dr/drill — запустить drill
		r.Post("/drill", drHandler.handleStartDrill)

		// GET /api/v1/dr/drill/active — текущий активный drill
		r.Get("/drill/active", drHandler.handleGetActiveDrill)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Request/Response Types
// ──────────────────────────────────────────────────────────────────────────────

type failoverRequest struct {
	TenantID    string `json:"tenant_id,omitempty"`
	Reason      string `json:"reason"`
	InitiatedBy string `json:"initiated_by,omitempty"`
}

type failoverApproveRequest struct {
	ApprovedBy string `json:"approved_by,omitempty"`
}

type failoverRejectRequest struct {
	RejectedBy string `json:"rejected_by,omitempty"`
	Reason     string `json:"reason"`
}

type drillRequest struct {
	Type        string `json:"type"` // "dns" | "db" | "nats" | "full"
	InitiatedBy string `json:"initiated_by,omitempty"`
}

// drHealthResponse — ответ GET /api/v1/dr/health.
type drHealthResponse struct {
	Status   dr.HealthStatus   `json:"status"`
	History  []dr.HealthRecord `json:"history,omitempty"`
	Failover *dr.FailoverEvent `json:"active_failover,omitempty"`
	Drill    *dr.DrillReport   `json:"active_drill,omitempty"`
	Metrics  *drHealthMetrics  `json:"metrics,omitempty"`
}

// drHealthMetrics — метрики RTO/RPO для dashboard.
type drHealthMetrics struct {
	UptimeSeconds  int64   `json:"uptime_seconds"`
	CheckCount     int     `json:"check_count"`
	FailureRate    float64 `json:"failure_rate"`
	LastFailoverAt string  `json:"last_failover_at,omitempty"`
	RTOCompliance  bool    `json:"rto_compliance"`
	RPOCompliance  bool    `json:"rpo_compliance"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────────────

// handleGetHealth возвращает текущий статус DR health checks.
//
// GET /api/v1/dr/health
// Соответствует: IEC 62443-3-3 SR 7.1 (Resource availability monitoring)
func (h *DRHandler) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthMonitor == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR health monitor not initialized"))
		return
	}

	status := h.healthMonitor.GetStatus()
	history := h.healthMonitor.GetHistory(10)

	resp := drHealthResponse{
		Status:  status,
		History: history,
	}

	if h.failoverManager != nil {
		resp.Failover = h.failoverManager.GetActiveFailover()
	}

	if h.drillRunner != nil {
		resp.Drill = h.drillRunner.GetActiveDrill()
	}

	// Метрики.
	resp.Metrics = &drHealthMetrics{
		UptimeSeconds: int64(time.Since(status.LastCheck).Seconds()),
		CheckCount:    len(history),
	}

	jsonResponse(w, http.StatusOK, resp)
}

// handleInitiateFailover инициирует процесс failover.
//
// POST /api/v1/dr/failover
// Body: {"reason": "string", "tenant_id": "optional"}
// Соответствует: ISO 27001 A.17.1.2 (DR procedures — initiation)
func (h *DRHandler) handleInitiateFailover(w http.ResponseWriter, r *http.Request) {
	if h.failoverManager == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR failover manager not initialized"))
		return
	}

	var req failoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewValidationError("invalid request body: "+err.Error()))
		return
	}

	if req.Reason == "" {
		req.Reason = "manual"
	}

	initiatedBy := req.InitiatedBy
	if initiatedBy == "" {
		initiatedBy = "admin"
	}

	event, err := h.failoverManager.InitiateFailover(r.Context(), req.Reason, initiatedBy)
	if err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failover initiation failed", err))
		return
	}

	jsonResponse(w, http.StatusAccepted, event)
}

// handleApproveFailover подтверждает failover (admin).
//
// POST /api/v1/dr/failover/{id}/approve
// Body: {"approved_by": "user_id"}
func (h *DRHandler) handleApproveFailover(w http.ResponseWriter, r *http.Request) {
	if h.failoverManager == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR failover manager not initialized"))
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		respond.RespondError(w, r, respond.NewValidationError("failover event id is required"))
		return
	}

	var req failoverApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewValidationError("invalid request body"))
		return
	}

	approvedBy := req.ApprovedBy
	if approvedBy == "" {
		approvedBy = "admin"
	}

	event, err := h.failoverManager.ApproveFailover(r.Context(), eventID, approvedBy)
	if err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failover approval failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, event)
}

// handleRejectFailover отклоняет failover (admin).
//
// POST /api/v1/dr/failover/{id}/reject
// Body: {"reason": "not approved"}
func (h *DRHandler) handleRejectFailover(w http.ResponseWriter, r *http.Request) {
	if h.failoverManager == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR failover manager not initialized"))
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		respond.RespondError(w, r, respond.NewValidationError("failover event id is required"))
		return
	}

	var req failoverRejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewValidationError("invalid request body"))
		return
	}

	rejectedBy := req.RejectedBy
	if rejectedBy == "" {
		rejectedBy = "admin"
	}
	if req.Reason == "" {
		req.Reason = "rejected by admin"
	}

	if err := h.failoverManager.RejectFailover(r.Context(), eventID, rejectedBy, req.Reason); err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failover rejection failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "rejected", "event_id": eventID})
}

// handleGetHistory возвращает историю failover.
//
// GET /api/v1/dr/history
func (h *DRHandler) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if h.failoverManager == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR failover manager not initialized"))
		return
	}

	active := h.failoverManager.GetActiveFailover()
	history := []*dr.FailoverEvent{}
	if active != nil {
		history = append(history, active)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"failover_history": history,
	})
}

// handleStartDrill запускает DR drill.
//
// POST /api/v1/dr/drill
// Body: {"type": "dns|db|nats|full"}
func (h *DRHandler) handleStartDrill(w http.ResponseWriter, r *http.Request) {
	if h.drillRunner == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR drill runner not initialized"))
		return
	}

	var req drillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewValidationError("invalid request body: "+err.Error()))
		return
	}

	if req.Type == "" {
		req.Type = "dns"
	}

	validTypes := map[string]bool{"dns": true, "db": true, "nats": true, "full": true}
	if !validTypes[req.Type] {
		respond.RespondError(w, r, respond.NewValidationError(
			"invalid drill type: must be one of dns, db, nats, full"))
		return
	}

	initiatedBy := req.InitiatedBy
	if initiatedBy == "" {
		initiatedBy = "admin"
	}

	report, err := h.drillRunner.StartDrill(r.Context(), req.Type, initiatedBy)
	if err != nil {
		respond.RespondError(w, r, respond.NewInternalError("drill execution failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, report)
}

// handleGetActiveDrill возвращает текущий активный drill.
//
// GET /api/v1/dr/drill/active
func (h *DRHandler) handleGetActiveDrill(w http.ResponseWriter, r *http.Request) {
	if h.drillRunner == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("DR drill runner not initialized"))
		return
	}

	drill := h.drillRunner.GetActiveDrill()
	if drill == nil {
		jsonResponse(w, http.StatusOK, map[string]string{"status": "no_active_drill"})
		return
	}

	jsonResponse(w, http.StatusOK, drill)
}
