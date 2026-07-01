package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"gb-telemetry-collector/internal/auth"
)

// cookieConfigForRequest возвращает конфиг cookie, адаптированный под протокол.
// На HTTP (dev) — Secure=false, чтобы браузер принимал cookies.
// На HTTPS (production) — Secure=true (OWASP ASVS V3.1).
func cookieConfigForRequest(r *http.Request) *auth.CookieConfig {
	if r.TLS != nil {
		return nil // production: DefaultCookieConfig (Secure=true)
	}
	return &auth.CookieConfig{
		Secure:          false,
		Path:            "/",
		SameSite:        http.SameSiteStrictMode,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
}

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

	// OWASP ASVS V5.1: Input validation (whitelist) before DB query
	if err := validateLoginRequest(req.Username, req.Password); err != nil {
		RespondError(w, r, NewValidationError(err.Error()))
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

	// P1-SEC.1: HttpOnly cookies (Secure=auto, SameSite=Strict) — для всех клиентов
	auth.SetAuthCookies(w, token, refreshToken, cookieConfigForRequest(r))

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

	// P1-SEC.1: HttpOnly cookies (Secure=auto, SameSite=Strict) для всех клиентов
	auth.SetAuthCookies(w, token, refreshToken, cookieConfigForRequest(r))

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

// handleRefreshToken обновляет токены с rotation + device fingerprinting + reuse detection.
//
// P1-HI-05: Refresh Token Rotation
//  1. Проверяет fingerprint устройства (User-Agent + IP хеш)
//  2. Выполняет rotation: инвалидирует старый токен, выдаёт новый
//  3. При reuse украденного токена — инвалидирует всю семью
//
// P1-SEC.1: Читает refresh_token из HttpOnly cookie (веб) или из JSON body (mobile).
//
// Compliance: OWASP ASVS V3.2.2, V3.2.3, V3.2.4
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// 1. Получаем refresh token из запроса
	refreshTokenValue, source, err := auth.ValidateRefreshRequest(r)
	if err != nil {
		RespondError(w, r, NewBadRequestError("invalid request: missing refresh token"))
		return
	}

	// 2. Вычисляем fingerprint устройства
	fingerprint := auth.ComputeFingerprint(r.UserAgent(), auth.ClientIP(r))
	tokenHash := auth.HashRefreshToken(refreshTokenValue)

	// 3. Выполняем rotation с проверками
	result, err := auth.RotateRefreshToken(s.db, tokenHash, fingerprint, "", auth.ClientIP(r), r.UserAgent())
	if err != nil {
		// P1-HI-05: Reuse detection — украденный токен
		if result != nil && result.ReuseDetected {
			s.logger.Error("refresh token reuse detected — token family revoked",
				"fingerprint", fingerprint,
				"ip", auth.ClientIP(r),
				"user_agent", r.UserAgent(),
				"trace_id", r.Header.Get("X-Request-ID"),
			)
			_ = s.db.SaveAudit("", "TOKEN_REUSE_DETECTED", "session", "", nil, nil)
			RespondError(w, r, NewUnauthorizedError("refresh token revoked (reuse detected)"))
			return
		}

		// Fingerprint mismatch — возможно, токен украден
		if err == auth.ErrFingerprintMismatch {
			s.logger.Warn("device fingerprint mismatch on refresh",
				"fingerprint", fingerprint,
				"ip", auth.ClientIP(r),
				"user_agent", r.UserAgent(),
				"trace_id", r.Header.Get("X-Request-ID"),
			)
			RespondError(w, r, NewUnauthorizedError("device fingerprint mismatch"))
			return
		}

		// Другие ошибки (истёк, не найден)
		s.logger.Error("refresh token rotation failed", "error", err)
		RespondError(w, r, NewUnauthorizedError("invalid or expired refresh token"))
		return
	}

	// 4. Получаем пользователя из сессии
	session, err := s.db.GetSessionByTokenHash(result.NewTokenHash)
	if err != nil {
		s.logger.Error("failed to get new session", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	user, err := s.db.GetUserByID(session.UserID)
	if err != nil {
		s.logger.Error("failed to get user for new session", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// 5. Генерируем новый access token
	newToken, err := auth.GenerateJWT(user.ID, user.Username, user.Role, user.TenantID)
	if err != nil {
		s.logger.Error("failed to generate new access token", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// 6. Устанавливаем новые HttpOnly cookies (Secure=auto, SameSite=Strict)
	auth.SetAuthCookies(w, newToken, result.NewToken, cookieConfigForRequest(r))

	user.PasswordHash = ""
	user.TOTPSecret = ""

	// 7. Audit log для успешной ротации
	_ = s.db.SaveAudit(session.UserID, "TOKEN_ROTATED", "session", result.NewSessionID, nil, nil)

	// P1-SEC.1: Для клиентов, приславших refresh_token в body — возвращаем токены в ответе
	if source == "body" {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"token":         newToken,
			"refresh_token": result.NewToken,
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

// issueTokenPair создаёт новую пару access + refresh token для пользователя.
// Используется при login и регистрации (initial token pair).
//
// P1-HI-05: Создаёт новую семью токенов для отслеживания rotation.
func (s *Server) issueTokenPair(r *http.Request, userID, username, role, tenantID string) (string, string, error) {
	token, err := auth.GenerateJWT(userID, username, role, tenantID)
	if err != nil {
		return "", "", err
	}

	refreshToken, tokenHash, expiresAt, err := auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// P1-HI-05: Вычисляем fingerprint и создаём сессию с новой семьёй токенов
	fingerprint := auth.ComputeFingerprint(r.UserAgent(), auth.ClientIP(r))
	tokenFamily := uuid.New()

	if _, err := s.db.CreateSession(
		userID, tokenHash, auth.ClientIP(r), r.UserAgent(),
		fingerprint, &tokenFamily, expiresAt,
	); err != nil {
		return "", "", err
	}

	return token, refreshToken, nil
}
