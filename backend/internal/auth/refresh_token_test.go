// Package auth — Refresh Token Rotation tests (P1-HI-05).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-HI-05: Refresh Token Rotation Tests
//
// Тесты:
//  1. ComputeFingerprint — корректность вычисления fingerprint
//  2. GenerateRefreshToken — генерация opaque токена
//  3. RotateRefreshToken — успешная ротация
//  4. RotateRefreshToken_ExpiredToken — отклонение истёкшего токена
//  5. RotateRefreshToken_ReuseDetection — обнаружение reuse
//  6. RotateRefreshToken_FingerprintMismatch — несовпадение fingerprint
//  7. RotateRefreshToken_WithFamily — ротация с семьёй токенов
//  8. ClientIP — извлечение IP из запроса
//
// Compliance:
//   - OWASP ASVS V3.2.2 — Refresh token rotation
//   - OWASP ASVS V3.2.3 — Reuse detection
//   - OWASP ASVS V3.2.4 — Device binding
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ────────────────────────────────────────────────────────────────────────────
// Mock RefreshTokenStore
// ────────────────────────────────────────────────────────────────────────────

// mockSession хранит данные сессии для тестов.
type mockSession struct {
	session *RefreshSession
}

// mockStore реализует RefreshTokenStore для тестов.
type mockStore struct {
	mu       sync.RWMutex
	sessions map[string]*mockSession // key: token_hash
	families map[uuid.UUID][]string  // key: token_family → []token_hash
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*mockSession),
		families: make(map[uuid.UUID][]string),
	}
}

func (m *mockStore) CreateSession(userID, tokenHash, ipAddress, userAgent, fingerprintHash string, tokenFamily *uuid.UUID, expiresAt time.Time) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()
	s := &RefreshSession{
		ID:              id,
		UserID:          userID,
		TokenHash:       tokenHash,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
		FingerprintHash: fingerprintHash,
		TokenFamily:     tokenFamily,
		IsRevoked:       false,
		ExpiresAt:       expiresAt,
		CreatedAt:       time.Now(),
	}
	m.sessions[tokenHash] = &mockSession{session: s}

	if tokenFamily != nil {
		m.families[*tokenFamily] = append(m.families[*tokenFamily], tokenHash)
	}

	return id, nil
}

func (m *mockStore) GetSessionByTokenHash(tokenHash string) (*RefreshSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms, ok := m.sessions[tokenHash]
	if !ok {
		return nil, errors.New("session not found")
	}
	// Return a copy to avoid data races
	s := *ms.session
	return &s, nil
}

func (m *mockStore) RevokeSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ms := range m.sessions {
		if ms.session.ID == sessionID {
			ms.session.IsRevoked = true
			return nil
		}
	}
	return errors.New("session not found")
}

func (m *mockStore) RevokeTokenFamily(tokenFamily uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ms := range m.sessions {
		if ms.session.TokenFamily != nil && *ms.session.TokenFamily == tokenFamily {
			ms.session.IsRevoked = true
		}
	}
	return nil
}

func (m *mockStore) GetActiveSessionsByFamily(tokenFamily uuid.UUID) ([]*RefreshSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*RefreshSession
	for _, ms := range m.sessions {
		if ms.session.TokenFamily != nil && *ms.session.TokenFamily == tokenFamily &&
			!ms.session.IsRevoked && time.Now().Before(ms.session.ExpiresAt) {
			s := *ms.session
			result = append(result, &s)
		}
	}
	return result, nil
}

