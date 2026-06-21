package audit

import (
	"testing"
)

func TestNewSigner(t *testing.T) {
	s := NewSigner("test-key-12345")
	if s == nil {
		t.Fatal("NewSigner returned nil")
	}
}

func TestSignAndVerify(t *testing.T) {
	s := NewSigner("my-secret-key")

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
	s := NewSigner("my-secret-key")

	data := "user1|create_wo|work_order|wo-001|{}|{}"
	signature := s.Sign(data)

	tampered := "user1|create_wo|work_order|wo-001|{}|{tampered}"
	if s.Verify(tampered, signature) {
		t.Error("Verify should return false for tampered data")
	}
}

func TestVerifyWrongSignature(t *testing.T) {
	s := NewSigner("my-secret-key")

	data := "user1|create_wo|work_order|wo-001|{}|{}"
	s.Sign(data)

	if s.Verify(data, "deadbeef") {
		t.Error("Verify should return false for wrong signature")
	}
}

func TestSignDifferentKeys(t *testing.T) {
	s1 := NewSigner("key-alpha")
	s2 := NewSigner("key-beta")

	data := "user1|action|entity|id|{}|{}"
	sig1 := s1.Sign(data)

	if s2.Verify(data, sig1) {
		t.Error("Verify should fail when using different keys")
	}
}

func TestSignDeterministic(t *testing.T) {
	s := NewSigner("consistent-key")

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

func TestSignEmptyKey(t *testing.T) {
	s := NewSigner("")

	data := "test|data"
	sig := s.Sign(data)

	if sig == "" {
		t.Fatal("Sign with empty key should still produce a signature")
	}

	if !s.Verify(data, sig) {
		t.Error("Verify should return true for empty-key signature")
	}
}
