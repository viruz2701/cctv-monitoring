// Package stb предоставляет abstraction layer для криптографии СТБ 34.101.30-2024.
//
// ═══════════════════════════════════════════════════════════════════════════
// СТБ Compliance Status
//
// Текущий статус: ✅ Phase 1 (Audit HMAC — SHA-256 placeholder)
// Блокирующий фактор: Нет сертифицированной Go-реализации bp2012/crypto
//
// План миграции:
//  1. Создан abstraction layer (CryptoProvider interface) — ТЕКУЩИЙ ФАЙЛ
//  2. Fallback на crypto/aes, crypto/sha256, crypto/hmac — временно
//  3. CGo wrapper: //go:build stb_certified — при получении SDK от ОАЦ
//  4. Замена одним PR: StandardCrypto → BeltCrypto/BignCrypto/BashCrypto
//
// Risk acceptance: Формальный exception от ИБ-отдела до получения SDK.
// ═══════════════════════════════════════════════════════════════════════════
package stb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// ────────────────────────────────────────────────────────────────────────────
// CryptoProvider interface
// ────────────────────────────────────────────────────────────────────────────

// CryptoProvider определяет интерфейс для всех криптографических операций.
// Текущая реализация: StandardCrypto (SHA-256, AES-GCM, HMAC).
//
// После получения сертифицированного СТБ-модуля:
//   - Hash → bash-256 (СТБ 34.101.77)
//   - HMAC → bash-256 HMAC (СТБ 34.101.77)
//   - Encrypt → belt-gcm (СТБ 34.101.31)
//   - Sign/Verify → bign-curve256v1 (СТБ 34.101.45)
type CryptoProvider interface {
	// Hash вычисляет хеш данных.
	// Сейчас: SHA-256 → Цель: bash-256
	Hash(data []byte) ([]byte, error)

	// HashHex возвращает hex-encoded хеш.
	HashHex(data []byte) (string, error)

	// HMAC вычисляет HMAC с заданным ключом.
	// Сейчас: HMAC-SHA256 → Цель: bash-256 HMAC
	HMAC(key, data []byte) ([]byte, error)

	// HMACHex возвращает hex-encoded HMAC.
	HMACHex(key, data []byte) (string, error)

	// Encrypt шифрует данные с использованием симметричного шифрования.
	// Сейчас: AES-256-GCM → Цель: belt-gcm
	Encrypt(key, plaintext []byte) ([]byte, error)

	// Decrypt расшифровывает данные.
	Decrypt(key, ciphertext []byte) ([]byte, error)

	// Sign подписывает данные.
	// Сейчас: HMAC-SHA256 → Цель: bign-curve256v1
	Sign(privateKey, data []byte) ([]byte, error)

	// Verify проверяет подпись.
	Verify(publicKey, data, signature []byte) (bool, error)

	// GenerateKey генерирует криптостойкий ключ.
	GenerateKey(length int) ([]byte, error)
}

// ────────────────────────────────────────────────────────────────────────────
// StandardCrypto — fallback реализация на стандартных алгоритмах
// ────────────────────────────────────────────────────────────────────────────

// StandardCrypto implements CryptoProvider using Go standard crypto.
//
// ⚠ ВРЕМЕННО: Использует SHA-256, AES-256-GCM, HMAC-SHA256.
// НЕ соответствует СТБ 34.101.30. Только для development.
//
// "СТБ COMPLIANCE: StandardCrypto не соответствует СТБ 34.101.30.
// Заменить на BeltCrypto/BignCrypto/BashCrypto при получении SDK."
type StandardCrypto struct{}

// NewStandardCrypto создаёт новый StandardCrypto.
func NewStandardCrypto() *StandardCrypto {
	return &StandardCrypto{}
}

func (s *StandardCrypto) Hash(data []byte) ([]byte, error) {
	h := sha256.Sum256(data)
	return h[:], nil
}

func (s *StandardCrypto) HashHex(data []byte) (string, error) {
	hash, err := s.Hash(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

func (s *StandardCrypto) HMAC(key, data []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil), nil
}

func (s *StandardCrypto) HMACHex(key, data []byte) (string, error) {
	mac, err := s.HMAC(key, data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac), nil
}

func (s *StandardCrypto) Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (s *StandardCrypto) Decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

func (s *StandardCrypto) Sign(privateKey, data []byte) ([]byte, error) {
	return s.HMAC(privateKey, data)
}

func (s *StandardCrypto) Verify(publicKey, data, signature []byte) (bool, error) {
	expected, err := s.HMAC(publicKey, data)
	if err != nil {
		return false, err
	}
	return hmac.Equal(signature, expected), nil
}

func (s *StandardCrypto) GenerateKey(length int) ([]byte, error) {
	if length < 32 {
		length = 32
	}
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return key, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Global instance
// ────────────────────────────────────────────────────────────────────────────

// DefaultCrypto — глобальный экземпляр CryptoProvider.
// Заменить на BeltCrypto при получении сертифицированного SDK.
var DefaultCrypto CryptoProvider = NewStandardCrypto()

// ────────────────────────────────────────────────────────────────────────────
// Convenience functions
// ────────────────────────────────────────────────────────────────────────────

func Hash(data []byte) ([]byte, error)               { return DefaultCrypto.Hash(data) }
func HashHex(data []byte) (string, error)            { return DefaultCrypto.HashHex(data) }
func HMAC(key, data []byte) ([]byte, error)          { return DefaultCrypto.HMAC(key, data) }
func HMACHex(key, data []byte) (string, error)       { return DefaultCrypto.HMACHex(key, data) }
func Encrypt(key, plaintext []byte) ([]byte, error)  { return DefaultCrypto.Encrypt(key, plaintext) }
func Decrypt(key, ciphertext []byte) ([]byte, error) { return DefaultCrypto.Decrypt(key, ciphertext) }
func Sign(privateKey, data []byte) ([]byte, error)   { return DefaultCrypto.Sign(privateKey, data) }
func Verify(publicKey, data, signature []byte) (bool, error) {
	return DefaultCrypto.Verify(publicKey, data, signature)
}
func GenerateKey(length int) ([]byte, error) { return DefaultCrypto.GenerateKey(length) }