// addSession добавляет сессию в хранилище (для настройки тестов).
func (m *mockStore) addSession(tokenHash, userID string, isRevoked bool, expiresAt time.Time, fingerprintHash string, tokenFamily *uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()
	m.sessions[tokenHash] = &mockSession{
		session: &RefreshSession{
			ID:              id,
			UserID:          userID,
			TokenHash:       tokenHash,
			FingerprintHash: fingerprintHash,
			TokenFamily:     tokenFamily,
			IsRevoked:       isRevoked,
			ExpiresAt:       expiresAt,
			CreatedAt:       time.Now(),
		},
	}

	if tokenFamily != nil {
		m.families[*tokenFamily] = append(m.families[*tokenFamily], tokenHash)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Tests
// ────────────────────────────────────────────────────────────────────────────

func TestComputeFingerprint(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		ip        string
		wantEmpty bool
	}{
		{
			name:      "full fingerprint",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			ip:        "192.168.1.1",
			wantEmpty: false,
		},
		{
			name:      "only user agent",
			userAgent: "curl/7.68.0",
			ip:        "",
			wantEmpty: false,
		},
		{
			name:      "only ip",
			userAgent: "",
			ip:        "10.0.0.1",
			wantEmpty: false,
		},
		{
			name:      "empty both",
			userAgent: "",
			ip:        "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := ComputeFingerprint(tt.userAgent, tt.ip)
			if tt.wantEmpty {
				if fp != "" {
					t.Errorf("expected empty fingerprint, got %q", fp)
				}
				return
			}
			if fp == "" {
				t.Error("expected non-empty fingerprint")
			}
			// Deterministic
			fp2 := ComputeFingerprint(tt.userAgent, tt.ip)
			if fp != fp2 {
				t.Error("fingerprint should be deterministic")
			}
			// Different UA/ip should produce different fingerprint
			fp3 := ComputeFingerprint(tt.userAgent+"diff", tt.ip)
			if fp == fp3 {
				t.Error("different inputs should produce different fingerprints")
			}
		})
	}
}

