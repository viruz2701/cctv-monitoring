// Package api — HTTP handlers для управления версиями API.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-API: API Versioning Strategy — CRUD Handlers
//
// Endpoints:
//
//	GET    /api/v1/versions              — список версий
//	POST   /api/v1/versions              — новая версия (admin)
//	PUT    /api/v1/versions/{version}    — обновить метаданные (admin)
//	GET    /api/v1/changelog             — changelog
//
// Соответствует:
//   - IEC 62443-3-3 SL-2 (Zone 2 — DMZ): Управление изменениями
//   - ISO 27001 A.12.4.1: Audit trail для изменений
//   - OWASP ASVS V1-V17 (полный спектр контролей)
//   - OWASP ASVS V5.1 (Whitelist validation)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

// versionKeyPattern — валидация ключа версии (v1, v2, v10, ...)
var versionKeyPattern = regexp.MustCompile(`^v[0-9]+$`)

// ── GET /api/v1/versions ──────────────────────────────────────────────────

// handleListVersions возвращает список всех версий API.
//
// Compliance:
//   - OWASP ASVS V4.1 (Access control — admin only via middleware)
//   - OWASP ASVS V7.1 (Error handling — стандартизированный ответ)
func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	if s.versionStore == nil {
		RespondError(w, r, NewInternalError("version store not initialized", nil))
		return
	}

	versions, err := s.versionStore.ListVersions()
	if err != nil {
		RespondError(w, r, NewInternalError("failed to list versions", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"versions": versions,
		"total":    len(versions),
	})
}

// ── POST /api/v1/versions ────────────────────────────────────────────────

// createVersionRequest — тело запроса для создания версии.
type createVersionRequest struct {
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
}

// handleCreateVersion создаёт новую версию API (admin only).
//
// Compliance:
//   - OWASP ASVS V4.1 (Access control — admin only)
//   - OWASP ASVS V5.1 (Whitelist validation — version format)
//   - OWASP ASVS V5.2 (Input validation — body parsing)
//   - ISO 27001 A.12.4 (Audit trail — мутация логируется)
func (s *Server) handleCreateVersion(w http.ResponseWriter, r *http.Request) {
	if s.versionStore == nil {
		RespondError(w, r, NewInternalError("version store not initialized", nil))
		return
	}

	var req createVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	// Валидация версии (OWASP ASVS V5.1)
	req.Version = strings.TrimSpace(strings.ToLower(req.Version))
	if !versionKeyPattern.MatchString(req.Version) {
		RespondError(w, r, NewValidationError("invalid version format; expected v{number}, e.g. v2"))
		return
	}

	version := APIVersion(req.Version)

	if err := s.versionStore.CreateVersion(version, req.Changelog); err != nil {
		RespondError(w, r, NewInternalError("failed to create version", err))
		return
	}

	// Audit log (ISO 27001 A.12.4)
	userID := ""
	if claims := auth.GetClaims(r); claims != nil {
		userID = claims.UserID
	}
	s.logAudit(userID, "api_version.create", "api_version", string(version), nil, map[string]interface{}{
		"version":   string(version),
		"changelog": req.Changelog,
	})

	jsonResponse(w, http.StatusCreated, map[string]string{
		"version": string(version),
		"status":  "created",
	})
}

// ── PUT /api/v1/versions/{version} ────────────────────────────────────────

// updateVersionRequest — тело запроса для обновления метаданных версии.
type updateVersionRequest struct {
	Deprecated   bool   `json:"deprecated"`
	Sunset       string `json:"sunset,omitempty"` // RFC 3339
	DeprecatedAt string `json:"deprecated_at,omitempty"`
	Changelog    string `json:"changelog"`
}

// handleUpdateVersion обновляет метаданные версии API (admin only).
//
// Compliance:
//   - OWASP ASVS V4.1 (Access control — admin only)
//   - OWASP ASVS V5.1 (Whitelist validation — version from URL param)
//   - OWASP ASVS V5.2 (Input validation — body parsing)
//   - ISO 27001 A.12.4 (Audit trail — мутация логируется)
func (s *Server) handleUpdateVersion(w http.ResponseWriter, r *http.Request) {
	if s.versionStore == nil {
		RespondError(w, r, NewInternalError("version store not initialized", nil))
		return
	}

	versionStr := chi.URLParam(r, "version")
	if !versionKeyPattern.MatchString(versionStr) {
		RespondError(w, r, NewValidationError("invalid version format; expected v{number}"))
		return
	}

	var req updateVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	// Валидация sunset date (RFC 3339)
	if req.Sunset != "" {
		if _, err := time.Parse(time.RFC3339, req.Sunset); err != nil {
			RespondError(w, r, NewValidationError("invalid sunset date; expected RFC 3339 format"))
			return
		}
	}
	if req.DeprecatedAt != "" {
		if _, err := time.Parse(time.RFC3339, req.DeprecatedAt); err != nil {
			RespondError(w, r, NewValidationError("invalid deprecated_at date; expected RFC 3339 format"))
			return
		}
	}

	info := VersionInfo{
		Version:      APIVersion(versionStr),
		Deprecated:   req.Deprecated,
		Sunset:       req.Sunset,
		DeprecatedAt: req.DeprecatedAt,
		Changelog:    req.Changelog,
	}

	if err := s.versionStore.UpdateVersion(APIVersion(versionStr), info); err != nil {
		RespondError(w, r, NewInternalError("failed to update version", err))
		return
	}

	// Audit log (ISO 27001 A.12.4)
	userID := ""
	if claims := auth.GetClaims(r); claims != nil {
		userID = claims.UserID
	}
	s.logAudit(userID, "api_version.update", "api_version", versionStr, nil, map[string]interface{}{
		"version":    versionStr,
		"deprecated": req.Deprecated,
		"sunset":     req.Sunset,
	})

	jsonResponse(w, http.StatusOK, map[string]string{
		"version": versionStr,
		"status":  "updated",
	})
}

// ── GET /api/v1/changelog ────────────────────────────────────────────────

// handleGetChangelog возвращает changelog API.
//
// Compliance:
//   - OWASP ASVS V7.1 (Error handling — стандартизированный ответ)
func (s *Server) handleGetChangelog(w http.ResponseWriter, r *http.Request) {
	if s.versionStore == nil {
		RespondError(w, r, NewInternalError("version store not initialized", nil))
		return
	}

	versions, err := s.versionStore.ListVersions()
	if err != nil {
		RespondError(w, r, NewInternalError("failed to load changelog", err))
		return
	}

	type changelogEntry struct {
		Version    string `json:"version"`
		Date       string `json:"date"`
		Change     string `json:"change"`
		Deprecated bool   `json:"deprecated"`
		Sunset     string `json:"sunset,omitempty"`
	}

	entries := make([]changelogEntry, 0, len(versions))
	for _, v := range versions {
		entries = append(entries, changelogEntry{
			Version:    string(v.Version),
			Date:       v.ReleasedAt,
			Change:     v.Changelog,
			Deprecated: v.Deprecated,
			Sunset:     v.Sunset,
		})
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"changelog": entries,
		"total":     len(entries),
	})
}
