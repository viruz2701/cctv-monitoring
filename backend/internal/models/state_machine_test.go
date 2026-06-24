package models

import (
	"context"
	"testing"
	"time"

	"github.com/looplab/fsm"
)

// ═══════════════════════════════════════════════════════════════════════
// State Machine Tests
// ═══════════════════════════════════════════════════════════════════════

func TestNewWorkOrderFSM_DefaultStatus(t *testing.T) {
	sm := NewWorkOrderFSM("", nil)
	if sm.Current() != StatusRequested {
		t.Errorf("expected REQUESTED, got %s", sm.Current())
	}
}

func TestNewWorkOrderFSM_CustomInitial(t *testing.T) {
	sm := NewWorkOrderFSM(StatusOpen, nil)
	if sm.Current() != StatusOpen {
		t.Errorf("expected OPEN, got %s", sm.Current())
	}
}

func TestWorkOrderFSM_FullLifecycle(t *testing.T) {
	sm := NewWorkOrderFSM("", nil)

	// REQUESTED → APPROVED
	if err := sm.Transition("approve"); err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	assertStatus(t, sm, StatusApproved)

	// APPROVED → OPEN
	if err := sm.Transition("open"); err != nil {
		t.Fatalf("open failed: %v", err)
	}
	assertStatus(t, sm, StatusOpen)

	// OPEN → IN_PROGRESS
	if err := sm.Transition("start"); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	assertStatus(t, sm, StatusInProgress)

	// IN_PROGRESS → COMPLETED
	if err := sm.Transition("complete"); err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	assertStatus(t, sm, StatusCompleted)

	// COMPLETED → VERIFIED
	if err := sm.Transition("verify"); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	assertStatus(t, sm, StatusVerified)

	// VERIFIED → CLOSED
	if err := sm.Transition("close"); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	assertStatus(t, sm, StatusClosed)
}

func TestWorkOrderFSM_RejectionFlow(t *testing.T) {
	sm := NewWorkOrderFSM(StatusRequested, nil)

	if err := sm.Transition("reject"); err != nil {
		t.Fatalf("reject failed: %v", err)
	}
	assertStatus(t, sm, StatusRejected)

	// REJECTED → REQUESTED (reopen)
	if err := sm.Transition("reopen"); err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	assertStatus(t, sm, StatusRequested)
}

func TestWorkOrderFSM_AwaitingFlow(t *testing.T) {
	sm := NewWorkOrderFSM(StatusInProgress, nil)

	// IN_PROGRESS → AWAITING_PARTS
	if err := sm.Transition("await_parts"); err != nil {
		t.Fatalf("await_parts failed: %v", err)
	}
	assertStatus(t, sm, StatusAwaitingParts)

	// AWAITING_PARTS → IN_PROGRESS
	if err := sm.Transition("parts_received"); err != nil {
		t.Fatalf("parts_received failed: %v", err)
	}
	assertStatus(t, sm, StatusInProgress)

	// IN_PROGRESS → ON_HOLD
	if err := sm.Transition("hold"); err != nil {
		t.Fatalf("hold failed: %v", err)
	}
	assertStatus(t, sm, StatusOnHold)

	// ON_HOLD → IN_PROGRESS
	if err := sm.Transition("resume"); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	assertStatus(t, sm, StatusInProgress)
}

func TestWorkOrderFSM_InvalidTransition(t *testing.T) {
	sm := NewWorkOrderFSM(StatusRequested, nil)

	// Cannot close from REQUESTED
	err := sm.Transition("close")
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	assertStatus(t, sm, StatusRequested) // status unchanged
}

func TestWorkOrderFSM_CanTransition(t *testing.T) {
	sm := NewWorkOrderFSM(StatusRequested, nil)

	if !sm.CanTransition("approve") {
		t.Error("expected 'approve' to be available")
	}
	if sm.CanTransition("close") {
		t.Error("expected 'close' to NOT be available")
	}
}

func TestWorkOrderFSM_AvailableTransitions(t *testing.T) {
	sm := NewWorkOrderFSM(StatusOpen, nil)
	available := sm.AvailableTransitions()

	expected := []string{"start", "reject_from_open"}
	for _, e := range expected {
		found := false
		for _, a := range available {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected transition '%s' to be available, got %v", e, available)
		}
	}
}

