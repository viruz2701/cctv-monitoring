// Package providers — AES-256-GCM Crypto Provider.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.2: AES-256-GCM Provider
//
// Используется для EU/US/INTL регионов (GDPR, ISO 27001).
//
// Соответствие:
//   - NIST SP 800-38D (GCM)
//   - ISO 27001 A.10.1 (Cryptographic controls)
//   - FIPS 140-3 (AES)
//   - OWASP ASVS V6 (Cryptographic storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"gb-telemetry-collector/internal/stb"
)

// ────────────────────────────────────────────────────────────────────────────
// Ensure interface compliance
// ────────────────────────────────────────────────────────────────────────────

var _ stb.CryptoProvider = (*AESCrypto)(nil)

// ────────────────────────────────────────────────────────────────────────────
// AESCrypto
// ────────────────────────────────────────────────────────────────────────────

// AESCrypto implements CryptoProvider using AES-256-GCM.
// Соответствует NIST SP 800-38D и FIPS 140-3.
type AESCrypto struct{}

// NewAESCrypto создаёт новый AES-256-GCM провайдер.
func NewAESCrypto() *AESCrypto {
	return &AESCrypto{}
}

func (a *AESCrypto) Hash(data []byte) ([]byte, error) {
	return stb.DefaultCrypto.Hash(data)
}

func (a *AESCrypto) HashHex(data []byte) (string, error) {
	return stb.DefaultCrypto.HashHex(data)
}

func (a *AESCrypto) HMAC(key, data []byte) ([]byte, error) {
	return stb.DefaultCrypto.HMAC(key, data)
}

func (a *AESCrypto) HMACHex(key, data []byte) (string, error) {
	return stb.DefaultCrypto.HMACHex(key, data)
}

// Encrypt шифрует данные с использованием AES-256-GCM.
// Возвращает: nonce || ciphertext || tag
func (a *AESCrypto) Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("aes-256-gcm: key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes-256-gcm: new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-256-gcm: new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("aes-256-gcm: nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt расшифровывает данные (ожидает: nonce || ciphertext || tag).
func (a *AESCrypto) Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("aes-256-gcm: key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes-256-gcm: new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-256-gcm: new gcm: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("aes-256-gcm: ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aes-256-gcm: decrypt: %w", err)
	}

	return plaintext, nil
}

func (a *AESCrypto) Sign(privateKey, data []byte) ([]byte, error) {
	return stb.DefaultCrypto.Sign(privateKey, data)
}

func (a *AESCrypto) Verify(publicKey, data, signature []byte) (bool, error) {
	return stb.DefaultCrypto.Verify(publicKey, data, signature)
}

func (a *AESCrypto) GenerateKey(length int) ([]byte, error) {
	return stb.DefaultCrypto.GenerateKey(length)
}
