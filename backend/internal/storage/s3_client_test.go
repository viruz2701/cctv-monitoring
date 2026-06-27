package storage

import (
	"testing"

	"gb-telemetry-collector/internal/compliance"
)

func TestNewS3ClientNilEnforcer(t *testing.T) {
	_, err := NewS3Client(S3ClientConfig{
		Enforcer:       nil,
		StorageContext: &StorageContext{Region: compliance.RegionBY},
	})
	if err == nil {
		t.Fatal("expected error for nil enforcer")
	}
}

func TestNewS3ClientNilContext(t *testing.T) {
	enforcer := NewResidencyEnforcer(nil)
	_, err := NewS3Client(S3ClientConfig{
		Enforcer:       enforcer,
		StorageContext: nil,
	})
	if err == nil {
		t.Fatal("expected error for nil storage context")
	}
}

func TestNewS3ClientUnknownRegion(t *testing.T) {
	enforcer := NewResidencyEnforcer(nil)
	_, err := NewS3Client(S3ClientConfig{
		Enforcer: enforcer,
		StorageContext: &StorageContext{
			Region: "UNKNOWN",
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown region")
	}
}

func TestNewS3ClientForProfileNilEnforcer(t *testing.T) {
	_, err := NewS3ClientForProfile(nil, compliance.NewINTLProfile(), compliance.RegionINTL, "", nil)
	if err == nil {
		t.Fatal("expected error for nil enforcer")
	}
}

func TestNewS3ClientForProfileNilProfile(t *testing.T) {
	enforcer := NewResidencyEnforcer(nil)
	_, err := NewS3ClientForProfile(enforcer, nil, compliance.RegionINTL, "", nil)
	if err == nil {
		t.Fatal("expected error for nil profile")
	}
}

func TestNewS3ClientForProfile(t *testing.T) {
	enforcer := NewResidencyEnforcer(nil)
	client, err := NewS3ClientForProfile(
		enforcer,
		compliance.NewINTLProfile(),
		compliance.RegionINTL,
		"test-tenant",
		nil,
	)
	if err != nil {
		t.Fatalf("NewS3ClientForProfile error: %v", err)
	}
	if client == nil {
		t.Fatal("client must not be nil")
	}
	if client.ctx == nil {
		t.Fatal("storage context must not be nil")
	}
	if client.ctx.Region != compliance.RegionINTL {
		t.Errorf("expected region INTL, got %s", client.ctx.Region)
	}
	if client.ctx.TenantID != "test-tenant" {
		t.Errorf("expected tenantID 'test-tenant', got %s", client.ctx.TenantID)
	}
}

func TestNewS3ClientBYRegion(t *testing.T) {
	enforcer := NewResidencyEnforcer(nil)
	client, err := NewS3Client(S3ClientConfig{
		Enforcer: enforcer,
		StorageContext: &StorageContext{
			Region:            compliance.RegionBY,
			ComplianceProfile: compliance.NewBYProfile(),
		},
	})
	if err != nil {
		t.Fatalf("NewS3Client error: %v", err)
	}
	if client == nil {
		t.Fatal("client must not be nil")
	}
	if client.ctx.Region != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", client.ctx.Region)
	}
}

func TestNewS3ClientEURegion(t *testing.T) {
	enforcer := NewResidencyEnforcer(nil)
	client, err := NewS3Client(S3ClientConfig{
		Enforcer: enforcer,
		StorageContext: &StorageContext{
			Region:            compliance.RegionEU,
			ComplianceProfile: compliance.NewEUProfile(),
		},
	})
	if err != nil {
		t.Fatalf("NewS3Client error: %v", err)
	}
	if client == nil {
		t.Fatal("client must not be nil")
	}
	if client.ctx.Region != compliance.RegionEU {
		t.Errorf("expected region EU, got %s", client.ctx.Region)
	}
}
