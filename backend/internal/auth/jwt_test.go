package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMain(m *testing.M) {
	// Set JWT secret for tests (legacy, for refresh tokens)
	os.Setenv("JWT_SECRET", "test-secret-key-min-32-chars-long-for-hs256!")
	// BIGN_PRIVATE_KEY не устанавливаем — будет использована автогенерация
	ResetBignPrivateKey()
	code := m.Run()
	os.Unsetenv("JWT_SECRET")
	ResetBignPrivateKey()
	os.Exit(code)
}

// getTestBignKey returns the generated ECDSA P-256 key for tests.
func getTestBignKey() (interface{}, error) {
	return GetBignPrivateKey()
}

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateJWT(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("expected UserID 'user-1', got '%s'", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("expected Username 'testuser', got '%s'", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("expected Role 'admin', got '%s'", claims.Role)
	}
	if claims.TenantID != "tenant-1" {
		t.Errorf("expected TenantID 'tenant-1', got '%s'", claims.TenantID)
	}
}

func TestJWTAccessTokenTTL(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	// Check that TTL is approximately 15 minutes
	expiresAt := claims.ExpiresAt.Time
	issuedAt := claims.IssuedAt.Time
	ttl := expiresAt.Sub(issuedAt)

	// Allow 1 second tolerance
	if ttl < 14*time.Minute || ttl > 16*time.Minute {
		t.Errorf("expected TTL ~15m, got %v", ttl)
	}
}

func TestInvalidJWT(t *testing.T) {
	_, err := ValidateJWT("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestExpiredJWT(t *testing.T) {
	// Verify expired token is rejected
	expiredClaims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "admin",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	key, err := GetBignPrivateKey()
	if err != nil {
		t.Fatalf("GetBignPrivateKey: %v", err)
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodES256, expiredClaims)
	tokenString, err := expiredToken.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = ValidateJWT(tokenString)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestExpiredJWT_HS256Rejected(t *testing.T) {
	// Ensure HS256 tokens are rejected by our ES256-only validation
	expiredClaims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "admin",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	hs256Token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	secret, _ := GetJWTSecret()
	tokenString, err := hs256Token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign HS256 token: %v", err)
	}

	_, err = ValidateJWT(tokenString)
	if err == nil {
		t.Error("expected error for HS256 token validated as ES256")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	GetJWTSecret() // ensure JWT_SECRET is set
	token, hash, expiresAt, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	// Check that hash matches
	if HashRefreshToken(token) != hash {
		t.Error("hash mismatch")
	}

	// Check expiration is ~30 days
	expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
	if expiresAt.Before(expectedExpiry.Add(-time.Minute)) {
		t.Errorf("refresh token expires too early: %v", expiresAt)
	}
	if expiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("refresh token expires too late: %v", expiresAt)
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	// Generate first token pair
	token1, hash1, _, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("first GenerateRefreshToken failed: %v", err)
	}

	// Generate second token pair (simulating rotation)
	token2, hash2, _, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("second GenerateRefreshToken failed: %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("refresh tokens should be unique")
	}

	// Hashes should be different
	if hash1 == hash2 {
		t.Error("refresh token hashes should be different")
	}

	// Each hash should match its token
	if HashRefreshToken(token1) != hash1 {
		t.Error("hash1 should match token1")
	}
	if HashRefreshToken(token2) != hash2 {
		t.Error("hash2 should match token2")
	}

	// Token1's hash should NOT match token2
	if HashRefreshToken(token1) == hash2 {
		t.Error("token1 hash should not match token2")
	}
}

func TestHashRefreshToken(t *testing.T) {
	token := "test-refresh-token-value"
	hash1 := HashRefreshToken(token)
	hash2 := HashRefreshToken(token)

	if hash1 == "" {
		t.Error("expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Error("hash should be deterministic")
	}

	// Different tokens should produce different hashes
	otherHash := HashRefreshToken("different-token")
	if hash1 == otherHash {
		t.Error("different tokens should produce different hashes")
	}
}

func TestGenerateTempToken(t *testing.T) {
	token, err := GenerateTempToken("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateTempToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestValidateTempToken(t *testing.T) {
	token, err := GenerateTempToken("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateTempToken failed: %v", err)
	}

	claims, err := ValidateTempToken(token)
	if err != nil {
		t.Fatalf("ValidateTempToken failed: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("expected UserID 'user-1', got '%s'", claims.UserID)
	}
	if claims.Subject != "2fa_pending" {
		t.Errorf("expected Subject '2fa_pending', got '%s'", claims.Subject)
	}
	if claims.TenantID != "tenant-1" {
		t.Errorf("expected TenantID 'tenant-1', got '%s'", claims.TenantID)
	}
}

func TestValidateRegularTokenAsTempToken(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	_, err = ValidateTempToken(token)
	if err == nil {
		t.Error("expected error when validating regular JWT as temp token")
	}
}

func TestJWTAlgorithmES256(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	// Парсим без валидации для проверки alg
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsed, _, err := parser.ParseUnverified(token, &Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified failed: %v", err)
	}

	alg := parsed.Header["alg"]
	if alg != "ES256" {
		t.Fatalf("expected alg=ES256, got %v", alg)
	}
}

func TestTempTokenAlgorithmES256(t *testing.T) {
	token, err := GenerateTempToken("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateTempToken failed: %v", err)
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsed, _, err := parser.ParseUnverified(token, &Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified failed: %v", err)
	}

	alg := parsed.Header["alg"]
	if alg != "ES256" {
		t.Fatalf("expected alg=ES256, got %v", alg)
	}
}

func TestJWTWithRegion(t *testing.T) {
	token, err := GenerateJWTWithRegion("user-1", "testuser", "tech", "tenant-1", "RU")
	if err != nil {
		t.Fatalf("GenerateJWTWithRegion failed: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.Region != "RU" {
		t.Fatalf("expected Region='RU', got '%s'", claims.Region)
	}
}
