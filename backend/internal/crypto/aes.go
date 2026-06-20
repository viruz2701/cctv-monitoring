// Package crypto provides AES-256-GCM encryption/decryption utilities
// for sensitive data at rest (push tokens, API keys, etc.)
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrInvalidKeyLength = errors.New("encryption key must be 32 bytes (64 hex chars) for AES-256")
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")
)

// getEncryptionKey retrieves the AES-256 key from the environment.
// Panics if not set — this is intentional: encrypted data cannot be decrypted without the key.
func getEncryptionKey() []byte {
	keyHex := os.Getenv("PUSH_TOKEN_ENCRYPTION_KEY")
	if keyHex == "" {
		panic("PUSH_TOKEN_ENCRYPTION_KEY environment variable is required")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		panic(fmt.Sprintf("PUSH_TOKEN_ENCRYPTION_KEY must be 64 hex characters (32 bytes): %v", err))
	}
	return key
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns hex-encoded ciphertext: nonce(12 bytes) + ciphertext + tag(16 bytes).
func Encrypt(plaintext string) (string, error) {
	key := getEncryptionKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded ciphertext produced by Encrypt.
func Decrypt(hexCiphertext string) (string, error) {
	key := getEncryptionKey()

	ciphertext, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("decode hex: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}
