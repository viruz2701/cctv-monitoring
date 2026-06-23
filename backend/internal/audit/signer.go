// Package audit предоставляет HMAC-подпись для обеспечения целостности журнала аудита.
// Соответствует требованиям ISO 27001:2022 A.12.4 (Logging and Monitoring),
// СТБ 34.101.30 (bash-256), СТБ 34.101.27 п. 7.2 (Защита журналов).
//
// ═══════════════════════════════════════════════════════════════════════════
// СТБ Compliance Status (СТБ 34.101.30-2024):
//
// Phase 1 (Текущий): ✅ Audit log HMAC — PLACEHOLDER (crypto/sha256)
//   - Временно используем crypto/sha256 как placeholder
//   - План: заменить на github.com/bp2012/crypto/bash (bash-256 HMAC)
//
// Phase 2 (Следующий): ❌ API key hashing — belt-hash
//   - Требует: github.com/bp2012/crypto/belt
//
// Phase 3 (Будущий):  ❌ JWT signing — bign-curve256v1
//   - Требует: github.com/bp2012/crypto/bign
//
// Блокирующий фактор: пакет github.com/bp2012/crypto не добавлен в go.mod
// До добавления: используем crypto/sha256 (НЕ СТБ) — только для development
// ═══════════════════════════════════════════════════════════════════════════
package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrKeyTooShort возвращается, если HMAC-ключ короче MinKeyLength байт.
var ErrKeyTooShort = errors.New("audit HMAC key must be at least 32 bytes (256 bits)")

// MinKeyLength — минимальная длина ключа в байтах для HMAC (СТБ 34.101.30: 256 бит = 32 байта).
// В production mode рекомендуется 32+ байт.
//
// ⚠ СТБ COMPLIANCE: После добавления github.com/bp2012/crypto/bash:
//
//	import "github.com/bp2012/crypto/bash"
//	func (s *Signer) Sign(data string) string {
//	    h := bash.NewHmac(s.key, bash.Size256)
//	    h.Write([]byte(data))
//	    return hex.EncodeToString(h.Sum(nil))
//	}
const MinKeyLength = 32

// Signer подписывает и верифицирует записи аудита с помощью HMAC.
// ⚠ Временно использует SHA256 — заменить на bash-256 после добавления bp2012/crypto
type Signer struct {
	key []byte
}

// NewSigner создаёт новый Signer с заданным ключом.
// Возвращает ошибку, если ключ короче MinKeyLength байт.
// Соответствует: СТБ 34.101.30 (256-bit key), ISO 27001 A.12.4.2
func NewSigner(key string) (*Signer, error) {
	if len(key) < MinKeyLength {
		return nil, fmt.Errorf("%w: got %d bytes, need %d", ErrKeyTooShort, len(key), MinKeyLength)
	}
	return &Signer{key: []byte(key)}, nil
}

// Sign вычисляет HMAC подпись для переданной строки данных.
// ⚠ Временно: SHA256 (не СТБ). Цель: bash-256 из СТБ 34.101.77.
//
// После миграции на bp2012/crypto:
//
//	func (s *Signer) Sign(data string) string {
//	    mac := bash.NewHmac(s.key, bash.Size256)
//	    mac.Write([]byte(data))
//	    return hex.EncodeToString(mac.Sum(nil))
//	}
func (s *Signer) Sign(data string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify проверяет HMAC подпись для переданных данных.
func (s *Signer) Verify(data, signature string) bool {
	expected := s.Sign(data)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// SignAuditEntry формирует строку для подписи из полей записи аудита.
func SignAuditEntry(userID, action, entityType, entityID string, oldValue, newValue []byte) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s", userID, action, entityType, entityID, string(oldValue), string(newValue))
}
