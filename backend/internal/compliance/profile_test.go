// Package compliance — unit tests for ComplianceProfile abstraction.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.1: ComplianceProfile Tests
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - OWASP ASVS V5 (Input validation testing)
//   - СТБ 34.101.27 п. 7.4 (Тестирование безопасности)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// ComplianceProfile interface tests — 8 policy methods
// ═══════════════════════════════════════════════════════════════════════════

func TestProfileHas8Policies(t *testing.T) {
	// Проверяем, что каждый профиль реализует все 8 policy методов
	profiles := []ComplianceProfile{
		NewBYProfile(),
		NewEUProfile(),
		NewINTLProfile(),
	}

	for _, p := range profiles {
		t.Run(p.Region(), func(t *testing.T) {
			// 1. Crypto
			crypto := p.Crypto()
			if crypto.Provider == "" {
				t.Error("CryptoPolicy.Provider must not be empty")
			}
			if crypto.KeySize <= 0 {
				t.Error("CryptoPolicy.KeySize must be > 0")
			}

			// 2. Hash
			hash := p.Hash()
			if hash.Provider == "" {
				t.Error("HashPolicy.Provider must not be empty")
			}
			if hash.OutputSizeBits <= 0 {
				t.Error("HashPolicy.OutputSizeBits must be > 0")
			}

			// 3. Signature
			sig := p.Signature()
			if sig.Provider == "" {
				t.Error("SignaturePolicy.Provider must not be empty")
			}
			if sig.Curve == "" {
				t.Error("SignaturePolicy.Curve must not be empty")
			}

			// 4. Password
			pwd := p.Password()
			if pwd.HashProvider == "" {
				t.Error("PasswordPolicy.HashProvider must not be empty")
			}
			if pwd.MinLength < 8 {
				t.Errorf("PasswordPolicy.MinLength must be >= 8, got %d", pwd.MinLength)
			}

			// 5. Data Residency
			res := p.DataResidency()
			if len(res.AllowedRegions) == 0 {
				t.Error("DataResidencyPolicy.AllowedRegions must not be empty")
			}

			// 6. Retention
			ret := p.Retention()
			if ret.AuditLogDays <= 0 {
				t.Error("RetentionPolicy.AuditLogDays must be > 0")
			}

			// 7. Audit
			audit := p.Audit()
			if audit.RetentionYears <= 0 {
				t.Error("AuditPolicy.RetentionYears must be > 0")
			}

			// 8. Session
			sess := p.Session()
			if sess.IdleTimeoutMinutes <= 0 {
				t.Error("SessionPolicy.IdleTimeoutMinutes must be > 0")
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Profile validation tests
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile ComplianceProfile
		wantErr bool
	}{
		{
			name:    "nil profile",
			profile: nil,
			wantErr: true,
		},
		{
			name:    "BY profile valid",
			profile: NewBYProfile(),
			wantErr: false,
		},
		{
			name:    "EU profile valid",
			profile: NewEUProfile(),
			wantErr: false,
		},
		{
			name:    "INTL profile valid",
			profile: NewINTLProfile(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfile() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Profile switching tests
// ═══════════════════════════════════════════════════════════════════════════

func TestProfileSwitching(t *testing.T) {
	registry := NewProfileRegistry(
		WithRequiredRegions(RegionINTL),
		WithProfile(NewBYProfile()),
		WithProfile(NewEUProfile()),
		WithProfile(NewINTLProfile()),
	)

	tests := []struct {
		name          string
		region        string
		wantCrypto    CryptoProviderType
		wantHash      HashProviderType
		wantSignature SignatureProviderType
	}{
		{
			name:          "BY profile → belt-gcm, bash-256, bign-curve256v1",
			region:        RegionBY,
			wantCrypto:    CryptoBeltGCM,
			wantHash:      HashBash256,
			wantSignature: SignatureBignCurve256,
		},
		{
			name:          "EU profile → aes-256-gcm, sha256, es256",
			region:        RegionEU,
			wantCrypto:    CryptoAES256GCM,
			wantHash:      HashSHA256,
			wantSignature: SignatureES256,
		},
		{
			name:          "INTL profile → aes-256-gcm, sha256, es256",
			region:        RegionINTL,
			wantCrypto:    CryptoAES256GCM,
			wantHash:      HashSHA256,
			wantSignature: SignatureES256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := registry.Get(tt.region)
			if err != nil {
				t.Fatalf("Get(%s) error: %v", tt.region, err)
			}

			if profile.Crypto().Provider != tt.wantCrypto {
				t.Errorf("Crypto provider = %s, want %s", profile.Crypto().Provider, tt.wantCrypto)
			}
			if profile.Hash().Provider != tt.wantHash {
				t.Errorf("Hash provider = %s, want %s", profile.Hash().Provider, tt.wantHash)
			}
			if profile.Signature().Provider != tt.wantSignature {
				t.Errorf("Signature provider = %s, want %s", profile.Signature().Provider, tt.wantSignature)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Region-specific policy tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBYProfileCIISPolicies(t *testing.T) {
	p := NewBYProfile()

	// КИИ: MFA обязателен
	if !p.Password().RequireMFA {
		t.Error("BY profile: MFA must be required for КИИ")
	}

	// КИИ: 30 мин idle timeout
	if p.Session().IdleTimeoutMinutes != 30 {
		t.Errorf("BY profile: idle timeout must be 30min, got %d", p.Session().IdleTimeoutMinutes)
	}

	// КИИ: 1 concurrent session
	if p.Session().MaxConcurrentSessions != 1 {
		t.Errorf("BY profile: max concurrent sessions must be 1, got %d", p.Session().MaxConcurrentSessions)
	}

	// КИИ: No cross-border transfer
	if p.DataResidency().CrossBorderTransferAllowed {
		t.Error("BY profile: cross-border transfer must be forbidden for КИИ")
	}

	// КИИ: 5 years audit retention
	if p.Audit().RetentionYears != 5 {
		t.Errorf("BY profile: audit retention must be 5 years, got %d", p.Audit().RetentionYears)
	}

	// КИИ: Chain hash required (tamper detection)
	if !p.Audit().ChainHashPrev {
		t.Error("BY profile: chain hash must be required for КИИ")
	}

	// КИИ: HMAC required
	if !p.Audit().HMACRequired {
		t.Error("BY profile: HMAC must be required for КИИ")
	}

	// КИИ: TLS 1.3 minimum
	if p.Crypto().TLSMinVersion != "1.3" {
		t.Errorf("BY profile: TLS min version must be 1.3, got %s", p.Crypto().TLSMinVersion)
	}
}

func TestEUProfileGDPRPolicies(t *testing.T) {
	p := NewEUProfile()

	// GDPR: Cross-border with SCC is allowed
	if !p.DataResidency().CrossBorderTransferAllowed {
		t.Error("EU profile: cross-border transfer must be allowed with SCC")
	}

	// GDPR: 2 years audit retention
	if p.Audit().RetentionYears != 2 {
		t.Errorf("EU profile: audit retention must be 2 years, got %d", p.Audit().RetentionYears)
	}

	// GDPR: Encryption at rest required
	if !p.DataResidency().RequireEncryptionAtRest {
		t.Error("EU profile: encryption at rest must be required")
	}

	// NIST: No forced password rotation
	if p.Password().MaxAgeDays != 0 {
		t.Errorf("EU profile: password max age should be 0 (no forced rotation), got %d", p.Password().MaxAgeDays)
	}

	// EU: only EU region
	allowed := p.DataResidency().AllowedRegions
	if len(allowed) != 1 || allowed[0] != RegionEU {
		t.Errorf("EU profile: allowed regions must be [EU], got %v", allowed)
	}
}

func TestINTLProfileBaselinePolicies(t *testing.T) {
	p := NewINTLProfile()

	// INTL: Flexibile data residency
	if !p.DataResidency().CrossBorderTransferAllowed {
		t.Error("INTL profile: cross-border transfer must be allowed")
	}

	// INTL: 1 year audit retention
	if p.Audit().RetentionYears != 1 {
		t.Errorf("INTL profile: audit retention must be 1 year, got %d", p.Audit().RetentionYears)
	}

	// INTL: HMAC optional
	if p.Audit().HMACRequired {
		t.Error("INTL profile: HMAC must be optional")
	}

	// INTL: 120 min idle timeout
	if p.Session().IdleTimeoutMinutes != 120 {
		t.Errorf("INTL profile: idle timeout must be 120min, got %d", p.Session().IdleTimeoutMinutes)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Default policies sanity checks
// ═══════════════════════════════════════════════════════════════════════════

func TestDefaultPoliciesAreValid(t *testing.T) {
	// Проверяем, что все default политики валидны и не nil
	if dp := DefaultCryptoPolicy(); dp.Provider == "" {
		t.Error("DefaultCryptoPolicy must have valid provider")
	}
	if dp := DefaultHashPolicy(); dp.Provider == "" {
		t.Error("DefaultHashPolicy must have valid provider")
	}
	if dp := DefaultSignaturePolicy(); dp.Provider == "" {
		t.Error("DefaultSignaturePolicy must have valid provider")
	}
	if dp := DefaultPasswordPolicy(); dp.HashProvider == "" {
		t.Error("DefaultPasswordPolicy must have valid hash provider")
	}
	if dp := DefaultDataResidencyPolicy(); len(dp.AllowedRegions) == 0 {
		t.Error("DefaultDataResidencyPolicy must have allowed regions")
	}
	if dp := DefaultRetentionPolicy(); dp.AuditLogDays <= 0 {
		t.Error("DefaultRetentionPolicy must have positive audit log days")
	}
	if dp := DefaultAuditPolicy(); dp.RetentionYears <= 0 {
		t.Error("DefaultAuditPolicy must have positive retention years")
	}
	if dp := DefaultSessionPolicy(); dp.IdleTimeoutMinutes <= 0 {
		t.Error("DefaultSessionPolicy must have positive idle timeout")
	}
}
