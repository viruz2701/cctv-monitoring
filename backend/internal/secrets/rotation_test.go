package secrets

import (
	"testing"
	"time"
)

// mockAuditLogger для тестов.
type mockAuditLogger struct {
	events []RotationEvent
}

func (m *mockAuditLogger) Log(event RotationEvent) {
	m.events = append(m.events, event)
}

func TestNewRotationManager(t *testing.T) {
	store := NewMemoryStore()
	audit := &mockAuditLogger{}
	rm := NewRotationManager(store, audit, nil)
	if rm == nil {
		t.Fatal("expected RotationManager")
	}
}

func TestGetCurrentSecret(t *testing.T) {
	store := NewMemoryStore()
	rm := NewRotationManager(store, nil, nil)

	secret, err := rm.GetCurrentSecret(SecretJWT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secret) == 0 {
		t.Fatal("expected non-empty secret")
	}
}

func TestGetCurrentSecret_Unknown(t *testing.T) {
	store := NewMemoryStore()
	rm := NewRotationManager(store, nil, nil)

	_, err := rm.GetCurrentSecret("unknown")
	if err == nil {
		t.Fatal("expected error for unknown secret type")
	}
}

func TestRotate_JWT(t *testing.T) {
	store := NewMemoryStore()
	audit := &mockAuditLogger{}
	rm := NewRotationManager(store, audit, nil)

	oldSecret, _ := rm.GetCurrentSecret(SecretJWT)

	err := rm.Rotate(SecretJWT, "manual")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSecret, _ := rm.GetCurrentSecret(SecretJWT)
	if newSecret == oldSecret {
		t.Fatal("expected secret to change after rotation")
	}

	// Проверяем audit log
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	if audit.events[0].Status != "success" {
		t.Fatalf("expected success status, got %s", audit.events[0].Status)
	}
	if audit.events[0].TriggeredBy != "manual" {
		t.Fatalf("expected manual trigger, got %s", audit.events[0].TriggeredBy)
	}
}

func TestRotate_HMAC(t *testing.T) {
	store := NewMemoryStore()
	audit := &mockAuditLogger{}
	rm := NewRotationManager(store, audit, nil)

	err := rm.Rotate(SecretHMAC, "scheduler")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
}

func TestRotate_VersionIncrement(t *testing.T) {
	store := NewMemoryStore()
	rm := NewRotationManager(store, nil, nil)

	rm.Rotate(SecretJWT, "test")
	rm.Rotate(SecretJWT, "test")
	rm.Rotate(SecretJWT, "test")

	secrets, _ := rm.GetValidSecrets(SecretJWT)
	_ = secrets
}

func TestGetValidSecrets_DuringGracePeriod(t *testing.T) {
	store := NewMemoryStore()
	rm := NewRotationManager(store, nil, nil)

	rm.Rotate(SecretJWT, "test")

	secrets, err := rm.GetValidSecrets(SecretJWT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Должны быть current + previous (grace period)
	if len(secrets) != 2 {
		t.Fatalf("expected 2 valid secrets during grace period, got %d", len(secrets))
	}
}

func TestRotate_UnknownType(t *testing.T) {
	store := NewMemoryStore()
	rm := NewRotationManager(store, nil, nil)

	err := rm.Rotate("unknown", "test")
	if err == nil {
		t.Fatal("expected error for unknown secret type")
	}
}

func TestRotate_StoresPrevious(t *testing.T) {
	store := NewMemoryStore()
	rm := NewRotationManager(store, nil, nil)

	first, _ := rm.GetCurrentSecret(SecretJWT)
	rm.Rotate(SecretJWT, "test")
	rm.Rotate(SecretJWT, "test")

	secrets, _ := rm.GetValidSecrets(SecretJWT)
	if secrets[0] == first {
		t.Fatal("expected current secret to change after rotation")
	}
}

func TestRotationEvent_Fields(t *testing.T) {
	event := RotationEvent{
		Timestamp:   time.Now(),
		SecretType:  SecretJWT,
		OldVersion:  1,
		NewVersion:  2,
		Status:      "success",
		TriggeredBy: "scheduler",
	}

	if event.SecretType != SecretJWT {
		t.Fatalf("expected JWT type, got %s", event.SecretType)
	}
	if event.Status != "success" {
		t.Fatalf("expected success, got %s", event.Status)
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := generateKey(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 64 { // 32 bytes = 64 hex chars
		t.Fatalf("expected key length 64 hex chars, got %d", len(key))
	}
}

func TestGenerateKey_Unique(t *testing.T) {
	k1, _ := generateKey(32)
	k2, _ := generateKey(32)
	if k1 == k2 {
		t.Fatal("expected unique keys")
	}
}

func TestFormatRotationInterval(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{90 * 24 * time.Hour, "3 months"},
		{365 * 24 * time.Hour, "1 years"},
		{60 * 24 * time.Hour, "2 months"},
		{30 * 24 * time.Hour, "1 months"},
	}

	for _, tt := range tests {
		result := FormatRotationInterval(tt.input)
		if result != tt.expected {
			t.Fatalf("FormatRotationInterval(%v) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Get(SecretJWT)
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestMemoryStore_SetAndGet(t *testing.T) {
	store := NewMemoryStore()
	entry := &SecretEntry{Current: "test-key", Version: 1}

	err := store.Set(SecretJWT, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := store.Get(SecretJWT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Current != "test-key" {
		t.Fatalf("expected test-key, got %s", got.Current)
	}
}
