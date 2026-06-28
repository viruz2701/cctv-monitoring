package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
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

func TestNewBignSigningMethod_EmptyKeyAutoGenerates(t *testing.T) {
	// Empty key should auto-generate ECDSA P-256 key for dev
	method, err := NewBignSigningMethod("")
	if err != nil {
		t.Fatalf("unexpected error with empty key: %v", err)
	}
	if method == nil {
		t.Fatal("method should not be nil")
	}

	// Should be able to sign with auto-generated key
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
}

func TestNewBignSigningMethod_ValidPEMKey(t *testing.T) {
	// Generate real ECDSA P-256 key and PEM-encode it
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	privDER, _ := x509.MarshalECPrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	method, err := NewBignSigningMethod(string(privPEM))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if method == nil {
		t.Fatal("method should not be nil")
	}
}

func TestNewBignSigningMethod_InvalidKey(t *testing.T) {
	_, err := NewBignSigningMethod("not-a-valid-pem-key")
	if err == nil {
		t.Fatal("should error on invalid key material")
	}
}

func TestBignSignAndVerify(t *testing.T) {
	// Auto-generate key
	method, err := NewBignSigningMethod("")
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

	t.Logf("JWT token (ES256): %s", token)

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
