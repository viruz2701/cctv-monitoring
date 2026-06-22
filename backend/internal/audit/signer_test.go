package audit

import (
	"testing"
)

// ── S1-07 Compliance Tests: Key Validation (ISO 27001 A.12.4.2, СТБ 34.101.30) ──

// validKey возвращает ключ длиной >= 32 байта для тестов.
func validKey() string {
	return "audit-hmac-key-32-bytes-12345678901" // 32 bytes
}

func TestNewSigner(t *testing.T) {
	s, err := NewSigner(validKey())
	if err != nil {
		t.Fatalf("NewSigner returned error: %v", err)
	}
	if s == nil {
		t.Fatal("NewSigner returned nil")
	}
}

func TestNewSignerKeyTooShort(t *testing.T) {
	_, err := NewSigner("short-key-16-bytes") // 16 bytes < 32
	if err == nil {
		t.Fatal("NewSigner should return error for key < 32 bytes")
	}
}

func TestNewSignerEmptyKey(t *testing.T) {
	_, err := NewSigner("")
	if err == nil {
		t.Fatal("NewSigner should return error for empty key")
	}
}

func TestNewSigner31BytesKey(t *testing.T) {
	// 31 bytes < 32 — должно быть ошибкой
	_, err := NewSigner("31-byte-key-1234567890abcd") // 31 bytes
	if err == nil {
		t.Fatal("NewSigner should return error for 31-byte key (need 32)")
	}
}

func TestNewSigner32BytesKey(t *testing.T) {
	// 32 bytes — минимально допустимо
	key := "this-is-a-32-byte-key-1234567890!" // 32 bytes
	s, err := NewSigner(key)
	if err != nil {
		t.Fatalf("NewSigner should accept 32-byte key: %v", err)
	}
	if s == nil {
		t.Fatal("NewSigner returned nil")
	}
}

func TestSignAndVerify(t *testing.T) {
	s, err := NewSigner(validKey())
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	data := "user1|create_wo|work_order|wo-001|{}|{}"
	signature := s.Sign(data)

	if signature == "" {
		t.Fatal("Sign returned empty signature")
	}
	if len(signature) != 64 {
		t.Errorf("expected 64 hex chars (SHA256), got %d", len(signature))
	}

	if !s.Verify(data, signature) {
		t.Error("Verify should return true for valid signature")
	}
}

func TestVerifyTamperedData(t *testing.T) {
	s, err := NewSigner(validKey())
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	data := "user1|create_wo|work_order|wo-001|{}|{}"
	signature := s.Sign(data)

	tampered := "user1|create_wo|work_order|wo-001|{}|{tampered}"
	if s.Verify(tampered, signature) {
		t.Error("Verify should return false for tampered data")
	}
}

func TestVerifyWrongSignature(t *testing.T) {
	s, err := NewSigner(validKey())
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	data := "user1|create_wo|work_order|wo-001|{}|{}"
	s.Sign(data)

	if s.Verify(data, "deadbeef") {
		t.Error("Verify should return false for wrong signature")
	}
}

func TestSignDifferentKeys(t *testing.T) {
	s1, err := NewSigner(validKey())
	if err != nil {
		t.Fatalf("NewSigner s1: %v", err)
	}
	s2, err := NewSigner("another-32-byte-key-for-test-123456") // 32 bytes
	if err != nil {
		t.Fatalf("NewSigner s2: %v", err)
	}

	data := "user1|action|entity|id|{}|{}"
	sig1 := s1.Sign(data)

	if s2.Verify(data, sig1) {
		t.Error("Verify should fail when using different keys")
	}
}

func TestSignDeterministic(t *testing.T) {
	s, err := NewSigner(validKey())
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	data := "user1|action|entity|id|{}|{}"
	sig1 := s.Sign(data)
	sig2 := s.Sign(data)

	if sig1 != sig2 {
		t.Error("Sign should be deterministic for same key and data")
	}
}

func TestSignAuditEntry(t *testing.T) {
	result := SignAuditEntry("user1", "create_wo", "work_order", "wo-001", []byte(`{}`), []byte(`{"status":"open"}`))

	expected := "user1|create_wo|work_order|wo-001|{}|{\"status\":\"open\"}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSignAuditEntryEmpty(t *testing.T) {
	result := SignAuditEntry("", "", "", "", nil, nil)
	expected := "|||||"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
