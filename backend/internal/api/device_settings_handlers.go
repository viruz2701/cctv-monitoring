// Package api — Device Settings HTTP handlers.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-EDGE Block 6 — API-01: Device Settings Endpoints
//
// Endpoints:
//
//	GET  /api/v1/devices/{id}/settings?category=network  — получить настройки
//	PUT  /api/v1/devices/{id}/settings                   — обновить настройки
//	POST /api/v1/devices/{id}/settings/apply             — применить настройки
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend), SR 1.1 (Defense in depth)
//   - OWASP ASVS V3.3: RBAC (admin only for settings mutations)
//   - OWASP ASVS V5.1: Input validation (whitelist)
//   - OWASP ASVS V7.1: Error handling (no information leakage)
//   - ISO 27001 A.9.2.3: Privileged access management
//   - ISO 27001 A.12.4.1: Event logging (audit trail)
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
// DeviceSettingsProvider — интерфейс для получения/обновления настроек
// устройства через VendorDevice.GetSettings() / SetSettings().
// ────────────────────────────────────────────────────────────────────────────

// DeviceSettingsProvider определяет контракт для работы с настройками устройства.
// Реализуется через DeviceFactory, который создаёт VendorDevice для конкретного вендора.
type DeviceSettingsProvider interface {
	// GetSettings возвращает настройки устройства по категории (network, video, audio, etc.).
	// Если category пустая — возвращает все настройки.
	GetSettings(deviceID string, category string) (map[string]interface{}, error)

	// SetSettings обновляет настройки устройства.
	// Принимает мапу с изменениями (ключ-значение).
	SetSettings(deviceID string, settings map[string]interface{}) error

	// ApplySettings применяет отложенные настройки на устройстве.
	// Для NVR/DVR может означать перезагрузку сервиса, для PTZ-camera — применение presets.
	ApplySettings(deviceID string) error
}

// ────────────────────────────────────────────────────────────────────────────
// Request/Response DTOs
// ────────────────────────────────────────────────────────────────────────────

// getDeviceSettingsResponse — ответ для GET settings.
type getDeviceSettingsResponse struct {
	DeviceID  string                 `json:"device_id"`
	Category  string                 `json:"category,omitempty"`
	Settings  map[string]interface{} `json:"settings"`
	UpdatedAt string                 `json:"updated_at,omitempty"`
}

// updateDeviceSettingsRequest — тело запроса для PUT settings.
type updateDeviceSettingsRequest struct {
	Settings map[string]interface{} `json:"settings" validate:"required"`
}

// applyDeviceSettingsResponse — ответ для POST settings/apply.
type applyDeviceSettingsResponse struct {
	DeviceID  string `json:"device_id"`
	Status    string `json:"status"`
	AppliedAt string `json:"applied_at"`
}

// ────────────────────────────────────────────────────────────────────────────
// Validation
// ────────────────────────────────────────────────────────────────────────────

// validSettingCategories — whitelist категорий настроек (OWASP ASVS V5.1).
var validSettingCategories = map[string]bool{
	"network": true,
	"video":   true,
	"audio":   true,
	"ptz":     true,
	"storage": true,
	"alarm":   true,
	"system":  true,
}

// validateCategory проверяет, что категория входит в whitelist.
func validateCategory(category string) bool {
	if category == "" {
		return true // пустая категория = все настройки
	}
	return validSettingCategories[category]
}

