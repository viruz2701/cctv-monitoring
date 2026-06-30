// Package api — Community Protocol Registry HTTP handlers.
//
// ═══════════════════════════════════════════════════════════════════════════
// PROTO-07: Community Protocol Registry (P2-EDGE)
//
// Публичный реестр Protocol Descriptor'ов (как Docker Hub),
// где community может публиковать и обмениваться дескрипторами
// для различных вендоров CCTV.
//
// Endpoints:
//
//	GET    /api/v1/community/descriptors              — список с пагинацией/поиском/фильтром
//	GET    /api/v1/community/descriptors/:vendor      — детали дескриптора
//	POST   /api/v1/community/descriptors              — публикация (auth required)
//	POST   /api/v1/community/descriptors/:vendor/rate — оценка (1-5)
//	GET    /api/v1/community/descriptors/:vendor/download — скачать (счётчик)
//
// Compliance:
//   - OWASP ASVS V1 (Input validation — whitelist)
//   - OWASP ASVS V2 (Session management — JWT in middleware)
//   - OWASP ASVS V5.1 (Input validation — whitelist approach)
//   - ISO 27001 A.12.4 (Audit trail)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Dependencies (injected via Server)
// ═══════════════════════════════════════════════════════════════════════

// communityRegistryService — интерфейс для Store.
// Позволяет подменять реализацию в тестах.
type communityRegistryService interface {
	List(ctx context.Context, filter models.CommunityDescriptorFilter) (*models.CommunityDescriptorListResponse, error)
	GetByVendor(ctx context.Context, vendor string) (*models.CommunityDescriptor, error)
	Publish(ctx context.Context, req models.PublishDescriptorRequest, authorID string) (*models.CommunityDescriptor, error)
	Rate(ctx context.Context, descriptorID, userID string, score int) error
	IncrementDownload(ctx context.Context, vendor string) error
	GetDescriptorIDByVendor(ctx context.Context, vendor string) (string, error)
}

// ═══════════════════════════════════════════════════════════════════════
// Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleCommunityDescriptorList — GET /api/v1/community/descriptors
//
// Query params: search, min_rating, verified, page, page_size, sort_by, sort_dir
func (s *Server) handleCommunityDescriptorList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := models.CommunityDescriptorFilter{
		Search:   q.Get("search"),
		Page:     queryParamInt(r, "page", 1),
		PageSize: queryParamInt(r, "page_size", 20),
		SortBy:   q.Get("sort_by"),
		SortDir:  q.Get("sort_dir"),
	}

	// MinRating
	if minRatingStr := q.Get("min_rating"); minRatingStr != "" {
		if v, err := strconv.ParseFloat(minRatingStr, 64); err == nil && v >= 0 && v <= 5 {
			filter.MinRating = v
		}
	}

	// Verified (bool pointer)
	if verifiedStr := q.Get("verified"); verifiedStr != "" {
		v := verifiedStr == "true"
		filter.Verified = &v
	}

	result, err := s.communityRegistry.List(r.Context(), filter)
	if err != nil {
		s.logger.Error("community list failed", "error", err)
		RespondError(w, r, NewInternalError("Failed to list community descriptors", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}

// handleCommunityDescriptorGet — GET /api/v1/community/descriptors/{vendor}
func (s *Server) handleCommunityDescriptorGet(w http.ResponseWriter, r *http.Request) {
	vendor := chi.URLParam(r, "vendor")
	if vendor == "" {
		RespondError(w, r, NewValidationError("vendor is required"))
		return
	}

	descriptor, err := s.communityRegistry.GetByVendor(r.Context(), vendor)
	if err != nil {
		s.logger.Error("community get failed", "vendor", vendor, "error", err)
		RespondError(w, r, NewInternalError("Failed to get descriptor", err))
		return
	}
	if descriptor == nil {
		RespondError(w, r, NewNotFoundError("Descriptor not found for vendor: "+vendor))
		return
	}

	jsonResponse(w, http.StatusOK, descriptor)
}

// handleCommunityDescriptorPublish — POST /api/v1/community/descriptors
//
// Body: { "vendor": "hikvision", "version": "1.0", "descriptor": {...} }
// Auth required: JWT middleware обеспечивает user_id в контексте.
func (s *Server) handleCommunityDescriptorPublish(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())

	var req models.PublishDescriptorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// OWASP ASVS V1: input validation
	v := NewValidator()
	v.Required("vendor", req.Vendor).
		MinLength("vendor", req.Vendor, 1).
		MaxLength("vendor", req.Vendor, 200).
		Required("version", req.Version).
		MaxLength("version", req.Version, 50)

	if req.Descriptor == nil || len(req.Descriptor) == 0 {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: "descriptor", Message: "required", Code: "REQUIRED",
		})
	}

	if !v.Valid() {
		respondValidationError(w, r, v.ToValidationErrors())
		return
	}

	descriptor, err := s.communityRegistry.Publish(r.Context(), req, userID)
	if err != nil {
		s.logger.Error("community publish failed",
			"vendor", req.Vendor,
			"user_id", userID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("Failed to publish descriptor", err))
		return
	}

	s.logger.Info("community descriptor published",
		"vendor", req.Vendor,
		"version", req.Version,
		"author", userID,
	)

	jsonResponse(w, http.StatusCreated, descriptor)
}

