// Package api — Auth domain routes: login, 2FA, Telegram, password reset, sessions, current user.
package api

import (
	"gb-telemetry-collector/internal/auth"
	"time"

	"github.com/go-chi/chi/v5"
)

// mountAuthRoutes регистрирует публичные и защищённые auth-маршруты.
func (s *Server) mountAuthRoutes(r chi.Router) {
	// Публичные (rate-limited: 5 req/min)
	r.With(s.rateLimitMiddleware).Post("/api/v1/auth/login", s.handleLogin)
	r.With(s.rateLimitMiddleware).Post("/api/v1/auth/refresh", s.handleRefreshToken)

	// 2FA login (rate-limited: 10 req/min для предотвращения brute force TOTP)
	// OWASP ASVS V2.2.1: защита от brute force аутентификации
	r.With(s.newRateLimiterMiddleware(10, time.Minute)).Post("/api/v1/auth/login/2fa", s.handleLogin2FA)

	// Telegram login (rate-limited: 5 req/min)
	r.With(s.rateLimitMiddleware).Post("/api/v1/auth/telegram/request-code", s.handleTelegramRequestCode)
	r.With(s.rateLimitMiddleware).Post("/api/v1/auth/telegram/verify", s.handleTelegramVerify)

	// Password reset
	r.Post("/api/v1/auth/forgot-password", s.handleForgotPassword)
	r.Post("/api/v1/auth/reset-password", s.handleResetPasswordWithToken)

	// P1-SEC.1: Logout — доступен с cookie или Authorization header
	// Используется CookieAuthMiddleware для чтения JWT из cookie
	r.With(auth.CookieAuthMiddleware, auth.AuthMiddleware).Post("/api/v1/auth/logout", s.handleLogout)
}

// mountProtectedAuthRoutes регистрирует auth-маршруты требующие JWT.
func (s *Server) mountProtectedAuthRoutes(r chi.Router) {
	// Current user
	r.Get("/api/v1/users/me", s.handleCurrentUser)

	// Password management
	r.Put("/api/v1/users/me/password", s.changeMyPassword)
	r.Put("/api/v1/users/{id}/reset-password", s.resetUserPassword)

	// 2FA management
	r.Post("/api/v1/users/me/2fa/setup", s.handle2FASetup)
	r.Post("/api/v1/users/me/2fa/verify", s.handle2FAVerify)
	r.Post("/api/v1/users/me/2fa/disable", s.handle2FADisable)

	// Telegram
	r.Post("/api/v1/users/me/telegram/generate-link", s.handleTelegramGenerateLink)
	r.Post("/api/v1/users/me/telegram/settings", s.handleTelegramUpdateSettings)
	r.Get("/api/v1/users/me/telegram/status", s.handleTelegramStatus)

	// Sessions
	r.Get("/api/v1/sessions", s.getUserSessions)
	r.Delete("/api/v1/sessions/{id}", s.revokeSession)
	r.Post("/api/v1/sessions/revoke-all", s.revokeAllOtherSessions)

	// User Management (Admin only — logic enforced in handlers)
	r.Get("/api/v1/users", s.listUsers)
	r.Post("/api/v1/users", s.createUser)
	r.Put("/api/v1/users/{id}", s.updateUser)
	r.Delete("/api/v1/users/{id}", s.deleteUser)

	// API Key Management (Admin only)
	r.Get("/api/v1/api-keys", s.handleListAPIKeys)
	r.Post("/api/v1/api-keys", s.handleCreateAPIKey)
	r.Delete("/api/v1/api-keys/{id}", s.handleRevokeAPIKey)

	// Settings
	r.Get("/api/v1/settings/services", s.getServicesSettings)
	r.Put("/api/v1/settings/services", s.updateServicesSettings)

	// Service status (health check on each protocol port)
	s.mountServicesStatusRoute(r)
}
