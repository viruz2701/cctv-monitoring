package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// getTestJWTSecret — вспомогательная функция для тестов, возвращает JWT_SECRET
// или паникует если не установлен (в тестах это ок).
func getTestJWTSecret() []byte {
	secret, err := GetJWTSecret()
	if err != nil {
		panic("JWT_SECRET not set: " + err.Error())
	}
	return secret
}

func TestMain(m *testing.M) {
	// Set JWT secret for tests
	os.Setenv("JWT_SECRET", "test-secret-key-min-32-chars-long-for-hs256!")
	code := m.Run()
	os.Unsetenv("JWT_SECRET")
	os.Exit(code)
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
	// Verify that our regular token has correct TTL
	_, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

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
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, err := expiredToken.SignedString(getTestJWTSecret())
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = ValidateJWT(tokenString)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	_ = getTestJWTSecret() // ensure JWT_SECRET is set
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
