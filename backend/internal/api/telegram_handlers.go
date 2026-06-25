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
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	// Generate random token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		s.logger.Error("failed to generate token", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Save token with 5-minute expiration
	expiresAt := time.Now().Add(5 * time.Minute)
	if err := s.db.SaveTelegramLinkToken(token, claims.UserID, expiresAt); err != nil {
		s.logger.Error("failed to save telegram link token", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
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
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	var req struct {
		Alerts bool `json:"alerts"`
		TFA    bool `json:"tfa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if err := s.db.UpdateTelegramSettings(claims.UserID, req.Alerts, req.TFA); err != nil {
		s.logger.Error("failed to update telegram settings", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleTelegramStatus returns current Telegram integration status
func (s *Server) handleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get user", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
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
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		respondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if user.TelegramChatID == "" {
		respondError(w, r, NewBadRequestError("telegram not linked"))
		return
	}

	if s.telegramBot == nil {
		respondError(w, r, NewExternalServiceError("telegram bot not configured"))
		return
	}

	// Generate and send code
	code, err := s.telegramBot.GenerateLoginCodeByUserID(user.ID)
	if err != nil {
		s.logger.Error("failed to generate telegram login code", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
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
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		respondError(w, r, NewNotFoundError("user not found"))
		return
	}

	valid, err := s.db.ValidateTelegramLoginCode(user.ID, req.Code)
	if err != nil {
		s.logger.Error("failed to validate telegram code", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	if !valid {
		respondError(w, r, NewUnauthorizedError("invalid or expired code"))
		return
	}

	token, refreshToken, err := s.issueTokenPair(r, user.ID, user.Username, user.Role, user.TenantID)
	if err != nil {
		s.logger.Error("failed to issue auth tokens", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	user.PasswordHash = ""
	user.TOTPSecret = ""
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
		"user":          user,
	})
}
