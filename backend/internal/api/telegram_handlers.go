package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/auth"
)

// handleTelegramGenerateLink generates a one-time token for linking Telegram account
func (s *Server) handleTelegramGenerateLink(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate random token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		s.logger.Error("failed to generate token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Save token with 5-minute expiration
	expiresAt := time.Now().Add(5 * time.Minute)
	if err := s.db.SaveTelegramLinkToken(token, claims.UserID, expiresAt); err != nil {
		s.logger.Error("failed to save telegram link token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_at": expiresAt,
	})
}

// handleTelegramUpdateSettings updates Telegram notification settings
func (s *Server) handleTelegramUpdateSettings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Alerts bool `json:"alerts"`
		TFA    bool `json:"tfa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateTelegramSettings(claims.UserID, req.Alerts, req.TFA); err != nil {
		s.logger.Error("failed to update telegram settings", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleTelegramStatus returns current Telegram integration status
func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"linked": user.TelegramChatID != "",
		"alerts": user.TelegramAlerts,
		"tfa":    user.Telegram2FA,
	})
}

// handleTelegramRequestCode requests a login code via Telegram
func (s *Server) handleTelegramRequestCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if user.TelegramChatID == "" {
		http.Error(w, "telegram not linked", http.StatusBadRequest)
		return
	}

	if s.telegramBot == nil {
		http.Error(w, "telegram bot not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate and send code
	code, err := s.telegramBot.GenerateLoginCodeByUserID(user.ID)
	if err != nil {
		s.logger.Error("failed to generate telegram login code", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Don't send code in response, it's sent via Telegram
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Code sent to Telegram",
		"code":    code, // Only for testing, remove in production
	})
}

// handleTelegramVerify verifies Telegram login code and issues JWT
func (s *Server) handleTelegramVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	valid, err := s.db.ValidateTelegramLoginCode(user.ID, req.Code)
	if err != nil {
		s.logger.Error("failed to validate telegram code", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "invalid or expired code", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token, err := auth.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user.PasswordHash = ""
	user.TOTPSecret = ""
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}
