package agent

import (
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"
)

func TestDefaultDecisionTree(t *testing.T) {
	dt := DefaultDecisionTree()
	if dt == nil {
		t.Fatal("DefaultDecisionTree returned nil")
	}
	if dt.MaxAutoFixRetries != 3 {
		t.Errorf("expected MaxAutoFixRetries=3, got %d", dt.MaxAutoFixRetries)
	}
	if dt.ApprovalTimeout != 5*time.Minute {
		t.Errorf("expected ApprovalTimeout=5m, got %v", dt.ApprovalTimeout)
	}
	if dt.BusinessHoursStart != 8 {
		t.Errorf("expected BusinessHoursStart=8, got %d", dt.BusinessHoursStart)
	}
	if dt.BusinessHoursEnd != 20 {
		t.Errorf("expected BusinessHoursEnd=20, got %d", dt.BusinessHoursEnd)
	}
}

func TestDecideFlappingDetection(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityLow,
			Method:   models.AlarmMethodVideoLoss,
		},
		FailureCount: 11,
		LastFixTime:  time.Now().Add(-1 * time.Minute),
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionIgnore {
		t.Errorf("expected DecisionIgnore for flapping, got %s", dec.Level)
	}
}

func TestDecideCriticalEscalates(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityHigh,
			Method:   models.AlarmMethodEquipmentFault,
		},
		FailureCount: 1,
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionEscalate {
		t.Errorf("expected DecisionEscalate for critical, got %s", dec.Level)
	}
	if dec.EscalateTo != "engineer_oncall" {
		t.Errorf("expected escalate to 'engineer_oncall', got %q", dec.EscalateTo)
	}
}

func TestDecideAutoFixVideoLoss(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityMedium,
			Method:   models.AlarmMethodVideoLoss,
		},
		Device: &models.Device{
			DeviceID:   "cam-001",
			VendorType: "Hikvision",
		},
		FailureCount: 1,
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionAutoFix {
		t.Errorf("expected DecisionAutoFix for VideoLoss, got %s", dec.Level)
	}
	if dec.PlaybookRef != "reboot_camera" {
		t.Errorf("expected playbook 'reboot_camera', got %q", dec.PlaybookRef)
	}
}

func TestDecideAutoFixEquipmentFaultHikvision(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityMedium,
			Method:   models.AlarmMethodEquipmentFault,
		},
		Device: &models.Device{
			DeviceID:   "cam-001",
			VendorType: "Hikvision",
		},
		FailureCount: 1,
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionAutoFix {
		t.Errorf("expected DecisionAutoFix for Hikvision EquipmentFault, got %s", dec.Level)
	}
	if dec.PlaybookRef != "hikvision_diagnostic" {
		t.Errorf("expected playbook 'hikvision_diagnostic', got %q", dec.PlaybookRef)
	}
}

func TestDecideAutoFixEquipmentFaultDahua(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityMedium,
			Method:   models.AlarmMethodEquipmentFault,
		},
		Device: &models.Device{
			DeviceID:   "cam-001",
			VendorType: "Dahua",
		},
		FailureCount: 1,
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionAutoFix {
		t.Errorf("expected DecisionAutoFix for Dahua EquipmentFault, got %s", dec.Level)
	}
	if dec.PlaybookRef != "camera_diagnostic" {
		t.Errorf("expected playbook 'camera_diagnostic', got %q", dec.PlaybookRef)
	}
}

func TestDecideAutoFixExceededRetries(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityMedium,
			Method:   models.AlarmMethodVideoLoss,
		},
		Device: &models.Device{
			DeviceID:   "cam-001",
			VendorType: "Hikvision",
		},
		FailureCount: 5, // > MaxAutoFixRetries=3
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionEscalate {
		t.Errorf("expected DecisionEscalate after max retries, got %s", dec.Level)
	}
}

func TestDecideApproveForNonAutofixable(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityLow,
			Method:   models.AlarmMethodMotionDetection,
		},
		Device: &models.Device{
			DeviceID:   "cam-001",
			VendorType: "Generic",
		},
		FailureCount:    1,
		IsBusinessHours: true,
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionApprove {
		t.Errorf("expected DecisionApprove for non-autofixable in business hours, got %s", dec.Level)
	}
	if dec.ApprovalTTL != 5*time.Minute {
		t.Errorf("expected ApprovalTTL=5m, got %v", dec.ApprovalTTL)
	}
}

func TestDecideEscalateNonBusinessHours(t *testing.T) {
	dt := DefaultDecisionTree()
	ctx := DecisionContext{
		Alarm: models.Alarm{
			DeviceID: "cam-001",
			Priority: models.AlarmPriorityLow,
			Method:   models.AlarmMethodMotionDetection,
		},
		Device: &models.Device{
			DeviceID:   "cam-001",
			VendorType: "Generic",
		},
		FailureCount:    1,
		IsBusinessHours: false,
	}

	dec := dt.Decide(ctx)
	if dec.Level != DecisionEscalate {
		t.Errorf("expected DecisionEscalate for non-business hours, got %s", dec.Level)
	}
}

func TestIsAutoFixable(t *testing.T) {
	dt := DefaultDecisionTree()

	tests := []struct {
		name     string
		ctx      DecisionContext
		expected bool
	}{
		{
			name: "VideoLoss always auto-fixable",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodVideoLoss},
				Device: &models.Device{VendorType: "Generic"},
			},
			expected: true,
		},
		{
			name: "EquipmentFault Hikvision auto-fixable",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodEquipmentFault},
				Device: &models.Device{VendorType: "Hikvision"},
			},
			expected: true,
		},
		{
			name: "EquipmentFault Dahua auto-fixable",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodEquipmentFault},
				Device: &models.Device{VendorType: "Dahua"},
			},
			expected: true,
		},
		{
			name: "EquipmentFault Generic not auto-fixable",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodEquipmentFault},
				Device: &models.Device{VendorType: "Generic"},
			},
			expected: false,
		},
		{
			name: "MotionDetection not auto-fixable",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodMotionDetection},
				Device: &models.Device{VendorType: "Hikvision"},
			},
			expected: false,
		},
		{
			name: "EquipmentFault nil device not auto-fixable",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodEquipmentFault},
				Device: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dt.isAutoFixable(tt.ctx)
			if result != tt.expected {
				t.Errorf("isAutoFixable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSelectPlaybook(t *testing.T) {
	dt := DefaultDecisionTree()

	tests := []struct {
		name     string
		ctx      DecisionContext
		expected string
	}{
		{
			name: "VideoLoss → reboot_camera",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodVideoLoss},
				Device: &models.Device{VendorType: "Hikvision"},
			},
			expected: "reboot_camera",
		},
		{
			name: "EquipmentFault Hikvision → hikvision_diagnostic",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodEquipmentFault},
				Device: &models.Device{VendorType: "Hikvision"},
			},
			expected: "hikvision_diagnostic",
		},
		{
			name: "EquipmentFault Dahua → camera_diagnostic",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodEquipmentFault},
				Device: &models.Device{VendorType: "Dahua"},
			},
			expected: "camera_diagnostic",
		},
		{
			name: "MotionDetection → default_diagnostic",
			ctx: DecisionContext{
				Alarm:  models.Alarm{Method: models.AlarmMethodMotionDetection},
				Device: &models.Device{VendorType: "Hikvision"},
			},
			expected: "default_diagnostic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dt.selectPlaybook(tt.ctx)
			if result != tt.expected {
				t.Errorf("selectPlaybook() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{100, "100"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tt := range tests {
		result := itoa(tt.n)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, result, tt.expected)
		}
	}
}
