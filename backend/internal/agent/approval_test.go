package agent

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestNewApprovalManager(t *testing.T) {
	am := NewApprovalManager(nil)
	if am == nil {
		t.Fatal("NewApprovalManager returned nil")
	}
	if len(am.GetPending()) != 0 {
		t.Errorf("expected 0 pending requests, got %d", len(am.GetPending()))
	}
}

func TestApprovalRequestApproval(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{
		Level:       DecisionApprove,
		Reason:      "test reason",
		ApprovalTTL: 5 * time.Minute,
	}

	req, err := am.RequestApproval(ctx, "cam-001", "Test Camera", "reboot", "test reason", dec, 30*time.Second)
	if err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}
	if req.Status != ApprovalPending {
		t.Errorf("expected status pending, got %s", req.Status)
	}
	if req.DeviceID != "cam-001" {
		t.Errorf("expected device 'cam-001', got %q", req.DeviceID)
	}
	if req.DeviceName != "Test Camera" {
		t.Errorf("expected name 'Test Camera', got %q", req.DeviceName)
	}
	if req.Action != "reboot" {
		t.Errorf("expected action 'reboot', got %q", req.Action)
	}

	pending := am.GetPending()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending request, got %d", len(pending))
	}
	if pending[0].ID != req.ID {
		t.Errorf("expected pending ID %q, got %q", req.ID, pending[0].ID)
	}
}

func TestApprovalRequestDefaultTTL(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, err := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 0)
	if err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}

	expectedExpiry := time.Now().Add(5 * time.Minute)
	if req.ExpiresAt.Before(expectedExpiry.Add(-1*time.Second)) || req.ExpiresAt.After(expectedExpiry.Add(1*time.Second)) {
		t.Errorf("expected TTL ~5m, got expiry at %v", req.ExpiresAt)
	}
}

func TestApprovalApprove(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	err := am.Approve(req.ID, "operator1")
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	// Check status via GetPending
	pending := am.GetPending()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after approve, got %d", len(pending))
	}
}

func TestApprovalReject(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	err := am.Reject(req.ID, "operator1", "not needed")
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}

	pending := am.GetPending()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after reject, got %d", len(pending))
	}
}

func TestApprovalDoubleResolve(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	am.Approve(req.ID, "operator1")
	err := am.Approve(req.ID, "operator2")
	if err == nil {
		t.Error("expected error for double approve, got nil")
	}
}

func TestApprovalApproveNonexistent(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	err := am.Approve("nonexistent", "operator1")
	if err == nil {
		t.Error("expected error for nonexistent request, got nil")
	}
}

func TestApprovalWaitApproval(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	// Approve in a goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		am.Approve(req.ID, "operator1")
	}()

	result := am.WaitApproval(ctx, req.ID)
	if !result.Approved {
		t.Errorf("expected approved, got rejected: %s", result.Reason)
	}
	if result.By != "operator1" {
		t.Errorf("expected approved by 'operator1', got %q", result.By)
	}
}

func TestApprovalWaitReject(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	go func() {
		time.Sleep(50 * time.Millisecond)
		am.Reject(req.ID, "operator1", "not needed")
	}()

	result := am.WaitApproval(ctx, req.ID)
	if result.Approved {
		t.Error("expected rejected, got approved")
	}
	if result.Reason != "not needed" {
		t.Errorf("expected reason 'not needed', got %q", result.Reason)
	}
}

func TestApprovalWaitTimeout(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 200*time.Millisecond)

	result := am.WaitApproval(ctx, req.ID)
	if result.Approved {
		t.Error("expected rejected due to timeout, got approved")
	}
	if result.Reason != "timeout expired" {
		t.Errorf("expected reason 'timeout expired', got %q", result.Reason)
	}
}

func TestApprovalWaitContextCancel(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx, cancel := context.WithCancel(context.Background())

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	// Cancel immediately
	cancel()

	result := am.WaitApproval(ctx, req.ID)
	if result.Approved {
		t.Error("expected rejected due to context cancel, got approved")
	}
}

func TestApprovalWaitNonexistent(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	result := am.WaitApproval(ctx, "nonexistent")
	if result.Approved {
		t.Error("expected rejected for nonexistent, got approved")
	}
}

func TestApprovalCleanupExpired(t *testing.T) {
	am := NewApprovalManager(slog.Default())
	ctx := context.Background()

	dec := Decision{Level: DecisionApprove}
	req, _ := am.RequestApproval(ctx, "cam-001", "Cam", "reboot", "test", dec, 30*time.Second)

	// Approve the request
	am.Approve(req.ID, "operator1")

	// Cleanup with 0 maxAge should remove it
	count := am.CleanupExpired(0)
	if count != 1 {
		t.Errorf("expected 1 cleaned up, got %d", count)
	}

	// Try to wait for cleaned up request
	result := am.WaitApproval(ctx, req.ID)
	if result.Approved {
		t.Error("expected rejected for cleaned up request, got approved")
	}
}

func TestBuildApprovalMessage(t *testing.T) {
	req := ApprovalRequest{
		ID:         "approval_test_123",
		DeviceID:   "cam-001",
		DeviceName: "Front Gate",
		Action:     "reboot",
		Reason:     "VideoLoss detected",
		Decision:   Decision{Level: DecisionApprove},
		ExpiresAt:  time.Date(2026, 1, 1, 15, 4, 5, 0, time.UTC),
	}

	msg := BuildApprovalMessage(req)

	if len(msg) == 0 {
		t.Error("BuildApprovalMessage returned empty string")
	}
	if !containsStr(msg, "Front Gate") {
		t.Error("message should contain device name")
	}
	if !containsStr(msg, "cam-001") {
		t.Error("message should contain device ID")
	}
	if !containsStr(msg, "reboot") {
		t.Error("message should contain action")
	}
	if !containsStr(msg, "VideoLoss") {
		t.Error("message should contain reason")
	}
	if !containsStr(msg, "/approve") {
		t.Error("message should contain approve command")
	}
	if !containsStr(msg, "/reject") {
		t.Error("message should contain reject command")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
