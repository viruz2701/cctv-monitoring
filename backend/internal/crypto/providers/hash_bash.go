// Package providers — bash-256 Hash Provider (СТБ 34.101.77).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.3: bash-256 Hash Provider
//
// Реализует хеширование bash-256 согласно СТБ 34.101.77.
// Используется для BY региона (audit log HMAC, подписи).
//
// ⚠ STUB: Требует github.com/bp2012/crypto в go.mod.
// Сейчас использует SHA-256 как временное решение.
//
// Цель после миграции:
//
//	import "github.com/bp2012/crypto/bash"
//	func Bash256(data []byte) []byte {
//	    h := bash.New(bash.Size256)
//	    h.Write(data)
//	    return h.Sum(nil)
//	}
//
// Compliance:
//   - СТБ 34.101.77 — bash-256
//   - СТБ 34.101.30 — Криптографические алгоритмы РБ
//   - Приказ ОАЦ № 66 п. 7.18 — Контроль целостности
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// Bash256Size — размер выхода bash-256 в байтах.
	Bash256Size = 32
	// Bash512Size — размер выхода bash-512 в байтах.
	Bash512Size = 64
)

// ────────────────────────────────────────────────────────────────────────────
// Hash functions
// ────────────────────────────────────────────────────────────────────────────

// Bash256 вычисляет bash-256 хеш.
// ⚠ Временно: SHA-256. Цель: bash-256 (СТБ 34.101.77).
//
// "СТБ COMPLIANCE: Bash256 использует SHA-256.
// Заменить на bp2012/crypto/bash при получении SDK."
func Bash256(data []byte) ([]byte, error) {
	h := sha256.Sum256(data)
	return h[:], nil
}

// Bash256Hex возвращает hex-encoded bash-256 хеш.
func Bash256Hex(data []byte) (string, error) {
	hash, err := Bash256(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// Bash256HMAC вычисляет bash-256 HMAC.
func Bash256HMAC(key, data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil), nil
}

// Bash256HMACHex возвращает hex-encoded bash-256 HMAC.
func Bash256HMACHex(key, data []byte) (string, error) {
	mac, err := Bash256HMAC(key, data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac), nil
}

// ────────────────────────────────────────────────────────────────────────────
// BashHashProvider — реализация CryptoProvider с bash-256
// ────────────────────────────────────────────────────────────────────────────

// BashHashProvider implements CryptoProvider using bash-256 (stub).
type BashHashProvider struct {
	status   string
	fallback *AESCrypto
}

// NewBashHashProvider создаёт bash-256 провайдер.
func NewBashHashProvider() *BashHashProvider {
	return &BashHashProvider{
		status:   "stub",
		fallback: NewAESCrypto(),
	}
}

func (b *BashHashProvider) Hash(data []byte) ([]byte, error) {
	return Bash256(data)
}

func (b *BashHashProvider) HashHex(data []byte) (string, error) {
	return Bash256Hex(data)
}

func (b *BashHashProvider) HMAC(key, data []byte) ([]byte, error) {
	return Bash256HMAC(key, data)
}

func (b *BashHashProvider) HMACHex(key, data []byte) (string, error) {
	return Bash256HMACHex(key, data)
}

func (b *BashHashProvider) Encrypt(key, plaintext []byte) ([]byte, error) {
	return b.fallback.Encrypt(key, plaintext)
}

func (b *BashHashProvider) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return b.fallback.Decrypt(key, ciphertext)
}

func (b *BashHashProvider) Sign(privateKey, data []byte) ([]byte, error) {
	return b.fallback.Sign(privateKey, data)
}

func (b *BashHashProvider) Verify(publicKey, data, signature []byte) (bool, error) {
	return b.fallback.Verify(publicKey, data, signature)
}

func (b *BashHashProvider) GenerateKey(length int) ([]byte, error) {
	return b.fallback.GenerateKey(length)
}

// Status возвращает статус реализации.
func (b *BashHashProvider) Status() string { return b.status }

// ────────────────────────────────────────────────────────────────────────────
// Audit Log HMAC (СТБ compliance)
// ────────────────────────────────────────────────────────────────────────────

// SignAuditLog подписывает запись аудита с помощью bash-256 HMAC.
// Соответствует ISO 27001 A.12.4, СТБ 34.101.27 п. 7.2.
func SignAuditLog(key []byte, entry string) (string, error) {
	return Bash256HMACHex(key, []byte(entry))
}

// VerifyAuditLog проверяет подпись записи аудита.
func VerifyAuditLog(key []byte, entry, signature string) (bool, error) {
	expected, err := SignAuditLog(key, entry)
	if err != nil {
		return false, fmt.Errorf("verify audit log: %w", err)
	}
	return hmac.Equal([]byte(expected), []byte(signature)), nil
}
