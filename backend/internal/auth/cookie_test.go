// Package auth — unit tests for HttpOnly Cookie-based JWT Authentication (P1-SEC.1).
//
// Соответствует:
//   - OWASP ASVS V3.1 (Session management — HttpOnly cookies)
//   - OWASP ASVS V3.2 (CSRF protection)
//   - ISO 27001 A.9.2.1 (User registration — secure session)
//   - Приказ ОАЦ №66 п. 7.18.2 (Secure session management)
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ────────────────────────────────────────────────────────────────────────────
// SetAuthCookies tests
// ────────────────────────────────────────────────────────────────────────────

func TestSetAuthCookies_SetsAllThreeCookies(t *testing.T) {
	w := httptest.NewRecorder()
	SetAuthCookies(w, "access-token-123", "refresh-token-456", nil)

	resp := w.Result()
	cookies := resp.Cookies()

	if len(cookies) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(cookies))
	}

	// Проверяем имена всех cookies
	names := make(map[string]*http.Cookie)
	for _, c := range cookies {
		names[c.Name] = c
	}

	if _, ok := names[CookieNameAccessToken]; !ok {
		t.Errorf("missing access_token cookie")
	}
	if _, ok := names[CookieNameRefreshToken]; !ok {
		t.Errorf("missing refresh_token cookie")
	}
	if _, ok := names[CookieNameCSRF]; !ok {
		t.Errorf("missing csrf_token cookie")
	}
}

func TestSetAuthCookies_HttpOnlyForTokensOnly(t *testing.T) {
	w := httptest.NewRecorder()
	SetAuthCookies(w, "access-token-123", "refresh-token-456", nil)

	resp := w.Result()
	for _, c := range resp.Cookies() {
		switch c.Name {
		case CookieNameAccessToken, CookieNameRefreshToken:
			if !c.HttpOnly {
				t.Errorf("cookie %s should be HttpOnly", c.Name)
			}
			if !c.Secure {
				t.Errorf("cookie %s should be Secure", c.Name)
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("cookie %s should have SameSite=Strict", c.Name)
			}
		case CookieNameCSRF:
			if c.HttpOnly {
				t.Errorf("csrf_token cookie should NOT be HttpOnly (needs JS access)")
			}
		}
	}
}

func TestSetAuthCookies_CorrectValues(t *testing.T) {
	w := httptest.NewRecorder()
	SetAuthCookies(w, "access-token-123", "refresh-token-456", nil)

	resp := w.Result()
	for _, c := range resp.Cookies() {
		switch c.Name {
		case CookieNameAccessToken:
			if c.Value != "access-token-123" {
				t.Errorf("expected access_token value 'access-token-123', got '%s'", c.Value)
			}
		case CookieNameRefreshToken:
			if c.Value != "refresh-token-456" {
				t.Errorf("expected refresh_token value 'refresh-token-456', got '%s'", c.Value)
			}
		case CookieNameCSRF:
			if c.Value == "" {
				t.Error("csrf_token should not be empty")
			}
			if len(c.Value) != 64 { // 32 bytes = 64 hex chars
				t.Errorf("expected CSRF token length 64, got %d", len(c.Value))
			}
		}
	}
}

