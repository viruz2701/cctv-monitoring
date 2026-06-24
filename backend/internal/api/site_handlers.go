// Package api — site domain handlers (sites + spare part categories).
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Sites Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listSites(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})
	if name := r.URL.Query().Get("name"); name != "" {
		filters["name"] = name
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}
	if city := r.URL.Query().Get("city"); city != "" {
		filters["city"] = city
	}

	sites, err := s.cmmsRouter.GetSites(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get sites", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if sites == nil {
		sites = []models.Site{}
	}
	jsonResponse(w, http.StatusOK, sites)
}

func (s *Server) getSite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	site, err := s.cmmsRouter.GetSite(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Site not found"))
		return
	}
	jsonResponse(w, http.StatusOK, site)
}

func (s *Server) createSite(w http.ResponseWriter, r *http.Request) {
	var site models.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if site.Name == "" {
		respondError(w, r, NewBadRequestError("name is required"))
		return
	}
	if site.Status == "" {
		site.Status = "active"
	}

	if err := s.cmmsRouter.CreateSite(r.Context(), &site); err != nil {
		s.logger.Error("Failed to create site", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_site", "site", site.ID, nil, site)
	jsonResponse(w, http.StatusCreated, site)
}

func (s *Server) updateSite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if err := s.cmmsRouter.UpdateSite(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update site", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_site", "site", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteSite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	if err := s.cmmsRouter.DeleteSite(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete site", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_site", "site", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Part Categories Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listSparePartCategories(w http.ResponseWriter, r *http.Request) {
	// Support pagination query params
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	_ = limit
	_ = offset

	categories, err := s.cmmsRouter.GetCategories(r.Context())
	if err != nil {
		s.logger.Error("Failed to get spare part categories", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if categories == nil {
		categories = []models.SparePartCategory{}
	}
	jsonResponse(w, http.StatusOK, categories)
}

func (s *Server) createSparePartCategory(w http.ResponseWriter, r *http.Request) {
	var cat models.SparePartCategory
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if cat.Name == "" {
		respondError(w, r, NewBadRequestError("name is required"))
		return
	}

	if err := s.cmmsRouter.CreateCategory(r.Context(), &cat); err != nil {
		s.logger.Error("Failed to create spare part category", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_spare_part_category", "spare_part_category", cat.ID, nil, cat)
	jsonResponse(w, http.StatusCreated, cat)
}

func (s *Server) updateSparePartCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if err := s.cmmsRouter.UpdateCategory(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update spare part category", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_spare_part_category", "spare_part_category", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteSparePartCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	if err := s.cmmsRouter.DeleteCategory(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete spare part category", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_spare_part_category", "spare_part_category", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// parsePagination извлекает лимит и offset из query-параметров.
func parsePagination(r *http.Request) (int, int) {
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}
