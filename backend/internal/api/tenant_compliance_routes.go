// Package api — Tenant Compliance Profile routes (P0-CE.5).
//
// Маршруты для управления compliance регионом tenant'ов.
// Доступны только admin пользователям.
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - IEC 62443 SR 5.1 (Zone-based access — региональные политики)
//   - ISO 27001 A.8.1 (Asset management — tenant classification)
//   - ISO 27001 A.9.2 (Access control — admin only)
//   - GDPR Art. 44-49 (Data transfer — region pinning)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа по tenant'ам)
//   - OWASP ASVS V4 (RBAC — admin only)
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/compliance"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V2 — Authentication (через JWT middleware)
// [x] V3 — Session Management (через AuthMiddleware)
// [x] V4 — Access Control (admin only)
// [x] V5 — Input Validation (whitelist region values)
// [x] V7 — Error Handling and Logging (через respondError)
// [x] V8 — Data Protection (compliance_region не sensitive)
// [x] V14 — Configuration (через config.Config)

// ═══════════════════════════════════════════════════════════════════════
// Response/request types
// ═══════════════════════════════════════════════════════════════════════

// tenantComplianceResponse — ответ с информацией о compliance регионе.
type tenantComplianceResponse struct {
	TenantID         string `json:"tenant_id"`
	ComplianceRegion string `json:"compliance_region"`
	ComplianceLocked bool   `json:"compliance_locked"`
}

// setComplianceRegionRequest — запрос на установку compliance региона.
type setComplianceRegionRequest struct {
	Region string `json:"region"`
}

// ═══════════════════════════════════════════════════════════════════════
// Route mounting
// ═══════════════════════════════════════════════════════════════════════

// mountTenantComplianceRoutes регистрирует маршруты Tenant Compliance Profile.
//
// Все маршруты доступны только для admin.
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.9.2
func (s *Server) mountTenantComplianceRoutes(r chi.Router) {
	r.Route("/api/v1/admin/tenants", func(r chi.Router) {
		// GET /api/v1/admin/tenants/{tenant_id}/compliance — получить compliance регион
		r.Get("/{tenant_id}/compliance", s.handleGetTenantCompliance)

		// PUT /api/v1/admin/tenants/{tenant_id}/compliance — установить/изменить compliance регион
		r.Put("/{tenant_id}/compliance", s.handleSetTenantCompliance)

		// POST /api/v1/admin/tenants/{tenant_id}/compliance/lock — принудительная блокировка
		r.Post("/{tenant_id}/compliance/lock", s.handleLockTenantCompliance)

		// GET /api/v1/admin/tenants/compliance/regions — список доступных compliance регионов
		r.Get("/compliance/regions", s.handleListComplianceRegions)
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleGetTenantCompliance возвращает compliance регион tenant'а.
//
// GET /api/v1/admin/tenants/{tenant_id}/compliance
//
// Access: admin
// Соответствует: OWASP ASVS V4 (RBAC), V5 (input validation)
func (s *Server) handleGetTenantCompliance(w http.ResponseWriter, r *http.Request) {
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

	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, NewValidationError("tenant_id is required"))
		return
	}

	if s.tenantComplianceStore == nil {
		RespondError(w, r, NewInternalError("compliance store not initialized", nil))
		return
	}

	region, locked, err := s.tenantComplianceStore.GetComplianceRegion(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, fmt.Errorf("get tenant compliance: %w", err))
		return
	}

	jsonResponse(w, http.StatusOK, tenantComplianceResponse{
		TenantID:         tenantID,
		ComplianceRegion: region,
		ComplianceLocked: locked,
	})
}

// handleSetTenantCompliance устанавливает compliance регион tenant'а.
//
// PUT /api/v1/admin/tenants/{tenant_id}/compliance
//
// Body: {"region": "BY|EU|INTL|RU|CN|US"}
//
// Access: admin
// Ошибка если регион уже заблокирован (immutable после first data).
// Соответствует: OWASP ASVS V4 (RBAC), V5 (input validation)
func (s *Server) handleSetTenantCompliance(w http.ResponseWriter, r *http.Request) {
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

	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, NewValidationError("tenant_id is required"))
		return
	}

	var req setComplianceRegionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewValidationError("invalid request body: "+err.Error()))
		return
	}

	// ── V5: Input Validation (whitelist) ──
	switch req.Region {
	case compliance.RegionBY, compliance.RegionEU, compliance.RegionINTL, compliance.RegionRU, compliance.RegionCN, compliance.RegionUS:
		// valid
	default:
		RespondError(w, r, NewValidationError(
			fmt.Sprintf("invalid region: %s. Allowed: BY, EU, INTL, RU, CN, US", req.Region),
		))
		return
	}

	if s.tenantComplianceStore == nil {
		RespondError(w, r, NewInternalError("compliance store not initialized", nil))
		return
	}

	if err := s.tenantComplianceStore.SetComplianceRegion(r.Context(), tenantID, req.Region); err != nil {
		RespondError(w, r, fmt.Errorf("set tenant compliance: %w", err))
		return
	}

	// Возвращаем обновлённый профиль
	region, locked, err := s.tenantComplianceStore.GetComplianceRegion(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, fmt.Errorf("get updated compliance: %w", err))
		return
	}

	jsonResponse(w, http.StatusOK, tenantComplianceResponse{
		TenantID:         tenantID,
		ComplianceRegion: region,
		ComplianceLocked: locked,
	})
}

// handleLockTenantCompliance принудительно блокирует compliance регион.
//
// POST /api/v1/admin/tenants/{tenant_id}/compliance/lock
//
// Access: admin
// Используется при первом data creation (вызывается автоматически,
// но доступен и для ручного вызова админом).
// Соответствует: OWASP ASVS V4 (RBAC)
func (s *Server) handleLockTenantCompliance(w http.ResponseWriter, r *http.Request) {
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

	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, NewValidationError("tenant_id is required"))
		return
	}

	if s.tenantComplianceStore == nil {
		RespondError(w, r, NewInternalError("compliance store not initialized", nil))
		return
	}

	if err := s.tenantComplianceStore.LockComplianceRegion(r.Context(), tenantID); err != nil {
		RespondError(w, r, fmt.Errorf("lock tenant compliance: %w", err))
		return
	}

	region, locked, err := s.tenantComplianceStore.GetComplianceRegion(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, fmt.Errorf("get updated compliance: %w", err))
		return
	}

	jsonResponse(w, http.StatusOK, tenantComplianceResponse{
		TenantID:         tenantID,
		ComplianceRegion: region,
		ComplianceLocked: locked,
	})
}

// handleListComplianceRegions возвращает список доступных compliance регионов.
//
// GET /api/v1/admin/tenants/compliance/regions
//
// Access: admin
// Соответствует: OWASP ASVS V4 (RBAC)
func (s *Server) handleListComplianceRegions(w http.ResponseWriter, r *http.Request) {
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

	if s.complianceRegistry == nil {
		RespondError(w, r, NewInternalError("compliance registry not initialized", nil))
		return
	}

	regions := s.complianceRegistry.List()
	type regionInfo struct {
		Region      string `json:"region"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}

	result := make([]regionInfo, 0, len(regions))
	for _, rgn := range regions {
		profile, err := s.complianceRegistry.Get(rgn)
		if err != nil {
			continue
		}
		result = append(result, regionInfo{
			Region:      rgn,
			Name:        profile.Name(),
			Description: profile.Description(),
		})
	}

	jsonResponse(w, http.StatusOK, result)
}
