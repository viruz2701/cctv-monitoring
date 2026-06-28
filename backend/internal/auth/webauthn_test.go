package auth

import (
	"strings"
	"testing"
)

func TestGenerateRecoveryCodes_DefaultCount(t *testing.T) {
	codes, raw := GenerateRecoveryCodes(0)
	if len(codes) != 8 {
		t.Fatalf("expected 8 codes, got %d", len(codes))
	}
	if len(raw) != 8 {
		t.Fatalf("expected 8 raw codes, got %d", len(raw))
	}
}

func TestGenerateRecoveryCodes_CustomCount(t *testing.T) {
	codes, raw := GenerateRecoveryCodes(5)
	if len(codes) != 5 {
		t.Fatalf("expected 5 codes, got %d", len(codes))
	}
	if len(raw) != 5 {
		t.Fatalf("expected 5 raw codes, got %d", len(raw))
	}
}

func TestGenerateRecoveryCodes_Unique(t *testing.T) {
	_, raw := GenerateRecoveryCodes(20)
	seen := make(map[string]bool)
	for _, code := range raw {
		if seen[code] {
			t.Fatal("duplicate recovery code generated")
		}
		seen[code] = true
	}
}

func TestGenerateRecoveryCodes_Length(t *testing.T) {
	_, raw := GenerateRecoveryCodes(1)
	if len(raw[0]) != 12 {
		t.Fatalf("expected code length 12, got %d", len(raw[0]))
	}
}

func TestValidateRecoveryCode_Valid(t *testing.T) {
	codes, raw := GenerateRecoveryCodes(3)
	updated, valid := ValidateRecoveryCode(codes, raw[1])
	if !valid {
		t.Fatal("expected valid recovery code")
	}
	if !updated[1].Used {
		t.Fatal("expected code to be marked as used")
	}
}

func TestValidateRecoveryCode_Invalid(t *testing.T) {
	codes, _ := GenerateRecoveryCodes(3)
	_, valid := ValidateRecoveryCode(codes, "invalid-code-123")
	if valid {
		t.Fatal("expected invalid recovery code")
	}
}

func TestValidateRecoveryCode_AlreadyUsed(t *testing.T) {
	codes, raw := GenerateRecoveryCodes(3)
	codes[0].Used = true
	_, valid := ValidateRecoveryCode(codes, raw[0])
	if valid {
		t.Fatal("expected already-used code to be rejected")
	}
}

func TestHasRecoveryCodesLeft_AllAvailable(t *testing.T) {
	codes, _ := GenerateRecoveryCodes(3)
	if !HasRecoveryCodesLeft(codes) {
		t.Fatal("expected codes to be available")
	}
}

func TestHasRecoveryCodesLeft_NoneAvailable(t *testing.T) {
	codes, _ := GenerateRecoveryCodes(3)
	for i := range codes {
		codes[i].Used = true
	}
	if HasRecoveryCodesLeft(codes) {
		t.Fatal("expected no codes available")
	}
}

func TestGenerateRecoveryCodesJSON(t *testing.T) {
	_, jsonStr, err := GenerateRecoveryCodesJSON(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(jsonStr, "[") || !strings.HasSuffix(jsonStr, "]") {
		t.Fatal("expected JSON array")
	}
}

func TestWebAuthnStore_SaveAndGetUser(t *testing.T) {
	store := NewWebAuthnStore()
	store.SaveUser(&WebAuthnUserData{UserID: "user1", DisplayName: "Test User"})
	user, ok := store.GetUser("user1")
	if !ok {
		t.Fatal("expected user to exist")
	}
	if user.DisplayName != "Test User" {
		t.Fatalf("expected 'Test User', got %s", user.DisplayName)
	}
}

func TestWebAuthnStore_UserNotFound(t *testing.T) {
	store := NewWebAuthnStore()
	_, ok := store.GetUser("nonexistent")
	if ok {
		t.Fatal("expected user to not exist")
	}
}

func TestWebAuthnStore_SessionLifecycle(t *testing.T) {
	store := NewWebAuthnStore()
	sid := store.GenerateSessionID()
	if sid == "" {
		t.Fatal("expected non-empty session ID")
	}

	store.SaveSession(sid, &WebAuthnSessionData{UserID: "user1"})
	data, ok := store.GetSession(sid)
	if !ok {
		t.Fatal("expected session to exist")
	}
	if data.UserID != "user1" {
		t.Fatalf("expected user1, got %s", data.UserID)
	}

	// Session should be single-use
	_, ok = store.GetSession(sid)
	if ok {
		t.Fatal("expected session to be deleted after single use")
	}
}

func TestWebAuthnStore_GenerateSessionID_Unique(t *testing.T) {
	store := NewWebAuthnStore()
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := store.GenerateSessionID()
		if ids[id] {
			t.Fatal("duplicate session ID")
		}
		ids[id] = true
	}
}
