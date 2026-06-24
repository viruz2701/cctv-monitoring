// Package api — Feature Flag handlers (F-0.2.4).
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — feature gating)
//   - ISO 27001 A.12.1.2 (Change management — controlled rollout)
//   - ISO/IEC 27019 PCC.A.12 (Change management for ICS)
//   - СТБ 34.101.27 (Защита информации — контроль доступа к функциям)
//   - OWASP ASVS L3 V1-V17 (полный спектр контролей)
//   - OWASP ASVS V5 (Input validation — whitelist for feature keys)
//   - OWASP ASVS V7 (Error handling — no information leakage)
package api

import (
	"encoding/json"
	"gb-telemetry-collector/internal/auth"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleGetFeatureFlags возвращает список всех фича-флагов.
// GET /api/v1/feature-flags
//
// Compliance:
//   - OWASP ASVS V4.1 (Access control — requires JWT auth)
//   - OWASP ASVS V7.1 (Error handling — стандартизированный ответ)
func (s *Server) handleGetFeatureFlags(w http.ResponseWriter, r *http.Request) {
	flags := s.featureFlags.GetAll()

	// Маскируем tenant_id = '*' как 'global' для клиента
	type flagResponse struct {
		Key         string `json:"key"`
		Enabled     bool   `json:"enabled"`
		Description string `json:"description"`
		TenantID    string `json:"tenant_id"`
	}

	response := make([]flagResponse, 0, len(flags))
	for _, f := range flags {
		tid := f.TenantID
		if tid == "*" {
			tid = "global"
		}
		response = append(response, flagResponse{
			Key:         f.Key,
			Enabled:     f.Enabled,
			Description: f.Description,
			TenantID:    tid,
		})
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"feature_flags": response,
		"total":         len(response),
	})
}

// handleUpdateFeatureFlag обновляет состояние фича-флага.
// PUT /api/v1/feature-flags/{key}
//
// Compliance:
//   - OWASP ASVS V4.1 (Access control — requires JWT auth)
//   - OWASP ASVS V5.1 (Whitelist validation — key from path)
//   - OWASP ASVS V5.2 (Parameterized queries — через DB слой)
//   - OWASP ASVS V7.1 (Error handling — no stack traces)
//   - ISO 27001 A.12.4 (Audit trail — мутация логируется)
func (s *Server) handleUpdateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		respondError(w, r, NewValidationError("feature flag key is required"))
		return
	}

	// Валидация key: только буквы, цифры, точки и подчёркивания (OWASP ASVS V5.1)
	if !isValidFeatureFlagKey(key) {
		respondError(w, r, NewValidationError("invalid feature flag key format"))
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	if err := s.featureFlags.SetEnabled(key, req.Enabled); err != nil {
		s.logger.Error("Failed to update feature flag",
			"key", key,
			"enabled", req.Enabled,
			"error", err,
		)
		respondError(w, r, NewInternalError("failed to update feature flag", err))
		return
	}

	s.logger.Info("Feature flag updated via API",
		"key", key,
		"enabled", req.Enabled,
		"updated_by", getRequestUserID(r),
	)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"key":     key,
		"enabled": req.Enabled,
	})
}

// isValidFeatureFlagKey проверяет key по whitelist (OWASP ASVS V5.1).
// Допустимы: буквы, цифры, точки, подчёркивания, дефисы.
func isValidFeatureFlagKey(key string) bool {
	if len(key) == 0 || len(key) > 255 {
		return false
	}
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// getRequestUserID извлекает user_id из контекста запроса (если есть).
func getRequestUserID(r *http.Request) string {
	if claims := auth.GetClaims(r); claims != nil {
		return claims.Subject
	}
	return "unknown"
}
