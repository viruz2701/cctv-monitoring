// Package api — Camera Specs routes (P0-9)
//
// Маршруты для каталога камер.
// Соответствует:
//   - OWASP ASVS V4 (Access Control — admin только для import/seed)
//   - OWASP ASVS V7 (Error Handling — через respondError)
//   - ISO 27001 A.8.1.2 (Asset inventory)
package api

import "github.com/go-chi/chi/v5"

// mountCameraModelRoutes регистрирует маршруты для каталога камер.
// Все маршруты защищены AuthMiddleware (вызывается из server.go).
func (s *Server) mountCameraModelRoutes(r chi.Router) {
	// ── Read endpoints (доступны всем аутентифицированным) ────────────
	// [x] V5 — Input validation в хендлерах
	// [x] V7 — Error handling через respondError

	// GET /api/v1/camera-models/brands — список брендов
	r.Get("/api/v1/camera-models/brands", s.handleListCameraBrands)

	// GET /api/v1/camera-models/models?brand=X — модели бренда
	r.Get("/api/v1/camera-models/models", s.handleListCameraModels)

	// GET /api/v1/camera-models/search?q=X — поиск по brand/model
	r.Get("/api/v1/camera-models/search", s.handleSearchCameraModels)

	// GET /api/v1/camera-models/{brand}/{model} — детали модели
	r.Get("/api/v1/camera-models/{brand}/{model}", s.handleGetCameraSpecs)

	// ── Write endpoints (Admin only — проверка в хендлерах) ──────────
	// [x] V4 — Access Control (RBAC: admin only)

	// POST /api/v1/camera-models/import — импорт JSON (admin only)
	r.Post("/api/v1/camera-models/import", s.handleImportCameraSpecs)

	// POST /api/v1/camera-models/seed — seed-данные (admin only)
	r.Post("/api/v1/camera-models/seed", s.handleSeedCameraSpecs)
}
