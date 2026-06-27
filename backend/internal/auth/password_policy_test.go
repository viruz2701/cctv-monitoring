// Package auth — unit tests for Regional Password Policies (P2-CR.3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.3: Regional Password Policy Tests
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing — table-driven)
//   - OWASP ASVS V2 (Authentication verification testing)
//   - СТБ 34.101.27 п. 7.4 (Тестирование безопасности)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// GetPasswordPolicy — regional policy tests
// ═══════════════════════════════════════════════════════════════════════════

func TestGetPasswordPolicyBY(t *testing.T) {
	policy := GetPasswordPolicy(RegionBY)

	if policy.MinLength != 12 {
		t.Errorf("BY MinLength = %d, want 12", policy.MinLength)
	}
	if policy.RotationDays != 90 {
		t.Errorf("BY RotationDays = %d, want 90", policy.RotationDays)
	}
	if policy.HistoryLength != 5 {
		t.Errorf("BY HistoryLength = %d, want 5", policy.HistoryLength)
	}
	if !policy.RequireUpper {
		t.Error("BY RequireUpper should be true")
	}
	if !policy.RequireLower {
		t.Error("BY RequireLower should be true")
	}
	if !policy.RequireDigit {
		t.Error("BY RequireDigit should be true")
	}
	if !policy.RequireSpecial {
		t.Error("BY RequireSpecial should be true")
	}
	if policy.MaxLength != 128 {
		t.Errorf("BY MaxLength = %d, want 128", policy.MaxLength)
	}
}

func TestGetPasswordPolicyRU(t *testing.T) {
	policy := GetPasswordPolicy(RegionRU)

	if policy.MinLength != 8 {
		t.Errorf("RU MinLength = %d, want 8", policy.MinLength)
	}
	if policy.RotationDays != 90 {
		t.Errorf("RU RotationDays = %d, want 90", policy.RotationDays)
	}
	if policy.HistoryLength != 5 {
		t.Errorf("RU HistoryLength = %d, want 5", policy.HistoryLength)
	}
	if !policy.RequireUpper {
		t.Error("RU RequireUpper should be true")
	}
}

func TestGetPasswordPolicyEU(t *testing.T) {
	policy := GetPasswordPolicy(RegionEU)

	if policy.MinLength != 8 {
		t.Errorf("EU MinLength = %d, want 8", policy.MinLength)
	}
	if policy.RotationDays != 0 {
		t.Errorf("EU RotationDays = %d, want 0 (no forced rotation)", policy.RotationDays)
	}
	if policy.RequireSpecial {
		t.Error("EU RequireSpecial should be false (NIST SP 800-63B)")
	}
	if policy.MaxLength != 0 {
		t.Errorf("EU MaxLength = %d, want 0 (no limit)", policy.MaxLength)
	}
}

func TestGetPasswordPolicyUS(t *testing.T) {
	policy := GetPasswordPolicy(RegionUS)

	if policy.MinLength != 8 {
		t.Errorf("US MinLength = %d, want 8", policy.MinLength)
	}
	if policy.RotationDays != 90 {
		t.Errorf("US RotationDays = %d, want 90", policy.RotationDays)
	}
	if policy.RequireSpecial {
		t.Error("US RequireSpecial should be false (NIST SP 800-63B)")
	}
}

func TestGetPasswordPolicyCN(t *testing.T) {
	policy := GetPasswordPolicy(RegionCN)

	if policy.MinLength != 8 {
		t.Errorf("CN MinLength = %d, want 8", policy.MinLength)
	}
	if policy.RotationDays != 90 {
		t.Errorf("CN RotationDays = %d, want 90", policy.RotationDays)
	}
	if policy.HistoryLength != 5 {
		t.Errorf("CN HistoryLength = %d, want 5", policy.HistoryLength)
	}
	if policy.MaxLength != 64 {
		t.Errorf("CN MaxLength = %d, want 64", policy.MaxLength)
	}
}

func TestGetPasswordPolicyDefault(t *testing.T) {
	policy := GetPasswordPolicy("XX")

	if policy.MinLength != 8 {
		t.Errorf("Default MinLength = %d, want 8", policy.MinLength)
	}
	if policy.RotationDays != 0 {
		t.Errorf("Default RotationDays = %d, want 0", policy.RotationDays)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ParseRegion
// ═══════════════════════════════════════════════════════════════════════════

func TestParseRegion(t *testing.T) {
	tests := []struct {
		input   string
		want    Region
		wantErr bool
	}{
		{"BY", RegionBY, false},
		{"RU", RegionRU, false},
		{"EU", RegionEU, false},
		{"US", RegionUS, false},
		{"CN", RegionCN, false},
		{"XX", "", true},
		{"", "", true},
		{"by", "", true}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseRegion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRegion(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseRegion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ValidRegions
// ═══════════════════════════════════════════════════════════════════════════

func TestValidRegions(t *testing.T) {
	if len(ValidRegions) != 5 {
		t.Errorf("ValidRegions length = %d, want 5", len(ValidRegions))
	}

	expected := []Region{RegionBY, RegionRU, RegionEU, RegionUS, RegionCN}
	for i, r := range expected {
		if ValidRegions[i] != r {
			t.Errorf("ValidRegions[%d] = %s, want %s", i, ValidRegions[i], r)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Region.String
// ═══════════════════════════════════════════════════════════════════════════

func TestRegionString(t *testing.T) {
	tests := []struct {
		region Region
		want   string
	}{
		{RegionBY, "Belarus (СТБ 34.101.27)"},
		{RegionRU, "Russia (ФСТЭК, 152-ФЗ)"},
		{RegionEU, "European Union (GDPR, NIST)"},
		{RegionUS, "United States (NIST SP 800-63B)"},
		{RegionCN, "China (MLPS 2.0)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.region), func(t *testing.T) {
			if got := tt.region.String(); got != tt.want {
				t.Errorf("Region.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
