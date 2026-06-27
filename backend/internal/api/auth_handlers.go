package api

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/pquerna/otp/totp"

	"gb-telemetry-collector/internal/auth"
)

// P1-SEC.1: Определяет, является ли клиент мобильным (токены в body) или веб (только cookies).
func isMobileClient(r *http.Request) bool {
	return r.Header.Get("X-Client-Type") == "mobile"
}

// ---------- Аутентификация ----------

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request"))
		return
	}
	// Ищем пользователя по username или email
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		// Пробуем найти по email
		user, err = s.db.GetUserByEmail(req.Username)
		if err != nil {
			RespondError(w, r, NewUnauthorizedError("invalid credentials"))
			return
		}
	}
	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		RespondError(w, r, NewUnauthorizedError("invalid credentials"))
		return
	}

	// If 2FA is enabled, return a temporary session token instead of the main JWT
	if user.TOTPEnabled {
		tempToken, err := auth.GenerateTempToken(user.ID, user.Username, user.Role, user.TenantID)
		if err != nil {
			s.logger.Error("failed to generate temp token", "error", err)
			RespondError(w, r, NewInternalError("internal error", nil))
			return
		}
		jsonResponse(w, http.StatusAccepted, map[string]interface{}{
			"requires_2fa":  true,
			"session_token": tempToken,
		})
		return
	}

	token, refreshToken, err := s.issueTokenPair(r, user.ID, user.Username, user.Role, user.TenantID)
	if err != nil {
		s.logger.Error("failed to issue auth tokens", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// P1-SEC.1: HttpOnly cookies (Secure, SameSite=Strict) — для всех клиентов
	auth.SetAuthCookies(w, token, refreshToken, nil)

	user.PasswordHash = ""

	// P1-SEC.1: Для веб-клиентов не возвращаем токены в body (только HttpOnly cookies)
	// Для мобильных клиентов возвращаем токены для secure storage (AsyncStorage/Keychain)
	if isMobileClient(r) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"token":         token,
			"refresh_token": refreshToken,
			"user":          user,
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// handleLogin2FA handles the second step of 2FA login.
func (s *Server) handleLogin2FA(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionToken string `json:"session_token"`
		Code         string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	claims, err := auth.ValidateTempToken(req.SessionToken)
	if err != nil {
		RespondError(w, r, NewUnauthorizedError("invalid or expired session"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		RespondError(w, r, NewUnauthorizedError("user not found"))
		return
	}

	if !user.TOTPEnabled || user.TOTPSecret == "" {
		RespondError(w, r, NewBadRequestError("2FA not enabled"))
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		RespondError(w, r, NewUnauthorizedError("invalid 2FA code"))
		return
	}

	token, refreshToken, err := s.issueTokenPair(r, user.ID, user.Username, user.Role, user.TenantID)
	if err != nil {
		s.logger.Error("failed to issue auth tokens", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// P1-SEC.1: HttpOnly cookies для всех клиентов
	auth.SetAuthCookies(w, token, refreshToken, nil)

	user.PasswordHash = ""
	user.TOTPSecret = ""

	// P1-SEC.1: Для мобильных клиентов возвращаем токены в body
	if isMobileClient(r) {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"token":         token,
			"refresh_token": refreshToken,
			"user":          user,
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// handle2FASetup generates a TOTP secret and returns the provisioning URI for QR code.
func (s *Server) handle2FASetup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if user.TOTPEnabled {
		RespondError(w, r, NewBadRequestError("2FA already enabled"))
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "CCTV Monitor",
		AccountName: user.Username,
	})
	if err != nil {
		s.logger.Error("failed to generate TOTP key", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	if err := s.db.UpdateTOTPSecret(user.ID, key.Secret()); err != nil {
		s.logger.Error("failed to save TOTP secret", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
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
		RespondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if user.TOTPEnabled {
		RespondError(w, r, NewBadRequestError("2FA already enabled"))
		return
	}

	if user.TOTPSecret == "" {
		RespondError(w, r, NewBadRequestError("2FA not set up. Call /2fa/setup first"))
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		RespondError(w, r, NewUnauthorizedError("invalid 2FA code"))
		return
	}

	if err := s.db.EnableTOTP(user.ID); err != nil {
		s.logger.Error("failed to enable TOTP", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "ENABLE_2FA", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "2fa_enabled"})
}

// handle2FADisable disables 2FA for the current user.
func (s *Server) handle2FADisable(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("user not found"))
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		RespondError(w, r, NewUnauthorizedError("invalid password"))
		return
	}

	if err := s.db.DisableTOTP(user.ID); err != nil {
		s.logger.Error("failed to disable TOTP", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "DISABLE_2FA", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "2fa_disabled"})
}

func (s *Server) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}
	user, err := s.db.GetUserByID(claims.UserID)
	if err != nil {
		RespondError(w, r, NewNotFoundError("user not found"))
		return
	}
	user.PasswordHash = ""
	jsonResponse(w, http.StatusOK, user)
}

// handleRefreshToken обновляет токены.
// P1-SEC.1: Читает refresh_token из HttpOnly cookie (веб) или из JSON body (mobile).
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// P1-SEC.1: Пробуем получить refresh_token из cookie
	refreshTokenValue := auth.GetRefreshTokenFromCookie(r)
	source := "cookie"

	// Если нет в cookie — пробуем из JSON body (mobile/API clients)
	if refreshTokenValue == "" {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			RespondError(w, r, NewBadRequestError("invalid request: missing refresh token"))
			return
		}
		refreshTokenValue = req.RefreshToken
		source = "body"
	}

	user, sessionID, err := s.db.GetUserByRefreshTokenHash(auth.HashRefreshToken(refreshTokenValue))
	if err != nil {
		RespondError(w, r, NewUnauthorizedError("invalid or expired refresh token"))
		return
	}

	if err := s.db.RevokeSession(sessionID); err != nil {
		s.logger.Error("failed to rotate refresh session", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	newToken, newRefreshToken, err := s.issueTokenPair(r, user.ID, user.Username, user.Role, user.TenantID)
	if err != nil {
		s.logger.Error("failed to issue refreshed tokens", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// P1-SEC.1: Устанавливаем новые HttpOnly cookies (для веб-клиентов)
	auth.SetAuthCookies(w, newToken, newRefreshToken, nil)

	user.PasswordHash = ""
	user.TOTPSecret = ""

	// P1-SEC.1: Для клиентов, приславших refresh_token в body — возвращаем токены в ответе
	if source == "body" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"token":         newToken,
			"refresh_token": newRefreshToken,
			"user":          user,
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// handleLogout очищает auth cookies и завершает сессию.
// P1-SEC.1: Вызов очищает HttpOnly cookies на клиенте.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Пытаемся получить claims для audit log
	claims := auth.GetClaims(r)
	if claims != nil {
		_ = s.db.SaveAudit(claims.UserID, "LOGOUT", "session", claims.UserID, nil, nil)
	}

	// P1-SEC.1: Очищаем все auth cookies
	auth.ClearAuthCookies(w, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (s *Server) issueTokenPair(r *http.Request, userID, username, role, tenantID string) (string, string, error) {
	token, err := auth.GenerateJWT(userID, username, role, tenantID)
	if err != nil {
		return "", "", err
	}

	refreshToken, tokenHash, expiresAt, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	if _, err := s.db.CreateUserSession(userID, tokenHash, clientIP(r), r.UserAgent(), expiresAt); err != nil {
		return "", "", err
	}

	return token, refreshToken, nil
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if host, _, err := net.SplitHostPort(forwardedFor); err == nil {
			return host
		}
		return forwardedFor
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
