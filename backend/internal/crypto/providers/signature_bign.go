// Package providers — bign-curve256v1 Signature Provider (СТБ 34.101.45).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.3: bign-curve256v1 Signature Provider
//
// Реализует цифровые подписи bign-curve256v1 согласно СТБ 34.101.45.
// Используется для BY региона (JWT подписи, документооборот).
//
// ⚠ STUB: Требует github.com/bp2012/crypto в go.mod.
// Сейчас использует HMAC-SHA256 как временное решение.
//
// Цель после миграции:
//
//	import "github.com/bp2012/crypto/bign"
//	privKey, _ := bign.GenerateKey(bign.Curve256v1)
//	signature, _ := bign.SignPKCS8(privKey, data)
//	ok := bign.VerifyPKCS8(&privKey.PublicKey, data, signature)
//
// Compliance:
//   - СТБ 34.101.45 — bign-curve256v1
//   - СТБ 34.101.30 — Криптографические алгоритмы РБ
//   - Приказ ОАЦ № 66 п. 7.18.1 — Сертификаты bign
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// BignCurve256v1KeySize — размер ключа bign-curve256v1 в байтах.
	BignCurve256v1KeySize = 32
	// BignCurve256v1SignatureSize — размер подписи в байтах.
	BignCurve256v1SignatureSize = 64
)

// ────────────────────────────────────────────────────────────────────────────
// BignKeyPair — пара ключей bign-curve256v1
// ────────────────────────────────────────────────────────────────────────────

// BignKeyPair представляет пару ключей bign-curve256v1.
// ⚠ STUB: После миграции использовать bign.GenerateKey(bign.Curve256v1).
type BignKeyPair struct {
	PrivateKey []byte `json:"private_key"`
	PublicKey  []byte `json:"public_key"`
}

// GenerateBignKeyPair генерирует новую пару ключей bign-curve256v1.
// ⚠ Временно: 32-байтовый ключ (HMAC-SHA256). Цель: bign-curve256v1.
func GenerateBignKeyPair() (*BignKeyPair, error) {
	privKey := make([]byte, BignCurve256v1KeySize)
	if _, err := rand.Read(privKey); err != nil {
		return nil, fmt.Errorf("generate bign key: %w", err)
	}

	// ⚠ Временно: public key = SHA-256(private key).
	// Цель: bign-curve256v1 извлечение публичного ключа.
	pubKey := sha256.Sum256(privKey)

	return &BignKeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey[:],
	}, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Sign/Verify functions
// ────────────────────────────────────────────────────────────────────────────

// BignSign подписывает данные с использованием bign-curve256v1.
// ⚠ Временно: HMAC-SHA256. Цель: bign-curve256v1.
func BignSign(privateKey, data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, privateKey)
	mac.Write(data)
	return mac.Sum(nil), nil
}

// BignVerify проверяет подпись bign-curve256v1.
// ⚠ Временно: использует тот же ключ, что и для подписи (HMAC-SHA256 симметричный).
// Цель: bign-curve256v1 VerifyPKCS8 с публичным ключом.
func BignVerify(publicKey, data, signature []byte) (bool, error) {
	// Временная HMAC-проверка — используем publicKey как ключ
	// (в асимметричной схеме будет отдельный public key)
	mac := hmac.New(sha256.New, publicKey)
	mac.Write(data)
	expected := mac.Sum(nil)
	return hmac.Equal(signature, expected), nil
}

// ────────────────────────────────────────────────────────────────────────────
// JWT signing
// ────────────────────────────────────────────────────────────────────────────

// BignJWT подписывает JWT claims с использованием bign-curve256v1.
// ⚠ Временно: HMAC-SHA256 (HS256). Цель: bign-curve256v1.
//
// После миграции:
//
//	func BignJWT(claims jwt.Claims, privKey *bign.PrivateKey) (string, error) {
//	    token := jwt.NewWithClaims(new(BignSigningMethod), claims)
//	    return token.SignedString(privKey)
//	}
func BignJWT(claims jwt.Claims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// BignJWTVerify проверяет JWT токен с bign-curve256v1 подписью.
func BignJWTVerify(tokenString string, secret []byte, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return fmt.Errorf("bign jwt verify: %w", err)
	}
	if !token.Valid {
		return fmt.Errorf("bign jwt: invalid token")
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// BignSignatureProvider — реализация CryptoProvider с bign-curve256v1
// ────────────────────────────────────────────────────────────────────────────

// BignSignatureProvider implements CryptoProvider using bign-curve256v1 (stub).
type BignSignatureProvider struct {
	status   string
	fallback *AESCrypto
}

// NewBignSignatureProvider создаёт bign-curve256v1 провайдер.
func NewBignSignatureProvider() *BignSignatureProvider {
	return &BignSignatureProvider{
		status:   "stub",
		fallback: NewAESCrypto(),
	}
}

func (b *BignSignatureProvider) Hash(data []byte) ([]byte, error) {
	return Bash256(data)
}

func (b *BignSignatureProvider) HashHex(data []byte) (string, error) {
	return Bash256Hex(data)
}

func (b *BignSignatureProvider) HMAC(key, data []byte) ([]byte, error) {
	return Bash256HMAC(key, data)
}

func (b *BignSignatureProvider) HMACHex(key, data []byte) (string, error) {
	return Bash256HMACHex(key, data)
}

func (b *BignSignatureProvider) Encrypt(key, plaintext []byte) ([]byte, error) {
	return b.fallback.Encrypt(key, plaintext)
}

func (b *BignSignatureProvider) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return b.fallback.Decrypt(key, ciphertext)
}

// Sign подписывает данные с использованием bign-curve256v1.
func (b *BignSignatureProvider) Sign(privateKey, data []byte) ([]byte, error) {
	return BignSign(privateKey, data)
}

// Verify проверяет подпись bign-curve256v1.
func (b *BignSignatureProvider) Verify(publicKey, data, signature []byte) (bool, error) {
	return BignVerify(publicKey, data, signature)
}

func (b *BignSignatureProvider) GenerateKey(length int) ([]byte, error) {
	return b.fallback.GenerateKey(length)
}

// Status возвращает статус реализации.
func (b *BignSignatureProvider) Status() string { return b.status }

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// BignPublicKeyHex возвращает hex-encoded публичный ключ.
func (k *BignKeyPair) BignPublicKeyHex() string {
	return hex.EncodeToString(k.PublicKey)
}

// BignPrivateKeyHex возвращает hex-encoded приватный ключ.
func (k *BignKeyPair) BignPrivateKeyHex() string {
	return hex.EncodeToString(k.PrivateKey)
}