func TestComputeFingerprint_UserAgentTruncated(t *testing.T) {
	longUA := string(make([]byte, 256))
	fp := ComputeFingerprint(longUA, "10.0.0.1")
	if fp == "" {
		t.Error("expected non-empty fingerprint even with long UA")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
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

	// Check prefix
	if len(token) < 3 || token[:3] != "rt_" {
		t.Errorf("expected token to start with 'rt_', got %q", token[:3])
	}

	// Hash should match
	if HashRefreshToken(token) != hash {
		t.Error("hash mismatch")
	}

	// Expiration should be ~30 days
	expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
	if expiresAt.Before(expectedExpiry.Add(-time.Minute)) {
		t.Errorf("expires too early: %v", expiresAt)
	}
	if expiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expires too late: %v", expiresAt)
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	token1, hash1, _, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("first GenerateRefreshToken failed: %v", err)
	}
	token2, hash2, _, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("second GenerateRefreshToken failed: %v", err)
	}

	if token1 == token2 {
		t.Error("tokens should be unique")
	}
	if hash1 == hash2 {
		t.Error("hashes should be different")
	}
	if HashRefreshToken(token1) != hash1 {
		t.Error("hash1 should match token1")
	}
	if HashRefreshToken(token2) != hash2 {
		t.Error("hash2 should match token2")
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

	otherHash := HashRefreshToken("different-token")
	if hash1 == otherHash {
		t.Error("different tokens should produce different hashes")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// RotateRefreshToken Tests
// ────────────────────────────────────────────────────────────────────────────

func TestRotateRefreshToken_Success(t *testing.T) {
	store := newMockStore()

	// Setup: create an active session
	oldHash := HashRefreshToken("old-token-value")
	store.addSession(oldHash, "user-1", false, time.Now().Add(24*time.Hour), "fp-hash-1", nil)

	// Execute rotation
	result, err := RotateRefreshToken(store, oldHash, "fp-hash-1", "user-1", "192.168.1.1", "test-agent")
	if err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}

	if result.NewToken == "" {
		t.Fatal("expected new token")
	}
	if result.NewTokenHash == "" {
		t.Fatal("expected new token hash")
	}
	if result.NewSessionID == "" {
		t.Fatal("expected new session ID")
	}
	if result.ReuseDetected {
		t.Error("should not detect reuse")
	}
	if result.RevokedFamily {
		t.Error("should not revoke family")
	}

	// Old session should be revoked
	oldSession, err := store.GetSessionByTokenHash(oldHash)
	if err != nil {
		t.Fatalf("get old session: %v", err)
	}
	if !oldSession.IsRevoked {
		t.Error("old session should be revoked")
	}

	// New session should be active
	newSession, err := store.GetSessionByTokenHash(result.NewTokenHash)
	if err != nil {
		t.Fatalf("get new session: %v", err)
	}
	if newSession.IsRevoked {
		t.Error("new session should not be revoked")
	}
	if newSession.FingerprintHash != "fp-hash-1" {
		t.Errorf("expected fingerprint 'fp-hash-1', got %q", newSession.FingerprintHash)
	}
}

func TestRotateRefreshToken_ExpiredToken(t *testing.T) {
	store := newMockStore()

	oldHash := HashRefreshToken("expired-token")
	store.addSession(oldHash, "user-1", false, time.Now().Add(-1*time.Hour), "fp-hash", nil)

	_, err := RotateRefreshToken(store, oldHash, "fp-hash", "user-1", "10.0.0.1", "agent")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !errors.Is(err, ErrRefreshTokenExpired) {
		t.Errorf("expected ErrRefreshTokenExpired, got %v", err)
	}
}

func TestRotateRefreshToken_ReuseDetection(t *testing.T) {
	store := newMockStore()

	// Setup: create a REVOKED session (simulating a rotated token being reused)
	family := uuid.New()
	oldHash := HashRefreshToken("stolen-token")
	store.addSession(oldHash, "user-1", true, time.Now().Add(24*time.Hour), "fp-hash", &family)

	// Add another active token in the same family
	activeHash := HashRefreshToken("active-family-token")
	store.addSession(activeHash, "user-1", false, time.Now().Add(24*time.Hour), "fp-hash-2", &family)

	// Attempt to use the revoked token — should detect reuse
	result, err := RotateRefreshToken(store, oldHash, "fp-hash", "user-1", "10.0.0.1", "agent")
	if err == nil {
		t.Fatal("expected error for reuse detection")
	}
	if !errors.Is(err, ErrRefreshTokenRevoked) {
		t.Errorf("expected ErrRefreshTokenRevoked, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.ReuseDetected {
		t.Error("expected reuse detected")
	}
	if !result.RevokedFamily {
		t.Error("expected entire family revoked")
	}

	// Verify the active token in the same family is also revoked
	activeSession, err := store.GetSessionByTokenHash(activeHash)
	if err != nil {
		t.Fatalf("get active session: %v", err)
	}
	if !activeSession.IsRevoked {
		t.Error("active session in same family should be revoked after reuse detection")
	}
}

func TestRotateRefreshToken_FingerprintMismatch(t *testing.T) {
	store := newMockStore()

	oldHash := HashRefreshToken("token-with-fingerprint")
	store.addSession(oldHash, "user-1", false, time.Now().Add(24*time.Hour), "original-fp", nil)

	// Try to rotate with DIFFERENT fingerprint
	_, err := RotateRefreshToken(store, oldHash, "different-fp", "user-1", "10.0.0.1", "agent")
	if err == nil {
		t.Fatal("expected error for fingerprint mismatch")
	}
	if !errors.Is(err, ErrFingerprintMismatch) {
		t.Errorf("expected ErrFingerprintMismatch, got %v", err)
	}
}

func TestRotateRefreshToken_FingerprintNoCheckWhenEmpty(t *testing.T) {
	store := newMockStore()

	// Session has no fingerprint (old token before migration)
	oldHash := HashRefreshToken("old-no-fp")
	store.addSession(oldHash, "user-1", false, time.Now().Add(24*time.Hour), "", nil)

	// No fingerprint provided either — should work
	_, err := RotateRefreshToken(store, oldHash, "", "user-1", "10.0.0.1", "agent")
	if err != nil {
		t.Fatalf("rotation should work without fingerprint: %v", err)
	}
}

func TestRotateRefreshToken_WithFamily(t *testing.T) {
	store := newMockStore()

	// Setup: create a session WITH token family
	family := uuid.New()
	oldHash := HashRefreshToken("family-token-1")
	store.addSession(oldHash, "user-1", false, time.Now().Add(24*time.Hour), "fp-hash", &family)

	// Rotate
	result, err := RotateRefreshToken(store, oldHash, "fp-hash", "user-1", "10.0.0.1", "agent")
	if err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}

	// New session should have the SAME family
	newSession, err := store.GetSessionByTokenHash(result.NewTokenHash)
	if err != nil {
		t.Fatalf("get new session: %v", err)
	}
	if newSession.TokenFamily == nil {
		t.Fatal("expected token family on new session")
	}
	if *newSession.TokenFamily != family {
		t.Errorf("expected same family %v, got %v", family, *newSession.TokenFamily)
	}
}

func TestRotateRefreshToken_MigratedTokenGetsFamily(t *testing.T) {
	store := newMockStore()

	// Session WITHOUT family (migrated from old system without family tracking)
	oldHash := HashRefreshToken("migrated-token")
	store.addSession(oldHash, "user-1", false, time.Now().Add(24*time.Hour), "fp-hash", nil)

	result, err := RotateRefreshToken(store, oldHash, "fp-hash", "user-1", "10.0.0.1", "agent")
	if err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}

	newSession, err := store.GetSessionByTokenHash(result.NewTokenHash)
	if err != nil {
		t.Fatalf("get new session: %v", err)
	}
	if newSession.TokenFamily == nil {
		t.Fatal("rotated token should have a new family")
	}
}

