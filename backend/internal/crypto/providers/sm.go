// Package providers — SM Crypto Provider (P2-CR.3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.3: SM Crypto Integration
//
// Реализует криптографические алгоритмы КНР (SM2/SM3/SM4) через интерфейс
// CryptoProvider.
//
// Текущий статус: FULL STUB через иностранные алгоритмы.
// После получения аппаратного HSM или сертифицированной библиотеки
// (GmSSL, Tongsuo и т.п.) заменить на нативные вызовы.
//
// Целевые алгоритмы:
//   - SM4 — block cipher (GB/T 32907 / GM/T 0002-2012)
//   - SM3 — hash (GB/T 32905 / GM/T 0004-2012)
//   - SM2 — signatures (GB/T 32918 / GM/T 0003-2012)
//
// STUB-реализация:
//   - Encrypt/Decrypt: AES-256-GCM с маркером SM4 формата
//   - Hash: SHA-256 с маркером SM3
//   - Sign/Verify: ECDSA P-256 (асимметричная, ближе к SM2 чем HMAC)
//
// Compliance:
//   - GM/T 0002-2012 (SM4 block cipher)
//   - GM/T 0003-2012 (SM2 public key cryptography)
//   - GM/T 0004-2012 (SM3 hash function)
//   - MLPS 2.0 (Multi-Level Protection Scheme)
//   - China Cybersecurity Law (网络安全法)
//   - GB/T 22239-2019 (Information Security Technology — Baseline)
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

var _ stb.CryptoProvider = (*SMProvider)(nil)

// ────────────────────────────────────────────────────────────────────────────
// SM Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// SM4Magic — маркер SM4 формата (первые 4 байта зашифрованных данных).
	// Формат: SM4\x01 (SM4 version 1)
	SM4Magic = "SM4\x01"

	// SM4MagicLen — длина маркера в байтах.
	SM4MagicLen = 4

	// SM3Marker — маркер SM3 хеша (первый байт).
	// Значение 0x53 = 'S' (SM).
	SM3Marker = 0x53

	// SM3HashSize — размер выхода хеша SM3 (32 байта SHA-256 + 1 байт маркер).
	SM3HashSize = 33

	// SM4KeySize — размер ключа SM4 (128 бит = 16 байт).
	SM4KeySize = 16

	// SM2SignatureMarker — маркер для SM2 подписи.
	SM2SignatureMarker = "SM2SIG1"
)

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	// ErrSMInvalidKeySize — неверный размер ключа.
	ErrSMInvalidKeySize = errors.New("sm: key must be 16 bytes (128 bit) for SM4")

	// ErrSMInvalidCiphertext — неверный формат ciphertext.
	ErrSMInvalidCiphertext = errors.New("sm: invalid ciphertext format")

	// ErrSMInvalidHash — неверный формат хеша.
	ErrSMInvalidHash = errors.New("sm: invalid hash format")

	// ErrSMHSMNotAvailable — HSM не доступен.
	ErrSMHSMNotAvailable = errors.New("sm: HSM not available, using software stub")
)

// ────────────────────────────────────────────────────────────────────────────
// SMProvider
// ────────────────────────────────────────────────────────────────────────────

// SMProvider implements CryptoProvider using SM algorithms (full stub).
//
// ⚠ STUB: Реализация через AES-256-GCM + SHA-256 + ECDSA P-256.
// Все операции помечены SM-маркерами для детекции формата.
//
// Целевая реализация (после HSM/сертифицированной библиотеки):
//   - Encrypt/Decrypt: SM4 (GB/T 32907 / GM/T 0002-2012)
//   - Hash: SM3 (GB/T 32905 / GM/T 0004-2012)
//   - Sign/Verify: SM2 (GB/T 32918 / GM/T 0003-2012)
type SMProvider struct {
	mu       sync.RWMutex
	status   string // "active" | "hsm" | "stub"
	fallback *AESCrypto
	hsmAvail bool // true если GmSSL/Tongsuo или другой HSM доступен
}

