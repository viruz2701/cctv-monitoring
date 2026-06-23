package stb

import (
	"bytes"
	"testing"
)

func TestHash(t *testing.T) {
	data := []byte("test data for hashing")
	hash, err := Hash(data)
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(hash))
	}

	// Deterministic
	hash2, _ := Hash(data)
	if !bytes.Equal(hash, hash2) {
		t.Fatal("hash should be deterministic")
	}
}

func TestHashHex(t *testing.T) {
	hex, err := HashHex([]byte("test"))
	if err != nil {
		t.Fatalf("HashHex error: %v", err)
	}
	if len(hex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(hex))
	}
}

func TestHMAC(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!") // exactly 32 bytes
	data := []byte("test data")

	mac, err := HMAC(key, data)
	if err != nil {
		t.Fatalf("HMAC error: %v", err)
	}
	if len(mac) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(mac))
	}
}

func TestHMACHex(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!") // exactly 32 bytes
	hex, err := HMACHex(key, []byte("test"))
	if err != nil {
		t.Fatalf("HMACHex error: %v", err)
	}
	if len(hex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(hex))
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!x") // exactly 32 bytes
	plaintext := []byte("sensitive CCTV data")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("decrypted data does not match original")
	}
}

func TestEncryptDecryptWithDifferentKey(t *testing.T) {
	key1 := []byte("this-is-a-32-byte-key-for-test!x")    // exactly 32 bytes
	key2 := []byte("this-is-another-32-byte-key-for-tes") // exactly 32 bytes
	plaintext := []byte("sensitive data")

	ciphertext, _ := Encrypt(key1, plaintext)
	_, err := Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestSignVerify(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!") // exactly 32 bytes
	data := []byte("data to sign")

	sig, err := Sign(key, data)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	valid, err := Verify(key, data, sig)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}

	// Wrong signature
	valid, _ = Verify(key, data, []byte("wrong"))
	if valid {
		t.Fatal("wrong signature should be invalid")
	}

	// Wrong key
	wrongKey := []byte("this-is-another-32-byte-key-for-tes") // exactly 32 bytes
	valid, _ = Verify(wrongKey, data, sig)
	if valid {
		t.Fatal("wrong key should produce invalid signature")
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey(32)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}

	// Auto-enforce minimum length
	short, _ := GenerateKey(1)
	if len(short) < 32 {
		t.Fatal("key should be at least 32 bytes")
	}

	// Uniqueness
	key2, _ := GenerateKey(32)
	if bytes.Equal(key, key2) {
		t.Fatal("keys should be unique")
	}
}

func TestDefaultCrypto(t *testing.T) {
	if DefaultCrypto == nil {
		t.Fatal("DefaultCrypto should not be nil")
	}

	// Verify it's the standard implementation
	hash, err := DefaultCrypto.Hash([]byte("test"))
	if err != nil {
		t.Fatalf("DefaultCrypto.Hash error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32 byte hash")
	}
}

func BenchmarkHash(b *testing.B) {
	data := []byte("benchmark test data for hashing")
	for i := 0; i < b.N; i++ {
		Hash(data)
	}
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	key := []byte("this-is-a-32-byte-key-for-test!") // exactly 32 bytes
	plaintext := []byte("sensitive CCTV monitoring data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := Encrypt(key, plaintext)
		Decrypt(key, ciphertext)
	}
}
