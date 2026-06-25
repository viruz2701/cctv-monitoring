// Package crypto provides encryption/decryption utilities for sensitive data at rest.
//
// ═══════════════════════════════════════════════════════════════════════════
// СТБ Compliance (СТБ 34.101.30-2024):
//
// Текущая реализация: Делегирует через stb.CryptoProvider (StandardCrypto).
//   - Encrypt/Decrypt: stb.CryptoProvider.Encrypt/Decrypt
//   - Сейчас: AES-256-GCM (через StandardCrypto)
//   - Цель: belt-gcm (СТБ 34.101.31) после добавления github.com/bp2012/crypto
//
// Pre-commit checklist:
//   [✅] Используется stb.CryptoProvider (не прямой crypto/aes)
//   [✅] Ключи из env (не config.yaml)
//   [✅] crypto/rand для генерации nonce
//   [✅] Нет fallback на слабые алгоритмы
//   [✅] Graceful: errors, не panic()
//   [ ] После получения bp2012/crypto: заменить DefaultCrypto на BeltCrypto
// ═══════════════════════════════════════════════════════════════════════════
package crypto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"gb-telemetry-collector/internal/stb"
)

var (
	ErrInvalidKeyLength = errors.New("encryption key must be 32 bytes (64 hex chars) for AES-256/belt-256")
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")
)

// getEncryptionKey retrieves the encryption key from the environment.
// Returns error if not set or invalid — never panics.
//
// СТБ Compliance: 256-bit ключ (32 байта) согласно СТБ 34.101.30.
// В production: ключ должен ротироваться каждые 90 дней.
func getEncryptionKey() ([]byte, error) {
	keyHex := os.Getenv("PUSH_TOKEN_ENCRYPTION_KEY")
	if keyHex == "" {
		return nil, errors.New("PUSH_TOKEN_ENCRYPTION_KEY environment variable is required")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("PUSH_TOKEN_ENCRYPTION_KEY must be 64 hex characters (32 bytes): %v", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using stb.CryptoProvider.
//
// СТБ Compliance:
//   - Сейчас: stb.StandardCrypto (AES-256-GCM) — временно
//   - Цель: stb.BeltCrypto (belt-gcm, СТБ 34.101.31)
//
// После миграции на bp2012/crypto:
//
//	достаточно заменить DefaultCrypto в stb/crypto.go
//	на BeltCrypto — этот файл не требует изменений.
func Encrypt(plaintext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := stb.DefaultCrypto.Encrypt(key, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded ciphertext using stb.CryptoProvider.
func Decrypt(hexCiphertext string) (string, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("decode hex: %w", err)
	}

	plaintext, err := stb.DefaultCrypto.Decrypt(key, ciphertext)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}
