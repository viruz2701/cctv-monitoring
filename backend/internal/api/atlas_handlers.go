package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/cmms"
)

// ── Atlas CMMS Integration Handlers ──────────────────────────────

// atlasHealthCheck проверяет доступность внешнего Atlas CMMS API.
func (s *Server) atlasHealthCheck(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	atlasAdapter, ok := s.cmmsRouter.Adapter().(*cmms.AtlasAdapter)
	if !ok {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"status":  "not_configured",
			"message": "Atlas adapter is not active; using internal CMMS",
		})
		return
	}

	if err := atlasAdapter.HealthCheck(ctx); err != nil {
		s.logger.Warn("atlas health check failed", "error", err)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// atlasFallbackStatus возвращает размер очереди отложенных операций.
func (s *Server) atlasFallbackStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	atlasAdapter, ok := s.cmmsRouter.Adapter().(*cmms.AtlasAdapter)
	if !ok {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"queue_size": 0,
			"message":    "Atlas adapter is not active",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"queue_size": atlasAdapter.FallbackQueueSize(),
	})
}

// atlasRetryFallback запускает повторную отправку операций из fallback-очереди.
func (s *Server) atlasRetryFallback(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	atlasAdapter, ok := s.cmmsRouter.Adapter().(*cmms.AtlasAdapter)
	if !ok {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"success": 0,
			"failed":  0,
			"message": "Atlas adapter is not active",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	success, failed := atlasAdapter.RetryFallback(ctx)

	_ = s.db.SaveAudit(claims.UserID, "ATLAS_RETRY_FALLBACK", "atlas", "fallback", nil,
		map[string]int{"success": success, "failed": failed})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": success,
		"failed":  failed,
	})
}

// atlasSyncAsset синхронизирует устройство с Atlas CMMS как актив.
func (s *Server) atlasSyncAsset(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	deviceID := chi.URLParam(r, "deviceId")
	if deviceID == "" {
		RespondError(w, r, NewBadRequestError("deviceId required"))
		return
	}

	dev, ok := s.stateManager.Get(deviceID)
	if !ok {
		RespondError(w, r, NewNotFoundError("device not found"))
		return
	}

	atlasAdapter, ok := s.cmmsRouter.Adapter().(*cmms.AtlasAdapter)
	if !ok {
		jsonResponse(w, http.StatusOK, map[string]string{
			"status":  "skipped",
			"message": "Atlas adapter is not active; using internal CMMS",
		})
		return
	}

	assetData := map[string]interface{}{
		"device_id":   dev.DeviceID,
		"name":        dev.Name,
		"location":    dev.Location,
		"vendor_type": dev.VendorType,
		"status":      dev.Status,
		"p2p_brand":   dev.P2PBrand,
		"p2p_serial":  dev.P2PSerial,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := atlasAdapter.SyncAsset(ctx, deviceID, assetData); err != nil {
		s.logger.Error("atlas sync asset failed", "device_id", deviceID, "error", err)
		jsonResponse(w, http.StatusOK, map[string]string{
			"status": "queued",
			"error":  err.Error(),
		})
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "ATLAS_SYNC_ASSET", "device", deviceID, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "synced"})
}
