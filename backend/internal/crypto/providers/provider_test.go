// Package providers — unit and benchmark tests.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.2: Provider Tests
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - OWASP ASVS V6 (Cryptographic storage testing)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"bytes"
	"testing"

	"gb-telemetry-collector/internal/compliance"
	"gb-telemetry-collector/internal/stb"
)

// testKey — 32-byte key для тестов.
var testKey = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}

// wrongKey — другой 32-byte key для тестов с неправильным ключом.
var wrongKey = []byte{0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8,
	0xf7, 0xf6, 0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0,
	0xef, 0xee, 0xed, 0xec, 0xeb, 0xea, 0xe9, 0xe8,
	0xe7, 0xe6, 0xe5, 0xe4, 0xe3, 0xe2, 0xe1, 0xe0}

// testData — тестовые данные.
var testData = []byte("sensitive CCTV monitoring data for encryption test")

// ═══════════════════════════════════════════════════════════════════════════
// Encrypt/Decrypt round-trip tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESEncryptDecrypt(t *testing.T) {
	p := NewAESCrypto()
	testEncryptDecryptRoundTrip(t, p)
}

func TestBeltEncryptDecrypt(t *testing.T) {
	p := NewBeltCrypto()
	testEncryptDecryptRoundTrip(t, p)
}

func TestGOSTEncryptDecrypt(t *testing.T) {
	p := NewGOSTCrypto()
	testEncryptDecryptRoundTrip(t, p)
}

// smTestKey — 16-byte key для SM4 тестов.
var smTestKey = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