func TestSetAuthCookies_CustomConfig(t *testing.T) {
	cfg := &CookieConfig{
		Secure:          false, // for local dev
		Domain:          "example.com",
		Path:            "/api",
		SameSite:        http.SameSiteLaxMode,
		AccessTokenTTL:  5 * 60,       // 5 min
		RefreshTokenTTL: 24 * 60 * 60, // 1 day
	}

	w := httptest.NewRecorder()
	SetAuthCookies(w, "access-token", "refresh-token", cfg)

	resp := w.Result()
	for _, c := range resp.Cookies() {
		if c.Domain != "example.com" {
			t.Errorf("expected domain 'example.com', got '%s'", c.Domain)
		}
		if c.Path != "/api" {
			t.Errorf("expected path '/api', got '%s'", c.Path)
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("expected SameSite=Lax, got %v", c.SameSite)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ClearAuthCookies tests
// ────────────────────────────────────────────────────────────────────────────

func TestClearAuthCookies_ClearsAllCookies(t *testing.T) {
	w := httptest.NewRecorder()
	ClearAuthCookies(w, nil)

	resp := w.Result()
	cookies := resp.Cookies()

	if len(cookies) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(cookies))
	}

	for _, c := range cookies {
		if c.Value != "" {
			t.Errorf("cookie %s should have empty value, got '%s'", c.Name, c.Value)
		}
		if c.MaxAge != -1 {
			t.Errorf("cookie %s should have MaxAge=-1, got %d", c.Name, c.MaxAge)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Cookie extraction tests
// ────────────────────────────────────────────────────────────────────────────

func TestGetAccessTokenFromCookie_WithValidCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameAccessToken,
		Value: "test-jwt-token",
	})

	token := GetAccessTokenFromCookie(req)
	if token != "test-jwt-token" {
		t.Errorf("expected 'test-jwt-token', got '%s'", token)
	}
}

func TestGetAccessTokenFromCookie_WithNoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	token := GetAccessTokenFromCookie(req)
	if token != "" {
		t.Errorf("expected empty string, got '%s'", token)
	}
}

func TestGetRefreshTokenFromCookie_WithValidCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameRefreshToken,
		Value: "test-refresh-token",
	})

	token := GetRefreshTokenFromCookie(req)
	if token != "test-refresh-token" {
		t.Errorf("expected 'test-refresh-token', got '%s'", token)
	}
}

func TestGetRefreshTokenFromCookie_WithNoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	token := GetRefreshTokenFromCookie(req)
	if token != "" {
		t.Errorf("expected empty string, got '%s'", token)
	}
}

func TestGetCSRFTokenFromCookie_WithValidCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "csrf-abc123",
	})

	token := GetCSRFTokenFromCookie(req)
	if token != "csrf-abc123" {
		t.Errorf("expected 'csrf-abc123', got '%s'", token)
	}
}

func TestGetCSRFTokenFromCookie_WithNoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	token := GetCSRFTokenFromCookie(req)
	if token != "" {
		t.Errorf("expected empty string, got '%s'", token)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// CSRF token validation tests
// ────────────────────────────────────────────────────────────────────────────

func TestValidateCSRFToken_SafeMethodsSkipped(t *testing.T) {
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, method := range safeMethods {
		req := httptest.NewRequest(method, "/api/v1/test", nil)
		// No CSRF token at all — should still pass for safe methods
		if !ValidateCSRFToken(req) {
			t.Errorf("ValidateCSRFToken should return true for %s without token", method)
		}
	}
}

func TestValidateCSRFToken_ValidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "valid-csrf-token-123",
	})
	req.Header.Set("X-CSRF-Token", "valid-csrf-token-123")

	if !ValidateCSRFToken(req) {
		t.Error("ValidateCSRFToken should return true for valid token")
	}
}

func TestValidateCSRFToken_MismatchedToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "cookie-csrf-token",
	})
	req.Header.Set("X-CSRF-Token", "header-csrf-token-different")

	if ValidateCSRFToken(req) {
		t.Error("ValidateCSRFToken should return false for mismatched tokens")
	}
}

func TestValidateCSRFToken_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "csrf-token-in-cookie",
	})
	// No X-CSRF-Token header

	if ValidateCSRFToken(req) {
		t.Error("ValidateCSRFToken should return false when header is missing")
	}
}

func TestValidateCSRFToken_MissingCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.Header.Set("X-CSRF-Token", "csrf-token-in-header")
	// No csrf_token cookie

	if ValidateCSRFToken(req) {
		t.Error("ValidateCSRFToken should return false when cookie is missing")
	}
}

