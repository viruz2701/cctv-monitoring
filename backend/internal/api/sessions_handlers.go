package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

// ---------- Password Management ----------

// changeMyPassword — пользователь меняет свой пароль (с проверкой текущего)
func (s *Server) changeMyPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	// Валидация нового пароля
	if len(req.NewPassword) < 6 {
		respondError(w, r, NewBadRequestError("new password must be at least 6 characters"))
		return
	}

	// Проверяем текущий пароль
	currentHash, err := s.db.GetPasswordHash(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get password hash", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	if !auth.CheckPasswordHash(req.CurrentPassword, currentHash) {
		respondError(w, r, NewUnauthorizedError("current password is incorrect"))
		return
	}

	// Хешируем новый пароль
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash new password", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// Обновляем в БД
	if err := s.db.UpdatePassword(claims.UserID, newHash); err != nil {
		s.logger.Error("failed to update password", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "CHANGE_PASSWORD", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "password_changed"})
}

// resetUserPassword — админ сбрасывает пароль пользователю (без проверки текущего)
func (s *Server) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" {
		respondError(w, r, NewForbiddenError("forbidden: admin only"))
		return
	}

	targetUserID := chi.URLParam(r, "id")
	if targetUserID == "" {
		respondError(w, r, NewBadRequestError("user id required"))
		return
	}

	// Защита от сброса пароля самому себе через этот эндпоинт
	if targetUserID == claims.UserID {
		respondError(w, r, NewBadRequestError("use /users/me/password to change your own password"))
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if len(req.NewPassword) < 6 {
		respondError(w, r, NewBadRequestError("new password must be at least 6 characters"))
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	if err := s.db.UpdatePassword(targetUserID, newHash); err != nil {
		s.logger.Error("failed to reset password", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "RESET_PASSWORD", "user", targetUserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "password_reset"})
}

// getUserSessions returns all active sessions for the current user.
func (s *Server) getUserSessions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	sessions, err := s.db.GetUserSessions(claims.UserID)
	if err != nil {
		s.logger.Error("failed to get user sessions", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	jsonResponse(w, http.StatusOK, sessions)
}

// revokeSession revokes a specific session.
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		respondError(w, r, NewBadRequestError("session id required"))
		return
	}

	if err := s.db.RevokeSession(sessionID); err != nil {
		s.logger.Error("failed to revoke session", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "REVOKE_SESSION", "session", sessionID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// revokeAllOtherSessions revokes all sessions for the current user except the current one.
func (s *Server) revokeAllOtherSessions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	var req struct {
		CurrentSessionID string `json:"current_session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if err := s.db.RevokeAllOtherSessions(claims.UserID, req.CurrentSessionID); err != nil {
		s.logger.Error("failed to revoke all other sessions", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.SaveAudit(claims.UserID, "REVOKE_ALL_SESSIONS", "user", claims.UserID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "revoked_all"})
}

// ---------- Password Reset (Forgot Password) ----------

// handleForgotPassword generates a reset token and returns it (in production, send via email).
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if req.Email == "" {
		respondError(w, r, NewBadRequestError("email required"))
		return
	}

	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil {
		// Не раскрываем, существует ли пользователь — всегда возвращаем успех
		jsonResponse(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "If the email exists, a reset link has been sent",
		})
		return
	}

	// Генерируем токен
	token := auth.GenerateResetToken()
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.db.CreatePasswordResetToken(user.ID, token, expiresAt); err != nil {
		s.logger.Error("failed to create reset token", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// В production здесь нужно отправить email с ссылкой.
	// Для простоты возвращаем токен в ответе (только для dev/test).
	s.logger.Info("Password reset token generated", "user_id", user.ID, "email", req.Email)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"message":     "Reset token generated (check logs for production)",
		"reset_token": token, // В production убрать! Отправлять только по email.
		"expires_at":  expiresAt,
	})
}

// handleResetPasswordWithToken resets password using a valid reset token.
func (s *Server) handleResetPasswordWithToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request"))
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		respondError(w, r, NewBadRequestError("token and new_password required"))
		return
	}

	if len(req.NewPassword) < 6 {
		respondError(w, r, NewBadRequestError("new password must be at least 6 characters"))
		return
	}

	userID, expiresAt, err := s.db.GetPasswordResetToken(req.Token)
	if err != nil {
		respondError(w, r, NewBadRequestError("invalid or expired token"))
		return
	}

	if time.Now().After(expiresAt) {
		_ = s.db.DeletePasswordResetToken(req.Token)
		respondError(w, r, NewBadRequestError("token expired"))
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	if err := s.db.UpdatePassword(userID, newHash); err != nil {
		s.logger.Error("failed to update password", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}

	_ = s.db.DeletePasswordResetToken(req.Token)
	_ = s.db.SaveAudit(userID, "RESET_PASSWORD_WITH_TOKEN", "user", userID, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "password_reset"})
}