func TestSMEncryptDecrypt(t *testing.T) {
	p := NewSMCrypto()

	ciphertext, err := p.Encrypt(smTestKey, testData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("ciphertext must not be empty")
	}

	if bytes.Equal(ciphertext, testData) {
		t.Fatal("ciphertext must not equal plaintext")
	}

	decrypted, err := p.Decrypt(smTestKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if !bytes.Equal(decrypted, testData) {
		t.Fatalf("decrypted data doesn't match original: got %v, want %v", decrypted, testData)
	}
}

func testEncryptDecryptRoundTrip(t *testing.T, p stb.CryptoProvider) {
	t.Helper()

	ciphertext, err := p.Encrypt(testKey, testData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("ciphertext must not be empty")
	}

	if bytes.Equal(ciphertext, testData) {
		t.Fatal("ciphertext must not equal plaintext")
	}

	decrypted, err := p.Decrypt(testKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if !bytes.Equal(decrypted, testData) {
		t.Fatalf("decrypted data doesn't match original: got %v, want %v", decrypted, testData)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Wrong key tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESWrongKey(t *testing.T) {
	p := NewAESCrypto()
	testWrongKey(t, p)
}

func testWrongKey(t *testing.T, p stb.CryptoProvider) {
	t.Helper()

	ciphertext, err := p.Encrypt(testKey, testData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	_, err = p.Decrypt(wrongKey, ciphertext)
	if err == nil {
		t.Fatal("Decrypt with wrong key should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Short key tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESShortKey(t *testing.T) {
	p := NewAESCrypto()
	testShortKey(t, p)
}

func testShortKey(t *testing.T, p stb.CryptoProvider) {
	t.Helper()

	shortKey := []byte("short-key")
	_, err := p.Encrypt(shortKey, testData)
	if err == nil {
		t.Fatal("Encrypt with short key should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Empty data tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESEmptyData(t *testing.T) {
	p := NewAESCrypto()

	ciphertext, err := p.Encrypt(testKey, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty data error: %v", err)
	}

	decrypted, err := p.Decrypt(testKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty data error: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("decrypted empty data should be empty, got %d bytes", len(decrypted))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Hash tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESHash(t *testing.T) {
	p := NewAESCrypto()
	testHash(t, p)
}

func testHash(t *testing.T, p stb.CryptoProvider) {
	t.Helper()

	hash, err := p.Hash(testData)
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32 bytes hash, got %d", len(hash))
	}

	// Deterministic
	hash2, _ := p.Hash(testData)
	if !bytes.Equal(hash, hash2) {
		t.Fatal("hash should be deterministic")
	}

	// Different input — different hash
	hash3, _ := p.Hash([]byte("different data"))
	if bytes.Equal(hash, hash3) {
		t.Fatal("different input should produce different hash")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HMAC tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESHMAC(t *testing.T) {
	p := NewAESCrypto()

	mac, err := p.HMAC(testKey, testData)
	if err != nil {
		t.Fatalf("HMAC error: %v", err)
	}
	if len(mac) == 0 {
		t.Fatal("HMAC must not be empty")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Provider selection from profile
// ═══════════════════════════════════════════════════════════════════════════

func TestNewFromProfileBY(t *testing.T) {
	p, err := NewFromProfile(compliance.NewBYProfile())
	if err != nil {
		t.Fatalf("NewFromProfile(BY) error: %v", err)
	}
	if p == nil {
		t.Fatal("NewFromProfile(BY) must return non-nil provider")
	}

	// BY profile should return BeltCrypto
	if _, ok := p.(*BeltCrypto); !ok {
		t.Fatalf("expected BeltCrypto for BY, got %T", p)
	}

	// Round-trip test
	testEncryptDecryptRoundTrip(t, p)
}

func TestNewFromProfileEU(t *testing.T) {
	p, err := NewFromProfile(compliance.NewEUProfile())
	if err != nil {
		t.Fatalf("NewFromProfile(EU) error: %v", err)
	}
	if p == nil {
		t.Fatal("NewFromProfile(EU) must return non-nil provider")
	}

	// EU profile should return AESCrypto
	if _, ok := p.(*AESCrypto); !ok {
		t.Fatalf("expected AESCrypto for EU, got %T", p)
	}

	testEncryptDecryptRoundTrip(t, p)
}

func TestNewFromProfileINTL(t *testing.T) {
	p, err := NewFromProfile(compliance.NewINTLProfile())
	if err != nil {
		t.Fatalf("NewFromProfile(INTL) error: %v", err)
	}
	if p == nil {
		t.Fatal("NewFromProfile(INTL) must return non-nil provider")
	}

	// INTL profile should return AESCrypto
	if _, ok := p.(*AESCrypto); !ok {
		t.Fatalf("expected AESCrypto for INTL, got %T", p)
	}

	testEncryptDecryptRoundTrip(t, p)
}

func TestNewFromProfileNil(t *testing.T) {
	_, err := NewFromProfile(nil)
	if err == nil {
		t.Fatal("NewFromProfile(nil) should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ProviderInfo tests
// ═══════════════════════════════════════════════════════════════════════════

func TestProviderInfo(t *testing.T) {
	providers := []stb.CryptoProvider{
		NewAESCrypto(),
		NewBeltCrypto(),
		NewGOSTCrypto(),
		NewSMCrypto(),
	}

	for _, p := range providers {
		info := Info(p)
		if info.Name == "" {
			t.Errorf("ProviderInfo.Name must not be empty for %T", p)
		}
		if info.Region == "" {
			t.Errorf("ProviderInfo.Region must not be empty for %T", p)
		}
		if info.KeySizeBits <= 0 {
			t.Errorf("ProviderInfo.KeySizeBits must be > 0 for %T", p)
		}
		if info.Status == "" {
			t.Errorf("ProviderInfo.Status must not be empty for %T", p)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BeltKDF tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBeltKDF(t *testing.T) {
	password := []byte("test-password-12345")
	salt := []byte("test-salt-67890")

	key, err := BeltKDF(password, salt, 32)
	if err != nil {
		t.Fatalf("BeltKDF error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes key, got %d", len(key))
	}

	// Deterministic
	key2, _ := BeltKDF(password, salt, 32)
	if !bytes.Equal(key, key2) {
		t.Fatal("BeltKDF should be deterministic")
	}

	// Different salt — different key
	key3, _ := BeltKDF(password, []byte("different-salt"), 32)
	if bytes.Equal(key, key3) {
		t.Fatal("different salt should produce different key")
	}
}

func TestBeltKDFInvalidKeyLen(t *testing.T) {
	_, err := BeltKDF([]byte("pwd"), []byte("salt"), 8)
	if err == nil {
		t.Fatal("BeltKDF with keyLen < 16 should return error")
	}

	_, err = BeltKDF([]byte("pwd"), []byte("salt"), 128)
	if err == nil {
		t.Fatal("BeltKDF with keyLen > 64 should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Sign/Verify tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESSignVerify(t *testing.T) {
	p := NewAESCrypto()

	sig, err := p.Sign(testKey, testData)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("signature must not be empty")
	}

	valid, err := p.Verify(testKey, testData, sig)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}

	// Wrong key should fail
	valid, _ = p.Verify(wrongKey, testData, sig)
	if valid {
		t.Fatal("signature with wrong key should be invalid")
	}

	// Wrong data should fail
	valid, _ = p.Verify(testKey, []byte("tampered data"), sig)
	if valid {
		t.Fatal("signature for tampered data should be invalid")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GenerateKey tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAESGenerateKey(t *testing.T) {
	p := NewAESCrypto()

	key, err := p.GenerateKey(32)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}

	// Minimum length enforcement
	shortKey, err := p.GenerateKey(1)
	if err != nil {
		t.Fatalf("GenerateKey(1) error: %v", err)
	}
	if len(shortKey) < 32 {
		t.Fatalf("expected at least 32 bytes, got %d", len(shortKey))
	}

	// Uniqueness
	key2, _ := p.GenerateKey(32)
	if len(key) == len(key2) && key[0] == key2[0] && key[16] == key2[16] {
		// Very unlikely to collide, but if it happens, it's suspicious
		t.Log("warning: generated keys might not be unique")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Status tests for stub providers
// ═══════════════════════════════════════════════════════════════════════════

func TestBeltStatus(t *testing.T) {
	p := NewBeltCrypto()
	if p.Status() != "stub" {
		t.Errorf("expected status 'stub', got '%s'", p.Status())
	}
}

func TestGOSTStatus(t *testing.T) {
	p := NewGOSTCrypto()
	// P2-MKT.1: GOST provider теперь gost-native (реальные алгоритмы)
	if p.Status() != "gost-native" {
		t.Errorf("P2-MKT.1: expected status 'gost-native', got '%s'", p.Status())
	}
}

func TestSMStatus(t *testing.T) {
	p := NewSMCrypto()
	// P2-CR.3: SM provider теперь active (не stub)
	if p.Status() != "active" {
		t.Errorf("P2-CR.3: expected status 'active', got '%s'", p.Status())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// MustFromProfile tests
// ═══════════════════════════════════════════════════════════════════════════

func TestMustFromProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustFromProfile should not panic for valid profile: %v", r)
		}
	}()

	p := MustFromProfile(compliance.NewINTLProfile())
	if p == nil {
		t.Fatal("MustFromProfile must return non-nil provider")
	}
}

func TestMustFromProfilePanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustFromProfile should panic on nil profile")
		}
	}()

	MustFromProfile(nil)
}

// ═══════════════════════════════════════════════════════════════════════════
// Benchmarks: belt vs AES vs GOST vs SM
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkAESEncryptDecrypt(b *testing.B) {
	p := NewAESCrypto()
	benchmarkEncryptDecrypt(b, p)
}

func BenchmarkBeltEncryptDecrypt(b *testing.B) {
	p := NewBeltCrypto()
	benchmarkEncryptDecrypt(b, p)
}

func BenchmarkGOSTEncryptDecrypt(b *testing.B) {
	p := NewGOSTCrypto()
	benchmarkEncryptDecrypt(b, p)
}

func BenchmarkSMEncryptDecrypt(b *testing.B) {
	p := NewSMCrypto()
	benchmarkEncryptDecrypt(b, p)
}

func benchmarkEncryptDecrypt(b *testing.B, p stb.CryptoProvider) {
	b.Helper()
	b.ReportAllocs()

	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, err := p.Encrypt(testKey, largeData)
		if err != nil {
			b.Fatalf("Encrypt error: %v", err)
		}
		_, err = p.Decrypt(testKey, ciphertext)
		if err != nil {
			b.Fatalf("Decrypt error: %v", err)
		}
	}
}

func BenchmarkAESHash(b *testing.B) {
	p := NewAESCrypto()
	benchmarkHash(b, p)
}

func benchmarkHash(b *testing.B, p stb.CryptoProvider) {
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
