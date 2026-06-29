// Package api — HTTP handlers для differential sync (P1-SYNC).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: Differential Sync for Mobile
//
// Endpoints:
//
//	GET /api/v1/sync/{entity}?since=ISO8601&compression=gzip|brotli
//	GET /api/v1/sync/status
//
// Соответствует:
//   - IEC 62443-3-3 SR 3.1, SR 5.1 (Network segmentation)
//   - ISO 27001 A.12.4 (Audit trail), A.13.1 (Network security)
//   - OWASP ASVS L3 V1 (Input validation), V2 (Authentication),
//     V3 (Session management), V4 (Access control), V5 (Validation)
//   - Приказ ОАЦ №66 п. 7.18.2 (mTLS, secure communication)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/api/sync"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/trace"
)

// mountSyncRoutes монтирует маршруты differential sync (P1-SYNC).
// Вызывается из MountRoutes внутри защищённой группы (JWT required).
func (s *Server) mountSyncRoutes(r chi.Router) {
	// Инициализируем DiffService при первом вызове
	if s.diffService == nil && s.db != nil && s.db.Pool != nil {
		s.diffService = sync.NewDiffService(s.db.Pool, s.logger)
	}

	if s.diffService == nil {
		s.logger.Warn("P1-SYNC: diff service not available, sync routes disabled")
		return
	}

	r.Get("/api/v1/sync/status", s.handleSyncStatus)
	r.Get("/api/v1/sync/{entity}", s.handleSyncGetDelta)
}

// handleSyncGetDelta обрабатывает GET /api/v1/sync/{entity}?since=ISO8601&compression=gzip.
//
// Параметры запроса:
//   - since: ISO8601 timestamp последней синхронизации (опционально)
//   - compression: gzip или brotli (опционально)
//   - page_size: количество записей на страницу (опционально, default: 500)
//
// Ответ: DeltaResponse (JSON).
//
// OWASP ASVS L3:
//   - V1: Input validation (entity path validation, since format validation)
//   - V2: Authentication (JWT via AuthMiddleware)
//   - V5: Output encoding (JSON response)
func (s *Server) handleSyncGetDelta(w http.ResponseWriter, r *http.Request) {
	traceID := trace.FromContext(r.Context())

	// ── 1. Извлекаем tenantID из контекста (устанавливается TenantMiddleware) ──
	tenantID := cmms.TenantIDFromContext(r.Context())
	if tenantID == "" {
		// Fallback: извлекаем из JWT claims
		claims := auth.GetClaims(r)
		if claims == nil || claims.TenantID == "" {
			RespondError(w, r, NewUnauthorizedError("Tenant ID required"))
			return
		}
		tenantID = claims.TenantID
	}

	// ── 2. Валидация path параметра entity ──────────────────────────────────
	entity := chi.URLParam(r, "entity")
	if entity == "" {
		RespondError(w, r, NewBadRequestError("Entity path parameter required"))
		return
	}

	// Whitelist validation (OWASP ASVS L3 V1)
	allowed := sync.AllowedEntities()
	if _, ok := allowed[entity]; !ok {
		RespondError(w, r, NewValidationError(fmt.Sprintf("Unsupported entity: %s", entity)))
		return
	}

	// ── 3. Парсим query параметры ──────────────────────────────────────────
	var since time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			// Пробуем nano-формат
			since, err = time.Parse(time.RFC3339Nano, sinceStr)
			if err != nil {
				RespondError(w, r, NewValidationError("Invalid since format: use ISO8601 (e.g., 2026-06-28T12:00:00Z)"))
				return
			}
		}
	}

	// Парсим compression
	compression := sync.ParseCompressionType(r.URL.Query().Get("compression"))

	// Парсим page_size
	pageSize := 0
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if _, err := fmt.Sscanf(ps, "%d", &pageSize); err != nil {
			RespondError(w, r, NewValidationError("Invalid page_size: must be integer"))
			return
		}
	}

	s.logger.Debug("sync: get delta request",
		"entity", entity,
		"since", since,
		"compression", compression,
		"page_size", pageSize,
		"trace_id", traceID,
	)

	// ── 4. Получаем delta ──────────────────────────────────────────────────
	opts := []sync.DeltaOption{
		sync.WithCompression(compression),
	}
	if pageSize > 0 {
		opts = append(opts, sync.WithPageSize(pageSize))
	}

	delta, err := s.diffService.GetDelta(r.Context(), tenantID, entity, since, opts...)
	if err != nil {
		s.logger.Error("sync: get delta failed",
			"entity", entity,
			"error", err,
			"trace_id", traceID,
		)
		RespondError(w, r, NewInternalError("Failed to get sync delta", err))
		return
	}

	// ── 5. Сериализуем ответ ──────────────────────────────────────────────
	var respData []byte
	respData, err = json.Marshal(delta)
	if err != nil {
		RespondError(w, r, NewInternalError("Failed to serialize response", err))
		return
	}

	// ── 6. Сжатие ответа (Content-Encoding) ────────────────────────────────
	setContentEncoding := ""
	switch compression {
	case sync.CompressionGzip:
		compressed, err := compressGzip(respData)
		if err != nil {
			s.logger.Error("sync: gzip compression failed",
				"error", err,
				"trace_id", traceID,
			)
			break
		}
		respData = compressed
		setContentEncoding = "gzip"
	case sync.CompressionBrotli:
		compressed, err := compressBrotli(respData)
		if err != nil {
			s.logger.Error("sync: brotli compression failed",
				"error", err,
				"trace_id", traceID,
			)
			break
		}
		respData = compressed
		setContentEncoding = "br"
	}

	// ── 7. Запись метрики bandwidth (async) ────────────────────────────────
	go func() {
		metric := sync.SyncMetricsEntry{
			Entity:     entity,
			BytesSent:  int64(len(respData)),
			Changes:    len(delta.Changes),
			SyncAt:     time.Now().UTC(),
			TenantID:   tenantID,
			Compressed: setContentEncoding != "",
		}
		if err := s.diffService.RecordSyncMetric(r.Context(), metric); err != nil {
			s.logger.Error("sync: failed to record metric",
				"entity", entity,
				"error", err,
				"trace_id", traceID,
			)
		}
	}()

	// ── 8. Отправляем ответ ───────────────────────────────────────────────
	w.Header().Set("Content-Type", "application/json")
	if setContentEncoding != "" {
		w.Header().Set("Content-Encoding", setContentEncoding)
	}
	w.Header().Set("X-Trace-ID", traceID)
	w.Header().Set("X-Sync-Timestamp", delta.Timestamp.Format(time.RFC3339))
	w.Header().Set("X-Sync-Has-More", fmt.Sprintf("%t", delta.HasMore))

	if delta.HasMore {
		w.Header().Set("X-Sync-Next-Page", fmt.Sprintf(
			"/api/v1/sync/%s?since=%s&compression=%s",
			entity,
			delta.Timestamp.Format(time.RFC3339Nano),
			compression,
		))
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respData)
}