func TestWorkOrderFSM_Callbacks(t *testing.T) {
	called := false
	callbacks := fsm.Callbacks{
		"before_approve": func(_ context.Context, e *fsm.Event) {
			called = true
		},
	}

	sm := NewWorkOrderFSM(StatusRequested, callbacks)
	if err := sm.Transition("approve"); err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if !called {
		t.Error("expected callback to be called")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Status Utility Tests
// ═══════════════════════════════════════════════════════════════════════

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		status WorkOrderStatus
		want   bool
	}{
		{StatusClosed, true},
		{StatusRejected, true},
		{StatusOpen, false},
		{StatusInProgress, false},
		{StatusCompleted, false},
	}

	for _, tt := range tests {
		if got := IsTerminal(tt.status); got != tt.want {
			t.Errorf("IsTerminal(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		status WorkOrderStatus
		want   bool
	}{
		{StatusOpen, true},
		{StatusInProgress, true},
		{StatusRequested, true},
		{StatusClosed, false},
		{StatusRejected, false},
		{StatusCompleted, false},
		{StatusVerified, false},
	}

	for _, tt := range tests {
		if got := IsActive(tt.status); got != tt.want {
			t.Errorf("IsActive(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestIsPaused(t *testing.T) {
	tests := []struct {
		status WorkOrderStatus
		want   bool
	}{
		{StatusOnHold, true},
		{StatusAwaitingParts, true},
		{StatusAwaitingVendor, true},
		{StatusAwaitingClient, true},
		{StatusOpen, false},
		{StatusInProgress, false},
	}

	for _, tt := range tests {
		if got := IsPaused(tt.status); got != tt.want {
			t.Errorf("IsPaused(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestStatusCategory(t *testing.T) {
	tests := []struct {
		status WorkOrderStatus
		want   string
	}{
		{StatusRequested, "requested"},
		{StatusApproved, "approved"},
		{StatusInProgress, "active"},
		{StatusOnHold, "paused"},
		{StatusCompleted, "completed"},
		{StatusVerified, "verified"},
		{StatusClosed, "closed"},
		{StatusRejected, "rejected"},
	}

	for _, tt := range tests {
		if got := StatusCategory(tt.status); got != tt.want {
			t.Errorf("StatusCategory(%s) = %s, want %s", tt.status, got, tt.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Validation Tests
// ═══════════════════════════════════════════════════════════════════════

func TestValidateWorkOrderStatus(t *testing.T) {
	valid := []string{
		"REQUESTED", "APPROVED", "OPEN", "IN_PROGRESS",
		"ON_HOLD", "AWAITING_PARTS", "AWAITING_VENDOR", "AWAITING_CLIENT",
		"COMPLETED", "VERIFIED", "CLOSED", "REJECTED",
	}
	invalid := []string{"open", "in_progress", "unknown", "", "PENDING"}

	for _, s := range valid {
		if !ValidateWorkOrderStatus(s) {
			t.Errorf("expected %s to be valid", s)
		}
	}
	for _, s := range invalid {
		if ValidateWorkOrderStatus(s) {
			t.Errorf("expected %s to be invalid", s)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	// Valid transitions
	if !ValidateTransition(StatusRequested, StatusApproved) {
		t.Error("REQUESTED → APPROVED should be valid")
	}
	if !ValidateTransition(StatusInProgress, StatusCompleted) {
		t.Error("IN_PROGRESS → COMPLETED should be valid")
	}
	if !ValidateTransition(StatusCompleted, StatusVerified) {
		t.Error("COMPLETED → VERIFIED should be valid")
	}

	// Invalid transitions
	if ValidateTransition(StatusRequested, StatusClosed) {
		t.Error("REQUESTED → CLOSED should be invalid")
	}
	if ValidateTransition(StatusOpen, StatusClosed) {
		t.Error("OPEN → CLOSED should be invalid")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SoftDeleteMixin Tests
// ═══════════════════════════════════════════════════════════════════════

func TestSoftDeleteMixin_IsDeleted(t *testing.T) {
	now := fixedTime()
	s := &SoftDeleteMixin{DeletedAt: &now}
	if !s.IsDeleted() {
		t.Error("expected IsDeleted() = true")
	}

	s2 := &SoftDeleteMixin{}
	if s2.IsDeleted() {
		t.Error("expected IsDeleted() = false")
	}
}

func TestSoftDeleteMixin_IsArchived(t *testing.T) {
	now := fixedTime()
	s := &SoftDeleteMixin{ArchivedAt: &now}
	if !s.IsArchived() {
		t.Error("expected IsArchived() = true")
	}

	s2 := &SoftDeleteMixin{}
	if s2.IsArchived() {
		t.Error("expected IsArchived() = false")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func assertStatus(t *testing.T, sm *WorkOrderStateMachine, expected WorkOrderStatus) {
	t.Helper()
	if sm.Current() != expected {
		t.Errorf("expected status %s, got %s", expected, sm.Current())
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 6, 24, 5, 0, 0, 0, time.UTC)
}