// handleCommunityDescriptorRate — POST /api/v1/community/descriptors/{vendor}/rate
//
// Body: { "score": 4 }
// Auth required.
func (s *Server) handleCommunityDescriptorRate(w http.ResponseWriter, r *http.Request) {
	vendor := chi.URLParam(r, "vendor")
	if vendor == "" {
		RespondError(w, r, NewValidationError("vendor is required"))
		return
	}

	userID := getUserIDFromContext(r.Context())

	var req models.RateDescriptorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// OWASP ASVS V1: input validation
	if req.Score < 1 || req.Score > 5 {
		RespondError(w, r, NewValidationError("score must be between 1 and 5"))
		return
	}

	// Получаем ID дескриптора по вендору
	descriptorID, err := s.communityRegistry.GetDescriptorIDByVendor(r.Context(), vendor)
	if err != nil {
		RespondError(w, r, NewNotFoundError("Descriptor not found for vendor: "+vendor))
		return
	}

	if err := s.communityRegistry.Rate(r.Context(), descriptorID, userID, req.Score); err != nil {
		s.logger.Error("community rate failed",
			"vendor", vendor,
			"user_id", userID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("Failed to rate descriptor", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "rated",
		"message": "Rating submitted successfully",
	})
}

// handleCommunityDescriptorDownload — GET /api/v1/community/descriptors/{vendor}/download
//
// Возвращает полный дескриптор и увеличивает счётчик скачиваний.
func (s *Server) handleCommunityDescriptorDownload(w http.ResponseWriter, r *http.Request) {
	vendor := chi.URLParam(r, "vendor")
	if vendor == "" {
		RespondError(w, r, NewValidationError("vendor is required"))
		return
	}

	// Увеличиваем счётчик (асинхронно — не блокируем ответ)
	if err := s.communityRegistry.IncrementDownload(r.Context(), vendor); err != nil {
		s.logger.Warn("community download count increment failed",
			"vendor", vendor,
			"error", err,
		)
		// Не возвращаем ошибку — счётчик может не обновиться,
		// но сам дескриптор должен быть отдан
	}

	// Получаем и возвращаем дескриптор
	descriptor, err := s.communityRegistry.GetByVendor(r.Context(), vendor)
	if err != nil {
		s.logger.Error("community download get failed", "vendor", vendor, "error", err)
		RespondError(w, r, NewInternalError("Failed to get descriptor for download", err))
		return
	}
	if descriptor == nil {
		RespondError(w, r, NewNotFoundError("Descriptor not found for vendor: "+vendor))
		return
	}

	jsonResponse(w, http.StatusOK, descriptor)
}
