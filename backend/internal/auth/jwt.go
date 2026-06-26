// backend/internal/auth/jwt.go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// AccessTokenTTL — время жизни access token (15 минут, OWASP ASVS V3.3.1).
const AccessTokenTTL = 15 * time.Minute

func GenerateJWT(userID, username, role, tenantID string) (string, error) {
	secret, err := GetJWTSecret()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateJWT(tokenString string) (*Claims, error) {
	secret, err := GetJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
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
func GenerateTempToken(userID, username, role, tenantID string) (string, error) {
	secret, err := GetJWTSecret()
	if err != nil {
		return "", err
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "2fa_pending",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateTempToken validates a temporary 2FA token.
func ValidateTempToken(tokenString string) (*Claims, error) {
	secret, err := GetJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
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

// P3-1.2: JWT → HttpOnly Cookies
//
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

const RefreshTokenTTL = 30 * 24 * time.Hour

func GenerateRefreshToken() (string, string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(RefreshTokenTTL)
	return token, HashRefreshToken(token), expiresAt, nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
