// Package providers — unit tests for hash and signature providers.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.3: Hash & Signature Provider Tests
//
// Соответствие:
//   - OWASP ASVS V2 (Authentication testing)
//   - OWASP ASVS V6 (Cryptographic storage testing)
//   - ISO 27001 A.14.2 (Security testing)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// ═══════════════════════════════════════════════════════════════════════════
// bash-256 hash tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBash256(t *testing.T) {
	hash, err := Bash256(testData)
	if err != nil {
		t.Fatalf("Bash256 error: %v", err)
	}
	if len(hash) != Bash256Size {
		t.Fatalf("expected %d bytes, got %d", Bash256Size, len(hash))
	}

	// Deterministic
	hash2, _ := Bash256(testData)
	if string(hash) != string(hash2) {
		t.Fatal("bash-256 should be deterministic")
	}

	// Different input — different hash
	hash3, _ := Bash256([]byte("different data"))
	if string(hash) == string(hash3) {
		t.Fatal("different input should produce different hash")
	}
}

func TestBash256Hex(t *testing.T) {
	hex, err := Bash256Hex(testData)
	if err != nil {
		t.Fatalf("Bash256Hex error: %v", err)
	}
	if len(hex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(hex))
	}
}

func TestBash256HMAC(t *testing.T) {
	mac, err := Bash256HMAC(testKey, testData)
	if err != nil {
		t.Fatalf("Bash256HMAC error: %v", err)
	}
	if len(mac) == 0 {
		t.Fatal("HMAC must not be empty")
	}

	// Deterministic
	mac2, _ := Bash256HMAC(testKey, testData)
	if string(mac) != string(mac2) {
		t.Fatal("HMAC should be deterministic")
	}

	// Wrong key
	mac3, _ := Bash256HMAC(wrongKey, testData)
	if string(mac) == string(mac3) {
		t.Fatal("different key should produce different HMAC")
	}
}

