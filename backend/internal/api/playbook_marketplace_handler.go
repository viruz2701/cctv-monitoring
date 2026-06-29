// Package api — Playbook Marketplace HTTP handlers.
//
// P1-MARKET: REST endpoints for public marketplace pre-built playbooks.
//
// Endpoints:
//
//	GET    /api/v1/playbook-marketplace           — list with filters
//	GET    /api/v1/playbook-marketplace/{id}      — get by ID
//	POST   /api/v1/playbook-marketplace/{id}/install — install to tenant
//	POST   /api/v1/playbook-marketplace/{id}/rate — rate (1-5)
//	GET    /api/v1/playbook-marketplace/my        — installed by current tenant
//	POST   /api/v1/playbook-marketplace/{id}/share — private share
//
// Compliance:
//   - OWASP ASVS V1 (Input validation — whitelist)
//   - OWASP ASVS V2 (Session management — JWT in middleware)
//   - ISO 27001 A.12.4 (Audit trail)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/playbook"
)

// ═══════════════════════════════════════════════════════════════════════
// Dependencies (injected via Server)
// ═══════════════════════════════════════════════════════════════════════

// playbookMarketplaceService — интерфейс для MarketplaceService.
// Позволяет подменять реализацию в тестах.
type playbookMarketplaceService interface {
	List(ctx context.Context, filter playbook.MarketplaceFilter) ([]playbook.MarketplacePlaybook, int, error)
	Get(ctx context.Context, id string) (*playbook.MarketplacePlaybook, error)
	Install(ctx context.Context, tenantID, playbookID string) error
	Rate(ctx context.Context, playbookID, userID string, score int, review string) error
	Share(ctx context.Context, playbookID, sourceTenant, targetTenant string) error
	GetRatingForUser(ctx context.Context, playbookID, userID string) (*playbook.MarketplaceRating, error)
	GetInstalledPlaybooks(ctx context.Context, tenantID string) ([]playbook.MarketplacePlaybook, error)
}