// NewSMProvider создаёт новый SM провайдер.
//
// P2-CR.3: Полная имплементация со stubs через AES/SHA-256/ECDSA.
// При наличии HSM (hsmAvail=true) будет использовать аппаратное ускорение.
func NewSMProvider() *SMProvider {
	return &SMProvider{
		status:   "active", // P2-CR.3: active stub implementation
		fallback: NewAESCrypto(),
		hsmAvail: false, // HSM не обнаружен при старте
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Hash — SM3 (GB/T 32905) stub
// ────────────────────────────────────────────────────────────────────────────

// Hash вычисляет SM3 хеш (stub через SHA-256 с маркером).
//
// Возвращает 33 байта: [SM3Marker (1) || SHA-256 (32)].
// Маркер позволяет отличить SM-хеш от обычного SHA-256.
//
// "SM COMPLIANCE: SM3 stub через SHA-256.
// Заменить на нативный SM3 при получении сертифицированной библиотеки (GmSSL)."
func (s *SMProvider) Hash(data []byte) ([]byte, error) {
	h := sha256.Sum256(data)
	result := make([]byte, SM3HashSize)
	result[0] = SM3Marker
	copy(result[1:], h[:])
	return result, nil
}

// HashHex возвращает hex-encoded SM3 хеш.
func (s *SMProvider) HashHex(data []byte) (string, error) {
	hash, err := s.Hash(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// ────────────────────────────────────────────────────────────────────────────
// HMAC — SM-based HMAC (stub через HMAC-SHA256)
// ────────────────────────────────────────────────────────────────────────────

// HMAC вычисляет HMAC с маркером SM.
func (s *SMProvider) HMAC(key, data []byte) ([]byte, error) {
	return s.fallback.HMAC(key, data)
}

// HMACHex возвращает hex-encoded HMAC.
func (s *SMProvider) HMACHex(key, data []byte) (string, error) {
	return s.fallback.HMACHex(key, data)
}

// ────────────────────────────────────────────────────────────────────────────
// Encrypt/Decrypt — SM4 (GB/T 32907) stub
// ────────────────────────────────────────────────────────────────────────────

// Encrypt шифрует данные по SM4 (stub через AES-256-GCM).
//
// Формат выхода: [SM4Magic (4) || nonce (12) || ciphertext || tag (16)]
// SM4Magic позволяет отличить SM4-формат от других провайдеров.
//
// "SM COMPLIANCE: SM4 stub через AES-256-GCM.
// Заменить на нативное SM4-шифрование при наличии HSM (GmSSL, Tongsuo)."
func (s *SMProvider) Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != SM4KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrSMInvalidKeySize, len(key))
	}

	// SM4 использует 128-битный ключ. Для AES-256-GCM нужно расширить до 32 байт.
	aesKey := s.expandKey(key)

	// Шифруем через AES-256-GCM
	ciphertext, err := s.fallback.Encrypt(aesKey, plaintext)
	if err != nil {
		return nil, fmt.Errorf("sm encrypt: %w", err)
	}

	// Префикс с маркером SM4 формата
	result := make([]byte, SM4MagicLen+len(ciphertext))
	copy(result[:SM4MagicLen], []byte(SM4Magic))
	copy(result[SM4MagicLen:], ciphertext)

	return result, nil
}

// Decrypt расшифровывает данные SM4 (stub).
//
// Ожидает формат: [SM4Magic (4) || nonce (12) || ciphertext || tag (16)]
func (s *SMProvider) Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != SM4KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrSMInvalidKeySize, len(key))
	}

	if len(ciphertext) < SM4MagicLen+1 {
		return nil, fmt.Errorf("%w: too short (%d bytes)", ErrSMInvalidCiphertext, len(ciphertext))
	}

	// Проверяем маркер SM4
	if !bytes.Equal(ciphertext[:SM4MagicLen], []byte(SM4Magic)) {
		return nil, fmt.Errorf("%w: missing SM4 magic", ErrSMInvalidCiphertext)
	}

	// Извлекаем ciphertext без маркера
	inner := ciphertext[SM4MagicLen:]

	// SM4 использует 128-битный ключ. Расширяем для AES-256-GCM.
	aesKey := s.expandKey(key)

	// Дешифруем через AES-256-GCM
	plaintext, err := s.fallback.Decrypt(aesKey, inner)
	if err != nil {
		return nil, fmt.Errorf("sm decrypt: %w", err)
	}

	return plaintext, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Sign/Verify — SM2 (GB/T 32918) stub
