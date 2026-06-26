// Package providers — SM4 Crypto Provider (stub).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.2: SM4 Crypto Provider (stub)
//
// ⚠ STUB: Полная реализация в P2-CN.1 (SM Crypto Integration).
// Сейчас использует AES-256-GCM как fallback.
//
// Целевые алгоритмы (P2-CN.1):
//   - SM4 — block cipher (GB/T 32907)
//   - SM3 — hash (GB/T 32905)
//   - SM2 — signatures (GB/T 32918)
//
// Compliance:
//   - GM/T 0002-2012 (SM4 block cipher)
//   - GM/T 0003-2012 (SM2 public key)
//   - GM/T 0004-2012 (SM3 hash)
//   - MLPS 2.0 (Multi-Level Protection Scheme)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"gb-telemetry-collector/internal/stb"
)

// ────────────────────────────────────────────────────────────────────────────
// Ensure interface compliance
// ────────────────────────────────────────────────────────────────────────────

var _ stb.CryptoProvider = (*SMCrypto)(nil)

// ────────────────────────────────────────────────────────────────────────────
// SMCrypto
// ────────────────────────────────────────────────────────────────────────────

// SMCrypto implements CryptoProvider using SM algorithms (stub).
//
// ⚠ STUB: Реализация через AES-256-GCM до P2-CN.1.
//
// Целевая реализация (P2-CN.1):
//   - Encrypt/Decrypt: SM4 (GB/T 32907)
//   - Hash: SM3 (GB/T 32905)
//   - Sign/Verify: SM2 (GB/T 32918)
type SMCrypto struct {
	status   string
	fallback *AESCrypto
}

// NewSMCrypto создаёт SM4 провайдер (stub).
func NewSMCrypto() *SMCrypto {
	return &SMCrypto{
		status:   "stub",
		fallback: NewAESCrypto(),
	}
}

func (s *SMCrypto) Hash(data []byte) ([]byte, error) {
	return s.fallback.Hash(data)
}

func (s *SMCrypto) HashHex(data []byte) (string, error) {
	return s.fallback.HashHex(data)
}

func (s *SMCrypto) HMAC(key, data []byte) ([]byte, error) {
	return s.fallback.HMAC(key, data)
}

func (s *SMCrypto) HMACHex(key, data []byte) (string, error) {
	return s.fallback.HMACHex(key, data)
}

func (s *SMCrypto) Encrypt(key, plaintext []byte) ([]byte, error) {
	return s.fallback.Encrypt(key, plaintext)
}

func (s *SMCrypto) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return s.fallback.Decrypt(key, ciphertext)
}

func (s *SMCrypto) Sign(privateKey, data []byte) ([]byte, error) {
	return s.fallback.Sign(privateKey, data)
}

func (s *SMCrypto) Verify(publicKey, data, signature []byte) (bool, error) {
	return s.fallback.Verify(publicKey, data, signature)
}

func (s *SMCrypto) GenerateKey(length int) ([]byte, error) {
	return s.fallback.GenerateKey(length)
}

// Status возвращает статус реализации.
func (s *SMCrypto) Status() string { return s.status }
