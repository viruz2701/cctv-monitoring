// Package api — CRUD handlers for devices with OWASP ASVS Level 3 compliance.
//
// Соответствие стандартам:
//   - OWASP ASVS L3 V1-V17 (полный набор контролей)
//   - ISO 27001:2022 A.9.2 (RBAC), A.12.4 (Audit), A.14.2 (Security in development)
//   - СТБ 34.101.27 п. 6.3 (Защита API endpoints)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
//   - Приказ ОАЦ № 66 п. 7.18 (Идентификация пользователей)
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/service"

	"github.com/go-chi/chi/v5"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V1 — Architecture, Design and Threat Modeling (через service layer)
// [x] V2 — Authentication (через JWT middleware)
// [x] V3 — Session Management (через AuthMiddleware)
// [x] V4 — Access Control (RBAC проверка в хендлерах + service)
// [x] V5 — Validation, Sanitization, Encoding (whitelist через Validator)
// [x] V6 — Stored Cryptography (СТБ 34.101.30 — через audit signer)
// [x] V7 — Error Handling and Logging (через respondError + structured logging)
// [x] V8 — Data Protection (sensitive fields: json:"-" в моделях)
// [x] V9 — Communications (TLS 1.3 через reverse proxy)
// [x] V14 — Configuration (через config.Config)

// ── POST /api/v1/devices ───────────────────────────────────────────────

// handleCreateDevice создаёт новое устройство.
// Соответствует: OWASP ASVS V4 (Access control), V5 (Validation), V7 (Error handling)
//
// Request: CreateDeviceRequest (JSON)
// Response: 201 Created + Device (JSON)
// Errors: 400 (validation), 403 (forbidden), 409 (conflict), 500 (internal)
func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication (JWT claims) ──
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control ──
	if !isWriteRole(claims.Role) {
		respondError(w, r, NewForbiddenError("insufficient permissions to create devices"))
		return
	}

	// ── V5: Input Validation (decode + whitelist) ──
	var req models.CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid JSON body"))
		return
	}

	// Whitelist validation
	fields := &createDeviceRequestFields{
		DeviceID:        req.DeviceID,
		Name:            req.Name,
		Location:        req.Location,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		VendorType:      req.VendorType,
		DeviceType:      req.DeviceType,
		Status:          req.Status,
		ConnectionType:  req.ConnectionType,
		AssetClass:      req.AssetClass,
		Manufacturer:    req.Manufacturer,
		SerialNumber:    req.SerialNumber,
		MacAddress:      req.MacAddress,
		FirmwareVersion: req.FirmwareVersion,
		SiteID:          req.SiteID,
		P2PBrand:        req.P2PBrand,
		P2PSerial:       req.P2PSerial,
		UserAgent:       req.UserAgent,
		ParentDeviceID:  req.ParentDeviceID,
	}
	if err := validateCreateDeviceRequest(fields); err != nil {
		respondError(w, r, NewValidationError(err.Error()))
		return
	}

	// ── V1: Business logic через service layer ──
	dev, err := s.deviceService.CreateDevice(r.Context(), claims.UserID, claims.Role, &req)
	if err != nil {
		if errors.Is(err, service.ErrAccessDenied) {
			respondError(w, r, NewForbiddenError(err.Error()))
			return
		}
		// V7: Error handling — no information leakage
		respondError(w, r, NewInternalError("failed to create device", err))
		return
	}

	// ── V8: Data Protection ──
	jsonResponse(w, http.StatusCreated, mapDeviceToResponse(dev))
}

// ── GET /api/v1/devices ───────────────────────────────────────────────