// handleSyncStatus обрабатывает GET /api/v1/sync/status.
//
// Возвращает:
//   - bandwidth_usage_bytes: использованный bandwidth за 24h
//   - last_sync_at: время последней синхронизации по каждой сущности
//   - total_syncs: общее количество синхронизаций
//   - total_changes: общее количество изменений
//
// OWASP ASVS L3:
//   - V2: Authentication (JWT required)
//   - V4: Access control (tenant-scoped)
func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	traceID := trace.FromContext(r.Context())

	// Извлекаем tenantID
	tenantID := cmms.TenantIDFromContext(r.Context())
	if tenantID == "" {
		claims := auth.GetClaims(r)
		if claims == nil || claims.TenantID == "" {
			RespondError(w, r, NewUnauthorizedError("Tenant ID required"))
			return
		}
		tenantID = claims.TenantID
	}

	if s.diffService == nil {
		RespondError(w, r, NewInternalError("Diff service not available", nil))
		return
	}

	status, err := s.diffService.GetSyncStatus(r.Context(), tenantID)
	if err != nil {
		s.logger.Error("sync: get status failed",
			"error", err,
			"trace_id", traceID,
		)
		RespondError(w, r, NewInternalError("Failed to get sync status", err))
		return
	}

	jsonResponse(w, http.StatusOK, status)
}

// ── Compression Helpers (handler-level) ─────────────────────────────────

// compressGzip сжимает данные с помощью gzip.
func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, fmt.Errorf("gzip writer: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return buf.Bytes(), nil
}

// compressBrotli сжимает данные с помощью brotli.
// Note: Go stdlib does not have brotli. Falls back to gzip.
// TODO: Заменить на github.com/andybalholm/brotli в production.
func compressBrotli(data []byte) ([]byte, error) {
	// Fallback to gzip until brotli dependency is added
	return compressGzip(data)
}
