package auth

import (
	"os"
	"testing"
)

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
	// Save and restore original value
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
