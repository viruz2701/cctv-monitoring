// Package api — Community Protocol Registry routes.
//
// ═══════════════════════════════════════════════════════════════════════════
// PROTO-07: Community Protocol Registry Routes
//
// Route mounting для community реестра дескрипторов.
// GET маршруты — публичные (без JWT).
// POST маршруты — защищённые (JWT required).
//
// Rate limiting для публикации: 5 req/min/user (OWASP ASVS V2.2.1)
//
// Compliance:
//   - OWASP ASVS V2.2.1 (Rate limiting для мутирующих запросов)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - IEC 62443-3-3 SL-3 (Zone separation)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/community"
)

// ═══════════════════════════════════════════════════════════════════════
// Initialization
// ═══════════════════════════════════════════════════════════════════════

// initCommunityRegistry инициализирует Community Protocol Registry (PROTO-07).
//
// Создаёт PostgreSQL store для community дескрипторов, если БД доступна.
func (s *Server) initCommunityRegistry() {
	if s.db == nil || s.db.Pool == nil {
		s.logger.Warn("PROTO-07: no database connection, community registry disabled")
		return
	}

	store := community.NewStore(s.db.Pool, s.logger)
	s.communityRegistry = store

	s.logger.Info("PROTO-07: community protocol registry initialized")
}

// ═══════════════════════════════════════════════════════════════════════
// Public Routes (без JWT — read-only)
// ═══════════════════════════════════════════════════════════════════════

// mountPublicCommunityRegistryRoutes монтирует публичные GET маршруты.
//
// Rate limiting: без ограничения (кэшируется браузером/CDN)
//
// Routes:
//
//	GET /api/v1/community/descriptors              — список с пагинацией/поиском/фильтром
//	GET /api/v1/community/descriptors/{vendor}      — детали дескриптора
//	GET /api/v1/community/descriptors/{vendor}/download — скачать (счётчик)
func (s *Server) mountPublicCommunityRegistryRoutes(r chi.Router) {
	r.Get("/api/v1/community/descriptors", s.handleCommunityDescriptorList)
	r.Get("/api/v1/community/descriptors/{vendor}", s.handleCommunityDescriptorGet)
	r.Get("/api/v1/community/descriptors/{vendor}/download", s.handleCommunityDescriptorDownload)
}

// ═══════════════════════════════════════════════════════════════════════
// Protected Routes (JWT required — mutations)
// ═══════════════════════════════════════════════════════════════════════

// mountProtectedCommunityRegistryRoutes монтирует защищённые POST маршруты.
//
// Rate limiting: глобальный rate limiter (100 read/30 write per minute)
//
// Routes:
//
//	POST /api/v1/community/descriptors              — публикация (auth required)
//	POST /api/v1/community/descriptors/{vendor}/rate — оценка (1-5)
func (s *Server) mountProtectedCommunityRegistryRoutes(r chi.Router) {
	r.Post("/api/v1/community/descriptors", s.handleCommunityDescriptorPublish)
	r.Post("/api/v1/community/descriptors/{vendor}/rate", s.handleCommunityDescriptorRate)
}
