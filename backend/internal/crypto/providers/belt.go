// Package providers — Belt-GCM Crypto Provider (СТБ 34.101.31).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.2: Belt-GCM Provider
//
// Реализует шифрование belt-gcm согласно СТБ 34.101.31.
// Используется для BY региона.
//
// ⚠ STUB: Требует github.com/bp2012/crypto в go.mod.
// Сейчас делегирует в AESCrypto (AES-256-GCM) как временное решение.
//
// План миграции:
//  1. Добавить github.com/bp2012/crypto в go.mod
//  2. Заменить делегацию на belt-gcm вызовы:
//     import "github.com/bp2012/crypto/belt"
//     belt.NewGCM(key) → Seal/Open
//  3. Удалить status="stub"
//
// Compliance:
//   - СТБ 34.101.30 — Криптографические алгоритмы РБ
//   - СТБ 34.101.31 — belt-gcm
//   - Приказ ОАЦ № 66 п. 7.18.3 — Шифрование данных КИИ
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"fmt"

	"gb-telemetry-collector/internal/stb"
)

// ────────────────────────────────────────────────────────────────────────────
// Ensure interface compliance
// ────────────────────────────────────────────────────────────────────────────

var _ stb.CryptoProvider = (*BeltCrypto)(nil)

// ────────────────────────────────────────────────────────────────────────────
// BeltCrypto
// ────────────────────────────────────────────────────────────────────────────

// BeltCrypto implements CryptoProvider using СТБ belt-gcm.
//
// ⚠ STUB: После добавления github.com/bp2012/crypto:
//
//	import "github.com/bp2012/crypto/belt"
//
//	type BeltCrypto struct {
//	    impl *belt.GCM
//	}
//
//	func (b *BeltCrypto) Encrypt(key, plaintext []byte) ([]byte, error) {
//	    gcm, err := belt.NewGCM(key)
//	    if err != nil { return nil, err }
//	    nonce := make([]byte, gcm.NonceSize())
//	    ...
//	    return gcm.Seal(nonce, nonce, plaintext, nil), nil
//	}
type BeltCrypto struct {
	// status указывает на статус имплементации.
	// "stub" — использует AES-256-GCM как fallback.
	// "active" — после интеграции bp2012/crypto.
	status string

	// fallback — временная AES реализация до получения bp2012/crypto.
	fallback *AESCrypto
}

// NewBeltCrypto создаёт belt-GCM провайдер.
// ⚠ Временная реализация: использует AES-256-GCM (НЕ СТБ).
//
// "СТБ COMPLIANCE: BeltCrypto использует AES fallback.
// Заменить на bp2012/crypto/belt при получении SDK."
func NewBeltCrypto() *BeltCrypto {
	return &BeltCrypto{
		status:   "stub",
		fallback: NewAESCrypto(),
	}
}

func (b *BeltCrypto) Hash(data []byte) ([]byte, error) {
	// ⚠ Временно: SHA-256. Цель: bash-256 (СТБ 34.101.77)
	return b.fallback.Hash(data)
}

func (b *BeltCrypto) HashHex(data []byte) (string, error) {
	return b.fallback.HashHex(data)
}

func (b *BeltCrypto) HMAC(key, data []byte) ([]byte, error) {
	// ⚠ Временно: HMAC-SHA256. Цель: bash-256 HMAC (СТБ 34.101.77)
	return b.fallback.HMAC(key, data)
}

func (b *BeltCrypto) HMACHex(key, data []byte) (string, error) {
	return b.fallback.HMACHex(key, data)
}

// Encrypt шифрует данные.
// ⚠ Временно: AES-256-GCM. Цель: belt-gcm (СТБ 34.101.31).
func (b *BeltCrypto) Encrypt(key, plaintext []byte) ([]byte, error) {
	return b.fallback.Encrypt(key, plaintext)
}

// Decrypt расшифровывает данные.
func (b *BeltCrypto) Decrypt(key, ciphertext []byte) ([]byte, error) {
	return b.fallback.Decrypt(key, ciphertext)
}

func (b *BeltCrypto) Sign(privateKey, data []byte) ([]byte, error) {
	return b.fallback.Sign(privateKey, data)
}

func (b *BeltCrypto) Verify(publicKey, data, signature []byte) (bool, error) {
	return b.fallback.Verify(publicKey, data, signature)
}

func (b *BeltCrypto) GenerateKey(length int) ([]byte, error) {
	return b.fallback.GenerateKey(length)
}

// Status возвращает статус реализации belt провайдера.
func (b *BeltCrypto) Status() string {
	return b.status
}

// ────────────────────────────────────────────────────────────────────────────
// Belt-specific key derivation (СТБ belt-kdf)
// ────────────────────────────────────────────────────────────────────────────

// BeltKDF — заглушка для belt-kdf (СТБ 34.101.31).
//
// После миграции на bp2012/crypto:
//
//	func BeltKDF(password, salt []byte, keyLen int) ([]byte, error) {
//	    return belt.KDF(password, salt, keyLen)
//	}
func BeltKDF(password, salt []byte, keyLen int) ([]byte, error) {
	// ⚠ Временно: SHA-256 based KDF. Цель: belt-kdf.
	if keyLen < 16 || keyLen > 64 {
		return nil, fmt.Errorf("belt-kdf: key length must be 16-64 bytes, got %d", keyLen)
	}

	h, err := stb.DefaultCrypto.Hash(append(password, salt...))
	if err != nil {
		return nil, fmt.Errorf("belt-kdf: %w", err)
	}

	if len(h) < keyLen {
		return nil, fmt.Errorf("belt-kdf: derived key too short")
	}

	return h[:keyLen], nil
}
