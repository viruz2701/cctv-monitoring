package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

// ---------- Users Management ----------

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}
	users, err := s.db.GetUsers()
	if err != nil {
		s.logger.Error("failed to get users", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	jsonResponse(w, http.StatusOK, users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	// Валидация роли
	validRoles := map[string]bool{"admin": true, "manager": true, "technician": true, "viewer": true, "support": true, "owner": true}
	if !validRoles[req.Role] {
		respondError(w, r, NewBadRequestError("invalid role"))
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	user, err := s.db.CreateUser(req.Username, hashed, req.Role, req.Email, nil)
	if err != nil {
		s.logger.Error("failed to create user", "error", err)
		respondError(w, r, NewConflictError("user already exists or db error"))
		return
	}

	// Аудит
	_ = s.db.SaveAudit(claims.UserID, "CREATE_USER", "user", user.ID, nil, map[string]string{"username": req.Username, "role": req.Role})

	user.PasswordHash = ""
	jsonResponse(w, http.StatusCreated, user)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}
	id := chi.URLParam(r, "id")
	var req struct {
		Role   string `json:"role"`
		Status string `json:"status"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	updates := make(map[string]interface{})
	if req.Role != "" {
		validRoles := map[string]bool{"admin": true, "manager": true, "technician": true, "viewer": true, "support": true, "owner": true}
		if !validRoles[req.Role] {
			respondError(w, r, NewBadRequestError("invalid role"))
			return
		}
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if err := s.db.UpdateUser(id, updates); err != nil {
		respondError(w, r, NewInternalError("failed to update user", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "UPDATE_USER", "user", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}
	id := chi.URLParam(r, "id")

	// Защита от удаления самого себя
	if id == claims.UserID {
		respondError(w, r, NewBadRequestError("cannot delete yourself"))
		return
	}

	if err := s.db.DeleteUser(id); err != nil {
		respondError(w, r, NewInternalError("failed to delete user", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "DELETE_USER", "user", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---------- Settings (Services) ----------

func (s *Server) getServicesSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "manager" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	settings, err := s.db.GetSystemSettings()
	if err != nil {
		s.logger.Error("failed to get services settings", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	jsonResponse(w, http.StatusOK, settings)
}

func (s *Server) updateServicesSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}
	if err := s.db.UpdateMultipleSettings(req, claims.UserID); err != nil {
		s.logger.Error("failed to update services settings", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// Если в запросе есть настройки P2P gateway с Xiongmai — пушим в P2P gateway
	if p2pSettings, ok := req["services_p2p_gateway"]; ok {
		if p2pMap, ok := p2pSettings.(map[string]interface{}); ok {
			s.syncP2PJftechConfig(p2pMap)
		}
	}

	_ = s.db.SaveAudit(claims.UserID, "UPDATE_SERVICES_SETTINGS", "system_settings", "services", nil, req)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// syncP2PJftechConfig отправляет Xiongmai (Jftech) конфигурацию в P2P gateway.
func (s *Server) syncP2PJftechConfig(p2pMap map[string]interface{}) {
	xiongmai, ok := p2pMap["xiongmai"]
	if !ok {
		return
	}
	xmMap, ok := xiongmai.(map[string]interface{})
	if !ok {
		return
	}

	uuid, _ := xmMap["uuid"].(string)
	appKey, _ := xmMap["app_key"].(string)
	appSecret, _ := xmMap["app_secret"].(string)
	endpoint, _ := xmMap["endpoint"].(string)
	region, _ := xmMap["region"].(string)
	moveCard, _ := xmMap["move_card"].(float64)

	if uuid == "" || appKey == "" || appSecret == "" {
		s.logger.Debug("Jftech config incomplete, skipping P2P gateway push")
		return
	}
	if endpoint == "" {
		endpoint = "api-cn.jftechws.com"
	}
	if region == "" {
		region = "Local"
	}

	p2pURL := s.p2pGatewayURL
	if p2pURL == "" {
		p2pURL = s.config.P2PGatewayURL
	}
	if p2pURL == "" {
		s.logger.Debug("P2P gateway URL not configured, skipping Jftech push")
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"uuid":       uuid,
		"app_key":    appKey,
		"app_secret": appSecret,
		"endpoint":   endpoint,
		"region":     region,
		"move_card":  int(moveCard),
	})

	resp, err := s.httpClient.Post(p2pURL+"/p2p/config/jftech", "application/json", bytes.NewReader(body))
	if err != nil {
		s.logger.Error("Failed to push Jftech config to P2P gateway", "error", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logger.Error("P2P gateway returned non-OK status for Jftech config", "status", resp.StatusCode)
		return
	}
	s.logger.Info("Jftech config pushed to P2P gateway successfully")
}