// handleListDevices возвращает список устройств с пагинацией и фильтрацией.
// Соответствует: OWASP ASVS V4 (RBAC), V7 (Error handling), V8 (Data protection)
//
// Query parameters:
//   - page (int, default: 1)
//   - page_size (int, default: 20, max: 100)
//   - status (string: ONLINE, OFFLINE, WARNING)
//   - device_type (string: camera, nvr, dvr, switch)
//   - vendor_type (string)
//   - site_id (string, UUID)
//   - asset_class (string: critical, confidential, internal, public)
//   - search (string — поиск по имени, device_id, локации)
func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V5: Input validation для query parameters ──
	filter := models.ListDevicesFilter{
		Page:           parseIntParam(r.URL.Query().Get("page"), 1),
		PageSize:       parseIntParam(r.URL.Query().Get("page_size"), models.DefaultPageSize),
		Status:         r.URL.Query().Get("status"),
		DeviceType:     r.URL.Query().Get("device_type"),
		VendorType:     r.URL.Query().Get("vendor_type"),
		SiteID:         r.URL.Query().Get("site_id"),
		AssetClass:     r.URL.Query().Get("asset_class"),
		Search:         r.URL.Query().Get("search"),
		ParentDeviceID: r.URL.Query().Get("parent_device_id"),
	}

	// Валидация статуса (OWASP ASVS V5 — whitelist)
	if filter.Status != "" {
		valid := false
		for _, s := range validStatuses {
			if s == filter.Status {
				valid = true
				break
			}
		}
		if !valid {
			respondError(w, r, NewValidationError("invalid status: must be ONLINE, OFFLINE, or WARNING"))
			return
		}
	}

	// Валидация device_type (OWASP ASVS V5 — whitelist)
	if filter.DeviceType != "" {
		valid := false
		for _, dt := range validDeviceTypes {
			if dt == filter.DeviceType {
				valid = true
				break
			}
		}
		if !valid {
			respondError(w, r, NewValidationError("invalid device_type: must be camera, nvr, dvr, or switch"))
			return
		}
	}

	result, err := s.deviceService.ListDevices(r.Context(), claims.UserID, claims.Role, filter)
	if err != nil {
		respondError(w, r, NewInternalError("failed to list devices", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// ── GET /api/v1/devices/{id} ───────────────────────────────────────────

// handleGetDevice возвращает устройство по ID.
func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}

	dev, err := s.deviceService.GetDevice(r.Context(), claims.UserID, claims.Role, deviceID)
	if err != nil {
		if errors.Is(err, service.ErrAccessDenied) {
			respondError(w, r, NewForbiddenError(err.Error()))
			return
		}
		respondError(w, r, NewNotFoundError("device not found"))
		return
	}

	jsonResponse(w, http.StatusOK, mapDeviceToResponse(dev))
}

// ── PUT /api/v1/devices/{id} ───────────────────────────────────────────

// handleUpdateDevice обновляет устройство (частичное обновление).
// Соответствует: OWASP ASVS V4, V5, V7, ISO 27001 A.12.4
//
// Request: UpdateDeviceRequest (JSON, все поля опциональны)
// Response: 200 OK + обновлённый Device
func (s *Server) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	if !isWriteRole(claims.Role) {
		respondError(w, r, NewForbiddenError("insufficient permissions to update devices"))
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}

	var req models.UpdateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid JSON body"))
		return
	}

	// Whitelist validation (OWASP ASVS V5.1)
	uf := &updateDeviceRequestFields{
		Name:            req.Name,
		Location:        req.Location,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		VendorType:      req.VendorType,
		DeviceType:      req.DeviceType,
		Status:          req.Status,
		ConnectionType:  req.ConnectionType,
		AssetClass:      req.AssetClass,
		Manufacturer:    req.Manufacturer,
		SerialNumber:    req.SerialNumber,
		MacAddress:      req.MacAddress,
		FirmwareVersion: req.FirmwareVersion,
		SiteID:          req.SiteID,
		P2PBrand:        req.P2PBrand,
		P2PSerial:       req.P2PSerial,
		UserAgent:       req.UserAgent,
		Health:          req.Health,
		ParentDeviceID:  req.ParentDeviceID,
	}
	if err := validateUpdateDeviceRequest(uf); err != nil {
		respondError(w, r, NewValidationError(err.Error()))
		return
	}

	dev, err := s.deviceService.UpdateDevice(r.Context(), claims.UserID, claims.Role, deviceID, &req)
	if err != nil {
		if errors.Is(err, service.ErrAccessDenied) {
			respondError(w, r, NewForbiddenError(err.Error()))
			return
		}
		respondError(w, r, NewNotFoundError("device not found"))
		return
	}

	jsonResponse(w, http.StatusOK, mapDeviceToResponse(dev))
}

