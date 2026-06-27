// Package compliance — unit tests for ProfileRegistry.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.1: Registry Tests
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - OWASP ASVS V5 (Input validation testing)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"sync"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Registry creation and basic operations
// ═══════════════════════════════════════════════════════════════════════════

func TestNewProfileRegistry(t *testing.T) {
	r := NewProfileRegistry()
	if r == nil {
		t.Fatal("NewProfileRegistry() must not return nil")
	}
	if r.Count() != 0 {
		t.Errorf("new registry must be empty, got %d profiles", r.Count())
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewProfileRegistry()

	if err := r.Register(NewBYProfile()); err != nil {
		t.Fatalf("Register BY profile error: %v", err)
	}

	if err := r.Register(NewEUProfile()); err != nil {
		t.Fatalf("Register EU profile error: %v", err)
	}

	// Get existing profile
	byProfile, err := r.Get(RegionBY)
	if err != nil {
		t.Fatalf("Get(BY) error: %v", err)
	}
	if byProfile.Region() != RegionBY {
		t.Errorf("expected region BY, got %s", byProfile.Region())
	}

	euProfile, err := r.Get(RegionEU)
	if err != nil {
		t.Fatalf("Get(EU) error: %v", err)
	}
	if euProfile.Region() != RegionEU {
		t.Errorf("expected region EU, got %s", euProfile.Region())
	}
}

func TestGetNonExistentProfile(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewINTLProfile())

	// Unknown region should return INTL fallback
	profile, err := r.Get("UNKNOWN")
	if err != nil {
		t.Fatalf("Get(UNKNOWN) should fallback to INTL, got error: %v", err)
	}
	if profile.Region() != RegionINTL {
		t.Errorf("fallback should return INTL profile, got %s", profile.Region())
	}
}

func TestGetNoFallback(t *testing.T) {
	r := NewProfileRegistry()
	// Empty registry, no INTL profile

	_, err := r.Get("UNKNOWN")
	if err == nil {
		t.Fatal("Get(UNKNOWN) should return error when no profiles registered")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Registration validation
// ═══════════════════════════════════════════════════════════════════════════

func TestRegisterDuplicateProfile(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewBYProfile())

	err := r.Register(NewBYProfile())
	if err == nil {
		t.Fatal("Register duplicate BY profile should return error")
	}
}

func TestRegisterNilProfile(t *testing.T) {
	r := NewProfileRegistry()
	err := r.Register(nil)
	if err == nil {
		t.Fatal("Register nil profile should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// MustRegister and MustGet
// ═══════════════════════════════════════════════════════════════════════════

func TestMustRegister(t *testing.T) {
	r := NewProfileRegistry()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustRegister should not panic for valid profile: %v", r)
		}
	}()

	r.MustRegister(NewINTLProfile())
}

func TestMustRegisterPanicsOnDuplicate(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewINTLProfile())

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustRegister should panic on duplicate registration")
		}
	}()

	r.MustRegister(NewINTLProfile())
}

func TestMustGet(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewINTLProfile())

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustGet should not panic for existing profile: %v", r)
		}
	}()

	_ = r.MustGet(RegionINTL)
}

func TestMustGetPanicsOnMissing(t *testing.T) {
	r := NewProfileRegistry()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustGet should panic for missing profile")
		}
	}()

	_ = r.MustGet(RegionBY)
}

// ═══════════════════════════════════════════════════════════════════════════
// IsRegistered, List, Count
// ═══════════════════════════════════════════════════════════════════════════

func TestIsRegistered(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewBYProfile())

	if !r.IsRegistered(RegionBY) {
		t.Error("IsRegistered(BY) should be true")
	}
	if r.IsRegistered(RegionEU) {
		t.Error("IsRegistered(EU) should be false")
	}
}

func TestList(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewBYProfile())
	r.MustRegister(NewEUProfile())
	r.MustRegister(NewINTLProfile())

	regions := r.List()
	if len(regions) != 3 {
		t.Errorf("List should return 3 regions, got %d: %v", len(regions), regions)
	}

	regionMap := make(map[string]bool)
	for _, region := range regions {
		regionMap[region] = true
	}

	if !regionMap[RegionBY] {
		t.Error("List should include BY")
	}
	if !regionMap[RegionEU] {
		t.Error("List should include EU")
	}
	if !regionMap[RegionINTL] {
		t.Error("List should include INTL")
	}
}