func TestBash256HMACHex(t *testing.T) {
	hex, err := Bash256HMACHex(testKey, testData)
	if err != nil {
		t.Fatalf("Bash256HMACHex error: %v", err)
	}
	if len(hex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(hex))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Audit log HMAC tests
// ═══════════════════════════════════════════════════════════════════════════

func TestSignAuditLog(t *testing.T) {
	entry := "user123|UPDATE|device|cam-001|old_status|new_status"
	sig, err := SignAuditLog(testKey, entry)
	if err != nil {
		t.Fatalf("SignAuditLog error: %v", err)
	}
	if sig == "" {
		t.Fatal("signature must not be empty")
	}
}

func TestVerifyAuditLog(t *testing.T) {
	entry := "user456|DELETE|camera|cam-002||"
	sig, _ := SignAuditLog(testKey, entry)

	valid, err := VerifyAuditLog(testKey, entry, sig)
	if err != nil {
		t.Fatalf("VerifyAuditLog error: %v", err)
	}
	if !valid {
		t.Fatal("audit log signature should be valid")
	}

	// Tampered entry
	valid, _ = VerifyAuditLog(testKey, "tampered-entry", sig)
	if valid {
		t.Fatal("audit log signature for tampered entry should be invalid")
	}

	// Wrong key
	valid, _ = VerifyAuditLog(wrongKey, entry, sig)
	if valid {
		t.Fatal("audit log signature with wrong key should be invalid")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// bign-curve256v1 signature tests
// ═══════════════════════════════════════════════════════════════════════════

func TestGenerateBignKeyPair(t *testing.T) {
	kp, err := GenerateBignKeyPair()
	if err != nil {
		t.Fatalf("GenerateBignKeyPair error: %v", err)
	}
	if kp == nil {
		t.Fatal("key pair must not be nil")
	}
	if len(kp.PrivateKey) != BignCurve256v1KeySize {
		t.Fatalf("expected %d bytes private key, got %d", BignCurve256v1KeySize, len(kp.PrivateKey))
	}
	if kp.BignPublicKeyHex() == "" {
		t.Fatal("public key hex must not be empty")
	}
	if kp.BignPrivateKeyHex() == "" {
		t.Fatal("private key hex must not be empty")
	}
}

func TestBignSignVerify(t *testing.T) {
	kp, _ := GenerateBignKeyPair()

	sig, err := BignSign(kp.PrivateKey, testData)
	if err != nil {
		t.Fatalf("BignSign error: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("signature must not be empty")
	}

	// ⚠ В HMAC stub-режиме sign/verify используют один ключ
	valid, err := BignVerify(kp.PrivateKey, testData, sig)
	if err != nil {
		t.Fatalf("BignVerify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}

	// Tampered data
	valid, _ = BignVerify(kp.PrivateKey, []byte("tampered"), sig)
	if valid {
		t.Fatal("signature for tampered data should be invalid")
	}

	// Wrong key
	valid, _ = BignVerify(wrongKey, testData, sig)
	if valid {
		t.Fatal("signature with wrong key should be invalid")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// bign-curve256v1 JWT tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBignJWT(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  "test-user",
		"role": "admin",
		"iat":  1700000000,
	}

	token, err := BignJWT(claims, testKey)
	if err != nil {
		t.Fatalf("BignJWT error: %v", err)
	}
	if token == "" {
		t.Fatal("JWT must not be empty")
	}

	t.Logf("Bign JWT: %s", token)
}

func TestBignJWTVerify(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  "test-user",
		"role": "admin",
		"iat":  1700000000,
	}

	token, _ := BignJWT(claims, testKey)

	verifiedClaims := jwt.MapClaims{}
	err := BignJWTVerify(token, testKey, &verifiedClaims)
	if err != nil {
		t.Fatalf("BignJWTVerify error: %v", err)
	}

	if verifiedClaims["sub"] != "test-user" {
		t.Fatalf("expected sub=test-user, got %v", verifiedClaims["sub"])
	}

	// Wrong key should fail
	wrongClaims := jwt.MapClaims{}
	err = BignJWTVerify(token, wrongKey, &wrongClaims)
	if err == nil {
		t.Fatal("BignJWTVerify with wrong key should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BashHashProvider tests (CryptoProvider interface)
// ═══════════════════════════════════════════════════════════════════════════

func TestBashHashProviderHash(t *testing.T) {
	p := NewBashHashProvider()

	hash, err := p.Hash(testData)
	if err != nil {
		t.Fatalf("BashHashProvider.Hash error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(hash))
	}

	if p.Status() != "stub" {
		t.Errorf("expected status 'stub', got '%s'", p.Status())
	}
}

func TestBashHashProviderEncryptDecrypt(t *testing.T) {
	p := NewBashHashProvider()

	ciphertext, err := p.Encrypt(testKey, testData)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	decrypted, err := p.Decrypt(testKey, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Fatal("decrypted data doesn't match original")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BignSignatureProvider tests (CryptoProvider interface)
// ═══════════════════════════════════════════════════════════════════════════

func TestBignSignatureProviderSignVerify(t *testing.T) {
	p := NewBignSignatureProvider()

	sig, err := p.Sign(testKey, testData)
	if err != nil {
		t.Fatalf("BignSignatureProvider.Sign error: %v", err)
	}

	valid, err := p.Verify(testKey, testData, sig)
	if err != nil {
		t.Fatalf("BignSignatureProvider.Verify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}

	if p.Status() != "stub" {
		t.Errorf("expected status 'stub', got '%s'", p.Status())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Password hashing tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBCryptHashPassword(t *testing.T) {
	p := NewBCryptHash()

	password := "test-password-123!"
	hash, err := p.Hash(password)
	if err != nil {
		t.Fatalf("BCrypt hash error: %v", err)
	}
	if hash == "" {
		t.Fatal("hash must not be empty")
	}

	// Verify correct password
	ok, err := p.Verify(password, hash)
	if err != nil {
		t.Fatalf("BCrypt verify error: %v", err)
	}
	if !ok {
		t.Fatal("correct password should verify")
	}

	// Verify wrong password
	ok, _ = p.Verify("wrong-password", hash)
	if ok {
		t.Fatal("wrong password should not verify")
	}

	if p.Name() != "bcrypt" {
		t.Errorf("expected name 'bcrypt', got '%s'", p.Name())
	}
}

func TestArgon2IDHashPassword(t *testing.T) {
	p := NewArgon2IDHash()

	password := "test-password-456@"
	hash, err := p.Hash(password)
	if err != nil {
		t.Fatalf("Argon2id hash error: %v", err)
	}
	if hash == "" {
		t.Fatal("hash must not be empty")
	}

	// Verify correct password
	ok, err := p.Verify(password, hash)
	if err != nil {
		t.Fatalf("Argon2id verify error: %v", err)
	}
	if !ok {
		t.Fatal("correct password should verify")
	}

	if p.Name() != "argon2id" {
		t.Errorf("expected name 'argon2id', got '%s'", p.Name())
	}
}

func TestBeltHashPassword(t *testing.T) {
	p := NewBeltHash()

	password := "test-password-789$"
	hash, err := p.Hash(password)
	if err != nil {
		t.Fatalf("BeltHash error: %v", err)
	}
	if hash == "" {
		t.Fatal("hash must not be empty")
	}

	ok, err := p.Verify(password, hash)
	if err != nil {
		t.Fatalf("BeltHash verify error: %v", err)
	}
	if !ok {
		t.Fatal("correct password should verify")
	}

	if p.Name() != "belt-hash" {
		t.Errorf("expected name 'belt-hash', got '%s'", p.Name())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Password migration tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPasswordMigratorCurrentOnly(t *testing.T) {
	migrator := NewPasswordMigrator(NewBCryptHash(), nil)

	password := "test-password-1"
	hash, _ := migrator.Current().Hash(password)

	ok, newHash := migrator.Verify(password, hash)
	if !ok {
		t.Fatal("verify should succeed with current provider")
	}
	if newHash != "" {
		t.Fatal("no rehash needed with current provider")
	}
}

func TestPasswordMigratorFallback(t *testing.T) {
	// Simulate migration from bcrypt to argon2id
	oldProvider := NewBCryptHash()
	newProvider := NewArgon2IDHash()
	migrator := NewPasswordMigrator(newProvider, []PasswordHashProvider{oldProvider})

	password := "test-migration-password"
	// Hash with old provider
	oldHash, _ := oldProvider.Hash(password)

	// Verify with migrator — should work and trigger rehash
	ok, newHash := migrator.Verify(password, oldHash)
	if !ok {
		t.Fatal("verify should succeed via fallback")
	}
	if newHash == "" {
		t.Fatal("rehash should be triggered")
	}

	// New hash should work with current provider
	ok, err := newProvider.Verify(password, newHash)
	if !ok {
		t.Fatalf("rehashed password should verify with new provider (err: %v)", err)
	}
}

func TestPasswordMigratorWrongPassword(t *testing.T) {
	migrator := NewPasswordMigrator(NewBCryptHash(), []PasswordHashProvider{NewArgon2IDHash()})

	hash, _ := migrator.Current().Hash("correct-password")

	ok, _ := migrator.Verify("wrong-password", hash)
	if ok {
		t.Fatal("wrong password should not verify")
	}
}

func TestMigratorFromProfile(t *testing.T) {
	migrator, err := MigratorFromProfile("argon2id", "bcrypt")
	if err != nil {
		t.Fatalf("MigratorFromProfile error: %v", err)
	}
	if migrator == nil {
		t.Fatal("migrator must not be nil")
	}

	password := "profile-migration-test"
	hash, _ := NewBCryptHash().Hash(password)

	ok, newHash := migrator.Verify(password, hash)
	if !ok {
		t.Fatal("verify should succeed via bcrypt fallback")
	}
	if newHash == "" {
		t.Fatal("rehash to argon2id should be triggered")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PasswordHashFromProfile tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPasswordHashFromProfile(t *testing.T) {
	tests := []struct {
		profile string
		want    string
	}{
		{"argon2id", "argon2id"},
		{"bcrypt", "bcrypt"},
		{"belt-hash", "belt-hash"},
		{"unknown", "argon2id"}, // unknown → argon2id fallback
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			p, err := PasswordHashFromProfile(tt.profile)
			if err != nil {
				t.Fatalf("PasswordHashFromProfile(%s) error: %v", tt.profile, err)
			}
			if p.Name() != tt.want {
				t.Errorf("expected provider '%s', got '%s'", tt.want, p.Name())
			}
		})
	}
}
