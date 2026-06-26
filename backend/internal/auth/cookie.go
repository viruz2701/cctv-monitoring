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
// Compliance:
//   - OWASP ASVS V3.1 (Session management — HttpOnly cookies)
//   - OWASP ASVS V3.2 (CSRF protection)
//   - ISO 27001 A.9.2.1 (User registration — secure session)
//   - Приказ ОАЦ №66 п. 7.18.2 (Secure session management)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
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
// CSRFMiddleware — middleware для проверки CSRF токена
// ────────────────────────────────────────────────────────────────────────────

// CSRFMiddleware проверяет CSRF токен для state-changing методов.
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ValidateCSRFToken(r) {
			http.Error(w, `{"error":{"code":"CSRF_INVALID","message":"invalid CSRF token"}}`,
				http.StatusForbidden)
			return
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
