// Package api — Admin routes for multi-region DR management (P3-1).
package api

import (
	"encoding/json"
	"fmt"
	"gb-telemetry-collector/internal/multiregion"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// mountAdminRoutes регистрирует admin-эндпоинты (P3-1).
// Все маршруты require JWT + роль admin.
func (s *Server) mountAdminRoutes(r chi.Router) {
	r.Route("/api/v1/admin", func(r chi.Router) {
		// P3-1: Tenant region management
		r.Get("/regions", s.handleListTenantRegions)
		r.Get("/regions/{tenant_id}", s.handleGetTenantRegion)
		r.Put("/regions/{tenant_id}", s.handleSetTenantRegion)

		// P3-1: Failover operations
		r.Post("/failover/{tenant_id}", s.handleFailoverTenant)
		r.Post("/failover/{tenant_id}/rollback", s.handleRollbackTenant)

		// P3-1: DR status
		r.Get("/dr/status", s.handleDRStatus)
	})
}

// handleListTenantRegions возвращает список всех tenant-region mapping.
func (s *Server) handleListTenantRegions(w http.ResponseWriter, r *http.Request) {
	if s.regionStore == nil {
		jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}

	regions, err := s.regionStore.ListAll(r.Context())
	if err != nil {
		RespondError(w, r, fmt.Errorf("list regions: %w", err))
		return
	}

	jsonResponse(w, http.StatusOK, regions)
}

// handleGetTenantRegion возвращает region для конкретного тенанта.
func (s *Server) handleGetTenantRegion(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, fmt.Errorf("tenant_id is required"))
		return
	}

	if s.regionStore == nil {
		RespondError(w, r, fmt.Errorf("region store not initialized"))
		return
	}

	tr, err := s.regionStore.GetTenantRegion(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, fmt.Errorf("get tenant region: %w", err))
		return
	}
	if tr == nil {
		RespondError(w, r, fmt.Errorf("tenant %s not found", tenantID))
		return
	}

	jsonResponse(w, http.StatusOK, tr)
}

// handleSetTenantRegion устанавливает или обновляет region для тенанта.
func (s *Server) handleSetTenantRegion(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, fmt.Errorf("tenant_id is required"))
		return
	}

	var req struct {
		PrimaryRegion  string `json:"primary_region"`
		FailoverRegion string `json:"failover_region"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}

	if s.regionStore == nil {
		RespondError(w, r, fmt.Errorf("region store not initialized"))
		return
	}

	tr := &multiregion.TenantRegion{
		TenantID:       tenantID,
		PrimaryRegion:  req.PrimaryRegion,
		FailoverRegion: req.FailoverRegion,
		Status:         "active",
	}

	if err := s.regionStore.SetTenantRegion(r.Context(), tr); err != nil {
		RespondError(w, r, fmt.Errorf("set tenant region: %w", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleFailoverTenant выполняет failover для указанного тенанта.
func (s *Server) handleFailoverTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, fmt.Errorf("tenant_id is required"))
		return
	}

	if s.failoverService == nil {
		RespondError(w, r, fmt.Errorf("failover service not initialized"))
		return
	}

	result, err := s.failoverService.ExecuteFailover(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, fmt.Errorf("failover: %w", err))
		return
	}

	statusCode := http.StatusOK
	if result.Status == "failed" {
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(result)
}

// handleRollbackTenant выполняет rollback failover для тенанта.
func (s *Server) handleRollbackTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenant_id")
	if tenantID == "" {
		RespondError(w, r, fmt.Errorf("tenant_id is required"))
		return
	}

	if s.regionStore == nil {
		RespondError(w, r, fmt.Errorf("region store not initialized"))
		return
	}

	// Rollback: возвращаем tenant в primary_region из failover_region
	tr, err := s.regionStore.GetTenantRegion(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, fmt.Errorf("get tenant: %w", err))
		return
	}
	if tr == nil {
		RespondError(w, r, fmt.Errorf("tenant %s not found", tenantID))
		return
	}

	// Сбрасываем статус на active
	if err := s.regionStore.UpdateTenantStatus(r.Context(), tenantID, "active"); err != nil {
		RespondError(w, r, fmt.Errorf("rollback: %w", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":         "rolled_back",
		"tenant_id":      tenantID,
		"primary_region": tr.PrimaryRegion,
	})
}

// handleDRStatus возвращает общий статус DR для всех регионов.
func (s *Server) handleDRStatus(w http.ResponseWriter, r *http.Request) {
	type regionStatus struct {
		Region        string `json:"region"`
		TenantCount   int    `json:"tenant_count"`
		FailoverCount int    `json:"failover_count"`
		Healthy       bool   `json:"healthy"`
	}

	var statuses []regionStatus

	if s.regionStore != nil {
		for _, region := range multiregion.ValidRegions {
			tenants, err := s.regionStore.ListByRegion(r.Context(), region)
			if err != nil {
				continue
			}

			failoverCount := 0
			for _, t := range tenants {
				if t.Status == "failover" {
					failoverCount++
				}
			}

			statuses = append(statuses, regionStatus{
				Region:        region,
				TenantCount:   len(tenants),
				FailoverCount: failoverCount,
				Healthy:       failoverCount == 0,
			})
		}
	}

	if statuses == nil {
		statuses = []regionStatus{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"regions":       statuses,
		"total_regions": len(multiregion.ValidRegions),
	})
}
