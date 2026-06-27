package retention

import (
	"testing"
	"time"
)

func TestGetProfile_BY_Audit(t *testing.T) {
	pm := NewProfileManager(nil)
	policy, err := pm.GetProfile(RegionBY, DataAudit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	if policy.Region != RegionBY {
		t.Errorf("expected region BY, got %s", policy.Region)
	}
	if policy.DataType != DataAudit {
		t.Errorf("expected data_type audit, got %s", policy.DataType)
	}
	expected := 1800 * 24 * time.Hour
	if policy.TotalTTL != expected {
		t.Errorf("expected TotalTTL %v, got %v", expected, policy.TotalTTL)
	}
}

func TestGetProfile_RU_Audit(t *testing.T) {
	pm := NewProfileManager(nil)
	policy, err := pm.GetProfile(RegionRU, DataAudit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	expected := 1095 * 24 * time.Hour
	if policy.TotalTTL != expected {
		t.Errorf("expected TotalTTL %v, got %v", expected, policy.TotalTTL)
	}
}

func TestGetProfile_EU_Telemetry(t *testing.T) {
	pm := NewProfileManager(nil)
	policy, err := pm.GetProfile(RegionEU, DataTelemetry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	expected := 30 * 24 * time.Hour
	if policy.TotalTTL != expected {
		t.Errorf("expected TotalTTL %v, got %v", expected, policy.TotalTTL)
	}
}

func TestGetProfile_US_Audit(t *testing.T) {
	pm := NewProfileManager(nil)
	policy, err := pm.GetProfile(RegionUS, DataAudit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	expected := 2555 * 24 * time.Hour
	if policy.TotalTTL != expected {
		t.Errorf("expected TotalTTL %v, got %v", expected, policy.TotalTTL)
	}
}

func TestGetProfile_CN_Telemetry(t *testing.T) {
	pm := NewProfileManager(nil)
	policy, err := pm.GetProfile(RegionCN, DataTelemetry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}
	expected := 30 * 24 * time.Hour
	if policy.TotalTTL != expected {
		t.Errorf("expected TotalTTL %v, got %v", expected, policy.TotalTTL)
	}
}

func TestGetProfile_UnknownRegion(t *testing.T) {
	pm := NewProfileManager(nil)
	_, err := pm.GetProfile("XX", DataAudit)
	if err == nil {
		t.Fatal("expected error for unknown region")
	}
}

func TestGetProfile_UnknownDataType(t *testing.T) {
	pm := NewProfileManager(nil)
	_, err := pm.GetProfile(RegionBY, "unknown_type")
	if err == nil {
		t.Fatal("expected error for unknown data type")
	}
}

func TestSetProfile(t *testing.T) {
	pm := NewProfileManager(nil)
	custom := &RetentionPolicy{
		Region:   RegionBY,
		DataType: DataTelemetry,
		HotTTL:   10 * 24 * time.Hour,
		TotalTTL: 10 * 24 * time.Hour,
	}
	if err := pm.SetProfile(custom); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	policy, err := pm.GetProfile(RegionBY, DataTelemetry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.HotTTL != 10*24*time.Hour {
		t.Errorf("expected HotTTL 10d, got %v", policy.HotTTL)
	}
}

func TestSetProfile_Invalid(t *testing.T) {
	pm := NewProfileManager(nil)
	tests := []struct {
		name   string
		policy *RetentionPolicy
	}{
		{"empty region", &RetentionPolicy{DataType: DataAudit, TotalTTL: time.Hour}},
		{"empty data_type", &RetentionPolicy{Region: RegionBY, TotalTTL: time.Hour}},
		{"zero total_ttl", &RetentionPolicy{Region: RegionBY, DataType: DataAudit}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := pm.SetProfile(tt.policy); err == nil {
				t.Error("expected error for invalid policy")
			}
		})
	}
}

func TestListProfiles(t *testing.T) {
	pm := NewProfileManager(nil)
	profiles := pm.ListProfiles()
	if len(profiles) == 0 {
		t.Fatal("expected non-empty profiles list")
	}
}

func TestListRegions(t *testing.T) {
	pm := NewProfileManager(nil)
	regions := pm.ListRegions()
	expected := 5
	if len(regions) != expected {
		t.Errorf("expected %d regions, got %d: %v", expected, len(regions), regions)
	}
}

func TestEvaluateLifecycle_Hot(t *testing.T) {
	policy := &RetentionPolicy{
		HotTTL:  30 * 24 * time.Hour,
		ColdTTL: 60 * 24 * time.Hour,
	}
	stage := EvaluateLifecycle(5*24*time.Hour, policy)
	if stage != StageHot {
		t.Errorf("expected StageHot, got %s", stage)
	}
}

func TestEvaluateLifecycle_Cold(t *testing.T) {
	policy := &RetentionPolicy{
		HotTTL:  30 * 24 * time.Hour,
		ColdTTL: 60 * 24 * time.Hour,
	}
	stage := EvaluateLifecycle(45*24*time.Hour, policy)
	if stage != StageCold {
		t.Errorf("expected StageCold, got %s", stage)
	}
}

func TestEvaluateLifecycle_Archive(t *testing.T) {
	policy := &RetentionPolicy{
		HotTTL:     30 * 24 * time.Hour,
		ColdTTL:    60 * 24 * time.Hour,
		ArchiveTTL: 90 * 24 * time.Hour,
		TotalTTL:   180 * 24 * time.Hour,
	}
	stage := EvaluateLifecycle(100*24*time.Hour, policy)
	if stage != StageArchive {
		t.Errorf("expected StageArchive, got %s", stage)
	}
}

func TestEvaluateLifecycle_Delete(t *testing.T) {
	policy := &RetentionPolicy{
		HotTTL:     30 * 24 * time.Hour,
		ColdTTL:    60 * 24 * time.Hour,
		ArchiveTTL: 90 * 24 * time.Hour,
		TotalTTL:   180 * 24 * time.Hour,
	}
	stage := EvaluateLifecycle(200*24*time.Hour, policy)
	if stage != StageDelete {
		t.Errorf("expected StageDelete, got %s", stage)
	}
}

// ── LegalHoldManager Tests ────────────────────────────────────────────────

func TestLegalHoldManager_AddAndIsHeld(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	hold := &LegalHold{
		TenantID:  "tenant-1",
		DataType:  DataAudit,
		Reason:    "Litigation case #123",
		CreatedBy: "legal-team",
	}
	if err := mgr.AddHold(hold); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mgr.IsHeld("tenant-1", DataAudit) {
		t.Error("expected tenant-1 to be held for audit data")
	}
}

func TestLegalHoldManager_IsHeld_WrongTenant(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	hold := &LegalHold{
		TenantID:  "tenant-1",
		DataType:  DataAudit,
		Reason:    "Litigation",
		CreatedBy: "legal-team",
	}
	if err := mgr.AddHold(hold); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr.IsHeld("tenant-2", DataAudit) {
		t.Error("expected tenant-2 to NOT be held")
	}
}

func TestLegalHoldManager_IsHeld_WrongDataType(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	hold := &LegalHold{
		TenantID:  "tenant-1",
		DataType:  DataAudit,
		Reason:    "Litigation",
		CreatedBy: "legal-team",
	}
	if err := mgr.AddHold(hold); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr.IsHeld("tenant-1", DataTelemetry) {
		t.Error("expected telemetry data to NOT be held")
	}
}

func TestLegalHoldManager_IsHeld_AllDataTypes(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	hold := &LegalHold{
		TenantID:  "tenant-1",
		DataType:  "", // empty = all data types
		Reason:    "Regulatory investigation",
		CreatedBy: "compliance-team",
	}
	if err := mgr.AddHold(hold); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mgr.IsHeld("tenant-1", DataAudit) {
		t.Error("expected all data to be held")
	}
	if !mgr.IsHeld("tenant-1", DataTelemetry) {
		t.Error("expected all data to be held")
	}
}

func TestLegalHoldManager_ReleaseHold(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	hold := &LegalHold{
		TenantID:  "tenant-1",
		Reason:    "Litigation case #123",
		CreatedBy: "legal-team",
	}
	if err := mgr.AddHold(hold); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mgr.ReleaseHold("tenant-1", "legal-team"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mgr.IsHeld("tenant-1", DataAudit) {
		t.Error("expected tenant to NOT be held after release")
	}
}

func TestLegalHoldManager_ReleaseHold_NoHolds(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	err := mgr.ReleaseHold("nonexistent-tenant", "admin")
	if err == nil {
		t.Fatal("expected error for releasing non-existent holds")
	}
}

func TestLegalHoldManager_ListActive(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	_ = mgr.AddHold(&LegalHold{TenantID: "t1", Reason: "Case 1", CreatedBy: "legal"})
	_ = mgr.AddHold(&LegalHold{TenantID: "t2", Reason: "Case 2", CreatedBy: "legal"})
	_ = mgr.ReleaseHold("t2", "admin")

	active := mgr.ListActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active hold, got %d", len(active))
	}
}

func TestLegalHoldManager_GetHolds(t *testing.T) {
	mgr := NewLegalHoldManager(nil)

	_ = mgr.AddHold(&LegalHold{TenantID: "t1", Reason: "Case 1", CreatedBy: "legal"})
	_ = mgr.AddHold(&LegalHold{TenantID: "t1", Reason: "Case 2", CreatedBy: "legal"})

	holds := mgr.GetHolds("t1")
	if len(holds) != 2 {
		t.Errorf("expected 2 holds, got %d", len(holds))
	}
}

func TestLegalHold_IsActive_Expired(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	hold := &LegalHold{
		TenantID:  "t1",
		Status:    LegalHoldActive,
		ExpiresAt: &past,
	}
	if hold.IsActive() {
		t.Error("expected expired hold to be inactive")
	}
}

func TestLegalHold_IsActive_Released(t *testing.T) {
	hold := &LegalHold{
		TenantID: "t1",
		Status:   LegalHoldReleased,
	}
	if hold.IsActive() {
		t.Error("expected released hold to be inactive")
	}
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  *RetentionPolicy
		wantErr bool
	}{
		{
			name:    "valid policy",
			policy:  &RetentionPolicy{Region: RegionBY, DataType: DataAudit, TotalTTL: time.Hour},
			wantErr: false,
		},
		{
			name:    "empty region",
			policy:  &RetentionPolicy{Region: "", DataType: DataAudit, TotalTTL: time.Hour},
			wantErr: true,
		},
		{
			name:    "empty data_type",
			policy:  &RetentionPolicy{Region: RegionBY, DataType: "", TotalTTL: time.Hour},
			wantErr: true,
		},
		{
			name:    "zero total_ttl",
			policy:  &RetentionPolicy{Region: RegionBY, DataType: DataAudit, TotalTTL: 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	pm := NewProfileManager(nil)
	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = pm.GetProfile(RegionBY, DataAudit)
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			_ = pm.SetProfile(&RetentionPolicy{
				Region:   RegionBY,
				DataType: DataTelemetry,
				HotTTL:   time.Hour,
				TotalTTL: time.Hour,
			})
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}