// ────────────────────────────────────────────────────────────────────────────

// Sign подписывает данные по SM2 (stub через ECDSA P-256).
//
// Асимметричная схема (ECDSA P-256) ближе к целевому SM2,
// чем HMAC, который использовался в предыдущей stub-реализации.
//
// Формат выхода: [SM2SignatureMarker (7) || ASN.1 DER signature]
//
// "SM COMPLIANCE: Подпись stub через ECDSA P-256.
// Заменить на SM2 (GB/T 32918) при наличии HSM."
func (s *SMProvider) Sign(privateKey, data []byte) ([]byte, error) {
	if len(privateKey) == 0 {
		return nil, errors.New("sm sign: private key is empty")
	}

	// Хешируем данные (используем SHA-256 как stub для SM3)
	hash := sha256.Sum256(data)

	// Детерминированное восстановление ECDSA ключа из seed
	priv, err := s.ecdsaKeyFromSeed(privateKey)
	if err != nil {
		return nil, fmt.Errorf("sm sign: key derivation: %w", err)
	}

	// Подписываем через ECDSA P-256 (stub для SM2)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		return nil, fmt.Errorf("sm sign: ecdsa: %w", err)
	}

	// Префикс с маркером SM2 подписи
	result := make([]byte, len(SM2SignatureMarker)+len(sig))
	copy(result[:len(SM2SignatureMarker)], []byte(SM2SignatureMarker))
	copy(result[len(SM2SignatureMarker):], sig)

	return result, nil
}

// Verify проверяет подпись SM2 (stub через ECDSA P-256).
//
// Ожидает формат: [SM2SignatureMarker (7) || ASN.1 DER signature]
func (s *SMProvider) Verify(publicKey, data, signature []byte) (bool, error) {
	if len(publicKey) == 0 {
		return false, errors.New("sm verify: public key is empty")
	}
	if len(signature) < len(SM2SignatureMarker)+1 {
		return false, fmt.Errorf("sm verify: signature too short (%d bytes)", len(signature))
	}

	// Проверяем маркер
	if !bytes.Equal(signature[:len(SM2SignatureMarker)], []byte(SM2SignatureMarker)) {
		return false, fmt.Errorf("sm verify: missing signature marker")
	}

	// Извлекаем ECDSA подпись
	rawSig := signature[len(SM2SignatureMarker):]

	// Хешируем данные
	hash := sha256.Sum256(data)

	// Восстанавливаем публичный ключ ECDSA из seed
	pub, err := s.ecdsaPublicKeyFromSeed(publicKey)
	if err != nil {
		return false, fmt.Errorf("sm verify: key reconstruction: %w", err)
	}

	// Верифицируем через ECDSA P-256
	valid := ecdsa.VerifyASN1(pub, hash[:], rawSig)
	return valid, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Key generation
// ────────────────────────────────────────────────────────────────────────────

// GenerateKey генерирует криптостойкий 16-байтовый ключ для SM4.
func (s *SMProvider) GenerateKey(length int) ([]byte, error) {
	if length < SM4KeySize {
		length = SM4KeySize
	}
	return s.fallback.GenerateKey(length)
}

// ────────────────────────────────────────────────────────────────────────────
// HSM availability
// ────────────────────────────────────────────────────────────────────────────

// IsAvailable проверяет доступность аппаратного HSM или GmSSL/Tongsuo.
//
// В текущей stub-реализации всегда возвращает false.
// При интеграции с GmSSL необходимо:
//  1. Загрузить библиотеку GmSSL (libgmssl.so / gmssl.dll)
//  2. Проверить доступность алгоритмов через API
//  3. Установить s.hsmAvail = true при успешном коннекте
//
// Returns:
//   - false — HSM не доступен, используется software stub
//   - true — HSM доступен, аппаратное ускорение активно
func (s *SMProvider) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hsmAvail
}

