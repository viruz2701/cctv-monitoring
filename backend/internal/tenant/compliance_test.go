// Package tenant — tests for Tenant Compliance Profile.
package tenant

import (
	"context"
	"testing"

	"gb-telemetry-collector/internal/compliance"
)

func TestContextWithComplianceRegion(t *testing.T) {
	ctx := ContextWithComplianceRegion(context.Background(), compliance.RegionBY)
	region := GetComplianceRegionFromContext(ctx)
	if region != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", region)
	}
}

func TestContextWithComplianceProfile(t *testing.T) {
	profile := compliance.NewBYProfile()
	ctx := ContextWithComplianceProfile(context.Background(), profile)

	got := GetComplianceProfileFromContext(ctx)
	if got == nil {
		t.Fatal("profile from context must not be nil")
	}
	if got.Region() != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", got.Region())
	}
}

func TestGetComplianceRegionFromContextEmpty(t *testing.T) {
	region := GetComplianceRegionFromContext(context.Background())
	if region != "" {
		t.Errorf("expected empty region, got %s", region)
	}
}

func TestGetComplianceProfileFromContextNil(t *testing.T) {
	profile := GetComplianceProfileFromContext(context.Background())
	if profile != nil {
		t.Fatal("expected nil profile from empty context")
	}
}

func TestContextRoundTrip(t *testing.T) {
	// BY profile
	byProfile := compliance.NewBYProfile()
	ctx := ContextWithComplianceRegion(context.Background(), compliance.RegionBY)
	ctx = ContextWithComplianceProfile(ctx, byProfile)

	if region := GetComplianceRegionFromContext(ctx); region != compliance.RegionBY {
		t.Errorf("expected BY, got %s", region)
	}
	if p := GetComplianceProfileFromContext(ctx); p == nil || p.Region() != compliance.RegionBY {
		t.Error("failed to retrieve BY profile from context")
	}

	// Override with EU
	euProfile := compliance.NewEUProfile()
	ctx = ContextWithComplianceRegion(ctx, compliance.RegionEU)
	ctx = ContextWithComplianceProfile(ctx, euProfile)

	if region := GetComplianceRegionFromContext(ctx); region != compliance.RegionEU {
		t.Errorf("expected EU, got %s", region)
	}
}

func TestGetTenantIDFromContext(t *testing.T) {
	ctx := context.Background()

	id := GetTenantIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty tenant ID, got %s", id)
	}
}

func TestNewTenantComplianceStore(t *testing.T) {
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewINTLProfile()),
	)

	// Without DB pool — store is functional for context operations
	store := NewTenantComplianceStore(nil, registry)
	if store == nil {
		t.Fatal("store must not be nil")
	}
}
