package api

import (
	"context"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/audit"
)

// handleAuditVerify проверяет целостность журнала аудита через HMAC-верификацию (ISO 27001 A.12.4).
func (s *Server) handleAuditVerify(w http.ResponseWriter, r *http.Request) {
	if s.auditSigner == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"status":  "skipped",
			"message": "HMAC signing not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, action, entity_type, entity_id,
			COALESCE(old_value::text, 'null'), COALESCE(new_value::text, 'null'),
			COALESCE(hmac_signature, '')
		FROM audit_log
		ORDER BY id DESC
		LIMIT 1000
	`)
	if err != nil {
		s.logger.Error("audit verify: query failed", "error", err)
		respondError(w, r, NewInternalError("query error", nil))
		return
	}
	defer rows.Close()

	var total int
	corrupted := 0
	corruptedIDs := []int64{}

	for rows.Next() {
		var id int64
		var userID, action, entityType, entityID, oldStr, newStr, hmacSig string
		if err := rows.Scan(&id, &userID, &action, &entityType, &entityID, &oldStr, &newStr, &hmacSig); err != nil {
			continue
		}

		data := audit.SignAuditEntry(userID, action, entityType, entityID, []byte(oldStr), []byte(newStr))
		if !s.auditSigner.Verify(data, hmacSig) {
			corrupted++
			corruptedIDs = append(corruptedIDs, id)
		}
		total++
	}

	status := "ok"
	if corrupted > 0 {
		status = "corrupted"
		s.logger.Warn("audit verify: integrity check failed",
			"total", total, "corrupted", corrupted,
			"corrupted_ids", corruptedIDs,
		)
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":        status,
		"total_checked": total,
		"corrupted":     corrupted,
		"corrupted_ids": corruptedIDs,
	})
}

// handleListAuditLog возвращает записи журнала аудита.
// GET /api/v1/audit/log?limit=N
// TODO: реализовать полноценный запрос к БД.
func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, []interface{}{})
}