func TestRotateRefreshToken_TokenNotFound(t *testing.T) {
	store := newMockStore()
	_, err := RotateRefreshToken(store, HashRefreshToken("nonexistent"), "fp", "user-1", "10.0.0.1", "agent")
	if err == nil {
		t.Fatal("expected error for non-existent token")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ClientIP Tests
// ────────────────────────────────────────────────────────────────────────────

func TestClientIP_ForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.195")

	ip := ClientIP(r)
	if ip != "203.0.113.195" {
		t.Errorf("expected '203.0.113.195', got %q", ip)
	}
}

func TestClientIP_ForwardedForChain(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.195, 198.51.100.1, 192.0.2.1")

	ip := ClientIP(r)
	if ip != "203.0.113.195" {
		t.Errorf("expected first IP '203.0.113.195', got %q", ip)
	}
}

func TestClientIP_RealIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Real-IP", "10.0.0.5")

	ip := ClientIP(r)
	if ip != "10.0.0.5" {
		t.Errorf("expected '10.0.0.5', got %q", ip)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100:54321"

	ip := ClientIP(r)
	if ip != "192.168.1.100" {
		t.Errorf("expected '192.168.1.100', got %q", ip)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ValidateRefreshRequest Tests
// ────────────────────────────────────────────────────────────────────────────

func TestValidateRefreshRequest_FromCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  CookieNameRefreshToken,
		Value: "rt_test-token-from-cookie",
	})

	token, source, err := ValidateRefreshRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "rt_test-token-from-cookie" {
		t.Errorf("expected 'rt_test-token-from-cookie', got %q", token)
	}
	if source != "cookie" {
		t.Errorf("expected source 'cookie', got %q", source)
	}
}

func TestValidateRefreshRequest_FromBody(t *testing.T) {
	body := `{"refresh_token": "rt_test-token-from-body"}`
	r := httptest.NewRequest("POST", "/", nil)
	r.Body = &readCloser{body}

	token, source, err := ValidateRefreshRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "rt_test-token-from-body" {
		t.Errorf("expected 'rt_test-token-from-body', got %q", token)
	}
	if source != "body" {
		t.Errorf("expected source 'body', got %q", source)
	}
}

func TestValidateRefreshRequest_Empty(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.Body = &readCloser{body: "{}"}

	_, _, err := ValidateRefreshRequest(r)
	if err == nil {
		t.Fatal("expected error for empty request")
	}
}

// readCloser implements io.ReadCloser for test request bodies.
type readCloser struct {
	body string
}

func (r *readCloser) Read(p []byte) (n int, err error) {
	return copy(p, r.body), nil
}

func (r *readCloser) Close() error {
	return nil
}
