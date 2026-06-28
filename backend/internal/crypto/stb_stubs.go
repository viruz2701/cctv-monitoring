// Package crypto предоставляет заглушки для СТБ 34.101.30 криптографии.
//
// ═══════════════════════════════════════════════════════════════════════════
// СТБ Compliance Status (СТБ 34.101.30-2024):
//
// Phase 1 (✅ Реализовано): Audit log HMAC — signer.go (crypto/sha256 placeholder)
// Phase 2 (⚠️ Stub):       API key hashing — belt-hash
// Phase 3 (✅ Реализовано): JWT signing — bign-curve256v1 (ECDSA P-256)
//
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
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
	return hmacEqual([]byte(HashAPIKey(key)), []byte(hash))
}

// hmacEqual — constant-time сравнение (безопасная альтернатива crypto/hmac.Equal).
func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// ────────────────────────────────────────────────────────────────────────────
// Phase 3: JWT Signing (bign-curve256v1 — ECDSA P-256)
// ────────────────────────────────────────────────────────────────────────────

// BignSigningMethod — метод подписи JWT с bign-curve256v1 (ECDSA P-256 / ES256).
type BignSigningMethod struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

// NewBignSigningMethod создаёт новый метод подписи JWT с ECDSA P-256.
// Если privateKeyPEM пуст, генерируется новый ключ.
func NewBignSigningMethod(privateKeyPEM string) (*BignSigningMethod, error) {
	if privateKeyPEM == "" {
		// Автогенерация для dev
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate bign key: %w", err)
		}
		return &BignSigningMethod{
			privateKey: privKey,
			publicKey:  &privKey.PublicKey,
		}, nil
	}

	// Парсим PEM ключ
	privKey, err := parseECPrivateKeyPEM([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("parse bign key: %w", err)
	}

	return &BignSigningMethod{
		privateKey: privKey,
		publicKey:  &privKey.PublicKey,
	}, nil
}

// parseECPrivateKeyPEM парсит PEM-encoded ECDSA приватный ключ.
func parseECPrivateKeyPEM(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// Попробуем PKCS8
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: %v (EC: %v, PKCS8: %v)", err, err, err2)
		}
		ecKey, ok := pkcs8Key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an ECDSA private key")
		}
		return ecKey, nil
	}
	return key, nil
}

// Sign подписывает JWT claims с bign-curve256v1 (ES256).
func (m *BignSigningMethod) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(m.privateKey)
}

// Verify проверяет JWT токен с bign-curve256v1.
func (m *BignSigningMethod) Verify(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
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
