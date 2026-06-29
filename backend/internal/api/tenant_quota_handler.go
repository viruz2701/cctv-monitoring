// Package api — Tenant Quota Management handlers (P1-QUOTA).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-QUOTA: Tenant Quota Management
//
// Маршруты для управления квотами tenant'ов.
// Доступны admin пользователям для управления лимитами.
// Обычные пользователи могут только просматривать свои квоты.
//
// API Endpoints:
//
//	GET    /api/v1/tenant/quota           — текущее использование
//	GET    /api/v1/tenant/quota/history   — история изменений
//	PUT    /api/v1/tenant/quota           — обновить лимиты (admin)
//	POST   /api/v1/tenant/quota/increase  — запрос на увеличение
//
// Compliance:
//   - IEC 62443 SR 3.1 (Resource management)
//   - ISO 27001 A.12.1.2 (Capacity management)
//   - ISO 27001 A.9.2 (Access control — admin only)
//   - OWASP ASVS V4 (RBAC — admin only for mutations)
//   - OWASP ASVS V5 (Input validation)
//   - OWASP ASVS V7 (Error handling)
//   - СТБ 34.101.27 п. 6.1 (Защита от DoS)
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/tenant"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V2 — Authentication (через JWT middleware)
// [x] V3 — Session Management (через AuthMiddleware)
// [x] V4 — Access Control (admin only for mutations)
// [x] V5 — Input Validation (whitelist quota types, positive ints)
// [x] V7 — Error Handling and Logging (через respondError)
// [x] V8 — Data Protection (quota данные не sensitive)
// [x] V14 — Configuration (через config.Config)

// ═══════════════════════════════════════════════════════════════════════
// Response/request types
// ═══════════════════════════════════════════════════════════════════════

// quotaResponse — полный ответ с информацией о квотах tenant'а.
type quotaResponse struct {
	TenantID   string                     `json:"tenant_id"`
	Quotas     map[string]*quotaStatusDTO `json:"quotas"`
	OnGrace    bool                       `json:"on_grace"`
	GraceUntil *time.Time                 `json:"grace_until,omitempty"`
	GraceDays  int                        `json:"grace_days"`
	SuspendAt  *time.Time                 `json:"suspend_at,omitempty"`
}

// quotaStatusDTO — DTO для ответа о статусе квоты.
type quotaStatusDTO struct {
	Type       string     `json:"type"`
	Current    int64      `json:"current"`
	SoftLimit  int64      `json:"soft_limit"`
	HardLimit  int64      `json:"hard_limit"`
	Usage      float64    `json:"usage_percent"`
	IsSoft     bool       `json:"is_soft"`
	IsHard     bool       `json:"is_hard"`
	OnGrace    bool       `json:"on_grace"`
	GraceUntil *time.Time `json:"grace_until,omitempty"`
}

// updateQuotaRequest — запрос на обновление лимитов (admin).
type updateQuotaRequest struct {
	QuotaType string `json:"quota_type"`
	HardLimit int64  `json:"hard_limit"`
}

// increaseQuotaRequest — запрос на увеличение квоты.
type increaseQuotaRequest struct {
	QuotaType string `json:"quota_type"`
	Reason    string `json:"reason"`
}

// quotaHistoryResponse — ответ с историей изменений квот.
type quotaHistoryResponse struct {
	History []quotaHistoryEntryDTO `json:"history"`
	Total   int                    `json:"total"`
}

