// Package providers — GOST 28147-89 Crypto Provider (stub).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.2: GOST Crypto Provider (stub)
//
// ⚠ STUB: Полная реализация в P2-RU.1 (GOST Crypto Integration).
// Сейчас использует AES-256-GCM как fallback.
//
// Целевые алгоритмы (P2-RU.1):
//   - ГОСТ 28147-89 / Магма / Кузнечик — encryption
//   - Стрибог-256 — hash
//   - ГОСТ Р 34.10-2012 — signatures
//   - КриптоПро HSM — hardware integration
//
// Compliance:
//   - ГОСТ 28147-89 (Магма), ГОСТ Р 34.12-2015 (Кузнечик)
//   - ГОСТ Р 34.11-2012 (Стрибог)
//   - ГОСТ Р 34.10-2012 (Криптография на эллиптических кривых)
//   - ФСТЭК — Сертификация средств КИИ
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"gb-telemetry-collector/internal/stb"
)

// ────────────────────────────────────────────────────────────────────────────
// Ensure interface compliance
// ────────────────────────────────────────────────────────────────────────────

var _ stb.CryptoProvider = (*GOSTCrypto)(nil)

// ────────────────────────────────────────────────────────────────────────────
// GOSTCrypto
// ────────────────────────────────────────────────────────────────────────────

// GOSTCrypto implements CryptoProvider using GOST algorithms (stub).
//
// ⚠ STUB: Реализация через AES-256-GCM до P2-RU.1.
//
// Целевая реализация (P2-RU.1):
//   - Encrypt/Decrypt: Кузнечик (ГОСТ Р 34.12-2015) или Магма (ГОСТ 28147-89)
//   - Hash: Стрибог-256 (ГОСТ Р 34.11-2012)
//   - Sign/Verify: ГОСТ Р 34.10-2012
type GOSTCrypto struct {
	status   string
	fallback *AESCrypto
}

// NewGOSTCrypto создаёт GOST провайдер (stub).
func NewGOSTCrypto() *GOSTCrypto {
	return &GOSTCrypto{
		status:   "stub",
		fallback: NewAESCrypto(),
	}
}

func (g *GOSTCrypto) Hash(data []byte) ([]byte, error) {
	return g.fallback.Hash(data)
}

func (g *GOSTCrypto) HashHex(data []byte) (string, error) {
	return g.fallback.HashHex(data)
}

func (g *GOSTCrypto) HMAC(key, data []byte) ([]byte, error) {
	return g.fallback.HMAC(key, data)
}

func (g *GOSTCrypto) HMACHex(key, data []byte) (string, error) {
	return g.fallback.HMACHex(key, data)
}

func (g *GOSTCrypto) Encrypt(key, plaintext []byte) ([]byte, error) {
	return g.fallback.Encrypt(key, plaintext)
}

func (g *GOSTCrypto) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return g.fallback.Decrypt(key, ciphertext)
}

func (g *GOSTCrypto) Sign(privateKey, data []byte) ([]byte, error) {
	return g.fallback.Sign(privateKey, data)
}

func (g *GOSTCrypto) Verify(publicKey, data, signature []byte) (bool, error) {
	return g.fallback.Verify(publicKey, data, signature)
}

func (g *GOSTCrypto) GenerateKey(length int) ([]byte, error) {
	return g.fallback.GenerateKey(length)
}

// Status возвращает статус реализации.
func (g *GOSTCrypto) Status() string { return g.status }
