// Package auth — unit tests for AuthMiddleware.
// Соответствует:
//   - OWASP ASVS V3 (Session Management — session timeout)
//   - ISO 27001 A.9.4 (Access Control — session timeout enforcement)
//   - СТБ 34.101.27 п. 6.1 (Аутентификация)
package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Создаём валидный JWT
	token, err := GenerateJWT("user-1", "testuser", "admin", "tenant-1")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			t.Error("expected claims in context")
		}
		if claims.UserID != "user-1" {
			t.Errorf("expected UserID 'user-1', got '%s'", claims.UserID)
		}
		if claims.Role != "admin" {
			t.Errorf("expected Role 'admin', got '%s'", claims.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	errObj := resp["error"].(map[string]interface{})
	if errObj["code"] != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED code, got %v", errObj["code"])
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"empty bearer", "Bearer "},
		{"invalid scheme", "Basic token123"},
		{"malformed", "Bearer token1 token2"},
		{"no bearer prefix", "justatoken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// Создаём просроченный токен
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "admin",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthMiddleware_SessionTimeout(t *testing.T) {
	// Создаём токен с IssuedAt > 30 минут назад (должен быть отклонён)
	// Используем role="technician" (не admin, т.к. admin bypass session timeout)
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "technician",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-31 * time.Minute)), // > 30 min timeout
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for idle session")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for idle session > 30min, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	errObj := resp["error"].(map[string]interface{})
	if !strings.Contains(errObj["message"].(string), "session expired") {
		t.Errorf("expected 'session expired' message, got '%s'", errObj["message"])
	}
}

// ────────────────────────────────────────────────────────────────────────────
// P2-CR.4: Regional Session Policy Tests
// ────────────────────────────────────────────────────────────────────────────

func TestAuthMiddleware_AdminBypassSessionTimeout(t *testing.T) {
	// Admin bypass: токен с IssuedAt > 30 мин назад, но role="admin" → должен пропустить
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "admin",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-31 * time.Minute)), // > idle timeout
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Admin bypass должен пропустить запрос
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin bypass, got %d", w.Code)
	}
}

func TestAuthMiddleware_SuperadminBypassSessionTimeout(t *testing.T) {
	// Superadmin bypass: аналогично admin
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "superadmin",
		TenantID: "tenant-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-31 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for superadmin bypass, got %d", w.Code)
	}
}

func TestAuthMiddleware_SessionHeaders(t *testing.T) {
	// Проверка наличия session headers в ответе
	token, err := GenerateJWTWithRegion("user-1", "testuser", "technician", "tenant-1", "BY")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Проверяем наличие заголовков сессии
	if w.Header().Get("X-Session-Timeout") == "" {
		t.Error("expected X-Session-Timeout header")
	}
	if w.Header().Get("X-Session-Age") == "" {
		t.Error("expected X-Session-Age header")
	}
	if w.Header().Get("X-Session-Remaining") == "" {
		t.Error("expected X-Session-Remaining header")
	}
	if w.Header().Get("X-Session-Region") == "" {
		t.Error("expected X-Session-Region header")
	}
}

func TestAuthMiddleware_SessionRegionHeader(t *testing.T) {
	// Проверка что X-Session-Region соответствует региону из claims
	token, err := GenerateJWTWithRegion("user-1", "testuser", "technician", "tenant-1", "RU")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	if region := w.Header().Get("X-Session-Region"); region != "RU" {
		t.Errorf("expected X-Session-Region 'RU', got '%s'", region)
	}
}

func TestAuthMiddleware_RUSessionTimeout(t *testing.T) {
	// RU регион: idle timeout = 15 минут.
	// Создаём токен с IssuedAt > 15 минут назад — должен быть отклонён
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "technician",
		TenantID: "tenant-1",
		Region:   "RU",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-16 * time.Minute)), // > 15 min RU timeout
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for idle session (RU policy)")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for RU session > 15min idle, got %d", w.Code)
	}
}

func TestAuthMiddleware_BYSessionWithinTimeout(t *testing.T) {
	// BY регион: idle timeout = 30 минут.
	// Создаём токен с IssuedAt 10 минут назад — должен быть пропущен
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "technician",
		TenantID: "tenant-1",
		Region:   "BY",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-10 * time.Minute)), // < 30 min BY timeout
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for BY session within timeout, got %d", w.Code)
	}
}

func TestAuthMiddleware_SessionWarningHeader(t *testing.T) {
	// Создаём токен с IssuedAt почти на idleTimeout (осталось < WarningThreshold)
	// BY: idle 30m, warning 3m → создаём токен с IssuedAt 28m назад
	claims := Claims{
		UserID:   "user-1",
		Username: "testuser",
		Role:     "technician",
		TenantID: "tenant-1",
		Region:   "BY",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-28 * time.Minute)), // < 3m remaining → warning
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("get JWT secret: %v", err)
	}
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with warning, got %d", w.Code)
	}

	if w.Header().Get("X-Session-Warning") != "true" {
		t.Error("expected X-Session-Warning header to be 'true'")
	}
	if w.Header().Get("X-Session-Warning-In") == "" {
		t.Error("expected X-Session-Warning-In header")
	}
}

func TestGetClaims_NilContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	claims := GetClaims(req)
	if claims != nil {
		t.Error("expected nil claims for request without auth context")
	}
}
