// Package providers — GOST Crypto Provider (P2-RU.1).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-RU.1: GOST Crypto Integration
//
// Реализует криптографические алгоритмы РФ (ГОСТ) через интерфейс CryptoProvider.
//
// Текущий статус: STUB через иностранные алгоритмы.
// После получения аппаратного HSM (КриптоПро и т.п.) заменить на нативные вызовы.
//
// Целевые алгоритмы:
//   - ГОСТ 28147-89 (Магма) / ГОСТ Р 34.12-2015 (Кузнечик) — encryption
//   - ГОСТ Р 34.11-2012 (Стрибог-256) — hash
//   - ГОСТ Р 34.10-2012 (на эллиптических кривых) — signatures
//
// STUB-реализация:
//   - Encrypt/Decrypt: AES-256-GCM с маркером GOST формата
//   - Hash: SHA-256 с маркером Стрибог
//   - Sign/Verify: ECDSA P-256 (асимметричная, ближе к ГОСТ Р 34.10-2012 чем HMAC)
//
// Compliance:
//   - ГОСТ 28147-89 (Магма) — Симметричное шифрование
//   - ГОСТ Р 34.11-2012 (Стрибог) — Хеширование
//   - ГОСТ Р 34.10-2012 — Цифровые подписи
//   - 152-ФЗ — Персональные данные РФ
//   - Приказ ФСТЭК № 17 — Защита информации
//   - IEC 62443-3-3 SR 5.1 (Zone-based access)
//   - OWASP ASVS V6 (Cryptographic storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"gb-telemetry-collector/internal/stb"
)

// ────────────────────────────────────────────────────────────────────────────
// Ensure interface compliance
// ────────────────────────────────────────────────────────────────────────────

var _ stb.CryptoProvider = (*GostProvider)(nil)

// ────────────────────────────────────────────────────────────────────────────
// GOST Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// GOSTMagic — маркер GOST формата (первые 4 байта зашифрованных данных).
	// Формат: GST\x01 (GOST version 1)
	GOSTMagic = "GST\x01"

	// GOSTMagicLen — длина маркера в байтах.
	GOSTMagicLen = 4

	// StribogMarker — маркер Стрибог-256 хеша (первый байт).
	// Значение 0x47 = 'G' (GOST).
	StribogMarker = 0x47

	// StribogHashSize — размер выхода хеша Стрибог-256 (32 байта SHA-256 + 1 байт маркер).
	StribogHashSize = 33

	// GOSTKeySize — размер ключа ГОСТ 28147-89 (256 бит = 32 байта).
	GOSTKeySize = 32

	// GostSignatureMarker — маркер для GOST подписи.
	GostSignatureMarker = "GSTSIG1"
)

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	// ErrGOSTInvalidKeySize — неверный размер ключа.
	ErrGOSTInvalidKeySize = errors.New("gost: key must be 32 bytes (256 bit)")

	// ErrGOSTInvalidCiphertext — неверный формат ciphertext.
	ErrGOSTInvalidCiphertext = errors.New("gost: invalid ciphertext format")

	// ErrGOSTInvalidHash — неверный формат хеша.
	ErrGOSTInvalidHash = errors.New("gost: invalid hash format")

	// ErrGOSTHSMNotAvailable — HSM не доступен.
	ErrGOSTHSMNotAvailable = errors.New("gost: HSM not available, using software stub")
)

// ────────────────────────────────────────────────────────────────────────────
// GostProvider
// ────────────────────────────────────────────────────────────────────────────

// GostProvider implements CryptoProvider using GOST algorithms (stub).
//
// ⚠ STUB: Реализация через AES-256-GCM + SHA-256 + ECDSA P-256.
// Все операции помечены GOST-маркерами для детекции формата.
//
// Целевая реализация (после HSM):
//   - Encrypt/Decrypt: Кузнечик (ГОСТ Р 34.12-2015) / Магма (ГОСТ 28147-89)
//   - Hash: Стрибог-256 (ГОСТ Р 34.11-2012)
//   - Sign/Verify: ГОСТ Р 34.10-2012 (кривая)
type GostProvider struct {
	mu       sync.RWMutex
	status   string // "active" | "hsm" | "stub"
	fallback *AESCrypto
	hsmAvail bool // true если КриптоПро или другой HSM доступен
}