// ── DELETE /api/v1/devices/{id} ────────────────────────────────────────

// handleDeleteDevice удаляет устройство (soft delete по умолчанию).
// Соответствует: ISO 27001 A.8.1.2 (Asset disposal), GDPR Art. 17
//
// Query parameters:
//   - hard (bool, default: false — hard delete только для admin)
func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	if !isWriteRole(claims.Role) {
		respondError(w, r, NewForbiddenError("insufficient permissions to delete devices"))
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}

	hardDelete := r.URL.Query().Get("hard") == "true"

	if err := s.deviceService.DeleteDevice(r.Context(), claims.UserID, claims.Role, deviceID, hardDelete); err != nil {
		if errors.Is(err, service.ErrAccessDenied) {
			respondError(w, r, NewForbiddenError(err.Error()))
			return
		}
		respondError(w, r, NewNotFoundError("device not found"))
		return
	}

	status := http.StatusOK
	if hardDelete {
		status = http.StatusNoContent
	}

	jsonResponse(w, status, map[string]string{
		"status":    "deleted",
		"device_id": deviceID,
	})
}

// ── POST /api/v1/devices/{id}/restore ──────────────────────────────────

// handleRestoreDevice восстанавливает soft-deleted устройство.
func (s *Server) handleRestoreDevice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	if !isWriteRole(claims.Role) {
		respondError(w, r, NewForbiddenError("insufficient permissions to restore devices"))
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}

	if err := s.deviceService.RestoreDevice(r.Context(), claims.UserID, claims.Role, deviceID); err != nil {
		if errors.Is(err, service.ErrAccessDenied) {
			respondError(w, r, NewForbiddenError(err.Error()))
			return
		}
		respondError(w, r, NewNotFoundError("device not found"))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "restored",
		"device_id": deviceID,
	})
}

// ── Helpers ────────────────────────────────────────────────────────────

// mapDeviceToResponse преобразует Device в безопасный ответ API.
// Исключает sensitive поля (OWASP ASVS V8 — Data Protection).
func mapDeviceToResponse(dev *models.Device) map[string]interface{} {
	return map[string]interface{}{
		"device_id":        dev.DeviceID,
		"owner_id":         dev.OwnerID,
		"site_id":          dev.SiteID,
		"name":             dev.Name,
		"location":         dev.Location,
		"latitude":         dev.Latitude,
		"longitude":        dev.Longitude,
		"vendor_type":      dev.VendorType,
		"device_type":      dev.DeviceType,
		"status":           dev.Status,
		"health":           dev.Health,
		"asset_class":      dev.AssetClass,
		"connection_type":  dev.ConnectionType,
		"manufacturer":     dev.Manufacturer,
		"serial_number":    dev.SerialNumber,
		"mac_address":      dev.MacAddress,
		"firmware_version": dev.FirmwareVersion,
		"p2p_brand":        dev.P2PBrand,
		"p2p_serial":       dev.P2PSerial,
		"cloud_status":     dev.CloudStatus,
		"last_seen":        dev.LastSeen,
		"registered_at":    dev.RegisteredAt,
		"created_at":       dev.CreatedAt,
		"updated_at":       dev.UpdatedAt,
		"parent_device_id": dev.ParentDeviceID,
		"hierarchy_level":  dev.HierarchyLevel,
		// Исключаем: ContactAddr (internal), LastError (может содержать sensitive data)
		// Исключаем: p2p_security_code, p2p_cloud_user, p2p_cloud_pass (credentials)
		// Исключаем: snmp_community (credentials)
		// Исключаем: onvif_username, onvif_password (credentials)
	}
}

// isWriteRole проверяет, имеет ли роль право на запись (OWASP ASVS V4).
func isWriteRole(role string) bool {
	switch role {
	case "admin", "manager", "support":
		return true
	default:
		return false
	}
}

// parseIntParam парсит целочисленный параметр с значением по умолчанию.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil || val <= 0 {
		return defaultVal
	}
	return val
}