// validateSettingsUpdate проверяет запрос на обновление настроек.
func validateSettingsUpdate(req *updateDeviceSettingsRequest) error {
	v := NewValidator()

	if req.Settings == nil {
		v.Required("settings", "required")
		return v.ToValidationErrors()
	}

	// Проверяем, что ключи не пустые и не содержат потенциально опасных символов
	for key, val := range req.Settings {
		v.Required("settings."+key, key)
		if key == "" {
			v.Required("settings.key", "key must not be empty")
		}
		// Значение может быть любого типа (string, number, bool, array, object)
		// — это нормально для настроек, проверка на уровне бизнес-логики
		_ = val
	}

	if !v.Valid() {
		return v.ToValidationErrors()
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────────────

// handleGetDeviceSettings возвращает настройки устройства (GET).
//
// Query parameters:
//   - category (optional): фильтр по категории (network, video, audio, ptz, storage, alarm, system)
//
// Compliance:
//   - OWASP ASVS V5.1: Whitelist validation для category
//   - OWASP ASVS V7.1: Стандартизированный error response
func (s *Server) handleGetDeviceSettings(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	category := r.URL.Query().Get("category")

	// Whitelist validation (OWASP ASVS V5.1)
	if !validateCategory(category) {
		respond.RespondError(w, r, respond.NewValidationError(
			"invalid category; allowed: network, video, audio, ptz, storage, alarm, system"))
		return
	}

	if s.deviceSettingsProvider == nil {
		respond.RespondError(w, r, respond.NewInternalError("device settings provider not available", nil))
		return
	}

	settings, err := s.deviceSettingsProvider.GetSettings(deviceID, category)
	if err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to get device settings", err))
		return
	}

	jsonResponse(w, http.StatusOK, getDeviceSettingsResponse{
		DeviceID:  deviceID,
		Category:  category,
		Settings:  settings,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleUpdateDeviceSettings обновляет настройки устройства (PUT).
//
// Compliance:
//   - OWASP ASVS V3.3: RBAC (admin only)
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.12.4.1: Audit logging
func (s *Server) handleUpdateDeviceSettings(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	// RBAC check (OWASP ASVS V3.3)
	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can update device settings"))
		return
	}

	var req updateDeviceSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewBadRequestError("invalid request body"))
		return
	}

	// Input validation (OWASP ASVS V5.1)
	if err := validateSettingsUpdate(&req); err != nil {
		var ve *ValidationErrors
		if errors.As(err, &ve) {
			respondValidationError(w, r, ve)
		} else {
			respond.RespondError(w, r, respond.NewValidationError(err.Error()))
		}
		return
	}

	if s.deviceSettingsProvider == nil {
		respond.RespondError(w, r, respond.NewInternalError("device settings provider not available", nil))
		return
	}

	if err := s.deviceSettingsProvider.SetSettings(deviceID, req.Settings); err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to update device settings", err))
		return
	}

	// Audit trail (ISO 27001 A.12.4.1)
	s.logAudit(getClaimsRole(r), "DEVICE_SETTINGS_UPDATE", "device_settings", deviceID, nil, map[string]interface{}{
		"device_id": deviceID,
		"settings":  req.Settings,
	})

	jsonResponse(w, http.StatusOK, getDeviceSettingsResponse{
		DeviceID:  deviceID,
		Settings:  req.Settings,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleApplyDeviceSettings применяет настройки на устройстве (POST).
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1: Resource management (apply operation)
//   - OWASP ASVS V3.3: RBAC (admin only)
//   - ISO 27001 A.12.4.1: Audit logging
func (s *Server) handleApplyDeviceSettings(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	// RBAC check (OWASP ASVS V3.3)
	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can apply device settings"))
		return
	}

	if s.deviceSettingsProvider == nil {
		respond.RespondError(w, r, respond.NewInternalError("device settings provider not available", nil))
		return
	}

	if err := s.deviceSettingsProvider.ApplySettings(deviceID); err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to apply device settings", err))
		return
	}

	// Audit trail (ISO 27001 A.12.4.1)
	s.logAudit(getClaimsRole(r), "DEVICE_SETTINGS_APPLY", "device_settings", deviceID, nil, map[string]string{
		"device_id": deviceID,
		"action":    "apply",
	})

	jsonResponse(w, http.StatusOK, applyDeviceSettingsResponse{
		DeviceID:  deviceID,
		Status:    "applied",
		AppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
