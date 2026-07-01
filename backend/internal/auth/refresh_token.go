// Package auth — аутентификация и управление доступом.
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-HI-05: Refresh Token Rotation + Device Fingerprinting + Reuse Detection
//
// Проблема: JWT refresh token используется без rotation — если refresh token
// скомпрометирован, злоумышленник может получать новые access token бесконечно.
//
// Решение:
//  1. Refresh token rotation — каждый раз при обмене refresh token на access
//     token, старый refresh token инвалидируется и выдаётся новый
//  2. Device fingerprinting — привязка refresh token к устройству через хеш
//     User-Agent + IP
//  3. Reuse detection — если старый refresh token используется повторно
//     (украден), инвалидируется вся семья токенов
//
// Compliance:
//   - OWASP ASVS V3.2.2 — Refresh token rotation
//   - OWASP ASVS V3.2.3 — Reuse detection (token family)
//   - OWASP ASVS V3.2.4 — Device binding (fingerprint)
//   - ISO 27001 A.9.2.1 — Device/user binding
//   - Приказ ОАЦ №66 п. 7.18.1 — Уникальная идентификация узлов
//   - IEC 62443 SR 2.1 — Account management (session rotation)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

// RefreshTokenTTL — время жизни refresh token (30 дней).
const RefreshTokenTTL = 30 * 24 * time.Hour

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	// ErrRefreshTokenExpired — refresh token истёк.
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	// ErrRefreshTokenRevoked — refresh token отозван (reuse detection).
	ErrRefreshTokenRevoked = errors.New("refresh token revoked (reuse detected)")
	// ErrFingerprintMismatch — fingerprint устройства не совпадает.
	ErrFingerprintMismatch = errors.New("device fingerprint mismatch")
	// ErrTokenFamilyRevoked — вся семья токенов отозвана из-за reuse.
	ErrTokenFamilyRevoked = errors.New("token family revoked (reuse detected)")
)

// ────────────────────────────────────────────────────────────────────────────
// RefreshTokenStore — интерфейс хранилища refresh токенов
// ────────────────────────────────────────────────────────────────────────────

// RefreshTokenStore определяет методы для работы с refresh токенами в БД.
// Реализация находится в backend/internal/db/repository.go.
type RefreshTokenStore interface {
	// CreateSession создаёт новую сессию/запись refresh токена.
	// Возвращает ID созданной сессии.
	CreateSession(userID, tokenHash, ipAddress, userAgent, fingerprintHash string, tokenFamily *uuid.UUID, expiresAt time.Time) (string, error)

	// GetSessionByTokenHash возвращает сессию по хешу токена.
	// Возвращает ErrNotFound если токен не найден или истёк.
	GetSessionByTokenHash(tokenHash string) (*RefreshSession, error)

	// RevokeSession помечает сессию как отозванную (is_revoked = TRUE).
	RevokeSession(sessionID string) error

	// RevokeTokenFamily помечает ВСЕ сессии в семье как отозванные.
	// Используется при reuse detection.
	RevokeTokenFamily(tokenFamily uuid.UUID) error

	// GetActiveSessionsByFamily возвращает активные (не отозванные) сессии в семье.
	GetActiveSessionsByFamily(tokenFamily uuid.UUID) ([]*RefreshSession, error)
}

