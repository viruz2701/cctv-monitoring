package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/gatekeeper"
)

// dbSiteProvider адаптирует db.DB к интерфейсу gatekeeper.SiteProvider.
type dbSiteProvider struct {
	database *db.DB
}

func (p *dbSiteProvider) GetSiteInfo(ctx context.Context, workOrderID string) (*gatekeeper.SiteInfo, error) {
	info, err := p.database.GetSiteInfo(ctx, workOrderID)
	if err != nil {
		return nil, err
	}
	return &gatekeeper.SiteInfo{
		SiteID:               info.SiteID,
		SiteName:             info.SiteName,
		Latitude:             info.Latitude,
		Longitude:            info.Longitude,
		GeofenceRadiusMeters: info.GeofenceRadiusMeters,
	}, nil
}

// handleVerifyWorkOrder — POST /api/v1/mobile/work-orders/{id}/verify
// Выполняет Gatekeeper-верификацию (GPS + EXIF + AI) и выпускает verification token.
func (s *Server) handleVerifyWorkOrder(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	workOrderID := chi.URLParam(r, "id")
	if workOrderID == "" {
		http.Error(w, "work order id is required", http.StatusBadRequest)
		return
	}

	var req gatekeeper.VerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Создаём SiteProvider и Verifier
	provider := &dbSiteProvider{database: s.db}
	verifier := gatekeeper.NewVerifier(provider)

	// Выполняем верификацию
	resp, err := verifier.Verify(r.Context(), req, workOrderID, claims.UserID)
	if err != nil {
		s.logger.Error("Gatekeeper verification failed", "error", err, "work_order", workOrderID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Логируем результат верификации
	s.logAudit(claims.UserID, "gatekeeper_verify", "work_order", workOrderID, nil, resp)

	respondJSON(w, http.StatusOK, resp)
}
