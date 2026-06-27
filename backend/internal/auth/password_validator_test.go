// Package auth — unit tests for Regional Password Validator (P2-CR.3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.3: Password Validator Tests
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing — table-driven)
//   - OWASP ASVS V2.1 (Authentication verification)
//   - NIST SP 800-63B (Verifier requirements)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// ValidatePassword — table-driven tests
// ═══════════════════════════════════════════════════════════════════════════

func TestValidatePasswordBY(t *testing.T) {
	policy := GetPasswordPolicy(RegionBY)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 12-char complex", "Abcd1234!@#$", false},
		{"valid with max length", "Abcd1234!@#$%^&*()_+", false},
		{"too short (11 chars)", "Abcd1234!@#", true},
		{"too short (8 chars)", "Ab1!abcd", true},
		{"no uppercase", "abcd1234!@#$", true},
		{"no lowercase", "ABCD1234!@#$", true},
		{"no digit", "Abcdefgh!@#$", true},
		{"no special", "Abcdefgh1234", true},
		{"empty password", "", true},
		{"over max length", makeLongString(129) + "Ab1!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(BY) error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordEU(t *testing.T) {
	policy := GetPasswordPolicy(RegionEU)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 8-char without special", "Abcd1234", false},
		{"valid with special", "Abcd1234!@#$", false},
		{"too short (7 chars)", "Ab1!abc", true},
		{"no uppercase", "abcd1234", true},
		{"no lowercase", "ABCD1234", true},
		{"no digit", "Abcdefgh", true},
		{"empty password", "", true},
		{"very long valid password", makeLongValidString(200), false}, // EU has no max length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(EU) error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordCN(t *testing.T) {
	policy := GetPasswordPolicy(RegionCN)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 8-char complex", "Abcd1234!", false},
		{"too short (7 chars)", "Ab1!abc", true},
		{"over max length", makeLongString(65) + "Ab1!", true},
		{"no special", "Abcd1234", true},
		{"no uppercase", "abcd1234!", true},
		{"empty password", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(CN) error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ValidatePasswordForRegion — convenience wrapper tests
// ═══════════════════════════════════════════════════════════════════════════

func TestValidatePasswordForRegion(t *testing.T) {
	tests := []struct {
		name     string
		region   Region
		password string
		wantErr  bool
	}{
		{"BY valid", RegionBY, "Abcd1234!@#$", false},
		{"BY invalid", RegionBY, "weak", true},
		{"RU valid", RegionRU, "Abcd1234!", false},
		{"EU valid no special", RegionEU, "Abcd1234", false}, // EU allows no special chars
		{"US valid no special", RegionUS, "Abcd1234", false}, // US allows no special chars
		{"CN valid", RegionCN, "Abcd1234!", false},
		{"unknown region uses default", "XX", "Abcd1234!", false}, // Default requires special chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordForRegion(tt.password, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordForRegion(%s) error = %v, wantErr = %v",
					tt.region, err, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// makeLongString creates a string of exactly n characters.
func makeLongString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'A'
	}
	return string(b)
}

// makeLongValidString creates a string of n characters that passes basic validation.
func makeLongValidString(n int) string {
	if n < 8 {
		n = 8
	}
	b := make([]byte, n)
	b[0] = 'A' // uppercase
	b[1] = 'b' // lowercase
	b[2] = '1' // digit
	b[3] = '!' // special
	for i := 4; i < n; i++ {
		b[i] = 'x' // lowercase filler
	}
	return string(b)
}