// NewGostProvider создаёт новый GOST провайдер.
//
// P2-RU.1: Полная имплементация со stubs через AES/SHA-256/ECDSA.
// При наличии HSM (hsmAvail=true) будет использовать аппаратное ускорение.
func NewGostProvider() *GostProvider {
	return &GostProvider{
		status:   "active", // P2-RU.1: active stub implementation
		fallback: NewAESCrypto(),
		hsmAvail: false, // HSM не обнаружен при старте
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Hash — Стрибог-256 (ГОСТ Р 34.11-2012) stub
// ────────────────────────────────────────────────────────────────────────────

// Hash вычисляет Стрибог-256 хеш (stub через SHA-256 с маркером).
//
// Возвращает 33 байта: [StribogMarker (1) || SHA-256 (32)].
// Маркер позволяет отличить GOST-хеш от обычного SHA-256.
//
// "СТБ COMPLIANCE: Stribog-256 stub через SHA-256.
// Заменить на нативный Стрибог-256 при получении HSM (КриптоПро)."
func (g *GostProvider) Hash(data []byte) ([]byte, error) {
	h := sha256.Sum256(data)
	result := make([]byte, StribogHashSize)
	result[0] = StribogMarker
	copy(result[1:], h[:])
	return result, nil
}

// HashHex возвращает hex-encoded Стрибог-256 хеш.
func (g *GostProvider) HashHex(data []byte) (string, error) {
	hash, err := g.Hash(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// ────────────────────────────────────────────────────────────────────────────
// HMAC — ГОСТ-based HMAC (stub через HMAC-SHA256)
// ────────────────────────────────────────────────────────────────────────────

// HMAC вычисляет HMAC с маркером GOST.
func (g *GostProvider) HMAC(key, data []byte) ([]byte, error) {
	// Используем fallback HMAC-SHA256 для stub
	return g.fallback.HMAC(key, data)
}

// HMACHex возвращает hex-encoded HMAC.
func (g *GostProvider) HMACHex(key, data []byte) (string, error) {
	return g.fallback.HMACHex(key, data)
}

// ────────────────────────────────────────────────────────────────────────────
// Encrypt/Decrypt — ГОСТ 28147-89 (Магма) stub
// ────────────────────────────────────────────────────────────────────────────

// Encrypt шифрует данные по ГОСТ 28147-89 (stub через AES-256-GCM).
//
// Формат выхода: [GOSTMagic (4) || nonce (12) || ciphertext || tag (16)]
// GOSTMagic позволяет отличить GOST-формат от других провайдеров.
//
// "ГОСТ COMPLIANCE: Магма stub через AES-256-GCM.
// Заменить на аппаратное шифрование при наличии HSM (КриптоПро CSP)."
func (g *GostProvider) Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != GOSTKeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrGOSTInvalidKeySize, len(key))
	}

	// Шифруем через AES-256-GCM
	ciphertext, err := g.fallback.Encrypt(key, plaintext)
	if err != nil {
		return nil, fmt.Errorf("gost encrypt: %w", err)
	}

	// Префикс с маркером GOST формата
	result := make([]byte, GOSTMagicLen+len(ciphertext))
	copy(result[:GOSTMagicLen], []byte(GOSTMagic))
	copy(result[GOSTMagicLen:], ciphertext)

	return result, nil
}

// Decrypt расшифровывает данные ГОСТ 28147-89 (stub).
//
// Ожидает формат: [GOSTMagic (4) || nonce (12) || ciphertext || tag (16)]
func (g *GostProvider) Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != GOSTKeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrGOSTInvalidKeySize, len(key))
	}

	if len(ciphertext) < GOSTMagicLen+1 {
		return nil, fmt.Errorf("%w: too short (%d bytes)", ErrGOSTInvalidCiphertext, len(ciphertext))
	}

	// Проверяем маркер GOST
	if !bytes.Equal(ciphertext[:GOSTMagicLen], []byte(GOSTMagic)) {
		return nil, fmt.Errorf("%w: missing GOST magic", ErrGOSTInvalidCiphertext)
	}

	// Извлекаем ciphertext без маркера
	inner := ciphertext[GOSTMagicLen:]

	// Дешифруем через AES-256-GCM
	plaintext, err := g.fallback.Decrypt(key, inner)
	if err != nil {
		return nil, fmt.Errorf("gost decrypt: %w", err)
	}

	return plaintext, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Sign/Verify — ГОСТ Р 34.10-2012 stub
