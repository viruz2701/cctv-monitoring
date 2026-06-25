// Package api — Camera Specs handlers (P0-9)
//
// API endpoints для каталога камер:
//   - GET /api/v1/camera-models/brands — список брендов
//   - GET /api/v1/camera-models/models?brand=X — модели бренда
//   - GET /api/v1/camera-models/search?q=X — поиск по brand/model
//   - GET /api/v1/camera-models/{brand}/{model} — детали модели
//   - POST /api/v1/camera-models/import — импорт JSON (admin only)
//   - POST /api/v1/camera-models/seed — вставка seed-данных (admin only)
//
// Соответствует:
//   - OWASP ASVS V5 (Input validation — whitelist params)
//   - OWASP ASVS V7 (Error handling — respondError с traceID)
//   - ISO 27001 A.8.1.2 (Asset inventory)
//   - IEC 62443 SR 3.1 (Identification of IACS devices)
package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/db"
)

// ── List Brands ──────────────────────────────────────────────────────────

// handleListCameraBrands возвращает список всех брендов камер.
// GET /api/v1/camera-models/brands
func (s *Server) handleListCameraBrands(w http.ResponseWriter, r *http.Request) {
	brands, err := s.db.ListBrands(r.Context())
	if err != nil {
		s.logger.Error("failed to list camera brands", "error", err)
		respondError(w, r, NewInternalError("failed to list camera brands", err))
		return
	}

	if brands == nil {
		brands = []db.CameraBrand{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"brands": brands,
	})
}

// ── List Models ─────────────────────────────────────────────────────────

// handleListCameraModels возвращает список моделей для указанного бренда.
// GET /api/v1/camera-models/models?brand=X
func (s *Server) handleListCameraModels(w http.ResponseWriter, r *http.Request) {
	brand := strings.TrimSpace(r.URL.Query().Get("brand"))
	if brand == "" {
		respondError(w, r, NewValidationError("brand query parameter is required"))
		return
	}

	models, err := s.db.ListModels(r.Context(), brand)
	if err != nil {
		s.logger.Error("failed to list camera models", "brand", brand, "error", err)
		respondError(w, r, NewInternalError("failed to list camera models", err))
		return
	}

	if models == nil {
		models = []db.CameraModelSummary{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"brand":  brand,
		"models": models,
	})
}

// ── Search Models ───────────────────────────────────────────────────────

// handleSearchCameraModels ищет камеры по brand или model.
// GET /api/v1/camera-models/search?q=X&limit=20
func (s *Server) handleSearchCameraModels(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" || len(query) < 2 {
		respondError(w, r, NewValidationError("search query (q) must be at least 2 characters"))
		return
	}

	limit := 20 // default

	models, err := s.db.SearchModels(r.Context(), query, limit)
	if err != nil {
		s.logger.Error("failed to search camera models", "query", query, "error", err)
		respondError(w, r, NewInternalError("failed to search camera models", err))
		return
	}

	if models == nil {
		models = []db.CameraModelSummary{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"query":  query,
		"models": models,
	})
}

// ── Get Camera Specs ────────────────────────────────────────────────────

// handleGetCameraSpecs возвращает детальные характеристики модели.
// GET /api/v1/camera-models/{brand}/{model}
func (s *Server) handleGetCameraSpecs(w http.ResponseWriter, r *http.Request) {
	brand := chi.URLParam(r, "brand")
	model := chi.URLParam(r, "model")

	if brand == "" || model == "" {
		respondError(w, r, NewValidationError("brand and model are required"))
		return
	}

	spec, err := s.db.GetCameraSpecs(r.Context(), brand, model)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			respondError(w, r, NewNotFoundError("camera model not found"))
			return
		}
		s.logger.Error("failed to get camera specs", "brand", brand, "model", model, "error", err)
		respondError(w, r, NewInternalError("failed to get camera specs", err))
		return
	}

	jsonResponse(w, http.StatusOK, spec)
}

// ── Import Camera Specs (Admin only) ────────────────────────────────────

// handleImportCameraSpecs импортирует камеры из JSON-массива.
// POST /api/v1/camera-models/import
// Body: [{"brand":"Hikvision","model":"DS-2CD2386G2-I",...}]
// Требует роль admin.
func (s *Server) handleImportCameraSpecs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, r, NewBadRequestError("failed to read request body"))
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		respondError(w, r, NewValidationError("request body is empty"))
		return
	}

	result, err := s.db.ImportFromJSON(r.Context(), body)
	if err != nil {
		s.logger.Error("failed to import camera specs", "error", err)
		respondError(w, r, NewInternalError("failed to import camera specs", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message":  "import completed",
		"inserted": result.Inserted,
		"updated":  result.Updated,
		"skipped":  result.Skipped,
		"errors":   result.Errors,
	})
}

// ── Seed Camera Specs (Admin only) ──────────────────────────────────────

// handleSeedCameraSpecs вставляет seed-данные (10 популярных моделей).
// POST /api/v1/camera-models/seed
// Требует роль admin.
func (s *Server) handleSeedCameraSpecs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil || claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	if err := s.db.SeedCameraSpecs(r.Context()); err != nil {
		s.logger.Error("failed to seed camera specs", "error", err)
		respondError(w, r, NewInternalError("failed to seed camera specs", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "seed data inserted successfully",
	})
}
