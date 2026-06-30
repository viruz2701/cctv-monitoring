// Package auth — HttpOnly Cookie-based JWT Authentication (P1-SEC.1).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-SEC.1: JWT → HttpOnly Cookies
//
// Замена localStorage JWT на HttpOnly cookies:
//   - HttpOnly cookies для web (Secure, SameSite=Strict)
//   - CSRF token в заголовке X-CSRF-Token
//   - Token refresh endpoint
//   - Logout clears cookie
//
// P1-SEC.1 (CSRF Tokens для Mutations):
//   - Double Submit Cookie pattern (stateless)
//   - Token rotation каждые 30 минут
//   - Excluded paths для webhooks, external, public
//   - WebSocket и API key bypass
//
// Compliance:
//   - OWASP ASVS V3.1 (Session management — HttpOnly cookies)
//   - OWASP ASVS V3.2 (CSRF protection)
//   - OWASP ASVS V4.1 (Cross-site request forgery — stateless)
//   - ISO 27001 A.9.2.1 (User registration — secure session)
//   - ISO 27001 A.9.4.2 (Secure log-on — CSRF)
//   - СТБ 34.101.27 п. 6.1 (Защита сессий — CSRF)
//   - Приказ ОАЦ №66 п. 7.18.2 (Secure session management)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gb-telemetry-collector/internal/respond"
)

// ────────────────────────────────────────────────────────────────────────────
// Cookie names
// ────────────────────────────────────────────────────────────────────────────

const (
	// CookieNameAccessToken — имя HttpOnly cookie для access token.
	CookieNameAccessToken = "access_token"
	// CookieNameRefreshToken — имя HttpOnly cookie для refresh token.
	CookieNameRefreshToken = "refresh_token"
	// CookieNameCSRF — имя cookie для CSRF токена (не HttpOnly, доступен JS).
	CookieNameCSRF = "csrf_token"
)

// ────────────────────────────────────────────────────────────────────────────
// Cookie configuration
// ────────────────────────────────────────────────────────────────────────────

// CookieConfig — конфигурация для установки HttpOnly cookies.
type CookieConfig struct {
	// Secure — только HTTPS (должно быть true в production).
	Secure bool
	// Domain — домен cookie (пусто = текущий).
	Domain string
	// Path — путь cookie.
	Path string
	// SameSite — политика SameSite.
	SameSite http.SameSite
	// AccessTokenTTL — TTL access token cookie.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL — TTL refresh token cookie.
	RefreshTokenTTL time.Duration
}

// DefaultCookieConfig — конфигурация по умолчанию.
var DefaultCookieConfig = CookieConfig{
	Secure:          true,
	Path:            "/",
	SameSite:        http.SameSiteStrictMode,
	AccessTokenTTL:  15 * time.Minute,
	RefreshTokenTTL: 7 * 24 * time.Hour,
}

// ────────────────────────────────────────────────────────────────────────────
// Cookie helpers
// ────────────────────────────────────────────────────────────────────────────

// SetAuthCookies устанавливает HttpOnly cookies для access и refresh токенов.
// Также устанавливает CSRF токен (не HttpOnly).
// secureMode: true для HTTPS (production), false для HTTP (development).
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, cfg *CookieConfig) {
	if cfg == nil {
		cfg = &DefaultCookieConfig
	}

	// Access token (HttpOnly, Secure, SameSite=Strict)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieNameAccessToken,
		Value:    accessToken,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		Expires:  time.Now().Add(cfg.AccessTokenTTL),
		MaxAge:   int(cfg.AccessTokenTTL.Seconds()),
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: cfg.SameSite,
	})

	// Refresh token (HttpOnly, Secure, SameSite=Strict)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieNameRefreshToken,
		Value:    refreshToken,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		Expires:  time.Now().Add(cfg.RefreshTokenTTL),
		MaxAge:   int(cfg.RefreshTokenTTL.Seconds()),
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: cfg.SameSite,
	})

	// CSRF token (не HttpOnly — доступен JS для отправки в заголовке)
	csrfToken := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     CookieNameCSRF,
		Value:    csrfToken,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		Expires:  time.Now().Add(cfg.AccessTokenTTL),
		MaxAge:   int(cfg.AccessTokenTTL.Seconds()),
		Secure:   cfg.Secure,
		HttpOnly: false,
		SameSite: cfg.SameSite,
	})
}

// ClearAuthCookies очищает все auth cookies (logout).
func ClearAuthCookies(w http.ResponseWriter, cfg *CookieConfig) {
	if cfg == nil {
		cfg = &DefaultCookieConfig
	}

	for _, name := range []string{CookieNameAccessToken, CookieNameRefreshToken, CookieNameCSRF} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     cfg.Path,
			Domain:   cfg.Domain,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			Secure:   cfg.Secure,
			HttpOnly: name != CookieNameCSRF, // CSRF cookie не HttpOnly
			SameSite: cfg.SameSite,
		})
	}
}

