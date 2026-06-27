package storage

import (
	"log/slog"
	"testing"

	"gb-telemetry-collector/internal/compliance"
)

func TestNewResidencyEnforcer(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	if e == nil {
		t.Fatal("enforcer must not be nil")
	}
}

func TestGetS3Endpoint(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	cfg, err := e.GetS3Endpoint(compliance.RegionBY)
	if err != nil {
		t.Fatalf("GetS3Endpoint(BY) error: %v", err)
	}
	if cfg.Endpoint != "s3.minsk.example.com:9000" {
		t.Errorf("expected BY endpoint, got %s", cfg.Endpoint)
	}
	if cfg.RetentionDays != 1825 {
		t.Errorf("expected BY retention 1825 days, got %d", cfg.RetentionDays)
	}

	cfg, err = e.GetS3Endpoint(compliance.RegionEU)
	if err != nil {
		t.Fatalf("GetS3Endpoint(EU) error: %v", err)
	}
	if cfg.RetentionDays != 730 {
		t.Errorf("expected EU retention 730 days, got %d", cfg.RetentionDays)
	}
}

func TestGetS3EndpointUnknown(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	_, err := e.GetS3Endpoint("UNKNOWN")
	if err == nil {
		t.Fatal("GetS3Endpoint for unknown region should error")
	}
}

func TestValidateDataAccessSameRegion(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	profile := compliance.NewINTLProfile()

	err := e.ValidateDataAccess(compliance.RegionINTL, compliance.RegionINTL, profile)
	if err != nil {
		t.Fatalf("same region access should be allowed: %v", err)
	}
}

func TestValidateDataAccessCrossBorderBY(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	// BY profile blocks cross-border
	profile := compliance.NewBYProfile()

	err := e.ValidateDataAccess(compliance.RegionEU, compliance.RegionBY, profile)
	if err == nil {
		t.Fatal("cross-border transfer from BY should be blocked")
	}
}

func TestValidateDataAccessCrossBorderINTL(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	// INTL allows cross-border
	profile := compliance.NewINTLProfile()

	err := e.ValidateDataAccess(compliance.RegionEU, compliance.RegionINTL, profile)
	if err != nil {
		t.Fatalf("INTL cross-border should be allowed: %v", err)
	}
}

func TestValidateDataAccessUnauthorizedRegion(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	// EU only allows EU region
	profile := compliance.NewEUProfile()

	err := e.ValidateDataAccess(compliance.RegionBY, compliance.RegionEU, profile)
	if err == nil {
		t.Fatal("access from BY to EU data should be blocked by EU profile")
	}
}

func TestViolationTracker(t *testing.T) {
	tracker := NewViolationTracker()

	stats := tracker.GetStats()
	if stats.TotalAttempts != 0 {
		t.Errorf("expected 0 attempts, got %d", stats.TotalAttempts)
	}

	tracker.Record(Violation{
		Type:    ViolationTypeCrossBorder,
		Blocked: true,
	})

	stats = tracker.GetStats()
	if stats.TotalAttempts != 1 {
		t.Errorf("expected 1 attempt, got %d", stats.TotalAttempts)
	}
	if stats.TotalBlocked != 1 {
		t.Errorf("expected 1 blocked, got %d", stats.TotalBlocked)
	}

	violations := tracker.GetViolations()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Type != ViolationTypeCrossBorder {
		t.Errorf("expected cross_border type, got %s", violations[0].Type)
	}
}

func TestViolationTrackerMaxCapacity(t *testing.T) {
	tracker := NewViolationTracker()

	// Record 1001 violations
	for i := 0; i < 1001; i++ {
		tracker.Record(Violation{Type: ViolationTypeStorageViolation})
	}

	violations := tracker.GetViolations()
	if len(violations) > 1000 {
		t.Errorf("violations should be capped at 1000, got %d", len(violations))
	}
}

func TestValidateStorageOperationNilContext(t *testing.T) {
	e := NewResidencyEnforcer(nil)
	err := e.ValidateStorageOperation(nil, compliance.RegionEU)
	if err == nil {
		t.Fatal("nil context should error")
	}
}

func TestCustomEndpoints(t *testing.T) {
	custom := map[string]S3EndpointConfig{
		"MY-REGION": {
			Endpoint: "custom.s3.example.com",
			Bucket:   "my-bucket",
			UseTLS:   true,
		},
	}
	e := NewResidencyEnforcer(custom)

	cfg, err := e.GetS3Endpoint("MY-REGION")
	if err != nil {
		t.Fatalf("GetS3Endpoint error: %v", err)
	}
	if cfg.Endpoint != "custom.s3.example.com" {
		t.Errorf("expected custom endpoint, got %s", cfg.Endpoint)
	}
}

func TestGetColdStorageEndpoint(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	cfg, err := e.GetColdStorageEndpoint(compliance.RegionBY)
	if err != nil {
		t.Fatalf("GetColdStorageEndpoint error: %v", err)
	}
	if cfg.Endpoint != "s3.minsk.example.com:9000" {
		t.Errorf("expected BY cold storage endpoint, got %s", cfg.Endpoint)
	}
}

func TestGetRetentionDays(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	days, err := e.GetRetentionDays(compliance.RegionBY)
	if err != nil {
		t.Fatalf("GetRetentionDays error: %v", err)
	}
	if days != 1825 {
		t.Errorf("expected 1825 days for BY, got %d", days)
	}
}

