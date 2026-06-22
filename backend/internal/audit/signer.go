// Package audit предоставляет HMAC-подпись для обеспечения целостности журнала аудита.
// Соответствует требованиям ISO 27001:2022 A.12.4 (Logging and Monitoring),
// СТБ 34.101.30 (bash-256), СТБ 34.101.27 п. 7.2 (Защита журналов).
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
// TODO: Мигрировать на github.com/bp2012/crypto/bash (bash-hmac) после добавления зависимости.
// Временно используем crypto/sha256 как placeholder до добавления СТБ-пакета.
const MinKeyLength = 32

// Signer подписывает и верифицирует записи аудита с помощью HMAC.
// В production использует SHA256 (будет заменён на bash-256 из github.com/bp2012/crypto).
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
// Использует SHA256 (TODO: заменить на bash-256 из СТБ 34.101.77).
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
