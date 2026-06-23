package crypto

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashAPIKey(t *testing.T) {
	key := "test-api-key-12345"
	hash := HashAPIKey(key)

	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	// Deterministic
	hash2 := HashAPIKey(key)
	if hash != hash2 {
		t.Fatal("hash should be deterministic")
	}

	// Different keys produce different hashes
	hash3 := HashAPIKey("different-key")
	if hash == hash3 {
		t.Fatal("different keys should produce different hashes")
	}

	t.Logf("API key hash: %s", hash)
}

func TestValidateAPIKey(t *testing.T) {
	key := "test-api-key-validate"
	hash := HashAPIKey(key)

	if !ValidateAPIKey(key, hash) {
		t.Fatal("should validate correct key")
	}

	if ValidateAPIKey("wrong-key", hash) {
		t.Fatal("should not validate wrong key")
	}
}

func TestNewBignSigningMethod(t *testing.T) {
	// Key too short
	_, err := NewBignSigningMethod("short")
	if err == nil {
		t.Fatal("should error on short key")
	}

	// Valid key
	validKey := "this-is-a-256-bit-key-that-is-long-enough!"
	method, err := NewBignSigningMethod(validKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method == nil {
		t.Fatal("method should not be nil")
	}
}

func TestBignSignAndVerify(t *testing.T) {
	secret := "this-is-a-secure-256-bit-key-for-testing-only!!"
	method, err := NewBignSigningMethod(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims := jwt.MapClaims{
		"sub":  "test-user",
		"role": "admin",
		"iat":  1700000000,
	}

	token, err := method.Sign(claims)
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}

	if token == "" {
		t.Fatal("token should not be empty")
	}

	t.Logf("JWT token: %s", token)

	// Verify
	verifiedClaims := jwt.MapClaims{}
	err = method.Verify(token, &verifiedClaims)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}

	if verifiedClaims["sub"] != "test-user" {
		t.Fatalf("expected sub=test-user, got %v", verifiedClaims["sub"])
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}

	// Minimum length enforcement
	shortKey, err := GenerateKey(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(shortKey) < MinKeyLength {
		t.Fatalf("expected at least %d bytes, got %d", MinKeyLength, len(shortKey))
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(key) != 64 { // 32 bytes hex-encoded = 64 chars
		t.Fatalf("expected 64 hex chars, got %d", len(key))
	}

	// Different calls produce different keys
	key2, _ := GenerateAPIKey()
	if key == key2 {
		t.Fatal("keys should be unique")
	}
}

func TestHashAPIKeyEmpty(t *testing.T) {
	hash := HashAPIKey("")
	if hash == "" {
		t.Fatal("empty key should still produce a hash")
	}
	t.Logf("Empty key hash: %s", hash)
}