func TestCount(t *testing.T) {
	r := NewProfileRegistry()
	if r.Count() != 0 {
		t.Errorf("empty registry count should be 0, got %d", r.Count())
	}

	r.MustRegister(NewBYProfile())
	if r.Count() != 1 {
		t.Errorf("count should be 1, got %d", r.Count())
	}

	r.MustRegister(NewEUProfile())
	if r.Count() != 2 {
		t.Errorf("count should be 2, got %d", r.Count())
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Startup validation
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateRegistryWithRequiredProfiles(t *testing.T) {
	r := NewProfileRegistry(
		WithRequiredRegions(RegionINTL),
	)
	r.MustRegister(NewINTLProfile())

	if err := r.Validate(); err != nil {
		t.Fatalf("Validate() should pass with INTL profile: %v", err)
	}
}

func TestValidateRegistryMissingRequiredProfile(t *testing.T) {
	r := NewProfileRegistry(
		WithRequiredRegions(RegionBY),
	)

	err := r.Validate()
	if err == nil {
		t.Fatal("Validate() should fail when BY profile is required but not registered")
	}
}

func TestValidateRegistryMultipleRequired(t *testing.T) {
	r := NewProfileRegistry(
		WithRequiredRegions(RegionBY, RegionEU, RegionINTL),
	)
	r.MustRegister(NewBYProfile())
	r.MustRegister(NewEUProfile())
	// INTL intentionally missing

	err := r.Validate()
	if err == nil {
		t.Fatal("Validate() should fail when INTL is required but not registered")
	}
}

func TestRegisterBaselineProfiles(t *testing.T) {
	// This should not panic
	registry := RegisterBaselineProfiles(nil)
	if registry == nil {
		t.Fatal("RegisterBaselineProfiles must return non-nil registry")
	}

	// BY, RU, EU, INTL = 4 baseline profiles
	if registry.Count() != 4 {
		t.Errorf("expected 4 baseline profiles, got %d", registry.Count())
	}

	// Verify all required profiles exist
	if err := registry.Validate(); err != nil {
		t.Fatalf("baseline registry validation failed: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Thread safety test
// ═══════════════════════════════════════════════════════════════════════════

func TestRegistryThreadSafety(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewINTLProfile())

	var wg sync.WaitGroup
	concurrentGets := 50

	// Concurrent reads
	for i := 0; i < concurrentGets; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			profile, err := r.Get(RegionINTL)
			if err != nil {
				t.Errorf("concurrent Get error: %v", err)
				return
			}
			if profile == nil {
				t.Error("concurrent Get returned nil profile")
			}
		}()
	}

	// Concurrent List
	for i := 0; i < concurrentGets; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.List()
			_ = r.Count()
			_ = r.IsRegistered(RegionINTL)
		}()
	}

	wg.Wait()
}

func TestRegistryConcurrentRegisterAndGet(t *testing.T) {
	r := NewProfileRegistry()
	r.MustRegister(NewINTLProfile())

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer
	go func() {
		defer wg.Done()
		_ = r.Register(NewBYProfile())
		_ = r.Register(NewEUProfile())
	}()

	// Reader
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			r.IsRegistered(RegionBY)
			r.IsRegistered(RegionEU)
			r.IsRegistered(RegionINTL)
			_ = r.List()
			_ = r.Count()
		}
	}()

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════════
// Registry with options
// ═══════════════════════════════════════════════════════════════════════════

func TestRegistryWithDefaultProfile(t *testing.T) {
	r := NewProfileRegistry(
		WithDefaultProfile(RegionBY),
	)

	// Verify default is set internally (BY should be defaultKey)
	// This is implicit — no direct accessor, but used for fallback logic
	if r.defaultKey != RegionBY {
		t.Errorf("expected default key BY, got %s", r.defaultKey)
	}
}

func TestRegistryWithProfileOption(t *testing.T) {
	r := NewProfileRegistry(
		WithProfile(NewBYProfile()),
		WithProfile(NewEUProfile()),
	)

	if r.Count() != 2 {
		t.Errorf("expected 2 profiles from options, got %d", r.Count())
	}

	if !r.IsRegistered(RegionBY) {
		t.Error("BY profile should be registered via WithProfile option")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// RegionFromRegistry
// ═══════════════════════════════════════════════════════════════════════════

func TestRegionFromRegistry(t *testing.T) {
	r := NewProfileRegistry(
		WithProfile(NewBYProfile()),
		WithProfile(NewEUProfile()),
	)

	region, err := RegionFromRegistry(r, "СТБ 34.101 (Республика Беларусь)")
	if err != nil {
		t.Fatalf("RegionFromRegistry error: %v", err)
	}
	if region != RegionBY {
		t.Errorf("expected region BY, got %s", region)
	}

	region, err = RegionFromRegistry(r, "GDPR / NIS2 (European Union)")
	if err != nil {
		t.Fatalf("RegionFromRegistry error: %v", err)
	}
	if region != RegionEU {
		t.Errorf("expected region EU, got %s", region)
	}
}

func TestRegionFromRegistryNotFound(t *testing.T) {
	r := NewProfileRegistry(
		WithProfile(NewINTLProfile()),
	)

	_, err := RegionFromRegistry(r, "NonExistentProfile")
	if err == nil {
		t.Fatal("RegionFromRegistry should error for non-existent profile name")
	}
}