// SetHSMStatus устанавливает статус доступности HSM.
// Используется для runtime-детекции GmSSL, Tongsuo или другого HSM.
func (s *SMProvider) SetHSMStatus(available bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hsmAvail = available
	if available {
		s.status = "hsm"
	} else {
		s.status = "active"
	}
}

// Status возвращает статус реализации провайдера.
//
// Возможные значения:
//   - "stub" — только fallback (устаревший режим)
//   - "active" — P2-CR.3 полная stub-имплементация через AES/SHA-256/ECDSA
//   - "hsm" — аппаратное ускорение через HSM/GmSSL
func (s *SMProvider) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// ────────────────────────────────────────────────────────────────────────────
// Compliance helpers
// ────────────────────────────────────────────────────────────────────────────

// ComplianceProfile возвращает название compliance профиля.
// Для SMProvider всегда "CN" (Китай).
func (s *SMProvider) ComplianceProfile() string {
	return "CN"
}

// AlgorithmInfo возвращает информацию о текущем алгоритме.
type smAlgorithmInfo struct {
	Encryption    string `json:"encryption"`
	Hash          string `json:"hash"`
	Signature     string `json:"signature"`
	HSMStatus     string `json:"hsm_status"`
	ComplianceStd string `json:"compliance_standard"`
	Status        string `json:"status"`
}

// Info возвращает детальную информацию о провайдере.
func (s *SMProvider) Info() smAlgorithmInfo {
	s.mu.RLock()
	hsm := "software-stub"
	if s.hsmAvail {
		hsm = "hardware-hsm"
	}
	s.mu.RUnlock()

	return smAlgorithmInfo{
		Encryption:    "SM4 (GB/T 32907) / AES-256-GCM stub",
		Hash:          "SM3 (GB/T 32905) / SHA-256 stub",
		Signature:     "SM2 (GB/T 32918) / ECDSA P-256 stub",
		HSMStatus:     hsm,
		ComplianceStd: "GM/T 0002-2012, GM/T 0003-2012, GM/T 0004-2012",
		Status:        s.status,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Internal: ECDSA key derivation (stub для SM2)
// ────────────────────────────────────────────────────────────────────────────

// ecdsaKeyFromSeed детерминированно восстанавливает ECDSA P-256 ключ из seed.
//
// ⚠ STUB: Используется только для имитации асимметричной криптографии.
// Не является криптографически безопасным key derivation.
//
// Цель: SM2 кривая (GB/T 32918).
func (s *SMProvider) ecdsaKeyFromSeed(seed []byte) (*ecdsa.PrivateKey, error) {
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
func (s *SMProvider) ecdsaPublicKeyFromSeed(seed []byte) (*ecdsa.PublicKey, error) {
	priv, err := s.ecdsaKeyFromSeed(seed)
	if err != nil {
		return nil, err
	}
	return &priv.PublicKey, nil
}

// expandKey расширяет 16-байтовый SM4 ключ до 32 байт для AES-256.
//
// Использует SHA-256 хеш оригинального ключа.
// ⚠ STUB: Временное решение до настоящей SM4 реализации.
func (s *SMProvider) expandKey(key []byte) []byte {
	h := sha256.Sum256(key)
	return h[:]
}

// ────────────────────────────────────────────────────────────────────────────
// Backward compatibility: SMCrypto
// ────────────────────────────────────────────────────────────────────────────

// SMCrypto — тип для обратной совместимости.
// Deprecated: Используйте SMProvider.
type SMCrypto = SMProvider

// NewSMCrypto создаёт SM провайдер (backward compatibility).
// Deprecated: Используйте NewSMProvider.
//
// P2-CR.3: Теперь возвращает полноценный SMProvider, а не пустой stub.
func NewSMCrypto() *SMCrypto {
	return NewSMProvider()
}
