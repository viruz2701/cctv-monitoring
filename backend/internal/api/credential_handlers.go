// Package api — Credential Management HTTP handlers.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRED-03: API Endpoints для управления credentials
//
// Endpoints:
//   POST   /api/v1/devices/{id}/credentials  — сохранить credentials (admin only)
//   GET    /api/v1/devices/{id}/credentials  — получить credentials (admin only, маскировать password)
//   PUT    /api/v1/devices/{id}/credentials  — обновить credentials (admin only)
//   DELETE /api/v1/devices/{id}/credentials  — удалить credentials (admin only)
//
// Compliance:
//   - OWASP ASVS V2.1: Verify credentials are stored using approved crypto
//   - OWASP ASVS V3.3: Role-based access control (RBAC)
//   - OWASP ASVS V5.1: Input validation (whitelist)
//   - OWASP ASVS V7.1: Error handling (no information leakage)
//   - ISO 27001 A.9.2.3: Management of privileged access rights
//   - ISO 27001 A.12.4.1: Event logging
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/respond"
)

// ────────────────────────────────────────────────────────────────────────────
// Request/Response DTOs
// ────────────────────────────────────────────────────────────────────────────

type credentialRequest struct {
	Username string `json:"username" validate:"required,min=1,max=255"`
	Password string `json:"password" validate:"required,min=1,max=255"`
}

type credentialResponse struct {
	DeviceID  string `json:"device_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Algorithm string `json:"algorithm,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

func getClaimsRole(r *http.Request) string {
	if claims, ok := r.Context().Value(auth.UserContextKey).(*auth.Claims); ok {
		return claims.Role
	}
	return ""
}

func isAdmin(r *http.Request) bool {
	role := getClaimsRole(r)
	return role == "admin" || role == "superadmin"
}

// ────────────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────────────

// handleStoreCredentials сохраняет credentials для устройства (POST).
func (s *Server) handleStoreCredentials(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can manage credentials"))
		return
	}

	var req credentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewBadRequestError("invalid request body"))
		return
	}
	if req.Username == "" || req.Password == "" {
		respond.RespondError(w, r, respond.NewValidationError("username and password are required"))
		return
	}
	if len(req.Username) > 255 || len(req.Password) > 255 {
		respond.RespondError(w, r, respond.NewValidationError("username and password must not exceed 255 characters"))
		return
	}

	if s.credentialManager == nil {
		respond.RespondError(w, r, respond.NewInternalError("credential manager not available", nil))
		return
	}

	if err := s.credentialManager.Store(r.Context(), deviceID, req.Username, req.Password); err != nil {
		respond.RespondError(w, r, respond.NewConflictError(err.Error()))
		return
	}

	s.logAudit(getClaimsRole(r), "CREDENTIAL_STORE", "credentials", deviceID, nil, map[string]string{
		"device_id": deviceID,
		"username":  req.Username,
	})

	jsonResponse(w, http.StatusCreated, credentialResponse{
		DeviceID:  deviceID,
		Username:  req.Username,
		Password:  "****",
		Algorithm: "aes-256-gcm",
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
}

// handleGetCredentials возвращает credentials для устройства (GET).
func (s *Server) handleGetCredentials(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can view credentials"))
		return
	}

	if s.credentialManager == nil {
		respond.RespondError(w, r, respond.NewInternalError("credential manager not available", nil))
		return
	}

	username, _, err := s.credentialManager.Retrieve(r.Context(), deviceID)
	if err != nil {
		respond.RespondError(w, r, respond.NewNotFoundError("credentials not found for device"))
		return
	}

	jsonResponse(w, http.StatusOK, credentialResponse{
		DeviceID: deviceID,
		Username: username,
		Password: "****",
	})
}

// handleRotateCredentials обновляет credentials для устройства (PUT).
func (s *Server) handleRotateCredentials(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can manage credentials"))
		return
	}

	var req credentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.RespondError(w, r, respond.NewBadRequestError("invalid request body"))
		return
	}
	if req.Username == "" || req.Password == "" {
		respond.RespondError(w, r, respond.NewValidationError("username and password are required"))
		return
	}

	if s.credentialManager == nil {
		respond.RespondError(w, r, respond.NewInternalError("credential manager not available", nil))
		return
	}

	if err := s.credentialManager.Rotate(r.Context(), deviceID, req.Username, req.Password); err != nil {
		respond.RespondError(w, r, respond.NewInternalError(err.Error(), err))
		return
	}

	s.logAudit(getClaimsRole(r), "CREDENTIAL_ROTATE", "credentials", deviceID, nil, map[string]string{
		"device_id": deviceID,
		"username":  req.Username,
	})

	jsonResponse(w, http.StatusOK, credentialResponse{
		DeviceID:  deviceID,
		Username:  req.Username,
		Password:  "****",
		Algorithm: "aes-256-gcm",
		UpdatedAt: time.Now().Format(time.RFC3339),
	})
}

// handleDeleteCredentials удаляет credentials для устройства (DELETE).
func (s *Server) handleDeleteCredentials(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	if !isAdmin(r) {
		respond.RespondError(w, r, respond.NewForbiddenError("only admin can manage credentials"))
		return
	}

	if s.credentialManager == nil {
		respond.RespondError(w, r, respond.NewInternalError("credential manager not available", nil))
		return
	}

	if err := s.credentialManager.Delete(r.Context(), deviceID); err != nil {
		respond.RespondError(w, r, respond.NewNotFoundError("credentials not found for device"))
		return
	}

	s.logAudit(getClaimsRole(r), "CREDENTIAL_DELETE", "credentials", deviceID, nil, map[string]string{
		"device_id": deviceID,
	})

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":    "deleted",
		"device_id": deviceID,
	})
}
