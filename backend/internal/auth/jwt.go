// Package auth — аутентификация и управление доступом.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-SEC.2: bign JWT — ECDSA P-256 (bign-curve256v1)
//
// Переход с HMAC-SHA256 (HS256) на ECDSA P-256 (ES256):
//   - GenerateJWT / ValidateJWT используют ES256 с bign-curve256v1
//   - GenerateTempToken / ValidateTempToken также используют ES256
//   - Refresh tokens остаются на HMAC-SHA256 (не JWT, а opaque tokens)
//
// Compliance:
//   - СТБ 34.101.45 — bign-curve256v1
//   - СТБ 34.101.30 — Криптографические алгоритмы РБ
//   - OWASP ASVS V6.2.2 — Асимметричная криптография
//   - Приказ ОАЦ №66 п. 7.18.1 — Сертификаты bign
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — кастомные JWT claims для CCTV Health Monitor.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	// Region — регион для применения региональных политик сессий (P2-CR.4).
	// Может быть пустым для обратной совместимости (тогда используется BY).
	Region string `json:"region,omitempty"`
	jwt.RegisteredClaims
}

// AccessTokenTTL — время жизни access token (15 минут, OWASP ASVS V3.3.1).
const AccessTokenTTL = 15 * time.Minute

// GenerateJWT создаёт JWT с подписью bign-curve256v1 (ECDSA P-256 / ES256).
func GenerateJWT(userID, username, role, tenantID string) (string, error) {
	return GenerateJWTWithRegion(userID, username, role, tenantID, "")
}

// GenerateJWTWithRegion создаёт JWT с указанием региона для региональных политик (P2-CR.4).
// Если region пустой, используется BY (наиболее строгие требования КИИ).
//
// Подпись: ECDSA P-256 (bign-curve256v1) через jwt.SigningMethodES256.
func GenerateJWTWithRegion(userID, username, role, tenantID, region string) (string, error) {
	key, err := GetBignPrivateKey()
	if err != nil {
		return "", err
	}

	if region == "" {
		region = string(RegionBY)
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
		Region:   region,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(key)
}

// ValidateJWT проверяет JWT и возвращает claims.
// Использует ECDSA P-256 (bign-curve256v1) верификацию.
func ValidateJWT(tokenString string) (*Claims, error) {
	key, err := GetBignPublicKey()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, errors.New("unexpected signing method: expected ES256")
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// GenerateTempToken generates a short-lived token for 2FA verification step (5 minutes).
// Использует ECDSA P-256 подпись.
func GenerateTempToken(userID, username, role, tenantID string) (string, error) {
	return GenerateTempTokenWithRegion(userID, username, role, tenantID, "")
}

// GenerateTempTokenWithRegion creates a 2FA temp token with region for session policies (P2-CR.4).
func GenerateTempTokenWithRegion(userID, username, role, tenantID, region string) (string, error) {
	key, err := GetBignPrivateKey()
	if err != nil {
		return "", err
	}

	if region == "" {
		region = string(RegionBY)
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
		Region:   region,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "2fa_pending",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(key)
}

// ValidateTempToken validates a temporary 2FA token.
func ValidateTempToken(tokenString string) (*Claims, error) {
	key, err := GetBignPublicKey()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, errors.New("unexpected signing method: expected ES256")
		}
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.Subject != "2fa_pending" {
			return nil, errors.New("not a 2FA temp token")
		}
		return claims, nil
	}
	return nil, errors.New("invalid temp token")
}

// ────────────────────────────────────────────────────────────────────────────
// JWT → HttpOnly Cookies (P0-SEC.3)
// ────────────────────────────────────────────────────────────────────────────

// SetAuthCookie записывает JWT в HttpOnly cookie.
//   - HttpOnly, Secure, SameSite=Strict
//   - CSRF token в заголовке X-CSRF-Token
const (
	AuthCookieName = "auth_token"
	CSRFHeaderName = "X-CSRF-Token"
	CookieMaxAge   = 24 * time.Hour * 7 // 7 days
)

// SetAuthCookie записывает JWT в HttpOnly cookie.
func SetAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(CookieMaxAge.Seconds()),
	})
}

// ClearAuthCookie удаляет cookie (для logout).
func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// ExtractTokenFromCookie извлекает JWT из cookie.
func ExtractTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// (Refresh token logic moved to refresh_token.go — P1-HI-05)