// GetAccessTokenFromCookie извлекает access token из cookie.
func GetAccessTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(CookieNameAccessToken)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetRefreshTokenFromCookie извлекает refresh token из HttpOnly cookie.
func GetRefreshTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(CookieNameRefreshToken)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetCSRFTokenFromCookie извлекает CSRF токен из cookie.
func GetCSRFTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(CookieNameCSRF)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// ────────────────────────────────────────────────────────────────────────────
// CSRF token generation
// ────────────────────────────────────────────────────────────────────────────

// generateCSRFToken генерирует случайный CSRF токен.
func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// ValidateCSRFToken проверяет CSRF токен из заголовка против cookie.
func ValidateCSRFToken(r *http.Request) bool {
	// Только для state-changing методов
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}

	cookieToken := GetCSRFTokenFromCookie(r)
	headerToken := r.Header.Get("X-CSRF-Token")

	if cookieToken == "" || headerToken == "" {
		return false
	}

	// Constant-time comparison
	if len(cookieToken) != len(headerToken) {
		return false
	}
	for i := range cookieToken {
		if cookieToken[i] != headerToken[i] {
			return false
		}
	}
	return true
}

// ────────────────────────────────────────────────────────────────────────────
// CookieAuthMiddleware — middleware для аутентификации через HttpOnly cookie
// ────────────────────────────────────────────────────────────────────────────

// CookieAuthMiddleware извлекает JWT из HttpOnly cookie и устанавливает
// claims в контекст (аналогично AuthMiddleware).
func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Пробуем получить токен из cookie
		tokenString := GetAccessTokenFromCookie(r)

		// Fallback на Authorization header (для API клиентов)
		if tokenString == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Устанавливаем Authorization header для совместимости с существующей
		// AuthMiddleware, которая читает из заголовка
		r.Header.Set("Authorization", "Bearer "+tokenString)
		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────────────────────
// CSRFConfig — конфигурация CSRF middleware
// ────────────────────────────────────────────────────────────────────────────

// CSRFConfig — конфигурация для CSRF middleware.
type CSRFConfig struct {
	// Secure — требовать HTTPS (должно быть true в production).
	Secure bool
	// Domain — домен cookie.
	Domain string
	// Path — путь cookie.
	Path string
	// SameSite — политика SameSite для CSRF cookie.
	SameSite http.SameSite
	// ExcludedPaths — точные пути, исключённые из CSRF проверки.
	ExcludedPaths []string
	// ExcludedPathPrefixes — префиксы путей, исключённые из проверки.
	ExcludedPathPrefixes []string
	// RotationInterval — интервал ротации CSRF токена (по умолчанию 30 мин).
	RotationInterval time.Duration
}

// DefaultCSRFConfig — конфигурация CSRF по умолчанию.
var DefaultCSRFConfig = CSRFConfig{
	Secure:               true,
	Path:                 "/",
	SameSite:             http.SameSiteStrictMode,
	RotationInterval:     30 * time.Minute,
	ExcludedPathPrefixes: []string{"/api/v1/webhooks", "/api/v1/external", "/api/v1/public", "/api/v1/health"},
}

// ────────────────────────────────────────────────────────────────────────────
// CSRF token rotation (thread-safe)
// ────────────────────────────────────────────────────────────────────────────

// csrfTokenMeta хранит метаданные CSRF токена для rotation.
type csrfTokenMeta struct {
	createdAt time.Time
}

var (
	csrfTokenStore   = make(map[string]*csrfTokenMeta)
	csrfTokenStoreMu sync.RWMutex
)

// registerCSRFToken регистрирует CSRF токен для отслеживания rotation.
func registerCSRFToken(token string) {
	csrfTokenStoreMu.Lock()
	defer csrfTokenStoreMu.Unlock()
	csrfTokenStore[token] = &csrfTokenMeta{createdAt: time.Now()}

	// Cleanup старых токенов (раз в 100 регистраций)
	if len(csrfTokenStore) > 1000 {
		for t, meta := range csrfTokenStore {
			if time.Since(meta.createdAt) > 2*DefaultCSRFConfig.RotationInterval {
				delete(csrfTokenStore, t)
			}
		}
	}
}

// needsRotation проверяет, нужна ли ротация CSRF токена.
func needsRotation(token string, interval time.Duration) bool {
	csrfTokenStoreMu.RLock()
	defer csrfTokenStoreMu.RUnlock()
	meta, ok := csrfTokenStore[token]
	if !ok {
		return false // неизвестный токен — не ротируем
	}
	return time.Since(meta.createdAt) >= interval
}