type quotaHistoryEntryDTO struct {
	ID        int64     `json:"id"`
	QuotaType string    `json:"quota_type"`
	OldLimit  int       `json:"old_limit"`
	NewLimit  int       `json:"new_limit"`
	Reason    string    `json:"reason"`
	ChangedBy string    `json:"changed_by"`
	CreatedAt time.Time `json:"created_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Route mounting
// ═══════════════════════════════════════════════════════════════════════

// mountTenantQuotaRoutes регистрирует маршруты Tenant Quota Management.
//
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.9.2
func (s *Server) mountTenantQuotaRoutes(r chi.Router) {
	r.Route("/api/v1/tenant/quota", func(r chi.Router) {
		// GET /api/v1/tenant/quota — текущее использование квот
		r.Get("/", s.handleGetCurrentQuota)

		// GET /api/v1/tenant/quota/history — история изменений
		r.Get("/history", s.handleGetQuotaHistory)

		// PUT /api/v1/tenant/quota — обновить лимиты (admin only)
		r.Put("/", s.handleUpdateQuotaLimits)

		// POST /api/v1/tenant/quota/increase — запрос на увеличение
		r.Post("/increase", s.handleIncreaseQuotaRequest)
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleGetCurrentQuota возвращает текущее использование всех квот tenant'а.
//
// GET /api/v1/tenant/quota
//
// Access: authenticated (любая роль)
// Соответствует: OWASP ASVS V4 (RBAC), V5 (input validation)
func (s *Server) handleGetCurrentQuota(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication ──
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control — tenantID из JWT ──
	tenantID := auth.GetTenantID(r)
	if tenantID == "" {
		RespondError(w, r, NewForbiddenError("tenant context required"))
		return
	}

	if s.tenantQuotaManager == nil {
		RespondError(w, r, NewInternalError("quota management not available", nil))
		return
	}

	usage, err := s.tenantQuotaManager.GetAll(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to get quota", err))
		return
	}

	resp := quotaResponse{
		TenantID:  usage.TenantID,
		OnGrace:   usage.OnGrace,
		GraceDays: usage.GraceDays,
		SuspendAt: usage.SuspendAt,
	}
	if usage.GraceUntil != nil {
		gu := *usage.GraceUntil
		resp.GraceUntil = &gu
	}
	resp.Quotas = make(map[string]*quotaStatusDTO)
	for qt, status := range usage.Quotas {
		dto := &quotaStatusDTO{
			Type:      string(qt),
			Current:   status.Current,
			SoftLimit: status.SoftLimit,
			HardLimit: status.HardLimit,
			Usage:     status.Usage,
			IsSoft:    status.IsSoft,
			IsHard:    status.IsHard,
			OnGrace:   status.OnGrace,
		}
		if status.GraceUntil != nil {
			gu := *status.GraceUntil
			dto.GraceUntil = &gu
		}
		resp.Quotas[string(qt)] = dto
	}

	jsonResponse(w, http.StatusOK, resp)
}

// handleGetQuotaHistory возвращает историю изменений квот tenant'а.
//
// GET /api/v1/tenant/quota/history
//
// Access: authenticated (любая роль, только свои данные)
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.12.4 (Audit trail)
func (s *Server) handleGetQuotaHistory(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication ──
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control — tenantID из JWT ──
	tenantID := auth.GetTenantID(r)
	if tenantID == "" {
		RespondError(w, r, NewForbiddenError("tenant context required"))
		return
	}

	// Admin может смотреть историю любого tenant'а
	if claims.Role == "admin" {
		if q := r.URL.Query().Get("tenant_id"); q != "" {
			tenantID = q
		}
	}

	history, err := s.getQuotaHistoryFromDB(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to get quota history", err))
		return
	}

	resp := quotaHistoryResponse{
		History: history,
		Total:   len(history),
	}

	jsonResponse(w, http.StatusOK, resp)
}

// handleUpdateQuotaLimits обновляет лимиты квоты для tenant'а.
//
// PUT /api/v1/tenant/quota
//
// Access: admin only
// Соответствует: OWASP ASVS V4 (RBAC — admin only), V5 (input validation)
func (s *Server) handleUpdateQuotaLimits(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication ──
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control (admin only) ──
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("insufficient permissions: admin role required"))
		return
	}

	if s.tenantQuotaManager == nil {
		RespondError(w, r, NewInternalError("quota management not available", nil))
		return
	}

	// ── V5: Input Validation ──
	var req updateQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewValidationError("invalid request body"))
		return
	}

	if req.HardLimit <= 0 {
		RespondError(w, r, NewValidationError("hard_limit must be positive"))
		return
	}

	qt, err := tenant.QuotaTypeFromString(req.QuotaType)
	if err != nil {
		RespondError(w, r, NewValidationError(fmt.Sprintf("invalid quota_type: %v", err)))
		return
	}

	// Tenant ID из query или из JWT
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = auth.GetTenantID(r)
	}
	if tenantID == "" {
		RespondError(w, r, NewBadRequestError("tenant_id is required"))
		return
	}

	// Сохраняем старый лимит для аудита
	oldLimit := int64(0)
	defaults := tenant.DefaultQuotaConfigs()
	if cfg, ok := defaults[qt]; ok {
		oldLimit = cfg.HardLimit
	}

	// Обновляем в Redis
	if err := s.tenantQuotaManager.SetLimits(r.Context(), tenantID, qt, req.HardLimit); err != nil {
		RespondError(w, r, NewInternalError("failed to set quota limits", err))
		return
	}

	// Сохраняем в БД для персистентности
	if err := s.saveQuotaLimitsToDB(r.Context(), tenantID, qt, req.HardLimit); err != nil {
		// Не фатально — Redis уже обновлён
		s.logger.Warn("quota: failed to persist limits to DB",
			"tenant_id", tenantID, "quota_type", qt, "error", err,
		)
	}

	// Аудит: записываем в историю
	if err := s.saveQuotaHistoryToDB(r.Context(), tenantID, string(qt), int(oldLimit), int(req.HardLimit), "admin update", claims.Subject); err != nil {
		s.logger.Warn("quota: failed to save quota history",
			"tenant_id", tenantID, "quota_type", qt, "error", err,
		)
	}

	s.logger.Info("quota: limits updated by admin",
		"admin_id", claims.Subject,
		"tenant_id", tenantID,
		"quota_type", qt,
		"old_limit", oldLimit,
		"new_limit", req.HardLimit,
	)

	jsonResponse(w, http.StatusOK, map[string]string{
		"status": "updated",
	})
}

// handleIncreaseQuotaRequest обрабатывает запрос на увеличение квоты.
//
// POST /api/v1/tenant/quota/increase
//
// Access: authenticated (любая роль)
// Соответствует: OWASP ASVS V5 (input validation)
func (s *Server) handleIncreaseQuotaRequest(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication ──
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control — tenantID из JWT ──
	tenantID := auth.GetTenantID(r)
	if tenantID == "" {
		RespondError(w, r, NewForbiddenError("tenant context required"))
		return
	}

	// ── V5: Input Validation ──
	var req increaseQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewValidationError("invalid request body"))
		return
	}

	qt, err := tenant.QuotaTypeFromString(req.QuotaType)
	if err != nil {
		RespondError(w, r, NewValidationError(fmt.Sprintf("invalid quota_type: %v", err)))
		return
	}

	if req.Reason == "" {
		RespondError(w, r, NewValidationError("reason is required"))
		return
	}

	// Сохраняем запрос в истории (для уведомления admin)
	if err := s.saveQuotaHistoryToDB(r.Context(), tenantID, string(qt), 0, 0, fmt.Sprintf("increase request: %s", req.Reason), claims.Subject); err != nil {
		s.logger.Warn("quota: failed to save increase request",
			"tenant_id", tenantID, "quota_type", qt, "error", err,
		)
	}

	s.logger.Info("quota: increase request submitted",
		"tenant_id", tenantID,
		"quota_type", qt,
		"reason", req.Reason,
		"requested_by", claims.Subject,
	)

	// TODO: P1-QUOTA.3: Отправить NATS event для уведомления admin
	// Пока просто возвращаем успех

	jsonResponse(w, http.StatusAccepted, map[string]string{
		"status":  "submitted",
		"message": "Quota increase request has been submitted for review",
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Database helpers
// ═══════════════════════════════════════════════════════════════════════

// saveQuotaLimitsToDB сохраняет лимиты квоты в PostgreSQL.
func (s *Server) saveQuotaLimitsToDB(ctx context.Context, tenantID string, qt tenant.QuotaType, hardLimit int64) error {
	query := fmt.Sprintf(`
		INSERT INTO tenant_quotas (tenant_id, %s, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET %s = EXCLUDED.%s, updated_at = NOW()
	`, string(qt), string(qt), string(qt))

	_, err := s.db.Pool.Exec(ctx, query, tenantID, int(hardLimit))
	return err
}

// saveQuotaHistoryToDB сохраняет запись в истории изменений квот.
func (s *Server) saveQuotaHistoryToDB(ctx context.Context, tenantID, quotaType string, oldLimit, newLimit int, reason, changedBy string) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO tenant_quota_history (tenant_id, quota_type, old_limit, new_limit, reason, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, quotaType, oldLimit, newLimit, reason, changedBy)
	return err
}

// getQuotaHistoryFromDB возвращает историю изменений квот из PostgreSQL.
func (s *Server) getQuotaHistoryFromDB(ctx context.Context, tenantID string) ([]quotaHistoryEntryDTO, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, quota_type, old_limit, new_limit, reason, changed_by, created_at
		FROM tenant_quota_history
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, tenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return []quotaHistoryEntryDTO{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var history []quotaHistoryEntryDTO
	for rows.Next() {
		var entry quotaHistoryEntryDTO
		if err := rows.Scan(&entry.ID, &entry.QuotaType, &entry.OldLimit, &entry.NewLimit, &entry.Reason, &entry.ChangedBy, &entry.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, entry)
	}

	return history, nil
}
