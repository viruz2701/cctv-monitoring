// Package crypto предоставляет заглушки для СТБ 34.101.30 криптографии.
//
// ═══════════════════════════════════════════════════════════════════════════
// СТБ Compliance Status (СТБ 34.101.30-2024):
//
// Phase 1 (✅ Реализовано): Audit log HMAC — signer.go (crypto/sha256 placeholder)
// Phase 2 (⚠️ Stub):       API key hashing — belt-hash
// Phase 3 (⚠️ Stub):       JWT signing — bign-curve256v1
//
// После добавления github.com/bp2012/crypto в go.mod:
//   - belt:   import "github.com/bp2012/crypto/belt"
//   - bign:   import "github.com/bp2012/crypto/bign"
//   - bash:   import "github.com/bp2012/crypto/bash"
//
// Временное решение: используем crypto/sha256, crypto/hmac, golang-jwt
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// ────────────────────────────────────────────────────────────────────────────
// Phase 2: API Key Hashing (belt-hash placeholder)
// ────────────────────────────────────────────────────────────────────────────

// ErrKeyTooShort возвращается если ключ короче минимальной длины.
var ErrKeyTooShort = errors.New("crypto: key must be at least 32 bytes")

// MinKeyLength — минимальная длина ключа (СТБ 34.101.30: 256 бит = 32 байта).
const MinKeyLength = 32

// HashAPIKey хеширует API ключ.
// ⚠ Временно: SHA-256 (НЕ СТБ). Цель: belt-hash из bp2012/crypto.
//
// После миграции:
//
//	import "github.com/bp2012/crypto/belt"
//	func HashAPIKey(key string) string {
//	    h := belt.NewHash()
//	    h.Write([]byte(key))
//	    return hex.EncodeToString(h.Sum(nil))
//	}
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// ValidateAPIKey проверяет API ключ против хеша.
func ValidateAPIKey(key, hash string) bool {
	return hmac.Equal([]byte(HashAPIKey(key)), []byte(hash))
}

// ────────────────────────────────────────────────────────────────────────────
// Phase 3: JWT Signing (bign-curve256v1 placeholder)
// ────────────────────────────────────────────────────────────────────────────

// BignSigningMethod — заглушка для bign-curve256v1 подписи JWT.
// ⚠ Временно: HMAC-SHA256 (НЕ СТБ). Цель: bign-curve256v1 из bp2012/crypto.
//
// После миграции:
//
//	import "github.com/bp2012/crypto/bign"
//	// Использовать bign.SignPKCS8(privKey, data) для подписи JWT
type BignSigningMethod struct {
	secret []byte
}

// NewBignSigningMethod создаёт новый метод подписи JWT.
func NewBignSigningMethod(secret string) (*BignSigningMethod, error) {
	if len(secret) < MinKeyLength {
		return nil, fmt.Errorf("%w: got %d bytes", ErrKeyTooShort, len(secret))
	}
	return &BignSigningMethod{secret: []byte(secret)}, nil
}

// Sign подписывает JWT claims.
// ⚠ Временно: HMAC-SHA256. Цель: bign-curve256v1.
func (m *BignSigningMethod) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Verify проверяет JWT токен.
func (m *BignSigningMethod) Verify(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}
	if !token.Valid {
		return errors.New("invalid token")
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Key Generation
// ────────────────────────────────────────────────────────────────────────────

// GenerateKey генерирует криптостойкий ключ указанной длины.
func GenerateKey(length int) ([]byte, error) {
	if length < MinKeyLength {
		length = MinKeyLength
	}
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateAPIKey генерирует API ключ в hex-формате.
func GenerateAPIKey() (string, error) {
	key, err := GenerateKey(32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}