// ────────────────────────────────────────────────────────────────────────────

// Sign подписывает данные по ГОСТ Р 34.10-2012 (stub через ECDSA P-256).
//
// Асимметричная схема (ECDSA P-256) ближе к целевому ГОСТ Р 34.10-2012,
// чем HMAC, который использовался в предыдущей stub-реализации.
//
// Формат выхода: [GostSignatureMarker (7) || ASN.1 DER signature]
//
// "ГОСТ COMPLIANCE: Подпись stub через ECDSA P-256.
// Заменить на ГОСТ Р 34.10-2012 при наличии HSM."
func (g *GostProvider) Sign(privateKey, data []byte) ([]byte, error) {
	if len(privateKey) == 0 {
		return nil, errors.New("gost sign: private key is empty")
	}

	// Хешируем данные (используем SHA-256 как stub для Стрибог-256)
	hash := sha256.Sum256(data)

	// Детерминированное восстановление ECDSA ключа из seed
	priv, err := g.ecdsaKeyFromSeed(privateKey)
	if err != nil {
		return nil, fmt.Errorf("gost sign: key derivation: %w", err)
	}

	// Подписываем через ECDSA P-256 (stub для ГОСТ Р 34.10-2012)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		return nil, fmt.Errorf("gost sign: ecdsa: %w", err)
	}

	// Префикс с маркером GOST подписи
	result := make([]byte, len(GostSignatureMarker)+len(sig))
	copy(result[:len(GostSignatureMarker)], []byte(GostSignatureMarker))
	copy(result[len(GostSignatureMarker):], sig)

	return result, nil
}

// Verify проверяет подпись ГОСТ Р 34.10-2012 (stub через ECDSA P-256).
//
// Ожидает формат: [GostSignatureMarker (7) || ASN.1 DER signature]
func (g *GostProvider) Verify(publicKey, data, signature []byte) (bool, error) {
	if len(publicKey) == 0 {
		return false, errors.New("gost verify: public key is empty")
	}
	if len(signature) < len(GostSignatureMarker)+1 {
		return false, fmt.Errorf("gost verify: signature too short (%d bytes)", len(signature))
	}

	// Проверяем маркер
	if !bytes.Equal(signature[:len(GostSignatureMarker)], []byte(GostSignatureMarker)) {
		return false, fmt.Errorf("gost verify: missing signature marker")
	}

	// Извлекаем ECDSA подпись
	rawSig := signature[len(GostSignatureMarker):]

	// Хешируем данные
	hash := sha256.Sum256(data)

	// Восстанавливаем публичный ключ ECDSA из seed
	pub, err := g.ecdsaPublicKeyFromSeed(publicKey)
	if err != nil {
		return false, fmt.Errorf("gost verify: key reconstruction: %w", err)
	}

	// Верифицируем через ECDSA P-256
	valid := ecdsa.VerifyASN1(pub, hash[:], rawSig)
	return valid, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Key generation
// ────────────────────────────────────────────────────────────────────────────

// GenerateKey генерирует криптостойкий 32-байтовый ключ для ГОСТ 28147-89.
func (g *GostProvider) GenerateKey(length int) ([]byte, error) {
	if length < GOSTKeySize {
		length = GOSTKeySize
	}
	return g.fallback.GenerateKey(length)
}

// ────────────────────────────────────────────────────────────────────────────
// HSM availability
// ────────────────────────────────────────────────────────────────────────────

// IsAvailable проверяет доступность аппаратного HSM или КриптоПро.
//
// В текущей stub-реализации всегда возвращает false.
// При интеграции с КриптоПро CSP необходимо:
//  1. Загрузить библиотеку КриптоПро (libcryptcp.so / cpapi.dll)
//  2. Вызвать CryptAcquireContext с PROV_GOST_2012_256
//  3. Проверить доступность алгоритмов через CryptGetProvParam
//  4. Установить g.hsmAvail = true при успешном коннекте
//
// Returns:
//   - false — HSM не доступен, используется software stub
//   - true — HSM доступен, аппаратное ускорение активно
func (g *GostProvider) IsAvailable() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.hsmAvail
}

