package workforce

import (
	"testing"
	"time"
)

func TestDefaultRBACMatrix(t *testing.T) {
	m := DefaultRBACMatrix()

	// Admin has all permissions on work_order
	if !m.CheckPermission(RoleAdmin, EntityWorkOrder, PermCreate) {
		t.Error("admin should be able to create work orders")
	}
	if !m.CheckPermission(RoleAdmin, EntityWorkOrder, PermDelete) {
		t.Error("admin should be able to delete work orders")
	}

	// Technician cannot delete
	if m.CheckPermission(RoleTechnician, EntityWorkOrder, PermDelete) {
		t.Error("technician should NOT be able to delete work orders")
	}

	// Viewer can only read
	if !m.CheckPermission(RoleViewer, EntityWorkOrder, PermRead) {
		t.Error("viewer should be able to read work orders")
	}
	if m.CheckPermission(RoleViewer, EntityWorkOrder, PermCreate) {
		t.Error("viewer should NOT be able to create work orders")
	}

	// Manager can approve
	if !m.CheckPermission(RoleManager, EntityWorkOrder, PermApprove) {
		t.Error("manager should be able to approve work orders")
	}
}

func TestRBACMatrix_InvalidRole(t *testing.T) {
	m := DefaultRBACMatrix()
	if m.CheckPermission("invalid_role", EntityWorkOrder, PermRead) {
		t.Error("invalid role should have no permissions")
	}
}

func TestValidateRole(t *testing.T) {
	if !ValidateRole("admin") {
		t.Error("admin should be valid")
	}
	if !ValidateRole("technician") {
		t.Error("technician should be valid")
	}
	if ValidateRole("superadmin") {
		t.Error("superadmin should NOT be valid")
	}
}

func TestValidateEntity(t *testing.T) {
	if !ValidateEntity("work_order") {
		t.Error("work_order should be valid")
	}
	if ValidateEntity("invalid") {
		t.Error("invalid should NOT be valid")
	}
}

func TestValidatePermission(t *testing.T) {
	if !ValidatePermission("create") {
		t.Error("create should be valid")
	}
	if ValidatePermission("sudo") {
		t.Error("sudo should NOT be valid")
	}
}

func TestGetRoleWeight(t *testing.T) {
	if GetRoleWeight(RoleAdmin) <= GetRoleWeight(RoleManager) {
		t.Error("admin should have higher weight than manager")
	}
	if GetRoleWeight(RoleTechnician) <= GetRoleWeight(RoleViewer) {
		t.Error("technician should have higher weight than viewer")
	}
}

func TestHasHigherRoleThan(t *testing.T) {
	if !HasHigherRoleThan(RoleAdmin, RoleTechnician) {
		t.Error("admin > technician")
	}
	if HasHigherRoleThan(RoleViewer, RoleManager) {
		t.Error("viewer is NOT > manager")
	}
}

func TestTechnicianWorkload(t *testing.T) {
	tw := &TechnicianWorkload{
		UserID:      "user-001",
		UserName:    "John Doe",
		Role:        RoleTechnician,
		ActiveWO:    3,
		MaxWorkload: 5,
	}

	if !tw.IsAvailable() {
		t.Error("3/5 should be available")
	}

	util := tw.Utilization()
	if util != 60.0 {
		t.Errorf("expected 60%% utilization, got %.1f%%", util)
	}

	// Overloaded
	tw.ActiveWO = 5
	if tw.IsAvailable() {
		t.Error("5/5 should NOT be available")
	}

	str := tw.String()
	if str == "" {
		t.Error("expected non-empty string representation")
	}
}

func TestTechnicianWorkload_Overloaded(t *testing.T) {
	tw := &TechnicianWorkload{
		UserID:      "user-002",
		ActiveWO:    6,
		MaxWorkload: 5,
	}
	if tw.IsAvailable() {
		t.Error("6/5 should NOT be available")
	}
	util := tw.Utilization()
	if util != 120.0 {
		t.Errorf("expected 120%% utilization, got %.1f%%", util)
	}
}

func TestUserCertification_Expired(t *testing.T) {
	future := time.Now().Add(365 * 24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	uc := &UserCertification{
		ExpiresAt: &future,
	}
	if uc.IsExpired() {
		t.Error("future expiry should not be expired")
	}

	uc.ExpiresAt = &past
	if !uc.IsExpired() {
		t.Error("past expiry should be expired")
	}

	uc.ExpiresAt = nil
	if uc.IsExpired() {
		t.Error("nil expiry should not be expired")
	}
}

func TestDefaultRBACMatrix_AllRolesPresent(t *testing.T) {
	m := DefaultRBACMatrix()
	roles := []Role{RoleAdmin, RoleManager, RoleTechnician, RoleViewer, RoleSupport}

	for _, role := range roles {
		if _, ok := m[role]; !ok {
			t.Errorf("role %s should be in matrix", role)
		}
	}
}

func TestDefaultRBACMatrix_ManagerPermissions(t *testing.T) {
	m := DefaultRBACMatrix()

	// Manager should NOT have settings access
	if m.CheckPermission(RoleManager, EntitySettings, PermRead) {
		t.Error("manager should NOT read settings")
	}
	if m.CheckPermission(RoleManager, EntitySettings, PermUpdate) {
		t.Error("manager should NOT update settings")
	}

	// Manager should NOT delete devices
	if m.CheckPermission(RoleManager, EntityDevice, PermDelete) {
		t.Error("manager should NOT delete devices")
	}
}