func TestNewResidencyEnforcerWithOptions(t *testing.T) {
	called := false
	auditFn := func(v Violation, tenantID string) {
		called = true
		if v.Type != ViolationTypeCrossBorder {
			t.Errorf("expected cross_border type, got %s", v.Type)
		}
		if tenantID != "test-tenant" {
			t.Errorf("expected tenantID 'test-tenant', got %s", tenantID)
		}
	}

	e := NewResidencyEnforcer(nil, WithAuditCallback(auditFn))
	if e == nil {
		t.Fatal("enforcer must not be nil")
	}

	// Trigger a cross-border violation which should call the audit callback
	_ = e.ValidateDataAccessWithTenant(
		compliance.RegionEU,
		compliance.RegionBY,
		compliance.NewBYProfile(),
		"test-tenant",
	)

	if !called {
		t.Error("audit callback was not called on cross-border violation")
	}
}

func TestGetMetrics(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	// Initially no violations
	stats := e.GetMetrics()
	if stats.TotalAttempts != 0 {
		t.Errorf("expected 0 attempts, got %d", stats.TotalAttempts)
	}

	// Trigger a violation
	_ = e.ValidateDataAccessWithTenant(
		compliance.RegionEU,
		compliance.RegionBY,
		compliance.NewBYProfile(),
		"tenant-1",
	)

	stats = e.GetMetrics()
	if stats.TotalAttempts != 1 {
		t.Errorf("expected 1 attempt, got %d", stats.TotalAttempts)
	}
	if stats.TotalBlocked != 1 {
		t.Errorf("expected 1 blocked, got %d", stats.TotalBlocked)
	}
}

func TestGetViolations(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	// Initially no violations
	violations := e.GetViolations()
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}

	// Trigger a violation
	_ = e.ValidateDataAccessWithTenant(
		compliance.RegionEU,
		compliance.RegionBY,
		compliance.NewBYProfile(),
		"tenant-1",
	)

	violations = e.GetViolations()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Type != ViolationTypeCrossBorder {
		t.Errorf("expected cross_border type, got %s", violations[0].Type)
	}
	if violations[0].RequestRegion != compliance.RegionEU {
		t.Errorf("expected request_region EU, got %s", violations[0].RequestRegion)
	}
	if violations[0].DataRegion != compliance.RegionBY {
		t.Errorf("expected data_region BY, got %s", violations[0].DataRegion)
	}
}

func TestValidateDataAccessWithTenant(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	// Same region — should pass
	err := e.ValidateDataAccessWithTenant(
		compliance.RegionBY,
		compliance.RegionBY,
		compliance.NewBYProfile(),
		"tenant-1",
	)
	if err != nil {
		t.Fatalf("same region access should be allowed: %v", err)
	}

	// Cross-border from BY — should be blocked
	err = e.ValidateDataAccessWithTenant(
		compliance.RegionEU,
		compliance.RegionBY,
		compliance.NewBYProfile(),
		"tenant-1",
	)
	if err == nil {
		t.Fatal("cross-border transfer from BY should be blocked")
	}

	// Cross-border INTL — should be allowed (INTL allows cross-border)
	err = e.ValidateDataAccessWithTenant(
		compliance.RegionEU,
		compliance.RegionINTL,
		compliance.NewINTLProfile(),
		"tenant-2",
	)
	if err != nil {
		t.Fatalf("INTL cross-border should be allowed: %v", err)
	}

	// RU profile blocks cross-border to non-RU
	err = e.ValidateDataAccessWithTenant(
		compliance.RegionEU,
		compliance.RegionRU,
		compliance.NewRUProfile(),
		"tenant-3",
	)
	if err == nil {
		t.Fatal("RU cross-border to EU should be blocked")
	}
}

func TestValidateDataAccessWithTenantNilProfile(t *testing.T) {
	e := NewResidencyEnforcer(nil)

	err := e.ValidateDataAccessWithTenant(compliance.RegionEU, compliance.RegionBY, nil, "tenant-1")
	if err == nil {
		t.Fatal("nil profile should error")
	}
}

func TestWithLoggerOption(t *testing.T) {
	e := NewResidencyEnforcer(nil, WithLogger(slog.Default()))
	if e == nil {
		t.Fatal("enforcer must not be nil")
	}

	// Verify it still works
	cfg, err := e.GetS3Endpoint(compliance.RegionBY)
	if err != nil {
		t.Fatalf("GetS3Endpoint error: %v", err)
	}
	if cfg.Endpoint != "s3.minsk.example.com:9000" {
		t.Errorf("expected BY endpoint, got %s", cfg.Endpoint)
	}
}

// TestAuditCallbackWithMultipleViolations проверяет, что callback
// вызывается для каждого нарушения, а не только для первого.
func TestAuditCallbackWithMultipleViolations(t *testing.T) {
	var callCount int
	auditFn := func(v Violation, tenantID string) {
		callCount++
	}

	e := NewResidencyEnforcer(nil, WithAuditCallback(auditFn))

	// Trigger 3 violations using BY profile (blocks all cross-border)
	_ = e.ValidateDataAccessWithTenant(compliance.RegionEU, compliance.RegionBY, compliance.NewBYProfile(), "t1")
	_ = e.ValidateDataAccessWithTenant(compliance.RegionINTL, compliance.RegionRU, compliance.NewBYProfile(), "t2")
	_ = e.ValidateDataAccessWithTenant(compliance.RegionUS, compliance.RegionEU, compliance.NewBYProfile(), "t3")

	if callCount != 3 {
		t.Errorf("expected 3 audit callback calls, got %d", callCount)
	}

	stats := e.GetMetrics()
	if stats.TotalAttempts != 3 {
		t.Errorf("expected 3 total attempts, got %d", stats.TotalAttempts)
	}
	if stats.TotalBlocked != 3 {
		t.Errorf("expected 3 blocked, got %d", stats.TotalBlocked)
	}
}
