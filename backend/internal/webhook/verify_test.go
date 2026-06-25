// Package webhook — тесты HMAC-верификации вебхуков.
//
// Соответствует:
//   - OWASP ASVS V6.3.1 (Integrity verification)
//   - IEC 62443 SR 3.1 (Communication integrity)
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// VerifyHMAC Tests
// ═══════════════════════════════════════════════════════════════════════

// computeHMAC вычисляет HMAC-SHA256 для тестов.
func computeHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyHMAC_ValidSignature(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`
	sig := computeHMAC(secret, body)

	if !VerifyHMAC(secret, sig, []byte(body)) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyHMAC_InvalidSignature(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`

	if VerifyHMAC(secret, "invalid-signature", []byte(body)) {
		t.Fatal("expected invalid signature")
	}
}

func TestVerifyHMAC_EmptySecret(t *testing.T) {
	body := `{"event":"test"}`
	// IEC 62443 SR 7.1: Empty secret = Fail Secure, reject
	if VerifyHMAC("", "any-signature", []byte(body)) {
		t.Fatal("expected reject with empty secret per IEC 62443 SR 7.1")
	}
}

func TestVerifyHMAC_EmptySignature(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`
	// Empty signature = reject
	if VerifyHMAC(secret, "", []byte(body)) {
		t.Fatal("expected fail with empty signature")
	}
}

func TestVerifyHMAC_TamperedBody(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`
	sig := computeHMAC(secret, body)

	// Tamper with body
	tamperedBody := `{"event":"test","data":"tampered"}`
	if VerifyHMAC(secret, sig, []byte(tamperedBody)) {
		t.Fatal("expected invalid signature for tampered body")
	}
}

func TestVerifyHMAC_WrongSecret(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`
	sig := computeHMAC(secret, body)

	// Different secret
	if VerifyHMAC("different-secret", sig, []byte(body)) {
		t.Fatal("expected invalid signature with wrong secret")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Signature Prefix Tests (as used by Jira with "sha256=")
// ═══════════════════════════════════════════════════════════════════════

func TestVerifyHMAC_WithPrefix(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`
	rawSig := computeHMAC(secret, body)
	prefixedSig := "sha256=" + rawSig

	if !VerifyHMAC(secret, prefixedSig, []byte(body), WithSignaturePrefix("sha256=")) {
		t.Fatal("expected valid signature with sha256= prefix")
	}
}

func TestVerifyHMAC_WithPrefixInvalid(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`
	rawSig := computeHMAC(secret, body)
	prefixedSig := "sha256=" + rawSig

	// Tampered body should still fail with prefix
	tampered := `{"event":"test","data":"tampered"}`
	if VerifyHMAC(secret, prefixedSig, []byte(tampered), WithSignaturePrefix("sha256=")) {
		t.Fatal("expected invalid signature for tampered body with prefix")
	}
}

func TestVerifyHMAC_WithPrefixWrongPrefix(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test","data":"hello"}`
	rawSig := computeHMAC(secret, body)
	prefixedSig := "md5=" + rawSig // Wrong prefix

	if VerifyHMAC(secret, prefixedSig, []byte(body), WithSignaturePrefix("sha256=")) {
		t.Fatal("expected invalid signature with wrong prefix (prefix not stripped)")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// VerifyMiddleware Tests
// ═══════════════════════════════════════════════════════════════════════

func TestVerifyMiddleware_ValidSignature(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`
	sig := computeHMAC(secret, body)

	handler := VerifyMiddleware(secret,
		WithSignatureHeader("X-Test-Signature"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Test-Signature", sig)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVerifyMiddleware_MissingSignature(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`

	handler := VerifyMiddleware(secret,
		WithSignatureHeader("X-Test-Signature"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(body)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "missing signature" {
		t.Errorf("expected 'missing signature' error, got '%s'", resp["error"])
	}
}

func TestVerifyMiddleware_InvalidSignature(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`

	handler := VerifyMiddleware(secret,
		WithSignatureHeader("X-Test-Signature"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Test-Signature", "invalid-signature")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestVerifyMiddleware_EmptySecret(t *testing.T) {
	// IEC 62443 SR 7.1: Empty secret = Fail Secure, reject with 500
	called := false
	handler := VerifyMiddleware("",
		WithSignatureHeader("X-Test-Signature"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/webhook", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("handler should NOT be called with empty secret per IEC 62443 SR 7.1")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// ServeHTTPWithVerify Tests
// ═══════════════════════════════════════════════════════════════════════

func TestServeHTTPWithVerify_Valid(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`
	sig := computeHMAC(secret, body)

	var capturedBody []byte
	handler := ServeHTTPWithVerify(secret, func(w http.ResponseWriter, r *http.Request, b []byte) {
		capturedBody = b
		JSONOK(w)
	}, WithSignatureHeader("X-Test-Signature"))

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Test-Signature", sig)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if string(capturedBody) != body {
		t.Errorf("expected body %q, got %q", body, string(capturedBody))
	}
}

func TestServeHTTPWithVerify_Invalid(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`

	handler := ServeHTTPWithVerify(secret, func(w http.ResponseWriter, r *http.Request, b []byte) {
		t.Error("handler should not be called")
	}, WithSignatureHeader("X-Test-Signature"))

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Test-Signature", "invalid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Edge Cases
// ═══════════════════════════════════════════════════════════════════════

func TestVerifyHMAC_EmptyBody(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := ""
	sig := computeHMAC(secret, body)

	if !VerifyHMAC(secret, sig, []byte(body)) {
		t.Fatal("expected valid signature for empty body")
	}
}

func TestVerifyHMAC_BinaryBody(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := []byte{0x00, 0x01, 0x02, 0xFF}
	sig := computeHMAC(secret, string(body))

	if !VerifyHMAC(secret, sig, body) {
		t.Fatal("expected valid signature for binary body")
	}
}

func TestDefaultHeaderName(t *testing.T) {
	secret := "test-secret-key-for-webhook-testing-12345"
	body := `{"event":"test"}`
	sig := computeHMAC(secret, body)

	handler := VerifyMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Signature-256", sig) // default header name
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with default header, got %d", rec.Code)
	}
}