// RefreshSession — модель сессии refresh токена.
type RefreshSession struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	TokenHash       string     `json:"token_hash"`
	IPAddress       string     `json:"ip_address"`
	UserAgent       string     `json:"user_agent"`
	FingerprintHash string     `json:"fingerprint_hash"`
	TokenFamily     *uuid.UUID `json:"token_family"`
	IsRevoked       bool       `json:"is_revoked"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ────────────────────────────────────────────────────────────────────────────
// Fingerprint Computation
// ────────────────────────────────────────────────────────────────────────────

// ComputeFingerprint вычисляет хеш устройства на основе User-Agent и IP.
//
// Формат: SHA-256(User-Agent || "|" || IP)
// Используется для device fingerprinting (OWASP ASVS V3.2.4).
//
// Приказ ОАЦ №66 п. 7.18.1: уникальная идентификация конечных узлов.
func ComputeFingerprint(userAgent, ip string) string {
	if userAgent == "" && ip == "" {
		return ""
	}
	h := sha256.Sum256([]byte(normalizeUserAgent(userAgent) + "|" + ip))
	return hex.EncodeToString(h[:])
}

// normalizeUserAgent нормализует User-Agent для fingerprinting.
// Обрезает до первых 128 символов для стабильности.
func normalizeUserAgent(ua string) string {
	if len(ua) > 128 {
		return ua[:128]
	}
	return ua
}

// ClientIP извлекает реальный IP клиента из запроса.
// Учитывает X-Forwarded-For, X-Real-IP, RemoteAddr.
func ClientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if host, _, err := net.SplitHostPort(forwardedFor); err == nil {
			return host
		}
		// Берём первый IP из цепочки
		if idx := strings.IndexByte(forwardedFor, ','); idx > 0 {
			return strings.TrimSpace(forwardedFor[:idx])
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

// ────────────────────────────────────────────────────────────────────────────
// Token Generation
// ────────────────────────────────────────────────────────────────────────────

// GenerateRefreshToken генерирует opaque refresh token.
//
// Возвращает: raw token, hash для БД, expiresAt, error.
// ⚠ Refresh tokens — opaque (не JWT), SHA-256 только для хеширования в БД.
func GenerateRefreshToken() (string, string, time.Time, error) {
	return generateRefreshTokenWithPrefix("rt_")
}

// generateRefreshTokenWithPrefix генерирует refresh token с префиксом.
func generateRefreshTokenWithPrefix(prefix string) (string, string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", time.Time{}, err
	}
	token := prefix + base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := time.Now().Add(RefreshTokenTTL)
	return token, HashRefreshToken(token), expiresAt, nil
}

// HashRefreshToken хеширует refresh token для хранения в БД.
// Использует SHA-256 (не криптографический, а для хеширования токена).
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ────────────────────────────────────────────────────────────────────────────
// Rotation Logic
// ────────────────────────────────────────────────────────────────────────────

// RotateResult — результат ротации refresh token.
type RotateResult struct {
	// NewToken — новый refresh token (opaque).
	NewToken string `json:"new_token"`
	// NewTokenHash — хеш нового токена для хранения в БД.
	NewTokenHash string `json:"new_token_hash"`
	// NewSessionID — ID новой сессии в БД.
	NewSessionID string `json:"new_session_id"`
	// ExpiresAt — время истечения нового токена.
	ExpiresAt time.Time `json:"expires_at"`
	// ReuseDetected — TRUE если был обнаружен reuse старого токена.
	ReuseDetected bool `json:"reuse_detected"`
	// RevokedFamily — TRUE если вся семья была отозвана из-за reuse.
	RevokedFamily bool `json:"revoked_family"`
}

// RotateRefreshToken выполняет rotation refresh token с проверками:
//  1. Проверяет, что токен существует и не истёк
//  2. Проверяет, что токен не был ранее отозван (reuse detection)
//  3. Проверяет fingerprint устройства
//  4. Если токен уже отозван — инвалидирует всю семью (reuse detection)
//  5. Инвалидирует старый токен
//  6. Создаёт новый токен в той же семье
//
// Compliance: OWASP ASVS V3.2.2, V3.2.3, V3.2.4
func RotateRefreshToken(
	store RefreshTokenStore,
	oldTokenHash string,
	fingerprintHash string,
	userID string,
	ipAddress string,
	userAgent string,
) (*RotateResult, error) {
	// 1. Получаем старую сессию
	session, err := store.GetSessionByTokenHash(oldTokenHash)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// 2. Проверяем, не истёк ли токен
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}

	// 3. REUSE DETECTION: Если токен уже отозван — это reuse атака
	if session.IsRevoked {
		// Инвалидируем ВСЮ семью токенов
		if session.TokenFamily != nil {
			if revokeErr := store.RevokeTokenFamily(*session.TokenFamily); revokeErr != nil {
				return nil, fmt.Errorf("revoke token family on reuse: %w", revokeErr)
			}
		}

		return &RotateResult{
			ReuseDetected: true,
			RevokedFamily: true,
		}, ErrRefreshTokenRevoked
	}

	// 4. Проверяем fingerprint (если есть)
	if session.FingerprintHash != "" && fingerprintHash != "" &&
		session.FingerprintHash != fingerprintHash {
		// Fingerprint не совпадает — возможно, токен украден
		// Не инвалидируем семью (может быть легитимная смена сети/IP),
		// но отклоняем запрос
		return nil, ErrFingerprintMismatch
	}

	// 5. Инвалидируем старый токен
	if err := store.RevokeSession(session.ID); err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}

	// 6. Определяем семью токенов
	var tokenFamily *uuid.UUID
	if session.TokenFamily != nil {
		// Продолжаем существующую семью
		tokenFamily = session.TokenFamily
	} else {
		// Создаём новую семью (для мигрированных токенов без семьи)
		fam := uuid.New()
		tokenFamily = &fam
	}

	// 7. Генерируем новый токен
	newToken, newTokenHash, expiresAt, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate new refresh token: %w", err)
	}

	// 8. Создаём новую сессию
	newSessionID, err := store.CreateSession(
		userID, newTokenHash, ipAddress, userAgent,
		fingerprintHash, tokenFamily, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create new session: %w", err)
	}

	return &RotateResult{
		NewToken:      newToken,
		NewTokenHash:  newTokenHash,
		NewSessionID:  newSessionID,
		ExpiresAt:     expiresAt,
		ReuseDetected: false,
		RevokedFamily: false,
	}, nil
}

// ValidateRefreshRequest проверяет refresh token из запроса.
// Пробует получить токен из HttpOnly cookie, затем из JSON body.
// Возвращает значение токена и источник ("cookie" или "body").
func ValidateRefreshRequest(r *http.Request) (token string, source string, err error) {
	// Пробуем cookie
	token = GetRefreshTokenFromCookie(r)
	if token != "" {
		return token, "cookie", nil
	}

	// Пробуем JSON body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		return "", "", errors.New("missing refresh token")
	}
	return req.RefreshToken, "body", nil
}
