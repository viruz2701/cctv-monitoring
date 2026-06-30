// Package api — HTTP handlers for P3-DB Database Optimization endpoints.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-DB: Database Optimization
//
// Endpoints:
//
//	GET /api/v1/db/pool                — pool stats (PoolMonitor)
//	GET /api/v1/db/slow-queries         — slow queries (SlowQueryDetector)
//	GET /api/v1/db/index-recommendations — index recommendations
//
// Соответствие:
//   - IEC 62443-3-3 SR 4.2 (Resource Limitation)
//   - ISO 27001 A.12.6.1 (Capacity Management)
//   - СТБ 34.101.27 п. 7.3 (Управление ресурсами)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"net/http"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/respond"

	"github.com/go-chi/chi/v5"
)

// mountDBRoutes монтирует P3-DB Database Optimization endpoints.
// Все маршруты require JWT + роль admin.
func (s *Server) mountDBRoutes(r chi.Router) {
	r.Route("/api/v1/db", func(r chi.Router) {
		// P3-DB: Pool statistics
		r.Get("/pool", s.handleDBPoolStats)

		// P3-DB: Slow queries list
		r.Get("/slow-queries", s.handleDBSlowQueries)

		// P3-DB: Index recommendations
		r.Get("/index-recommendations", s.handleDBIndexRecommendations)
	})
}

// handleDBPoolStats возвращает текущую статистику пулов соединений.
//
// Response: PoolStats (JSON)
//   - primary_max_conns, primary_acquired, primary_idle
//   - replica_count, replica_acquired, replica_idle
//   - active_conns, idle_conns, wait_count, wait_duration_ms
//   - replica_fallbacks, replica_degradations
//   - latency_p50_ms, latency_p95_ms, latency_p99_ms
//
// Соответствует:
//   - IEC 62443-3-3 SR 7.2 (Performance Monitoring)
//   - ISO 27001 A.12.6.1 (Capacity Management)
func (s *Server) handleDBPoolStats(w http.ResponseWriter, r *http.Request) {
	if s.poolManager == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("pool manager not configured"))
		return
	}

	stats := s.poolManager.Monitor().Stats()
	if stats == nil {
		respond.RespondError(w, r, respond.NewInternalError("pool stats unavailable", nil))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"pool":       stats,
		"trace_id":   TraceIDFromContext(r.Context()),
		"prometheus": s.poolManager.Monitor().PrometheusMetrics(),
	})
}

// handleDBSlowQueries возвращает список медленных запросов.
//
// Response: SlowQuery[] (JSON)
//   - query_id, query (truncated), calls, mean_time_ms, max_time_ms
//   - rows, shared_reads, shared_writes
//   - query_plan (если доступен)
//
// Соответствует:
//   - ISO 27001 A.12.6.1 (Capacity Management)
func (s *Server) handleDBSlowQueries(w http.ResponseWriter, r *http.Request) {
	if s.slowQueryDetector == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("slow query detector not configured"))
		return
	}

	queries := s.slowQueryDetector.GetSlowQueries()
	if queries == nil {
		queries = []*db.SlowQuery{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"slow_queries": queries,
		"count":        len(queries),
		"trace_id":     TraceIDFromContext(r.Context()),
	})
}

// handleDBIndexRecommendations возвращает рекомендации по индексам.
//
// Response: IndexRecommendation[] (JSON)
//   - table, columns, index_type, issue_type
//   - create_index_sql, estimated_improvement, confidence
//   - status (pending/applied/rejected)
//
// Соответствует:
//   - IEC 62443-3-3 SR 7.2 (Performance Monitoring)
func (s *Server) handleDBIndexRecommendations(w http.ResponseWriter, r *http.Request) {
	if s.slowQueryDetector == nil {
		respond.RespondError(w, r, respond.NewNotFoundError("slow query detector not configured"))
		return
	}

	recommendations := s.slowQueryDetector.GetIndexRecommendations()
	if recommendations == nil {
		recommendations = []*db.IndexRecommendation{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"recommendations": recommendations,
		"count":           len(recommendations),
		"trace_id":        TraceIDFromContext(r.Context()),
	})
}
