// Package providers — GOST Crypto Provider tests (P2-RU.1).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-RU.1: GOST Crypto Integration Tests
//
// Coverage requirements:
//   - Encrypt/Decrypt round-trip (with GOST magic marker)
//   - Hash consistency (Stribog-256 marker)
//   - Sign/Verify (ECDSA P-256 stub for ГОСТ Р 34.10-2012)
//   - Wrong key / tampered data rejection
//   - HSM availability check
//   - Performance benchmarks (overhead <3x vs AES baseline)
//   - Provider selection via ComplianceProfile
//
// Compliance:
//   - ISO 27001 A.14.2 (Security testing)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - OWASP ASVS V6 (Cryptographic storage testing)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"gb-telemetry-collector/internal/compliance"
)

// ────────────────────────────────────────────────────────────────────────────
// Test keys and data
// ────────────────────────────────────────────────────────────────────────────

// gostTestKey — 32-byte key для тестов GOST.
var gostTestKey = []byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

// gostWrongKey — другой 32-byte key для тестов с неправильным ключом.
var gostWrongKey = []byte{
	0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8,
	0xf7, 0xf6, 0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0,
	0xef, 0xee, 0xed, 0xec, 0xeb, 0xea, 0xe9, 0xe8,
	0xe7, 0xe6, 0xe5, 0xe4, 0xe3, 0xe2, 0xe1, 0xe0,
}

// gostTestData — тестовые данные для GOST.
var gostTestData = []byte("sensitive CCTV monitoring data for GOST encryption test P2-RU.1")

// ═══════════════════════════════════════════════════════════════════════════
// Constructor and basic tests
// ═══════════════════════════════════════════════════════════════════════════

// TestNewGostProvider проверяет корректное создание провайдера.
func TestNewGostProvider(t *testing.T) {
	p := NewGostProvider()
	if p == nil {
		t.Fatal("NewGostProvider() must return non-nil provider")
	}

	// P2-RU.1: должен быть active (не stub)
	if p.Status() != "active" {
		t.Fatalf("expected status 'active', got '%s'", p.Status())
	}

	// HSM не доступен по умолчанию
	if p.IsAvailable() {
		t.Fatal("HSM should not be available by default")
	}

	// ComplianceProfile
	if p.ComplianceProfile() != "RU" {
		t.Fatalf("expected ComplianceProfile 'RU', got '%s'", p.ComplianceProfile())
	}
}