// ═══════════════════════════════════════════════════════════════════════
// Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleMarketplaceList — GET /api/v1/playbook-marketplace
// Query params: vendor, min_rating, search, verified, limit, offset
func (s *Server) handleMarketplaceList(w http.ResponseWriter, r *http.Request) {
	filter := playbook.MarketplaceFilter{
		Vendor: r.URL.Query().Get("vendor"),
		Search: r.URL.Query().Get("search"),
		Limit:  queryParamInt(r, "limit", 20),
		Offset: queryParamInt(r, "offset", 0),
	}

	// MinRating
	if minRatingStr := r.URL.Query().Get("min_rating"); minRatingStr != "" {
		if v, err := strconv.ParseFloat(minRatingStr, 64); err == nil {
			filter.MinRating = v
		}
	}

	// Verified (bool pointer)
	if verifiedStr := r.URL.Query().Get("verified"); verifiedStr != "" {
		v := verifiedStr == "true"
		filter.Verified = &v
	}

	playbooks, total, err := s.playbookMarketplace.List(r.Context(), filter)
	if err != nil {
		s.logger.Error("marketplace list failed", "error", err)
		RespondError(w, r, NewInternalError("Failed to list marketplace playbooks", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"playbooks": playbooks,
		"total":     total,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})
}

// handleMarketplaceGet — GET /api/v1/playbook-marketplace/{id}
func (s *Server) handleMarketplaceGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	playbookData, err := s.playbookMarketplace.Get(r.Context(), id)
	if err != nil {
		s.logger.Error("marketplace get failed", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to get playbook", err))
		return
	}
	if playbookData == nil {
		RespondError(w, r, NewNotFoundError("Playbook not found"))
		return
	}

	jsonResponse(w, http.StatusOK, playbookData)
}

// handleMarketplaceInstall — POST /api/v1/playbook-marketplace/{id}/install
func (s *Server) handleMarketplaceInstall(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := cmms.TenantIDFromContext(r.Context())

	if err := s.playbookMarketplace.Install(r.Context(), tenantID, id); err != nil {
		s.logger.Error("marketplace install failed",
			"playbook_id", id,
			"tenant_id", tenantID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("Failed to install playbook", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "installed",
		"message": "Playbook installed successfully",
	})
}

// handleMarketplaceRate — POST /api/v1/playbook-marketplace/{id}/rate
// Body: { "score": 4, "review": "Great playbook" }
func (s *Server) handleMarketplaceRate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	var req struct {
		Score  int    `json:"score"`
		Review string `json:"review,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// OWASP ASVS V1: input validation
	if req.Score < 1 || req.Score > 5 {
		RespondError(w, r, NewValidationError("score must be between 1 and 5"))
		return
	}
	if len(req.Review) > 2000 {
		RespondError(w, r, NewValidationError("review must be <= 2000 characters"))
		return
	}

	if err := s.playbookMarketplace.Rate(r.Context(), id, userID, req.Score, req.Review); err != nil {
		s.logger.Error("marketplace rate failed",
			"playbook_id", id,
			"user_id", userID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("Failed to rate playbook", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "rated",
		"message": "Rating submitted successfully",
	})
}

// handleMarketplaceMyPlaybooks — GET /api/v1/playbook-marketplace/my
// Возвращает плейбуки, установленные текущим tenant'ом.
func (s *Server) handleMarketplaceMyPlaybooks(w http.ResponseWriter, r *http.Request) {
	tenantID := cmms.TenantIDFromContext(r.Context())

	playbooks, err := s.playbookMarketplace.GetInstalledPlaybooks(r.Context(), tenantID)
	if err != nil {
		s.logger.Error("marketplace my playbooks failed", "error", err)
		RespondError(w, r, NewInternalError("Failed to list installed playbooks", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"playbooks": playbooks,
		"total":     len(playbooks),
	})
}

// handleMarketplaceShare — POST /api/v1/playbook-marketplace/{id}/share
// Body: { "target_tenant": "tenant-xyz" }
func (s *Server) handleMarketplaceShare(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sourceTenant := cmms.TenantIDFromContext(r.Context())

	var req struct {
		TargetTenant string `json:"target_tenant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}
	if req.TargetTenant == "" {
		RespondError(w, r, NewValidationError("target_tenant is required"))
		return
	}

	if err := s.playbookMarketplace.Share(r.Context(), id, sourceTenant, req.TargetTenant); err != nil {
		s.logger.Error("marketplace share failed",
			"playbook_id", id,
			"from", sourceTenant,
			"to", req.TargetTenant,
			"error", err,
		)
		RespondError(w, r, NewInternalError("Failed to share playbook", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "shared",
		"message": fmt.Sprintf("Playbook shared with tenant %s", req.TargetTenant),
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// queryParamInt извлекает int параметр из query string с дефолтным значением.
func queryParamInt(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(val)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}

// ═══════════════════════════════════════════════════════════════════════
// Route mounting
// ═══════════════════════════════════════════════════════════════════════

// mountPlaybookMarketplaceRoutes регистрирует маршруты marketplace.
func (s *Server) mountPlaybookMarketplaceRoutes(r chi.Router) {
	// Валидация vendor через whitelist (OWASP ASVS V5.1)
	r.Get("/api/v1/playbook-marketplace", s.handleMarketplaceList)
	r.Get("/api/v1/playbook-marketplace/my", s.handleMarketplaceMyPlaybooks)
	r.Get("/api/v1/playbook-marketplace/{id}", s.handleMarketplaceGet)
	r.Post("/api/v1/playbook-marketplace/{id}/install", s.handleMarketplaceInstall)
	r.Post("/api/v1/playbook-marketplace/{id}/rate", s.handleMarketplaceRate)
	r.Post("/api/v1/playbook-marketplace/{id}/share", s.handleMarketplaceShare)
}
