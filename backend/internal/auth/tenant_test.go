// Package auth — unit tests for TenantMiddleware (F-0.2.3).
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - ISO 27001 A.9.1.2 (Access control — tenant data separation)
//   - ISO 27001 A.9.2.3 (Privilege management — admin bypass)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testClaims возвращает Claims для тестов.
func testClaims(userID, username, role, tenantID string) *Claims {
	return &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		TenantID: tenantID,
	}
}

// setupTenantTest создаёт цепочку AuthMiddleware → TenantMiddleware → handler
// и возвращает ResponseRecorder.
func setupTenantTest(claims *Claims, handler http.HandlerFunc) *httptest.ResponseRecorder {
	// Сначала AuthMiddleware (кладёт claims в контекст)
	authHandler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Затем TenantMiddleware извлекает tenantID
		TenantMiddleware(handler).ServeHTTP(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	if claims != nil {
		// Создаём JWT с нужными claims
		token, err := GenerateJWT(claims.UserID, claims.Username, claims.Role, claims.TenantID)
		if err != nil {
			panic("failed to generate test JWT: " + err.Error())
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	authHandler.ServeHTTP(w, req)
	return w
}

func TestTenantMiddleware_ExtractsTenantID(t *testing.T) {
	var capturedTenantID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenantID = GetTenantID(r)
		w.WriteHeader(http.StatusOK)
	})

	_ = setupTenantTest(testClaims("user-1", "testuser", "technician", "tenant-42"), handler)

	if capturedTenantID != "tenant-42" {
		t.Errorf("expected TenantID 'tenant-42', got '%s'", capturedTenantID)
	}
}

func TestTenantMiddleware_AdminBypass(t *testing.T) {
	var capturedTenantID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenantID = GetTenantID(r)
		w.WriteHeader(http.StatusOK)
	})

	_ = setupTenantTest(testClaims("admin-1", "admin", "admin", "tenant-42"), handler)

	// Admin bypass: tenantID должен быть "*"
	if capturedTenantID != "*" {
		t.Errorf("expected TenantID '*' for admin bypass, got '%s'", capturedTenantID)
	}
}

func TestTenantMiddleware_NoClaims(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when claims are missing")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// Без JWT — TenantMiddleware не должен пропустить
	w := httptest.NewRecorder()
	TenantMiddleware(handler).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when claims are missing, got %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "UNAUTHORIZED" {
		t.Errorf("expected UNAUTHORIZED code, got %v", errObj["code"])
	}
}

func TestGetTenantID_NoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	tenantID := GetTenantID(req)
	if tenantID != "" {
		t.Errorf("expected empty tenantID, got '%s'", tenantID)
	}
}

func TestGetTenantRole_NoContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	role := GetTenantRole(req)
	if role != "" {
		t.Errorf("expected empty role, got '%s'", role)
	}
}

func TestGetTenantRole_ExtractsRole(t *testing.T) {
	var capturedRole string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRole = GetTenantRole(r)
		w.WriteHeader(http.StatusOK)
	})

	_ = setupTenantTest(testClaims("user-1", "testuser", "technician", "tenant-42"), handler)

	if capturedRole != "technician" {
		t.Errorf("expected role 'technician', got '%s'", capturedRole)
	}
}

func TestContextWithTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	ctx := ContextWithTenantID(req.Context(), "tenant-99")
	req = req.WithContext(ctx)

	tenantID := GetTenantID(req)
	if tenantID != "tenant-99" {
		t.Errorf("expected TenantID 'tenant-99', got '%s'", tenantID)
	}
}

func TestContextWithTenantRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	ctx := ContextWithTenantRole(req.Context(), "admin")
	req = req.WithContext(ctx)

	role := GetTenantRole(req)
	if role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", role)
	}
}

func TestTenantMiddleware_ChainsWithAuthMiddleware(t *testing.T) {
	// Полная цепочка: AuthMiddleware → TenantMiddleware → handler
	var capturedTenantID string
	var capturedRole string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTenantID = GetTenantID(r)
		capturedRole = GetTenantRole(r)
		w.WriteHeader(http.StatusOK)
	})

	w := setupTenantTest(testClaims("user-1", "testuser", "technician", "tenant-42"), handler)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedTenantID != "tenant-42" {
		t.Errorf("expected TenantID 'tenant-42', got '%s'", capturedTenantID)
	}
	if capturedRole != "technician" {
		t.Errorf("expected role 'technician', got '%s'", capturedRole)
	}
}