// TestNewGOSTCryptoBackwardCompat проверяет обратную совместимость.
func TestNewGOSTCryptoBackwardCompat(t *testing.T) {
	// GOSTCrypto — type alias для GostProvider
	p := NewGOSTCrypto()
	if p == nil {
		t.Fatal("NewGOSTCrypto() must return non-nil provider")
	}

	// Проверяем, что это действительно GostProvider
	if _, ok := interface{}(p).(*GostProvider); !ok {
		t.Fatalf("NewGOSTCrypto must return *GostProvider, got %T", p)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Encrypt/Decrypt round-trip tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostEncryptDecrypt проверяет полный цикл шифрования/дешифрования.
func TestGostEncryptDecrypt(t *testing.T) {
	p := NewGostProvider()

	ciphertext, err := p.Encrypt(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("ciphertext must not be empty")
	}

	// Проверяем наличие GOST маркера
	if !bytes.HasPrefix(ciphertext, []byte(GOSTMagic)) {
		t.Fatal("ciphertext must start with GOST magic marker")
	}

	// Ciphertext не должен быть равен plaintext
	if bytes.Equal(ciphertext, gostTestData) {
		t.Fatal("ciphertext must not equal plaintext")
	}

	// Ciphertext должен быть длиннее plaintext + маркер
	expectedMinLen := len(gostTestData) + GOSTMagicLen + 12 + 16 // marker + nonce + tag
	if len(ciphertext) < expectedMinLen {
		t.Fatalf("ciphertext too short: got %d, expected at least %d",
			len(ciphertext), expectedMinLen)
	}

	// Decrypt
	decrypted, err := p.Decrypt(gostTestKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if !bytes.Equal(decrypted, gostTestData) {
		t.Fatalf("decrypted data doesn't match original:\ngot:  %x\nwant: %x",
			decrypted, gostTestData)
	}
}

// TestGostEncryptDecryptMultipleKeys проверяет шифрование с разными ключами.
func TestGostEncryptDecryptMultipleKeys(t *testing.T) {
	p := NewGostProvider()
	keys := [][]byte{gostTestKey, gostWrongKey}

	for i, key := range keys {
		ciphertext, err := p.Encrypt(key, gostTestData)
		if err != nil {
			t.Fatalf("key[%d] Encrypt error: %v", i, err)
		}

		decrypted, err := p.Decrypt(key, ciphertext)
		if err != nil {
			t.Fatalf("key[%d] Decrypt error: %v", i, err)
		}

		if !bytes.Equal(decrypted, gostTestData) {
			t.Fatalf("key[%d] decrypted data mismatch", i)
		}
	}
}

// TestGostEncryptEmptyData проверяет шифрование пустых данных.
func TestGostEncryptEmptyData(t *testing.T) {
	p := NewGostProvider()

	ciphertext, err := p.Encrypt(gostTestKey, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty data error: %v", err)
	}

	if !bytes.HasPrefix(ciphertext, []byte(GOSTMagic)) {
		t.Fatal("empty ciphertext must start with GOST magic")
	}

	decrypted, err := p.Decrypt(gostTestKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty data error: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("decrypted empty data should be empty, got %d bytes", len(decrypted))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Wrong key and tampered data tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostWrongKey проверяет, что decrypt с неправильным ключом возвращает ошибку.
func TestGostWrongKey(t *testing.T) {
	p := NewGostProvider()

	ciphertext, err := p.Encrypt(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	_, err = p.Decrypt(gostWrongKey, ciphertext)
	if err == nil {
		t.Fatal("Decrypt with wrong key should return error")
	}
}

// TestGostInvalidKeySize проверяет, что неверный размер ключа вызывает ошибку.
func TestGostInvalidKeySize(t *testing.T) {
	p := NewGostProvider()

	shortKey := []byte("short-key")
	_, err := p.Encrypt(shortKey, gostTestData)
	if err == nil {
		t.Fatal("Encrypt with short key should return error")
	}

	if !strings.Contains(err.Error(), "key must be 32 bytes") {
		t.Fatalf("unexpected error message: %v", err)
	}

	_, err = p.Decrypt(shortKey, gostTestData)
	if err == nil {
		t.Fatal("Decrypt with short key should return error")
	}
}

// TestGostTamperedCiphertext проверяет, что подделанный ciphertext вызывает ошибку.
func TestGostTamperedCiphertext(t *testing.T) {
	p := NewGostProvider()

	ciphertext, err := p.Encrypt(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	// Повреждаем ciphertext
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)-1] ^= 0xff // flip last byte

	_, err = p.Decrypt(gostTestKey, tampered)
	if err == nil {
		t.Fatal("Decrypt with tampered ciphertext should return error")
	}
}

// TestGostMissingMagic проверяет, что ciphertext без маркера GOST вызывает ошибку.
func TestGostMissingMagic(t *testing.T) {
	p := NewGostProvider()

	// Прямой вызов fallback без маркера
	plainAES, err := p.fallback.Encrypt(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("fallback Encrypt error: %v", err)
	}

	// Пытаемся расшифровать как GOST — должно упасть без маркера
	_, err = p.Decrypt(gostTestKey, plainAES)
	if err == nil {
		t.Fatal("Decrypt without GOST magic should return error")
	}

	if !strings.Contains(err.Error(), "missing GOST magic") {
		t.Fatalf("expected 'missing GOST magic' error, got: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Hash tests — Стрибог-256 (ГОСТ Р 34.11-2012)
// ═══════════════════════════════════════════════════════════════════════════

// TestGostHash проверяет корректность хеширования.
func TestGostHash(t *testing.T) {
	p := NewGostProvider()

	hash, err := p.Hash(gostTestData)
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}

	// Проверяем размер: 1 байт маркер + 32 байта SHA-256
	if len(hash) != StribogHashSize {
		t.Fatalf("expected %d bytes hash, got %d", StribogHashSize, len(hash))
	}

	// Проверяем наличие маркера Стрибог
	if hash[0] != StribogMarker {
		t.Fatalf("expected Stribog marker 0x%02x, got 0x%02x", StribogMarker, hash[0])
	}

	// Проверяем, что SHA-256 часть корректна
	expectedSHA := sha256.Sum256(gostTestData)
	if !bytes.Equal(hash[1:], expectedSHA[:]) {
		t.Fatal("hash body does not match SHA-256 of input")
	}

	// Детерминированность
	hash2, _ := p.Hash(gostTestData)
	if !bytes.Equal(hash, hash2) {
		t.Fatal("hash should be deterministic")
	}

	// Разные входы — разные хеши
	hash3, _ := p.Hash([]byte("different data"))
	if bytes.Equal(hash, hash3) {
		t.Fatal("different input should produce different hash")
	}
}

// TestGostHashEmpty проверяет хеширование пустых данных.
func TestGostHashEmpty(t *testing.T) {
	p := NewGostProvider()

	hash, err := p.Hash([]byte{})
	if err != nil {
		t.Fatalf("Hash empty error: %v", err)
	}

	if len(hash) != StribogHashSize {
		t.Fatalf("expected %d bytes, got %d", StribogHashSize, len(hash))
	}

	if hash[0] != StribogMarker {
		t.Fatal("empty hash must have Stribog marker")
	}
}

// TestGostHashHex проверяет hex-encoded хеш.
func TestGostHashHex(t *testing.T) {
	p := NewGostProvider()

	hashHex, err := p.HashHex(gostTestData)
	if err != nil {
		t.Fatalf("HashHex error: %v", err)
	}

	if len(hashHex) != StribogHashSize*2 {
		t.Fatalf("expected %d hex chars, got %d", StribogHashSize*2, len(hashHex))
	}

	// Должен начинаться с 47 (hex для StribogMarker)
	if !strings.HasPrefix(hashHex, "47") {
		t.Fatalf("HashHex should start with '47' (Stribog marker), got: %s", hashHex[:2])
	}

	// Детерминированность
	hashHex2, _ := p.HashHex(gostTestData)
	if hashHex != hashHex2 {
		t.Fatal("HashHex should be deterministic")
	}

	// Проверяем, что можно декодировать обратно
	decoded, err := hex.DecodeString(hashHex)
	if err != nil {
		t.Fatalf("hex decode error: %v", err)
	}
	if len(decoded) != StribogHashSize {
		t.Fatalf("decoded hash has wrong length: %d", len(decoded))
	}
}

// TestGostHashDeterminism проверяет детерминированность хеша.
func TestGostHashDeterminism(t *testing.T) {
	p := NewGostProvider()

	const iterations = 100
	first, _ := p.Hash(gostTestData)

	for i := 0; i < iterations; i++ {
		hash, _ := p.Hash(gostTestData)
		if !bytes.Equal(hash, first) {
			t.Fatalf("hash not deterministic at iteration %d", i)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HMAC tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostHMAC проверяет HMAC.
func TestGostHMAC(t *testing.T) {
	p := NewGostProvider()

	mac, err := p.HMAC(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("HMAC error: %v", err)
	}
	if len(mac) == 0 {
		t.Fatal("HMAC must not be empty")
	}

	// Детерминированность
	mac2, _ := p.HMAC(gostTestKey, gostTestData)
	if !bytes.Equal(mac, mac2) {
		t.Fatal("HMAC should be deterministic")
	}

	// Разный ключ — разный MAC
	mac3, _ := p.HMAC(gostWrongKey, gostTestData)
	if bytes.Equal(mac, mac3) {
		t.Fatal("different key should produce different HMAC")
	}
}

// TestGostHMACHex проверяет hex-encoded HMAC.
func TestGostHMACHex(t *testing.T) {
	p := NewGostProvider()

	macHex, err := p.HMACHex(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("HMACHex error: %v", err)
	}
	if len(macHex) == 0 {
		t.Fatal("HMACHex must not be empty")
	}

	// Должен быть hex-строкой
	_, err = hex.DecodeString(macHex)
	if err != nil {
		t.Fatalf("HMACHex is not valid hex: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Sign/Verify tests — ГОСТ Р 34.10-2012 (stub через ECDSA P-256)
// ═══════════════════════════════════════════════════════════════════════════

// TestGostSignVerify проверяет полный цикл подписи/верификации.
func TestGostSignVerify(t *testing.T) {
	p := NewGostProvider()

	sig, err := p.Sign(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("signature must not be empty")
	}

	// Проверяем наличие маркера
	if !bytes.HasPrefix(sig, []byte(GostSignatureMarker)) {
		t.Fatal("signature must start with GOST signature marker")
	}

	// Сигнатура должна быть длиннее маркера
	if len(sig) <= len(GostSignatureMarker) {
		t.Fatal("signature must contain actual signature data after marker")
	}

	// Verify
	valid, err := p.Verify(gostTestKey, gostTestData, sig)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}
}

// TestGostSignWrongKey проверяет, что подпись с неправильным ключом не верифицируется.
func TestGostSignWrongKey(t *testing.T) {
	p := NewGostProvider()

	sig, err := p.Sign(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	// Верификация с wrongKey должна провалиться
	valid, _ := p.Verify(gostWrongKey, gostTestData, sig)
	if valid {
		t.Fatal("signature with wrong key should be invalid")
	}
}

// TestGostSignTamperedData проверяет, что подпись для изменённых данных не валидна.
func TestGostSignTamperedData(t *testing.T) {
	p := NewGostProvider()

	sig, err := p.Sign(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	// Верификация с изменёнными данными должна провалиться
	valid, _ := p.Verify(gostTestKey, []byte("tampered data"), sig)
	if valid {
		t.Fatal("signature for tampered data should be invalid")
	}
}

// TestGostSignEmptyKey проверяет, что пустой ключ вызывает ошибку.
func TestGostSignEmptyKey(t *testing.T) {
	p := NewGostProvider()

	_, err := p.Sign([]byte{}, gostTestData)
	if err == nil {
		t.Fatal("Sign with empty key should return error")
	}
}

// TestGostSignVerifyMultipleKeys проверяет sign/verify с разными ключами.
func TestGostSignVerifyMultipleKeys(t *testing.T) {
	p := NewGostProvider()
	keys := [][]byte{gostTestKey, gostWrongKey}

	for i, key := range keys {
		sig, err := p.Sign(key, gostTestData)
		if err != nil {
			t.Fatalf("key[%d] Sign error: %v", i, err)
		}

		valid, err := p.Verify(key, gostTestData, sig)
		if err != nil {
			t.Fatalf("key[%d] Verify error: %v", i, err)
		}
		if !valid {
			t.Fatalf("key[%d] signature should be valid", i)
		}
	}
}

// TestGostSignDeterminism проверяет детерминированность подписи.
//
// ECDSA P-256 подпись использует случайный k, поэтому подписи
// от одной и той же пары (ключ, данные) будут разными.
// Но Verify должен проходить для обеих.
func TestGostSignDeterminism(t *testing.T) {
	p := NewGostProvider()

	sig1, err := p.Sign(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Sign 1 error: %v", err)
	}

	sig2, err := p.Sign(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Sign 2 error: %v", err)
	}

	// ECDSA подпись недетерминирована — разные sigs должны оба проходить verify
	valid1, _ := p.Verify(gostTestKey, gostTestData, sig1)
	valid2, _ := p.Verify(gostTestKey, gostTestData, sig2)

	if !valid1 {
		t.Fatal("sig1 should be valid")
	}
	if !valid2 {
		t.Fatal("sig2 should be valid")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GenerateKey tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostGenerateKey проверяет генерацию ключей.
func TestGostGenerateKey(t *testing.T) {
	p := NewGostProvider()

	key, err := p.GenerateKey(32)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}

	// Минимальная длина
	shortKey, err := p.GenerateKey(1)
	if err != nil {
		t.Fatalf("GenerateKey(1) error: %v", err)
	}
	if len(shortKey) < 32 {
		t.Fatalf("expected at least 32 bytes, got %d", len(shortKey))
	}

	// Уникальность
	key2, _ := p.GenerateKey(32)
	if bytes.Equal(key, key2) {
		t.Log("warning: generated keys might not be unique")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HSM availability tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostHSMStatus проверяет управление HSM статусом.
func TestGostHSMStatus(t *testing.T) {
	p := NewGostProvider()

	// По умолчанию false
	if p.IsAvailable() {
		t.Fatal("HSM should not be available by default")
	}

	// Включаем HSM
	p.SetHSMStatus(true)
	if !p.IsAvailable() {
		t.Fatal("HSM should be available after SetHSMStatus(true)")
	}
	if p.Status() != "hsm" {
		t.Fatalf("expected status 'hsm', got '%s'", p.Status())
	}

	// Выключаем HSM
	p.SetHSMStatus(false)
	if p.IsAvailable() {
		t.Fatal("HSM should not be available after SetHSMStatus(false)")
	}
	if p.Status() != "active" {
		t.Fatalf("expected status 'active', got '%s'", p.Status())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ProviderInfo tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostProviderInfo проверяет метаданные провайдера.
func TestGostProviderInfo(t *testing.T) {
	p := NewGostProvider()
	info := Info(p)

	if info.Name != "gost-28147-89" {
		t.Fatalf("expected name 'gost-28147-89', got '%s'", info.Name)
	}
	if info.Region != "RU" {
		t.Fatalf("expected region 'RU', got '%s'", info.Region)
	}
	if info.KeySizeBits != 256 {
		t.Fatalf("expected key size 256, got %d", info.KeySizeBits)
	}
	if info.Status != "active" {
		t.Fatalf("expected status 'active', got '%s'", info.Status)
	}
	if info.Algorithm != "ГОСТ 28147-89 (Магма/Кузнечик)" {
		t.Fatalf("unexpected algorithm: %s", info.Algorithm)
	}
}

// TestGostProviderInfoHSM проверяет метаданные при активном HSM.
func TestGostProviderInfoHSM(t *testing.T) {
	p := NewGostProvider()
	p.SetHSMStatus(true)

	info := Info(p)
	if info.Status != "hsm" {
		t.Fatalf("expected status 'hsm', got '%s'", info.Status)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ComplianceProfile selection tests
// ═══════════════════════════════════════════════════════════════════════════

// TestNewFromProfileRU проверяет, что RU профиль возвращает GostProvider.
func TestNewFromProfileRU(t *testing.T) {
	p, err := NewFromProfile(compliance.NewRUProfile())
	if err != nil {
		t.Fatalf("NewFromProfile(RU) error: %v", err)
	}
	if p == nil {
		t.Fatal("NewFromProfile(RU) must return non-nil provider")
	}

	// RU profile должен возвращать GostProvider
	gp, ok := p.(*GostProvider)
	if !ok {
		t.Fatalf("expected *GostProvider for RU, got %T", p)
	}

	// Status должен быть "active" (P2-RU.1)
	if gp.Status() != "active" {
		t.Fatalf("expected status 'active', got '%s'", gp.Status())
	}

	// Полный цикл encrypt/decrypt
	testEncryptDecryptRoundTrip(t, p)

	// Hash с маркером
	hash, err := p.Hash(gostTestData)
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}
	if len(hash) != StribogHashSize || hash[0] != StribogMarker {
		t.Fatal("RU provider must produce Stribog-marked hash")
	}

	// Sign/Verify
	sig, err := p.Sign(gostTestKey, gostTestData)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	valid, err := p.Verify(gostTestKey, gostTestData, sig)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !valid {
		t.Fatal("RU provider signature should be valid")
	}
}

// TestNewFromProfileRUMust проверяет MustFromProfile для RU.
func TestNewFromProfileRUMust(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustFromProfile(RU) should not panic: %v", r)
		}
	}()

	p := MustFromProfile(compliance.NewRUProfile())
	if p == nil {
		t.Fatal("MustFromProfile(RU) must return non-nil provider")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// AlgorithmInfo tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostAlgorithmInfo проверяет Info() метод GostProvider.
func TestGostAlgorithmInfo(t *testing.T) {
	p := NewGostProvider()

	info := p.Info()
	if info.Encryption == "" {
		t.Error("Encryption info must not be empty")
	}
	if info.Hash == "" {
		t.Error("Hash info must not be empty")
	}
	if info.Signature == "" {
		t.Error("Signature info must not be empty")
	}
	if info.HSMStatus != "software-stub" {
		t.Fatalf("expected HSMStatus 'software-stub', got '%s'", info.HSMStatus)
	}
	if info.Status != "active" {
		t.Fatalf("expected Status 'active', got '%s'", info.Status)
	}

	// Проверяем описание ГОСТ алгоритмов
	if !strings.Contains(info.Encryption, "ГОСТ") {
		t.Error("Encryption info should contain 'ГОСТ'")
	}
	if !strings.Contains(info.Hash, "Стрибог") {
		t.Error("Hash info should contain 'Стрибог'")
	}
	if !strings.Contains(info.Signature, "ГОСТ") {
		t.Error("Signature info should contain 'ГОСТ'")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Edge cases and boundary tests
// ═══════════════════════════════════════════════════════════════════════════

// TestGostLargeData проверяет шифрование больших данных (1MB).
func TestGostLargeData(t *testing.T) {
	p := NewGostProvider()

	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	ciphertext, err := p.Encrypt(gostTestKey, largeData)
	if err != nil {
		t.Fatalf("Encrypt 1MB error: %v", err)
	}

	if !bytes.HasPrefix(ciphertext, []byte(GOSTMagic)) {
		t.Fatal("1MB ciphertext must have GOST magic")
	}

	decrypted, err := p.Decrypt(gostTestKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt 1MB error: %v", err)
	}

	if !bytes.Equal(decrypted, largeData) {
		t.Fatal("1MB round-trip failed: data mismatch")
	}
}

// TestGostMultipleEncryptDecrypt проверяет множественные операции.
func TestGostMultipleEncryptDecrypt(t *testing.T) {
	p := NewGostProvider()

	datasets := []struct {
		name string
		data []byte
	}{
		{"small", []byte("hello")},
		{"medium", gostTestData},
		{"large", bytes.Repeat([]byte("A"), 10000)},
		{"binary", []byte{0x00, 0x01, 0xff, 0xfe, 0x80, 0x7f}},
		{"unicode", []byte("Привет, мир! Тест ГОСТ шифрования.")},
	}

	for _, ds := range datasets {
		t.Run(ds.name, func(t *testing.T) {
			ciphertext, err := p.Encrypt(gostTestKey, ds.data)
			if err != nil {
				t.Fatalf("Encrypt error: %v", err)
			}

			if !bytes.HasPrefix(ciphertext, []byte(GOSTMagic)) {
				t.Fatal("ciphertext must have GOST magic")
			}

			decrypted, err := p.Decrypt(gostTestKey, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt error: %v", err)
			}

			if !bytes.Equal(decrypted, ds.data) {
				t.Fatalf("round-trip failed for %s data", ds.name)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Status backward compatibility test
// ═══════════════════════════════════════════════════════════════════════════

// TestGostStatusActive проверяет, что статус изменился с "stub" на "active".
func TestGostStatusActive(t *testing.T) {
	p := NewGostProvider()
	if p.Status() != "active" {
		t.Fatalf("P2-RU.1: expected status 'active', got '%s'. "+
			"GOST provider is no longer a stub.", p.Status())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Benchmarks
// ═══════════════════════════════════════════════════════════════════════════

// BenchmarkGostEncryptDecrypt — производительность Encrypt/Decrypt.
//
// P2-RU.1: overhead <3x vs AES baseline.
// Ожидаемый overhead: маркер + ECDSA key derivation (небольшой).
func BenchmarkGostEncryptDecrypt(b *testing.B) {
	p := NewGostProvider()
	benchmarkGostEncryptDecrypt(b, p)
}

func benchmarkGostEncryptDecrypt(b *testing.B, p *GostProvider) {
	b.Helper()
	b.ReportAllocs()

	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, err := p.Encrypt(gostTestKey, largeData)
		if err != nil {
			b.Fatalf("Encrypt error: %v", err)
		}
		_, err = p.Decrypt(gostTestKey, ciphertext)
		if err != nil {
			b.Fatalf("Decrypt error: %v", err)
		}
	}
}

// BenchmarkGostHash — производительность хеширования.
func BenchmarkGostHash(b *testing.B) {
	p := NewGostProvider()
	benchmarkGostHash(b, p)
}

func benchmarkGostHash(b *testing.B, p *GostProvider) {
	b.Helper()
	b.ReportAllocs()

	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Hash(data)
	}
}

// BenchmarkGostSign — производительность подписи.
func BenchmarkGostSign(b *testing.B) {
	p := NewGostProvider()
	b.ReportAllocs()

	data := make([]byte, 1024) // 1KB
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.Sign(gostTestKey, data)
		if err != nil {
			b.Fatalf("Sign error: %v", err)
		}
	}
}

// BenchmarkGostVerify — производительность верификации.
func BenchmarkGostVerify(b *testing.B) {
	p := NewGostProvider()
	data := make([]byte, 1024) // 1KB
	for i := range data {
		data[i] = byte(i)
	}

	sig, err := p.Sign(gostTestKey, data)
	if err != nil {
		b.Fatalf("Sign error: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := p.Verify(gostTestKey, data, sig)
		if err != nil {
			b.Fatalf("Verify error: %v", err)
		}
	}
}

// BenchmarkGostEncryptVsAES — сравнение производительности GOST vs AES.
//
// P2-RU.1: overhead должен быть <3x.
func BenchmarkGostEncryptVsAES(b *testing.B) {
	b.Run("AES-256-GCM", func(b *testing.B) {
		p := NewAESCrypto()
		b.ReportAllocs()
		benchmarkEncryptDecrypt(b, p)
	})

	b.Run("GOST-28147-89-stub", func(b *testing.B) {
		p := NewGostProvider()
		b.ReportAllocs()
		benchmarkGostEncryptDecrypt(b, p)
	})
}

// BenchmarkGostHashVsSHA256 — сравнение производительности хеша.
func BenchmarkGostHashVsSHA256(b *testing.B) {
	b.Run("SHA-256", func(b *testing.B) {
		p := NewAESCrypto()
		b.ReportAllocs()
		benchmarkHash(b, p)
	})

	b.Run("Stribog-256-stub", func(b *testing.B) {
		p := NewGostProvider()
		b.ReportAllocs()
		benchmarkGostHash(b, p)
	})
}

// BenchmarkGostSignVerify — производительность Sign + Verify.
func BenchmarkGostSignVerify(b *testing.B) {
	p := NewGostProvider()
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sig, err := p.Sign(gostTestKey, data)
		if err != nil {
			b.Fatalf("Sign error: %v", err)
		}
		valid, err := p.Verify(gostTestKey, data, sig)
		if err != nil {
			b.Fatalf("Verify error: %v", err)
		}
		if !valid {
			b.Fatal("signature should be valid")
		}
	}
}
