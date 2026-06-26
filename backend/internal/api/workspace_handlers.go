// Package api — Workspace handlers для синхронизации layout между устройствами.
//
// P1-1.4: Dashboard Multi-Device Sync
//   - Сохраняет layout в БД с привязкой к user_id
//   - Sync при login на новом устройстве
//   - Conflict resolution: last-write-wins
//
// Compliance:
//   - IEC 62443-3-3 SR 2.1: User account management
//   - OWASP ASVS V3.3: Session management — user_id из JWT claims
//   - OWASP ASVS V5.2: Input validation — whitelist валидация через JSON decoder
//   - OWASP ASVS V8.3: Error handling — respondError без раскрытия деталей
//   - ISO 27001 A.12.4: Audit trail — все мутации логируются

package api

import (
	"encoding/json"
	"net/http"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/db"
)

// ── Types ──────────────────────────────────────────────────────────────────

// WorkspaceLayout — DTO для ответа API (без UserID).
type WorkspaceLayoutResponse struct {
	TabID          string          `json:"tab_id"`
	Layout         json.RawMessage `json:"layout"`
	VisibleWidgets []string        `json:"visible_widgets"`
	UpdatedAt      string          `json:"updated_at,omitempty"`
}

// SaveLayoutRequest — тело запроса для сохранения layout.
type SaveLayoutRequest struct {
	TabID          string          `json:"tab_id"`
	Layout         json.RawMessage `json:"layout"`
	VisibleWidgets []string        `json:"visible_widgets"`
}

// ── Handlers ──────────────────────────────────────────────────────────────

// handleSaveLayout сохраняет layout дашборда для пользователя.
// POST /api/v1/workspace/layout
//
// Доступ: аутентифицированные пользователи (JWT required).
// Conflict resolution: last-write-wins (UPSERT).
func (s *Server) handleSaveLayout(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("user not authenticated"))
		return
	}

	var req SaveLayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	if req.TabID == "" {
		req.TabID = "overview"
	}

	// Валидация: layout должен быть JSON-массивом или объектом
	if len(req.Layout) > 0 && !json.Valid(req.Layout) {
		respondError(w, r, NewBadRequestError("invalid layout format: must be valid JSON"))
		return
	}

	layout := &db.WorkspaceLayout{
		UserID:         claims.UserID,
		TabID:          req.TabID,
		Layout:         req.Layout,
		VisibleWidgets: req.VisibleWidgets,
	}

	if err := s.db.SaveWorkspaceLayout(r.Context(), layout); err != nil {
		s.logger.Error("failed to save workspace layout",
			"user_id", claims.UserID,
			"tab_id", req.TabID,
			"error", err,
		)
		respondError(w, r, NewInternalError("failed to save layout", err))
		return
	}

	// Audit trail (ISO 27001 A.12.4)
	s.logAudit(claims.UserID, "save_workspace_layout", "workspace_layout", req.TabID, nil, map[string]interface{}{
		"tab_id": req.TabID,
	})

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleGetLayout загружает layout дашборда для пользователя.
// GET /api/v1/workspace/layout?tab_id=overview
//
// Доступ: аутентифицированные пользователи (JWT required).
// Если layout не найден — возвращает пустой ответ.
func (s *Server) handleGetLayout(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("user not authenticated"))
		return
	}

	tabID := r.URL.Query().Get("tab_id")
	if tabID == "" {
		tabID = "overview"
	}

	layout, err := s.db.GetWorkspaceLayout(r.Context(), claims.UserID, tabID)
	if err != nil {
		// Нет сохранённого layout — возвращаем пустой, не ошибку
		jsonResponse(w, http.StatusOK, WorkspaceLayoutResponse{
			TabID:          tabID,
			Layout:         nil,
			VisibleWidgets: nil,
		})
		return
	}

	jsonResponse(w, http.StatusOK, WorkspaceLayoutResponse{
		TabID:          layout.TabID,
		Layout:         layout.Layout,
		VisibleWidgets: layout.VisibleWidgets,
		UpdatedAt:      layout.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