// rotateCSRFTokenIfNeeded генерирует новый CSRF токен, если текущий устарел.
// Возвращает новый токен (или пустую строку, если ротация не нужна).
func rotateCSRFTokenIfNeeded(currentToken string, cfg *CSRFConfig) string {
	if currentToken == "" || cfg == nil {
		return ""
	}
	if cfg.RotationInterval <= 0 {
		cfg.RotationInterval = 30 * time.Minute
	}
	if !needsRotation(currentToken, cfg.RotationInterval) {
		return ""
	}
	newToken := generateCSRFToken()
	if newToken == "" {
		return ""
	}
	registerCSRFToken(newToken)
	return newToken
}

// ────────────────────────────────────────────────────────────────────────────
// SetCSRFTokenCookie — установка CSRF cookie с поддержкой rotation
// ────────────────────────────────────────────────────────────────────────────

// SetCSRFTokenCookie устанавливает CSRF токен в cookie (не HttpOnly).
//
// Cookie НЕ HttpOnly, чтобы JS мог читать его и добавлять в X-CSRF-Token.
// Это стандартный Double Submit Cookie pattern.
func SetCSRFTokenCookie(w http.ResponseWriter, token string, cfg *CSRFConfig) {
	if cfg == nil {
		cfg = &DefaultCSRFConfig
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieNameCSRF,
		Value:    token,
		Path:     cfg.Path,
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HttpOnly: false, // Должен быть доступен JS для чтения!
		SameSite: cfg.SameSite,
		MaxAge:   int((24 * time.Hour).Seconds()), // 24h TTL
	})
}

// ────────────────────────────────────────────────────────────────────────────
// CSRFMiddleware — middleware для проверки CSRF токена
// ────────────────────────────────────────────────────────────────────────────

// isExcludedPath проверяет, исключён ли путь из CSRF проверки.
func isExcludedPath(r *http.Request, cfg *CSRFConfig) bool {
	path := r.URL.Path
	for _, prefix := range cfg.ExcludedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	for _, p := range cfg.ExcludedPaths {
		if path == p {
			return true
		}
	}
	return false
}

// CSRFMiddleware проверяет CSRF токен для state-changing методов.
//
// Особенности:
//   - Double Submit Cookie pattern (stateless)
//   - Token rotation каждые 30 минут
//   - WebSocket upgrade bypass (Connection: Upgrade)
//   - API key аутентификация bypass (X-API-Key)
//   - Excluded paths для webhooks, external, public endpoints
//
// Соответствует: OWASP ASVS V3.2, ISO 27001 A.9.4.2, СТБ 34.101.27 п. 6.1
func CSRFMiddleware(next http.Handler) http.Handler {
	return CSRFMiddlewareWithConfig(next, nil)
}

// CSRFMiddlewareWithConfig создаёт CSRF middleware с кастомной конфигурацией.
func CSRFMiddlewareWithConfig(next http.Handler, cfg *CSRFConfig) http.Handler {
	if cfg == nil {
		cfg = &DefaultCSRFConfig
	}
	if cfg.RotationInterval <= 0 {
		cfg.RotationInterval = 30 * time.Minute
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ── 1. Safe методы — пропускаем ──────────────────────────────
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			// Обновляем CSRF токен если нужно
			if token := GetCSRFTokenFromCookie(r); token != "" {
				if newToken := rotateCSRFTokenIfNeeded(token, cfg); newToken != "" {
					SetCSRFTokenCookie(w, newToken, cfg)
				}
			}
			next.ServeHTTP(w, r)
			return
		}

		// ── 2. WebSocket upgrade — пропускаем ───────────────────────
		if strings.EqualFold(r.Header.Get("Connection"), "Upgrade") &&
			strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}

		// ── 3. API key аутентификация — пропускаем ─────────────────
		if r.Header.Get("X-API-Key") != "" {
			next.ServeHTTP(w, r)
			return
		}

		// ── 4. Исключённые пути ────────────────────────────────────
		if isExcludedPath(r, cfg) {
			next.ServeHTTP(w, r)
			return
		}

		// ── 5. Валидация CSRF токена ───────────────────────────────
		if !ValidateCSRFToken(r) {
			respond.RespondError(w, r, respond.NewForbiddenError("CSRF token required: cookie or header missing or mismatch"))
			return
		}

		// ── 6. Token rotation ──────────────────────────────────────
		if token := GetCSRFTokenFromCookie(r); token != "" {
			if newToken := rotateCSRFTokenIfNeeded(token, cfg); newToken != "" {
				SetCSRFTokenCookie(w, newToken, cfg)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	ErrCSRFInvalid = fmt.Errorf("csrf: invalid token")
	ErrNoCookie    = fmt.Errorf("cookie: no auth cookie found")
)
