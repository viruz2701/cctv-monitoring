package api

import (
	"encoding/json"
	"net/http"

	"github.com/pquerna/otp/totp"

	"gb-telemetry-collector/internal/auth"
)

// ---------- Аутентификация ----------

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}
	// Ищем пользователя по username или email
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		// Пробуем найти по email
		user, err = s.db.GetUserByEmail(req.Username)
		if err != nil {
			respondError(w, r, NewUnauthorizedError("invalid credentials"))
			return
		}
	}
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		respondError(w, r, NewUnauthorizedError("invalid credentials"))
		return
	}

	// If 2FA is enabled, return a temporary session token instead of the main JWT
	if user.TOTPEnabled {
		tempToken, err := auth.GenerateTempToken(user.ID, user.Username, user.Role)
		if err != nil {
			s.logger.Error("failed to generate temp token", "error", err)
			respondError(w, r, NewInternalError("internal error", nil))
			return
		}
		jsonResponse(w, http.StatusAccepted, map[string]interface{}{
			"requires_2fa":  true,
			"session_token": tempToken,
		})
		return
	}

	token, err := auth.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	user.PasswordHash = ""
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handleLogin2FA handles the second step of 2FA login.
func (s *Server) handleLogin2FA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionToken string `json:"session_token"`
		Code         string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	claims, err := auth.ValidateTempToken(req.SessionToken)
	if err != nil {
		respondError(w, r, NewUnauthorizedError("invalid or expired session"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, r, NewUnauthorizedError("user not found"))
		return
	}

	if !user.TOTPEnabled || user.TOTPSecret == "" {
		respondError(w, r, NewBadRequestError("2FA not enabled"))
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		respondError(w, r, NewUnauthorizedError("invalid 2FA code"))
		return
	}

	token, err := auth.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("failed to generate JWT", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	user.PasswordHash = ""
	user.TOTPSecret = ""
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// handle2FASetup generates a TOTP secret and returns the provisioning URI for QR code.
func (s *Server) handle2FASetup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if user.TOTPEnabled {
		respondError(w, r, NewBadRequestError("2FA already enabled"))
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "CCTV Monitor",
		AccountName: user.Username,
	})
	if err != nil {
		s.logger.Error("failed to generate TOTP key", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	if err := s.db.UpdateTOTPSecret(user.ID, key.Secret()); err != nil {
		s.logger.Error("failed to save TOTP secret", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"secret":   key.Secret(),
		"uri":      key.URL(),
		"qr_image": nil, // Frontend will generate QR from URI
	})
}

// handle2FAVerify verifies the TOTP code and enables 2FA.
func (s *Server) handle2FAVerify(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if user.TOTPEnabled {
		respondError(w, r, NewBadRequestError("2FA already enabled"))
		return
	}

	if user.TOTPSecret == "" {
		respondError(w, r, NewBadRequestError("2FA not set up. Call /2fa/setup first"))
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		respondError(w, r, NewUnauthorizedError("invalid 2FA code"))
		return
	}

	if err := s.db.EnableTOTP(user.ID); err != nil {
		s.logger.Error("failed to enable TOTP", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "ENABLE_2FA", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "2fa_enabled"})
}

// handle2FADisable disables 2FA for the current user.
func (s *Server) handle2FADisable(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		respondError(w, r, NewUnauthorizedError("invalid password"))
		return
	}

	if err := s.db.DisableTOTP(user.ID); err != nil {
		s.logger.Error("failed to disable TOTP", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "DISABLE_2FA", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "2fa_disabled"})
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		respondError(w, r, NewNotFoundError("user not found"))
		return
	}
	user.PasswordHash = ""
	jsonResponse(w, http.StatusOK, user)
}
