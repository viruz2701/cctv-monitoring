// Package auth — unit tests for SessionPolicy (P2-CR.4).
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — session timeout)
//   - ISO 27001 A.9.4 (Access control — session management)
//   - СТБ 34.101.27 п. 6.1 (Аутентификация — таймауты сессий)
//   - OWASP ASVS V3 (Session Management)
//   - Приказ ОАЦ №66 п. 7.18.2 (Защита сетей — управление сессиями)
package auth

import (
	"testing"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// SessionPolicy Tests
// ────────────────────────────────────────────────────────────────────────────

func TestGetSessionPolicy_BY(t *testing.T) {
	p := GetSessionPolicy(RegionBY)

	if p.IdleTimeout != 30*time.Minute {
		t.Errorf("BY IdleTimeout: expected 30m, got %s", p.IdleTimeout)
	}
	if p.AbsoluteTimeout != 8*time.Hour {
		t.Errorf("BY AbsoluteTimeout: expected 8h, got %s", p.AbsoluteTimeout)
	}
	if p.MaxConcurrentSessions != 3 {
		t.Errorf("BY MaxConcurrentSessions: expected 3, got %d", p.MaxConcurrentSessions)
	}
	if p.FailedLoginLockout != 5 {
		t.Errorf("BY FailedLoginLockout: expected 5, got %d", p.FailedLoginLockout)
	}
	if p.LockoutDuration != 15*time.Minute {
		t.Errorf("BY LockoutDuration: expected 15m, got %s", p.LockoutDuration)
	}
}

func TestGetSessionPolicy_RU(t *testing.T) {
	p := GetSessionPolicy(RegionRU)

	if p.IdleTimeout != 15*time.Minute {
		t.Errorf("RU IdleTimeout: expected 15m, got %s", p.IdleTimeout)
	}
	if p.AbsoluteTimeout != 4*time.Hour {
		t.Errorf("RU AbsoluteTimeout: expected 4h, got %s", p.AbsoluteTimeout)
	}
	if p.MaxConcurrentSessions != 2 {
		t.Errorf("RU MaxConcurrentSessions: expected 2, got %d", p.MaxConcurrentSessions)
	}
	if p.FailedLoginLockout != 5 {
		t.Errorf("RU FailedLoginLockout: expected 5, got %d", p.FailedLoginLockout)
	}
	if p.LockoutDuration != 30*time.Minute {
		t.Errorf("RU LockoutDuration: expected 30m, got %s", p.LockoutDuration)
	}
}

func TestGetSessionPolicy_EU(t *testing.T) {
	p := GetSessionPolicy(RegionEU)

	if p.IdleTimeout != 8*time.Hour {
		t.Errorf("EU IdleTimeout: expected 8h, got %s", p.IdleTimeout)
	}
	if p.AbsoluteTimeout != 24*time.Hour {
		t.Errorf("EU AbsoluteTimeout: expected 24h, got %s", p.AbsoluteTimeout)
	}
	if p.MaxConcurrentSessions != 5 {
		t.Errorf("EU MaxConcurrentSessions: expected 5, got %d", p.MaxConcurrentSessions)
	}
	if p.FailedLoginLockout != 10 {
		t.Errorf("EU FailedLoginLockout: expected 10, got %d", p.FailedLoginLockout)
	}
	if p.LockoutDuration != 15*time.Minute {
		t.Errorf("EU LockoutDuration: expected 15m, got %s", p.LockoutDuration)
	}
}

func TestGetSessionPolicy_US(t *testing.T) {
	p := GetSessionPolicy(RegionUS)

	if p.IdleTimeout != 30*time.Minute {
		t.Errorf("US IdleTimeout: expected 30m, got %s", p.IdleTimeout)
	}
	if p.AbsoluteTimeout != 8*time.Hour {
		t.Errorf("US AbsoluteTimeout: expected 8h, got %s", p.AbsoluteTimeout)
	}
	if p.MaxConcurrentSessions != 3 {
		t.Errorf("US MaxConcurrentSessions: expected 3, got %d", p.MaxConcurrentSessions)
	}
	if p.FailedLoginLockout != 5 {
		t.Errorf("US FailedLoginLockout: expected 5, got %d", p.FailedLoginLockout)
	}
	if p.LockoutDuration != 15*time.Minute {
		t.Errorf("US LockoutDuration: expected 15m, got %s", p.LockoutDuration)
	}
}

func TestGetSessionPolicy_CN(t *testing.T) {
	p := GetSessionPolicy(RegionCN)

	if p.IdleTimeout != 15*time.Minute {
		t.Errorf("CN IdleTimeout: expected 15m, got %s", p.IdleTimeout)
	}
	if p.AbsoluteTimeout != 4*time.Hour {
		t.Errorf("CN AbsoluteTimeout: expected 4h, got %s", p.AbsoluteTimeout)
	}
	if p.MaxConcurrentSessions != 2 {
		t.Errorf("CN MaxConcurrentSessions: expected 2, got %d", p.MaxConcurrentSessions)
	}
	if p.FailedLoginLockout != 5 {
		t.Errorf("CN FailedLoginLockout: expected 5, got %d", p.FailedLoginLockout)
	}
	if p.LockoutDuration != 15*time.Minute {
		t.Errorf("CN LockoutDuration: expected 15m, got %s", p.LockoutDuration)
	}
}

func TestGetSessionPolicy_UnknownRegion(t *testing.T) {
	// Неизвестный регион должен возвращать BY (fail secure)
	p := GetSessionPolicy("UNKNOWN")

	if p.IdleTimeout != 30*time.Minute {
		t.Errorf("unknown region IdleTimeout: expected 30m (BY default), got %s", p.IdleTimeout)
	}
	if p.AbsoluteTimeout != 8*time.Hour {
		t.Errorf("unknown region AbsoluteTimeout: expected 8h (BY default), got %s", p.AbsoluteTimeout)
	}
}

func TestDefaultSessionPolicy(t *testing.T) {
	p := DefaultSessionPolicy()

	// Default должен быть BY (наиболее строгие требования КИИ)
	if p.IdleTimeout != 30*time.Minute {
		t.Errorf("Default IdleTimeout: expected 30m (BY), got %s", p.IdleTimeout)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// WarningThreshold Tests
// ────────────────────────────────────────────────────────────────────────────

func TestWarningThreshold_BY(t *testing.T) {
	p := GetSessionPolicy(RegionBY)
	// BY: idle 30m → warning = 3m (10%)
	expected := 3 * time.Minute
	if p.WarningThreshold() != expected {
		t.Errorf("BY WarningThreshold: expected %s, got %s", expected, p.WarningThreshold())
	}
}

func TestWarningThreshold_RU(t *testing.T) {
	p := GetSessionPolicy(RegionRU)
	// RU: idle 15m → warning = 1m30s (10%), но минимум 1m
	expected := 1*time.Minute + 30*time.Second
	if p.WarningThreshold() != expected {
		t.Errorf("RU WarningThreshold: expected %s, got %s", expected, p.WarningThreshold())
	}
}

func TestWarningThreshold_EU(t *testing.T) {
	p := GetSessionPolicy(RegionEU)
	// EU: idle 8h → warning = 48m (10%)
	expected := 48 * time.Minute
	if p.WarningThreshold() != expected {
		t.Errorf("EU WarningThreshold: expected %s, got %s", expected, p.WarningThreshold())
	}
}

func TestWarningThreshold_Minimum(t *testing.T) {
	// Минимальный WarningThreshold — 1 минута
	policy := SessionPolicy{
		IdleTimeout: 30 * time.Second, // 10% = 3s, но минимум 1m
	}
	if policy.WarningThreshold() != 1*time.Minute {
		t.Errorf("minimum WarningThreshold: expected 1m, got %s", policy.WarningThreshold())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// IsAdminOverride Tests
// ────────────────────────────────────────────────────────────────────────────

func TestIsAdminOverride_Admin(t *testing.T) {
	if !IsAdminOverride("admin") {
		t.Error("expected true for admin role")
	}
}

func TestIsAdminOverride_Superadmin(t *testing.T) {
	if !IsAdminOverride("superadmin") {
		t.Error("expected true for superadmin role")
	}
}

func TestIsAdminOverride_NonAdmin(t *testing.T) {
	if IsAdminOverride("technician") {
		t.Error("expected false for technician role")
	}
}

func TestIsAdminOverride_Empty(t *testing.T) {
	if IsAdminOverride("") {
		t.Error("expected false for empty role")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// GenerateJWTWithRegion Tests
// ────────────────────────────────────────────────────────────────────────────

func TestGenerateJWTWithRegion(t *testing.T) {
	token, err := GenerateJWTWithRegion("user-1", "testuser", "technician", "tenant-1", "RU")
	if err != nil {
		t.Fatalf("GenerateJWTWithRegion: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}

	if claims.Region != "RU" {
		t.Errorf("expected Region 'RU', got '%s'", claims.Region)
	}
}

func TestGenerateJWTWithRegion_EmptyDefaultsToBY(t *testing.T) {
	token, err := GenerateJWTWithRegion("user-1", "testuser", "technician", "tenant-1", "")
	if err != nil {
		t.Fatalf("GenerateJWTWithRegion: %v", err)
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}

	if claims.Region != "BY" {
		t.Errorf("expected Region 'BY' for empty, got '%s'", claims.Region)
	}
}