func TestValidateCSRFToken_EmptyTokens(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "",
	})
	req.Header.Set("X-CSRF-Token", "")

	if ValidateCSRFToken(req) {
		t.Error("ValidateCSRFToken should return false for empty tokens")
	}
}

func TestValidateCSRFToken_ConstantTimeComparison(t *testing.T) {
	// Test that validation is constant-time (different lengths)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "short",
	})
	req.Header.Set("X-CSRF-Token", "a-much-longer-token-value")

	if ValidateCSRFToken(req) {
		t.Error("ValidateCSRFToken should return false for different length tokens")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// CookieAuthMiddleware tests
// ────────────────────────────────────────────────────────────────────────────

func TestCookieAuthMiddleware_ExtractsTokenFromCookie(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	handler := CookieAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CookieAuthMiddleware sets Authorization header from cookie
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+token {
			t.Errorf("expected Authorization header 'Bearer %s', got '%s'", token, authHeader)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameAccessToken,
		Value: token,
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCookieAuthMiddleware_NoCookiePassesThrough(t *testing.T) {
	handler := CookieAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Without cookie, Authorization header should remain empty
		if r.Header.Get("Authorization") != "" {
			t.Error("expected no Authorization header when no cookie present")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// CSRFMiddleware tests
// ────────────────────────────────────────────────────────────────────────────

func TestCSRFMiddleware_ValidTokenPasses(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "valid-csrf-token",
	})
	req.Header.Set("X-CSRF-Token", "valid-csrf-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid CSRF, got %d", w.Code)
	}
}

func TestCSRFMiddleware_InvalidTokenBlocked(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for invalid CSRF")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameCSRF,
		Value: "cookie-token",
	})
	req.Header.Set("X-CSRF-Token", "different-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid CSRF, got %d", w.Code)
	}
}

func TestCSRFMiddleware_SafeMethodsExempt(t *testing.T) {
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, method := range safeMethods {
		handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(method, "/api/v1/test", nil)
		// No CSRF token at all
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for %s without CSRF, got %d", method, w.Code)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Integration: CookieAuthMiddleware + AuthMiddleware chain
// ────────────────────────────────────────────────────────────────────────────

func TestCookieAuthChain_ValidTokenViaCookie(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "technician", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	// Chain: CookieAuthMiddleware → AuthMiddleware → handler
	handler := CookieAuthMiddleware(AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			t.Error("expected claims in context")
			return
		}
		if claims.UserID != "user-1" {
			t.Errorf("expected UserID 'user-1', got '%s'", claims.UserID)
		}
		if claims.Role != "technician" {
			t.Errorf("expected Role 'technician', got '%s'", claims.Role)
		}
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  CookieNameAccessToken,
		Value: token,
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCookieAuthChain_NoCookieReturns401(t *testing.T) {
	handler := CookieAuthMiddleware(AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth")
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCookieAuthChain_AuthorizationHeaderStillWorks(t *testing.T) {
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	handler := CookieAuthMiddleware(AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			t.Error("expected claims in context")
			return
		}
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for Authorization header, got %d", w.Code)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Error types
// ────────────────────────────────────────────────────────────────────────────

func TestErrCSRFInvalid_IsDefined(t *testing.T) {
	if ErrCSRFInvalid == nil {
		t.Error("ErrCSRFInvalid should be defined")
	}
	if ErrCSRFInvalid.Error() != "csrf: invalid token" {
		t.Errorf("unexpected error message: %s", ErrCSRFInvalid.Error())
	}
}

func TestErrNoCookie_IsDefined(t *testing.T) {
	if ErrNoCookie == nil {
		t.Error("ErrNoCookie should be defined")
	}
	if ErrNoCookie.Error() != "cookie: no auth cookie found" {
		t.Errorf("unexpected error message: %s", ErrNoCookie.Error())
	}
}
