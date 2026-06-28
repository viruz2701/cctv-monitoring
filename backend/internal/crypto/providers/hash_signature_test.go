// Package providers — unit tests for hash and signature providers.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-SEC.2: bign-curve256v1 Signature Tests
//
// Соответствие:
//   - OWASP ASVS V2 (Authentication testing)
//   - OWASP ASVS V6 (Cryptographic storage testing)
//   - ISO 27001 A.14.2 (Security testing)
//   - СТБ 34.101.45 — bign-curve256v1
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
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
// bign-curve256v1 (ECDSA P-256) signature tests
// ═══════════════════════════════════════════════════════════════════════════

// testBignKey — глобальная тестовая пара ключей bign.
var testBignKey *BignKeyPair

func init() {
	var err error
	testBignKey, err = GenerateBignKeyPair()
	if err != nil {
		panic("failed to generate test bign key: " + err.Error())
	}
}

func TestGenerateBignKeyPair(t *testing.T) {
	kp, err := GenerateBignKeyPair()
	if err != nil {
		t.Fatalf("GenerateBignKeyPair error: %v", err)
	}
	if kp == nil {
		t.Fatal("key pair must not be nil")
	}
	if kp.PrivateKey == nil {
		t.Fatal("private key must not be nil")
	}
	if kp.PublicKey == nil {
		t.Fatal("public key must not be nil")
	}
	if len(kp.PrivateKeyPEM) == 0 {
		t.Fatal("private key PEM must not be empty")
	}
	if len(kp.PublicKeyPEM) == 0 {
		t.Fatal("public key PEM must not be empty")
	}

	// Проверяем кривую
	if kp.PrivateKey.Curve != elliptic.P256() {
		t.Fatal("expected P-256 curve")
	}

	// Проверяем hex helpers
	privHex := BignPrivateKeyHex(kp.PrivateKey)
	if privHex == "" {
		t.Fatal("private key hex must not be empty")
	}
	pubHex := BignPublicKeyHex(kp.PublicKey)
	if pubHex == "" {
		t.Fatal("public key hex must not be empty")
	}
	if len(pubHex) != 130 { // 65 bytes uncompressed = 130 hex chars
		t.Fatalf("expected 130 hex chars for uncompressed public key, got %d", len(pubHex))
	}
}

func TestParseBignPrivateKey(t *testing.T) {
	kp, err := GenerateBignKeyPair()
	if err != nil {
		t.Fatalf("GenerateBignKeyPair: %v", err)
	}

	parsed, err := ParseBignPrivateKey(kp.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("ParseBignPrivateKey error: %v", err)
	}
	if parsed == nil {
		t.Fatal("parsed key must not be nil")
	}
	if parsed.Curve != elliptic.P256() {
		t.Fatal("expected P-256 curve")
	}

	// Проверяем что D совпадает
	if parsed.D.Cmp(kp.PrivateKey.D) != 0 {
		t.Fatal("parsed private key D does not match original")
	}
}

func TestParseBignPublicKey(t *testing.T) {
	kp, err := GenerateBignKeyPair()
	if err != nil {
		t.Fatalf("GenerateBignKeyPair: %v", err)
	}

	parsed, err := ParseBignPublicKey(kp.PublicKeyPEM)
	if err != nil {
		t.Fatalf("ParseBignPublicKey error: %v", err)
	}
	if parsed == nil {
		t.Fatal("parsed public key must not be nil")
	}
	if parsed.Curve != elliptic.P256() {
		t.Fatal("expected P-256 curve")
	}
	if parsed.X.Cmp(kp.PublicKey.X) != 0 || parsed.Y.Cmp(kp.PublicKey.Y) != 0 {
		t.Fatal("parsed public key does not match original")
	}
}

