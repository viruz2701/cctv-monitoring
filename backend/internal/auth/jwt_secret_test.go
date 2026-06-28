package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// JWT_SECRET tests (legacy symmetric secret)
// ═══════════════════════════════════════════════════════════════════════════

func TestGetJWTSecret_Success(t *testing.T) {
	orig := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", orig)

	os.Setenv("JWT_SECRET", "this-is-a-256-bit-key-that-is-long-enough-for-testing!")
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty secret")
	}
}

func TestGetJWTSecret_Missing(t *testing.T) {
	orig := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", orig)

	os.Unsetenv("JWT_SECRET")

	_, err := GetJWTSecret()
	if err == nil {
		t.Fatal("expected error for missing JWT_SECRET")
	}
	if err != ErrJWTSecretMissing {
		t.Fatalf("expected ErrJWTSecretMissing, got: %v", err)
	}
}

func TestIsJWTSecretSet(t *testing.T) {
	orig := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", orig)

	os.Unsetenv("JWT_SECRET")
	if IsJWTSecretSet() {
		t.Fatal("expected false when JWT_SECRET is not set")
	}

	os.Setenv("JWT_SECRET", "test-secret")
	if !IsJWTSecretSet() {
		t.Fatal("expected true when JWT_SECRET is set")
	}
}

func TestGetJWTSecret_EmptyString(t *testing.T) {
	orig := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", orig)

	os.Setenv("JWT_SECRET", "")
	_, err := GetJWTSecret()
	if err == nil {
		t.Fatal("expected error for empty JWT_SECRET")
	}
}

func TestGetJWTSecret_Uniqueness(t *testing.T) {
	orig := os.Getenv("JWT_SECRET")
	defer os.Setenv("JWT_SECRET", orig)

	os.Setenv("JWT_SECRET", "different-test-secret-key-for-uniqueness-check!")

	secret1, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	secret2, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Deterministic — same key should return same secret
	if string(secret1) != string(secret2) {
		t.Fatal("GetJWTSecret should be deterministic")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BIGN_PRIVATE_KEY tests (ECDSA P-256)
// ═══════════════════════════════════════════════════════════════════════════

func TestGetBignPrivateKey_GeneratesOnMissing(t *testing.T) {
	ResetBignPrivateKey()
	origKey := os.Getenv("BIGN_PRIVATE_KEY")
	defer os.Setenv("BIGN_PRIVATE_KEY", origKey)
	os.Unsetenv("BIGN_PRIVATE_KEY")

	key, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("GetBignPrivateKey error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.Curve != elliptic.P256() {
		t.Fatalf("expected P-256 curve, got %s", key.Curve.Params().Name)
	}
}

func TestGetBignPrivateKey_FromEnv(t *testing.T) {
	ResetBignPrivateKey()

	// Generate a real ECDSA P-256 key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	// PEM-encode it
	privDER, _ := x509.MarshalECPrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	orig := os.Getenv("BIGN_PRIVATE_KEY")
	defer os.Setenv("BIGN_PRIVATE_KEY", orig)
	os.Setenv("BIGN_PRIVATE_KEY", string(privPEM))

	key, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("GetBignPrivateKey error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.D.Cmp(privKey.D) != 0 {
		t.Fatal("loaded key does not match original")
	}
}

func TestGetBignPrivateKey_FromEnvPKCS8(t *testing.T) {
	ResetBignPrivateKey()

	// Generate key and marshal as PKCS8
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	pkcs8DER, _ := x509.MarshalPKCS8PrivateKey(privKey)
	pkcs8PEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8DER})

	orig := os.Getenv("BIGN_PRIVATE_KEY")
	defer os.Setenv("BIGN_PRIVATE_KEY", orig)
	os.Setenv("BIGN_PRIVATE_KEY", string(pkcs8PEM))

	key, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("GetBignPrivateKey from PKCS8 error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.D.Cmp(privKey.D) != 0 {
		t.Fatal("PKCS8 loaded key does not match original")
	}
}

func TestGetBignPrivateKey_Caches(t *testing.T) {
	ResetBignPrivateKey()

	// First call generates or loads
	key1, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call should return cached
	key2, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if key1 != key2 {
		t.Fatal("GetBignPrivateKey should cache the key")
	}
}

func TestGetBignPublicKey(t *testing.T) {
	ResetBignPrivateKey()

	pubKey, err := GetBignPublicKey()
	if err != nil {
		t.Fatalf("GetBignPublicKey error: %v", err)
	}
	if pubKey == nil {
		t.Fatal("expected non-nil public key")
	}
	if pubKey.Curve != elliptic.P256() {
		t.Fatalf("expected P-256 curve, got %s", pubKey.Curve.Params().Name)
	}
}

func TestGetBignPublicKey_MatchesPrivateKey(t *testing.T) {
	ResetBignPrivateKey()

	privKey, _ := GetBignPrivateKey()
	pubKey, _ := GetBignPublicKey()

	if pubKey.X.Cmp(privKey.PublicKey.X) != 0 || pubKey.Y.Cmp(privKey.PublicKey.Y) != 0 {
		t.Fatal("public key should match private key's public key")
	}
}

func TestParseBignPrivateKey_InvalidPEM(t *testing.T) {
	_, err := parseBignPrivateKeyPEM([]byte("not-valid-pem"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestParseBignPrivateKey_WrongCurve(t *testing.T) {
	// Generate P-521 key (wrong curve)
	privKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey P-521: %v", err)
	}
	privDER, _ := x509.MarshalECPrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	_, err = parseBignPrivateKeyPEM(privPEM)
	if err == nil {
		t.Fatal("expected error for wrong curve (P-521 != P-256)")
	}
}

func TestGetBignPrivateKey_FromFile(t *testing.T) {
	ResetBignPrivateKey()

	// Generate a key and write to temp file
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privDER, _ := x509.MarshalECPrivateKey(privKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	tmpFile, err := os.CreateTemp("", "bign-key-*.pem")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(privPEM); err != nil {
		t.Fatalf("Write temp file: %v", err)
	}
	tmpFile.Close()

	origKey := os.Getenv("BIGN_PRIVATE_KEY")
	origFile := os.Getenv("BIGN_PRIVATE_KEY_FILE")
	defer func() {
		os.Setenv("BIGN_PRIVATE_KEY", origKey)
		os.Setenv("BIGN_PRIVATE_KEY_FILE", origFile)
	}()
	os.Unsetenv("BIGN_PRIVATE_KEY")
	os.Setenv("BIGN_PRIVATE_KEY_FILE", tmpFile.Name())

	key, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("GetBignPrivateKey from file error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if key.D.Cmp(privKey.D) != 0 {
		t.Fatal("key from file does not match original")
	}
}
