// Package audit предоставляет HMAC-подпись для обеспечения целостности журнала аудита.
// Соответствует требованиям ISO 27001:2022 A.12.4 (Logging and Monitoring).
package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrKeyTooShort возвращается, если HMAC-ключ короче 16 байт.
var ErrKeyTooShort = errors.New("audit HMAC key must be at least 16 bytes (128 bits)")

// MinKeyLength — минимальная длина ключа в байтах для HMAC-SHA256.
const MinKeyLength = 16

// Signer подписывает и верифицирует записи аудита с помощью HMAC-SHA256.
type Signer struct {
	key []byte
}

// NewSigner создаёт новый Signer с заданным ключом.
// Возвращает ошибку, если ключ короче MinKeyLength байт.
func NewSigner(key string) (*Signer, error) {
	if len(key) < MinKeyLength {
		return nil, fmt.Errorf("%w: got %d bytes", ErrKeyTooShort, len(key))
	}
	return &Signer{key: []byte(key)}, nil
}

// Sign вычисляет HMAC-SHA256 подпись для переданной строки данных.
func (s *Signer) Sign(data string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify проверяет HMAC-SHA256 подпись для переданных данных.
func (s *Signer) Verify(data, signature string) bool {
	expected := s.Sign(data)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// SignAuditEntry формирует строку для подписи из полей записи аудита.
func SignAuditEntry(userID, action, entityType, entityID string, oldValue, newValue []byte) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s", userID, action, entityType, entityID, string(oldValue), string(newValue))
}