// SetHSMStatus устанавливает статус доступности HSM.
// Используется для runtime-детекции КриптоПро или другого HSM.
func (g *GostProvider) SetHSMStatus(available bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.hsmAvail = available
	if available {
		g.status = "hsm"
	} else {
		g.status = "active"
	}
}

// Status возвращает статус реализации провайдера.
//
// Возможные значения:
//   - "stub" — только fallback (устаревший режим)
//   - "active" — P2-RU.1 полная stub-имплементация через AES/SHA-256/ECDSA
//   - "hsm" — аппаратное ускорение через HSM/КриптоПро
func (g *GostProvider) Status() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.status
}

// ────────────────────────────────────────────────────────────────────────────
// Compliance helpers
// ────────────────────────────────────────────────────────────────────────────

// ComplianceProfile возвращает название compliance профиля.
// Для GostProvider всегда "RU" (ГОСТ).
func (g *GostProvider) ComplianceProfile() string {
	return "RU"
}

// AlgorithmInfo возвращает информацию о текущем алгоритме.
type gostAlgorithmInfo struct {
	Encryption    string `json:"encryption"`
	Hash          string `json:"hash"`
	Signature     string `json:"signature"`
	HSMStatus     string `json:"hsm_status"`
	ComplianceStd string `json:"compliance_standard"`
	Status        string `json:"status"`
}

// Info возвращает детальную информацию о провайдере.
func (g *GostProvider) Info() gostAlgorithmInfo {
	g.mu.RLock()
	hsm := "software-stub"
	if g.hsmAvail {
		hsm = "hardware-hsm"
	}
	g.mu.RUnlock()

	return gostAlgorithmInfo{
		Encryption:    "ГОСТ 28147-89 (Магма) / AES-256-GCM stub",
		Hash:          "ГОСТ Р 34.11-2012 (Стрибог-256) / SHA-256 stub",
		Signature:     "ГОСТ Р 34.10-2012 / ECDSA P-256 stub",
		HSMStatus:     hsm,
		ComplianceStd: "ГОСТ Р 34.12-2015, ГОСТ Р 34.11-2012, ГОСТ Р 34.10-2012",
		Status:        g.status,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Internal: ECDSA key derivation (stub для ГОСТ Р 34.10-2012)
// ────────────────────────────────────────────────────────────────────────────

// ecdsaKeyFromSeed детерминированно восстанавливает ECDSA P-256 ключ из seed.
//
// ⚠ STUB: Используется только для имитации асимметричной криптографии.
// Не является криптографически безопасным key derivation.
//
// Цель: bign-curve256v1 (СТБ 34.101.45) или ГОСТ Р 34.10-2012 кривая.
func (g *GostProvider) ecdsaKeyFromSeed(seed []byte) (*ecdsa.PrivateKey, error) {
	// SHA-256 хеш seed как детерминированный скаляр
	h := sha256.Sum256(seed)
	d := new(big.Int).SetBytes(h[:])

	curve := elliptic.P256()

	// Проверяем, что скаляр в допустимом диапазоне
	order := curve.Params().N
	if d.Cmp(order) >= 0 {
		d.Mod(d, order) // reduce mod order
	}

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = curve
	priv.D = d

	// Вычисляем публичный ключ: Q = d * G
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())

	return priv, nil
}

// ecdsaPublicKeyFromSeed восстанавливает ECDSA P-256 публичный ключ из seed.
func (g *GostProvider) ecdsaPublicKeyFromSeed(seed []byte) (*ecdsa.PublicKey, error) {
	priv, err := g.ecdsaKeyFromSeed(seed)
	if err != nil {
		return nil, err
	}
	return &priv.PublicKey, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Backward compatibility: GOSTCrypto
// ────────────────────────────────────────────────────────────────────────────

// GOSTCrypto — тип для обратной совместимости.
// Deprecated: Используйте GostProvider.
type GOSTCrypto = GostProvider

// NewGOSTCrypto создаёт GOST провайдер (backward compatibility).
// Deprecated: Используйте NewGostProvider.
//
// P2-RU.1: Теперь возвращает полноценный GostProvider, а не пустой stub.
func NewGOSTCrypto() *GOSTCrypto {
	return NewGostProvider()
}
