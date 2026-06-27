package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gb-telemetry-collector/internal/audit"
)

// ────────────────────────────────────────────────────────────────────────────
// Audit Routes
// ────────────────────────────────────────────────────────────────────────────

func (s *Server) mountAuditRoutes() {
	// Mounted from server.go under auth middleware
}

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
			COALESCE(hmac_signature, ''), COALESCE(prev_hash, '')
		FROM audit_log
		ORDER BY id ASC
	`)
	if err != nil {
		s.logger.Error("audit verify: query failed", "error", err)
		RespondError(w, r, NewInternalError("query error", nil))
		return
	}
	defer rows.Close()

	var total int
	corrupted := 0
	corruptedIDs := []int64{}

	var prevHMAC string

	for rows.Next() {
		var id int64
		var userID, action, entityType, entityID, oldStr, newStr, hmacSig, prevHash string
		if err := rows.Scan(&id, &userID, &action, &entityType, &entityID, &oldStr, &newStr, &hmacSig, &prevHash); err != nil {
			continue
		}

		// Check 1: HMAC of current record
		data := audit.SignAuditEntry(userID, action, entityType, entityID, []byte(oldStr), []byte(newStr))
		if prevHash != "" {
			data = prevHash + "|" + data
		}

		if !s.auditSigner.Verify(data, hmacSig) {
			corrupted++
			corruptedIDs = append(corruptedIDs, id)
		}

		// Check 2: Chain integrity (prev_hash matches previous record's HMAC)
		if id > 1 && prevHash != prevHMAC {
			// Only flag if not already corrupted
			if corrupted == 0 || corruptedIDs[len(corruptedIDs)-1] != id {
				corrupted++
				corruptedIDs = append(corruptedIDs, id)
			}
		}

		prevHMAC = hmacSig
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
		"status":         status,
		"total_checked":  total,
		"corrupted":      corrupted,
		"corrupted_ids":  corruptedIDs,
		"chain_verified": true,
	})
}

// handleListAuditLog возвращает записи журнала аудита с пагинацией.
// GET /api/v1/audit/log?limit=N&offset=N&action=X&entity_type=Y
func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	action := r.URL.Query().Get("action")
	entityType := r.URL.Query().Get("entity_type")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	query := `
		SELECT id, timestamp, user_id, action, entity_type, entity_id,
		       old_value, new_value, ip_address, user_agent,
		       COALESCE(hmac_signature, ''), COALESCE(prev_hash, ''), COALESCE(trace_id, '')
		FROM audit_log
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	if entityType != "" {
		query += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, entityType)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error("audit list: query failed", "error", err)
		RespondError(w, r, NewInternalError("query error", nil))
		return
	}
	defer rows.Close()

	type auditRecord struct {
		ID            int64           `json:"id"`
		Timestamp     time.Time       `json:"timestamp"`
		UserID        string          `json:"user_id"`
		Action        string          `json:"action"`
		EntityType    string          `json:"entity_type"`
		EntityID      string          `json:"entity_id"`
		OldValue      json.RawMessage `json:"old_value,omitempty"`
		NewValue      json.RawMessage `json:"new_value,omitempty"`
		IPAddress     string          `json:"ip_address,omitempty"`
		UserAgent     string          `json:"user_agent,omitempty"`
		HMACSignature string          `json:"hmac_signature,omitempty"`
		PrevHash      string          `json:"prev_hash,omitempty"`
		TraceID       string          `json:"trace_id,omitempty"`
	}

	var records []auditRecord
	for rows.Next() {
		var rec auditRecord
		var oldVal, newVal []byte
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.UserID, &rec.Action,
			&rec.EntityType, &rec.EntityID, &oldVal, &newVal,
			&rec.IPAddress, &rec.UserAgent, &rec.HMACSignature,
			&rec.PrevHash, &rec.TraceID); err != nil {
			continue
		}
		if oldVal != nil {
			rec.OldValue = json.RawMessage(oldVal)
		}
		if newVal != nil {
			rec.NewValue = json.RawMessage(newVal)
		}
		records = append(records, rec)
	}

	if records == nil {
		records = []auditRecord{}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"records": records,
		"limit":   limit,
		"offset":  offset,
	})
}

// handleAuditCompliance возвращает compliance-отчёт по audit trail.
// GET /api/v1/audit/compliance
func (s *Server) handleAuditCompliance(w http.ResponseWriter, r *http.Request) {
	if s.auditChainStore == nil {
		RespondError(w, r, fmt.Errorf("audit chain store not initialized"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	report, err := s.auditChainStore.GetComplianceReport(ctx)
	if err != nil {
		s.logger.Error("audit compliance: get report failed", "error", err)
		RespondError(w, r, NewInternalError("failed to get compliance report", nil))
		return
	}

	jsonResponse(w, http.StatusOK, report)
}

// handleAuditArchive запускает архивацию записей старше N лет.
// POST /api/v1/audit/archive?retention_years=7
func (s *Server) handleAuditArchive(w http.ResponseWriter, r *http.Request) {
	retentionYears, _ := strconv.Atoi(r.URL.Query().Get("retention_years"))
	if retentionYears <= 0 {
		retentionYears = 7
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	var archivedCount int64
	err := s.db.Pool.QueryRow(ctx, `SELECT archive_audit_logs($1)`, retentionYears).Scan(&archivedCount)
	if err != nil {
		s.logger.Error("audit archive: failed", "error", err)
		RespondError(w, r, NewInternalError("archive failed", nil))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":          "ok",
		"archived_count":  archivedCount,
		"retention_years": retentionYears,
	})
}