func TestBignSignVerify(t *testing.T) {
	sig, err := BignSign(testBignKey.PrivateKey, testData)
	if err != nil {
		t.Fatalf("BignSign error: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("signature must not be empty")
	}

	// Verify with public key (асимметричная схема)
	valid, err := BignVerify(testBignKey.PublicKey, testData, sig)
	if err != nil {
		t.Fatalf("BignVerify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}

	// Tampered data
	valid, _ = BignVerify(testBignKey.PublicKey, []byte("tampered"), sig)
	if valid {
		t.Fatal("signature for tampered data should be invalid")
	}

	// Wrong key — другая пара ключей
	otherKey, _ := GenerateBignKeyPair()
	valid, _ = BignVerify(otherKey.PublicKey, testData, sig)
	if valid {
		t.Fatal("signature with wrong public key should be invalid")
	}
}

func TestBignSignVerifyRaw(t *testing.T) {
	sig, err := BignSignRaw(testBignKey.PrivateKey, testData)
	if err != nil {
		t.Fatalf("BignSignRaw error: %v", err)
	}
	if len(sig) != 64 {
		t.Fatalf("expected 64 bytes raw signature, got %d", len(sig))
	}

	valid, err := BignVerifyRaw(testBignKey.PublicKey, testData, sig)
	if err != nil {
		t.Fatalf("BignVerifyRaw error: %v", err)
	}
	if !valid {
		t.Fatal("raw signature should be valid")
	}

	// Tampered data
	valid, _ = BignVerifyRaw(testBignKey.PublicKey, []byte("tampered"), sig)
	if valid {
		t.Fatal("raw signature for tampered data should be invalid")
	}
}

func TestBignSignVerifyDeterministic(t *testing.T) {
	// ECDSA P-256 не детерминирован (использует random k),
	// но проверка должна работать для любой подписи
	sig1, _ := BignSign(testBignKey.PrivateKey, testData)
	sig2, _ := BignSign(testBignKey.PrivateKey, testData)

	// Подписи должны быть разными (из-за random k)
	if string(sig1) == string(sig2) {
		t.Log("note: ECDSA signatures may collide but very unlikely")
	}

	// Обе должны верифицироваться
	valid1, _ := BignVerify(testBignKey.PublicKey, testData, sig1)
	valid2, _ := BignVerify(testBignKey.PublicKey, testData, sig2)
	if !valid1 || !valid2 {
		t.Fatal("both signatures should be valid")
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

	token, err := BignJWT(claims, testBignKey.PrivateKey)
	if err != nil {
		t.Fatalf("BignJWT error: %v", err)
	}
	if token == "" {
		t.Fatal("JWT must not be empty")
	}

	t.Logf("Bign JWT (ES256): %s", token)
}

func TestBignJWTVerify(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  "test-user",
		"role": "admin",
		"iat":  1700000000,
	}

	token, _ := BignJWT(claims, testBignKey.PrivateKey)

	verifiedClaims := jwt.MapClaims{}
	err := BignJWTVerify(token, testBignKey.PublicKey, &verifiedClaims)
	if err != nil {
		t.Fatalf("BignJWTVerify error: %v", err)
	}

	if verifiedClaims["sub"] != "test-user" {
		t.Fatalf("expected sub=test-user, got %v", verifiedClaims["sub"])
	}

	// Wrong key should fail
	otherKey, _ := GenerateBignKeyPair()
	wrongClaims := jwt.MapClaims{}
	err = BignJWTVerify(token, otherKey.PublicKey, &wrongClaims)
	if err == nil {
		t.Fatal("BignJWTVerify with wrong public key should return error")
	}
}

func TestBignJWTRoundTripWithClaims(t *testing.T) {
	claims := &jwt.RegisteredClaims{
		Subject: "test-subject",
	}
	token, err := BignJWT(claims, testBignKey.PrivateKey)
	if err != nil {
		t.Fatalf("BignJWT error: %v", err)
	}

	parsed := &jwt.RegisteredClaims{}
	err = BignJWTVerify(token, testBignKey.PublicKey, parsed)
	if err != nil {
		t.Fatalf("BignJWTVerify error: %v", err)
	}

	if parsed.Subject != "test-subject" {
		t.Fatalf("expected subject 'test-subject', got '%s'", parsed.Subject)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BignSigningMethod tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBignSigningMethodAlg(t *testing.T) {
	m := NewBignSigningMethod()
	if m.Alg() != "ES256" {
		t.Fatalf("expected Alg='ES256', got '%s'", m.Alg())
	}
}

func TestBignSigningMethodSignVerify(t *testing.T) {
	m := NewBignSigningMethod()
	claims := jwt.MapClaims{
		"sub": "test",
		"iat": 1700000000,
	}

	token, err := m.Sign(claims, testBignKey.PrivateKey)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	parsed := jwt.MapClaims{}
	err = m.Verify(token, testBignKey.PublicKey, &parsed)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}

	if parsed["sub"] != "test" {
		t.Fatalf("expected sub='test', got '%v'", parsed["sub"])
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

	// Используем PEM-encoded приватный ключ
	sig, err := p.Sign(testBignKey.PrivateKeyPEM, testData)
	if err != nil {
		t.Fatalf("BignSignatureProvider.Sign error: %v", err)
	}

	valid, err := p.Verify(testBignKey.PublicKeyPEM, testData, sig)
	if err != nil {
		t.Fatalf("BignSignatureProvider.Verify error: %v", err)
	}
	if !valid {
		t.Fatal("signature should be valid")
	}

	// Статус теперь "active" (real ECDSA)
	if p.Status() != "active" {
		t.Errorf("expected status 'active', got '%s'", p.Status())
	}
}

func TestBignSignatureProviderRejectsWrongKey(t *testing.T) {
	p := NewBignSignatureProvider()
	otherKey, _ := GenerateBignKeyPair()

	sig, err := p.Sign(testBignKey.PrivateKeyPEM, testData)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	// Wrong public key
	valid, err := p.Verify(otherKey.PublicKeyPEM, testData, sig)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if valid {
		t.Fatal("signature with wrong public key should be invalid")
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

// ═══════════════════════════════════════════════════════════════════════════
// Benchmark: ECDSA P-256 sign/verify
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkBignSign(b *testing.B) {
	kp, _ := GenerateBignKeyPair()
	data := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		BignSign(kp.PrivateKey, data)
	}
}

func BenchmarkBignVerify(b *testing.B) {
	kp, _ := GenerateBignKeyPair()
	data := make([]byte, 1024)
	sig, _ := BignSign(kp.PrivateKey, data)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		BignVerify(kp.PublicKey, data, sig)
	}
}

func BenchmarkBignSignRaw(b *testing.B) {
	kp, _ := GenerateBignKeyPair()
	data := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		BignSignRaw(kp.PrivateKey, data)
	}
}

func BenchmarkBignKeyGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
}

func BenchmarkBignJWT(b *testing.B) {
	kp, _ := GenerateBignKeyPair()
	claims := jwt.MapClaims{"sub": "bench", "iat": 1700000000}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		BignJWT(claims, kp.PrivateKey)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Edge case tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBignSignEmptyData(t *testing.T) {
	sig, err := BignSign(testBignKey.PrivateKey, []byte{})
	if err != nil {
		t.Fatalf("BignSign empty data error: %v", err)
	}

	valid, err := BignVerify(testBignKey.PublicKey, []byte{}, sig)
	if err != nil {
		t.Fatalf("BignVerify empty data error: %v", err)
	}
	if !valid {
		t.Fatal("signature for empty data should be valid")
	}
}

func TestBignSignLargeData(t *testing.T) {
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	sig, err := BignSign(testBignKey.PrivateKey, largeData)
	if err != nil {
		t.Fatalf("BignSign 10MB error: %v", err)
	}

	// Sign+verify with SHA-256 hashing internally, so large data should work
	valid, err := BignVerify(testBignKey.PublicKey, largeData, sig)
	if err != nil {
		t.Fatalf("BignVerify 10MB error: %v", err)
	}
	if !valid {
		t.Fatal("signature for 10MB data should be valid")
	}
}

func TestBignSignNilKey(t *testing.T) {
	_, err := BignSign(nil, testData)
	if err == nil {
		t.Fatal("expected error for nil private key")
	}
}

func TestBignVerifyNilKey(t *testing.T) {
	sig, _ := BignSign(testBignKey.PrivateKey, testData)
	_, err := BignVerify(nil, testData, sig)
	if err == nil {
		t.Fatal("expected error for nil public key")
	}
}

func TestBignVerifyWrongSignatureLength(t *testing.T) {
	valid, err := BignVerifyRaw(testBignKey.PublicKey, testData, []byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for short signature")
	}
	if valid {
		t.Fatal("short signature should not be valid")
	}
}

func TestParseBignPrivateKeyInvalidPEM(t *testing.T) {
	_, err := ParseBignPrivateKey([]byte("invalid-pem"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestParseBignPublicKeyInvalidPEM(t *testing.T) {
	_, err := ParseBignPublicKey([]byte("not-a-pem"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestBignPublicKeyHexNil(t *testing.T) {
	if BignPublicKeyHex(nil) != "" {
		t.Fatal("expected empty string for nil public key")
	}
}

func TestBignPrivateKeyHexNil(t *testing.T) {
	if BignPrivateKeyHex(nil) != "" {
		t.Fatal("expected empty string for nil private key")
	}
}

func TestBignJWTWithInvalidClaims(t *testing.T) {
	_, err := BignJWT(nil, testBignKey.PrivateKey)
	if err == nil {
		t.Fatal("expected error for nil claims")
	}
}

func TestBignJWTVerifyWithWrongAlg(t *testing.T) {
	// Создаём HMAC-SHA256 JWT вместо ES256
	claims := jwt.MapClaims{"sub": "test"}
	hmacToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := hmacToken.SignedString([]byte("secret"))

	// Проверяем что ES256 верификация отвергает HS256 токен
	parsed := jwt.MapClaims{}
	err := BignJWTVerify(token, testBignKey.PublicKey, &parsed)
	if err == nil {
		t.Fatal("expected error for wrong algorithm (HS256 vs ES256)")
	}
}

func TestBignSignHashesCorrectly(t *testing.T) {
	// Проверяем что подпись использует SHA-256
	data := []byte("test data for hash verification")

	sig, err := BignSign(testBignKey.PrivateKey, data)
	if err != nil {
		t.Fatalf("BignSign error: %v", err)
	}

	// Проверяем через стандартную ecdsa.VerifyASN1
	hash := sha256.Sum256(data)
	valid := ecdsa.VerifyASN1(testBignKey.PublicKey, hash[:], sig)
	if !valid {
		t.Fatal("ECDSA verification failed with SHA-256 hash")
	}
}

func TestBignSignatureProviderStatus(t *testing.T) {
	p := NewBignSignatureProvider()
	// Статус должен быть "active" (real ECDSA, не stub)
	if p.Status() != "active" {
		t.Errorf("expected status 'active', got '%s'", p.Status())
	}
}

func TestBignSignatureProviderHash(t *testing.T) {
	p := NewBignSignatureProvider()

	hash, err := p.Hash(testData)
	if err != nil {
		t.Fatalf("Hash error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(hash))
	}

	hashHex, err := p.HashHex(testData)
	if err != nil {
		t.Fatalf("HashHex error: %v", err)
	}
	if len(hashHex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(hashHex))
	}
}

func TestBignSignatureProviderHMAC(t *testing.T) {
	p := NewBignSignatureProvider()

	mac, err := p.HMAC(testKey, testData)
	if err != nil {
		t.Fatalf("HMAC error: %v", err)
	}
	if len(mac) == 0 {
		t.Fatal("HMAC must not be empty")
	}

	macHex, err := p.HMACHex(testKey, testData)
	if err != nil {
		t.Fatalf("HMACHex error: %v", err)
	}
	if len(macHex) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(macHex))
	}
}

func TestBignSignatureProviderEncryptDecrypt(t *testing.T) {
	p := NewBignSignatureProvider()

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
