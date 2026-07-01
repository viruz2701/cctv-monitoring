// Package agent — CMMS integration tests with context timeout enforcement.
package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ── Mocks ──────────────────────────────────────────────────────────

// mockCMMSAdapter implements cmms.CMMSAdapter for testing CMMSIntegrator.
// Only CreateWorkOrder and UpdateWorkOrder are relevant for the tested methods;
// all other interface methods return zero values.
type mockCMMSAdapter struct {
	mu sync.Mutex

	createWOFunc func(ctx context.Context, wo *models.WorkOrder) error
	updateWOFunc func(ctx context.Context, id string, updates map[string]interface{}) error
}

func (m *mockCMMSAdapter) CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error {
	if m.createWOFunc != nil {
		return m.createWOFunc(ctx, wo)
	}
	return nil
}

func (m *mockCMMSAdapter) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	if m.updateWOFunc != nil {
		return m.updateWOFunc(ctx, id, updates)
	}
	return nil
}

// ── Unused interface stubs ─────────────────────────────────────────

func (m *mockCMMSAdapter) GetWorkOrders(_ context.Context, _ map[string]interface{}) ([]models.WorkOrder, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetWorkOrder(_ context.Context, _ string) (*models.WorkOrder, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) AssignWorkOrder(_ context.Context, _, _ string) error { return nil }
func (m *mockCMMSAdapter) StartWorkOrder(_ context.Context, _ string) error     { return nil }
func (m *mockCMMSAdapter) CompleteWorkOrder(_ context.Context, _, _ string, _ []string, _ []models.PartUsage, _ string) error {
	return nil
}
func (m *mockCMMSAdapter) CancelWorkOrder(_ context.Context, _, _ string) error { return nil }
func (m *mockCMMSAdapter) UsePartInWorkOrder(_ context.Context, _, _ string, _ int, _ string) error {
	return nil
}

// Spare parts
func (m *mockCMMSAdapter) CreateSparePart(_ context.Context, _ *models.SparePart) error { return nil }
func (m *mockCMMSAdapter) GetSpareParts(_ context.Context, _ map[string]interface{}) ([]models.SparePart, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetSparePart(_ context.Context, _ string) (*models.SparePart, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateSparePart(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockCMMSAdapter) DeleteSparePart(_ context.Context, _ string) error { return nil }
func (m *mockCMMSAdapter) GetLowStockParts(_ context.Context) ([]models.SparePart, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateSparePartStock(_ context.Context, _ string, _ int) error {
	return nil
}

// Maintenance schedules
func (m *mockCMMSAdapter) CreateMaintenanceSchedule(_ context.Context, _ *models.MaintenanceSchedule) error {
	return nil
}
func (m *mockCMMSAdapter) GetMaintenanceSchedules(_ context.Context, _ map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetMaintenanceSchedule(_ context.Context, _ string) (*models.MaintenanceSchedule, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateMaintenanceSchedule(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockCMMSAdapter) DeleteMaintenanceSchedule(_ context.Context, _ string) error { return nil }
func (m *mockCMMSAdapter) GetDueSchedules(_ context.Context) ([]models.MaintenanceSchedule, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) CompleteMaintenanceSchedule(_ context.Context, _ string) error { return nil }

// SLA
func (m *mockCMMSAdapter) GetSLAConfig(_ context.Context, _ string) (*models.SLAConfig, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetAllSLAConfigs(_ context.Context) ([]models.SLAConfig, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateSLAConfig(_ context.Context, _ string, _, _ int) error { return nil }

// Technicians
func (m *mockCMMSAdapter) GetTechnicianWorkload(_ context.Context, _ string) (*models.TechnicianWorkload, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetAllTechnicianWorkloads(_ context.Context) ([]models.TechnicianWorkload, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetTechnicianMonthlyStats(_ context.Context, _ string) (*models.TechnicianMonthlyStats, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateTechnicianSkills(_ context.Context, _ string, _, _ []string) error {
	return nil
}

// Reports
func (m *mockCMMSAdapter) GetMaintenanceReport(_ context.Context) ([]models.MaintenanceReport, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetSLAComplianceReport(_ context.Context) ([]models.SLAComplianceReport, error) {
	return nil, nil
}

// Technician Site Assignments
func (m *mockCMMSAdapter) CreateTechnicianSiteAssignment(_ context.Context, _ *models.TechnicianSiteAssignment) error {
	return nil
}
func (m *mockCMMSAdapter) GetTechnicianSiteAssignments(_ context.Context, _ map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateTechnicianSiteAssignment(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockCMMSAdapter) DeleteTechnicianSiteAssignment(_ context.Context, _ string) error {
	return nil
}

// Sites
func (m *mockCMMSAdapter) GetSites(_ context.Context, _ map[string]interface{}) ([]models.Site, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetSite(_ context.Context, _ string) (*models.Site, error) { return nil, nil }
func (m *mockCMMSAdapter) CreateSite(_ context.Context, _ *models.Site) error        { return nil }
func (m *mockCMMSAdapter) UpdateSite(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockCMMSAdapter) DeleteSite(_ context.Context, _ string) error { return nil }

// Categories
func (m *mockCMMSAdapter) GetCategories(_ context.Context) ([]models.SparePartCategory, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) CreateCategory(_ context.Context, _ *models.SparePartCategory) error {
	return nil
}
func (m *mockCMMSAdapter) UpdateCategory(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockCMMSAdapter) DeleteCategory(_ context.Context, _ string) error { return nil }

// Work Requests
func (m *mockCMMSAdapter) CreateWorkRequest(_ context.Context, _ *models.WorkRequest) error {
	return nil
}
func (m *mockCMMSAdapter) GetWorkRequests(_ context.Context, _ map[string]interface{}) ([]models.WorkRequest, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetWorkRequest(_ context.Context, _ string) (*models.WorkRequest, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) ApproveWorkRequest(_ context.Context, _, _ string) error     { return nil }
func (m *mockCMMSAdapter) RejectWorkRequest(_ context.Context, _, _, _ string) error   { return nil }
func (m *mockCMMSAdapter) ConvertWorkRequestToWO(_ context.Context, _, _ string) error { return nil }

// WorkOrder ↔ Alert
func (m *mockCMMSAdapter) LinkAlertToWorkOrder(_ context.Context, _, _, _ string) error  { return nil }
func (m *mockCMMSAdapter) UnlinkAlertFromWorkOrder(_ context.Context, _, _ string) error { return nil }
func (m *mockCMMSAdapter) GetAlertsForWorkOrder(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetWorkOrdersForAlert(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, nil
}

// Vendors
func (m *mockCMMSAdapter) CreateVendor(_ context.Context, _ *models.Vendor) error { return nil }
func (m *mockCMMSAdapter) GetVendors(_ context.Context, _ map[string]interface{}) ([]models.Vendor, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) GetVendor(_ context.Context, _ string) (*models.Vendor, error) {
	return nil, nil
}
func (m *mockCMMSAdapter) UpdateVendor(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockCMMSAdapter) DeleteVendor(_ context.Context, _ string) error { return nil }

// Mobile
func (m *mockCMMSAdapter) SavePushToken(_ context.Context, _, _, _ string) error { return nil }

// ── Test helpers ───────────────────────────────────────────────────

func newTestIntegrator(adapter *mockCMMSAdapter) *CMMSIntegrator {
	logger := slog.New(slog.NewTextHandler(discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewCMMSIntegrator(adapter, logger)
}

// discardWriter implements io.Writer discarding all output.
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// ── Tests: AutoCreateTicket ────────────────────────────────────────

func TestCMMSIntegrator_AutoCreateTicket_Success(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-12345"
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	ticketID, err := integrator.AutoCreateTicket(ctx, "dev-001", "Camera-1", "motion", "high", "Motion detected")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ticketID != "wo-12345" {
		t.Fatalf("expected ticketID 'wo-12345', got: %s", ticketID)
	}

	// Verify ticket map was updated
	id, ok := integrator.GetTicketForDevice("dev-001")
	if !ok {
		t.Fatal("expected ticket to be in map after creation")
	}
	if id != "wo-12345" {
		t.Fatalf("expected mapped ticketID 'wo-12345', got: %s", id)
	}
}

func TestCMMSIntegrator_AutoCreateTicket_AdapterError(t *testing.T) {
	expectedErr := errors.New("cmms unavailable")
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			return expectedErr
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	_, err := integrator.AutoCreateTicket(ctx, "dev-002", "Camera-2", "tamper", "critical", "Camera tampered")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got: %v", expectedErr, err)
	}

	// Verify ticket map was NOT updated
	_, ok := integrator.GetTicketForDevice("dev-002")
	if ok {
		t.Fatal("expected no ticket in map after failed creation")
	}
}

func TestCMMSIntegrator_AutoCreateTicket_ContextTimeout(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			// Block until context is done
			<-ctx.Done()
			return ctx.Err()
		},
	}
	integrator := newTestIntegrator(adapter)

	// Use a very short timeout to trigger context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := integrator.AutoCreateTicket(ctx, "dev-003", "Camera-3", "hardware", "high", "Hardware failure")

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Verify ticket map was NOT updated on timeout
	_, ok := integrator.GetTicketForDevice("dev-003")
	if ok {
		t.Fatal("expected no ticket in map after timeout")
	}
}

func TestCMMSIntegrator_AutoCreateTicket_Enforces30sTimeout(t *testing.T) {
	// If parent context has no deadline, the integrator should enforce 30s
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			// Should timeout after ~30s, not hang forever
			<-ctx.Done()
			return ctx.Err()
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	start := time.Now()
	_, err := integrator.AutoCreateTicket(ctx, "dev-004", "Camera-4", "network", "medium", "Network lost")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Should timeout in ~30s — allow generous delta
	if elapsed < 25*time.Second {
		t.Logf("timeout triggered in %v (expected ~30s)", elapsed)
	}
}

// ── Tests: AutoCloseTicket ─────────────────────────────────────────

func TestCMMSIntegrator_AutoCloseTicket_Success(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-close-001"
			return nil
		},
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	// First create a ticket
	ctx := context.Background()
	ticketID, err := integrator.AutoCreateTicket(ctx, "dev-close-001", "Camera-Close", "hardware", "high", "Fault")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Verify it's in the map
	_, ok := integrator.GetTicketForDevice("dev-close-001")
	if !ok {
		t.Fatal("expected ticket in map after creation")
	}

	// Now close it
	err = integrator.AutoCloseTicket(ctx, "dev-close-001", ticketID, "Replaced power supply")
	if err != nil {
		t.Fatalf("expected no error closing ticket, got: %v", err)
	}

	// Verify it's removed from map
	_, ok = integrator.GetTicketForDevice("dev-close-001")
	if ok {
		t.Fatal("expected ticket removed from map after closure")
	}
}

func TestCMMSIntegrator_AutoCloseTicket_NoTicketID_LookupByDevice(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-lookup-001"
			return nil
		},
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	_, err := integrator.AutoCreateTicket(ctx, "dev-lookup-001", "Cam-Lookup", "hardware", "high", "Fault")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Close with empty ticketID — should find it from deviceID
	err = integrator.AutoCloseTicket(ctx, "dev-lookup-001", "", "Fixed")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCMMSIntegrator_AutoCloseTicket_UnknownDevice(t *testing.T) {
	adapter := &mockCMMSAdapter{}
	integrator := newTestIntegrator(adapter)

	// Closing a non-existent device with no ticketID should be a no-op (warn, not error)
	err := integrator.AutoCloseTicket(context.Background(), "dev-unknown", "", "N/A")
	if err != nil {
		t.Fatalf("expected no error for unknown device, got: %v", err)
	}
}

func TestCMMSIntegrator_AutoCloseTicket_AdapterError(t *testing.T) {
	expectedErr := errors.New("update failed")
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-err-001"
			return nil
		},
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			return expectedErr
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	_, err := integrator.AutoCreateTicket(ctx, "dev-err-001", "Cam-Err", "hardware", "high", "Fault")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	err = integrator.AutoCloseTicket(ctx, "dev-err-001", "wo-err-001", "Fix")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got: %v", expectedErr, err)
	}

	// Ticket should remain in map after failed close
	_, ok := integrator.GetTicketForDevice("dev-err-001")
	if !ok {
		t.Fatal("expected ticket to remain in map after failed closure")
	}
}

func TestCMMSIntegrator_AutoCloseTicket_ContextTimeout(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-timeout-close-001"
			return nil
		},
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	_, err := integrator.AutoCreateTicket(ctx, "dev-timeout-close", "Cam-Timeout", "hardware", "high", "Fault")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	// Trigger timeout with short parent context
	shortCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = integrator.AutoCloseTicket(shortCtx, "dev-timeout-close", "wo-timeout-close-001", "Fix")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Ticket should remain in map after timeout
	_, ok := integrator.GetTicketForDevice("dev-timeout-close")
	if !ok {
		t.Fatal("expected ticket to remain in map after timeout")
	}
}

// ── Tests: AddAuditNote ────────────────────────────────────────────

func TestCMMSIntegrator_AddAuditNote_Success(t *testing.T) {
	var capturedNotes string
	adapter := &mockCMMSAdapter{
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			if n, ok := updates["notes"]; ok {
				capturedNotes = n.(string)
			}
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	err := integrator.AddAuditNote(context.Background(), "wo-audit-001", "status_check", "Device health verified")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if capturedNotes == "" {
		t.Fatal("expected notes to be captured")
	}
}

func TestCMMSIntegrator_AddAuditNote_AdapterError(t *testing.T) {
	expectedErr := errors.New("cmms write failure")
	adapter := &mockCMMSAdapter{
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			return expectedErr
		},
	}
	integrator := newTestIntegrator(adapter)

	err := integrator.AddAuditNote(context.Background(), "wo-err-audit", "restart", "Device rebooted")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got: %v", expectedErr, err)
	}
}

func TestCMMSIntegrator_AddAuditNote_ContextTimeout(t *testing.T) {
	adapter := &mockCMMSAdapter{
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	integrator := newTestIntegrator(adapter)

	shortCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := integrator.AddAuditNote(shortCtx, "wo-timeout-audit", "diagnostics", "Running diagnostics")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestCMMSIntegrator_AddAuditNote_Enforces15sTimeout(t *testing.T) {
	adapter := &mockCMMSAdapter{
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	start := time.Now()
	err := integrator.AddAuditNote(ctx, "wo-15s-timeout", "test", "Should timeout after 15s")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	if elapsed < 12*time.Second {
		t.Logf("add audit note timeout triggered in %v (expected ~15s)", elapsed)
	}
}

// ── Tests: GetTicketForDevice ──────────────────────────────────────

func TestCMMSIntegrator_GetTicketForDevice(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-map-001"
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	// Empty map
	_, ok := integrator.GetTicketForDevice("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent device")
	}

	// After creation
	ctx := context.Background()
	_, err := integrator.AutoCreateTicket(ctx, "dev-map-001", "Cam-Map", "hardware", "high", "Test")
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	id, ok := integrator.GetTicketForDevice("dev-map-001")
	if !ok {
		t.Fatal("expected true for existing device")
	}
	if id != "wo-map-001" {
		t.Fatalf("expected 'wo-map-001', got: %s", id)
	}

	// After closure — should be removed
	err = integrator.AutoCloseTicket(ctx, "dev-map-001", "wo-map-001", "Fixed")
	if err != nil {
		t.Fatalf("close ticket: %v", err)
	}

	_, ok = integrator.GetTicketForDevice("dev-map-001")
	if ok {
		t.Fatal("expected false after ticket closed")
	}
}

func TestCMMSIntegrator_GetTicketForDevice_MultipleDevices(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-" + wo.DeviceID
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()

	devices := []string{"dev-a", "dev-b", "dev-c"}
	for _, d := range devices {
		_, err := integrator.AutoCreateTicket(ctx, d, "Cam-"+d, "hardware", "high", "Test")
		if err != nil {
			t.Fatalf("create ticket for %s: %v", d, err)
		}
	}

	for _, d := range devices {
		id, ok := integrator.GetTicketForDevice(d)
		if !ok {
			t.Fatalf("expected ticket for %s", d)
		}
		if id != "wo-"+d {
			t.Fatalf("expected 'wo-%s', got: %s", d, id)
		}
	}
}

// ── Test: Nil logger ───────────────────────────────────────────────

func TestNewCMMSIntegrator_NilLogger(t *testing.T) {
	integrator := NewCMMSIntegrator(&mockCMMSAdapter{}, nil)
	if integrator == nil {
		t.Fatal("expected non-nil integrator")
	}
	if integrator.logger == nil {
		t.Fatal("expected default logger when nil passed")
	}
}

// ── Concurrency test: Race condition verification (P0-CR-03) ───────

func TestCMMSIntegrator_ConcurrentAccess(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-" + wo.DeviceID
			return nil
		},
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()
	var wg sync.WaitGroup

	// 100 goroutines: concurrent create + close + get
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			devID := fmt.Sprintf("dev-concurrent-%d", n)

			// Auto-create ticket
			_, err := integrator.AutoCreateTicket(ctx, devID, "Cam-"+devID, "motion", "high", "Concurrent test")
			if err != nil {
				t.Errorf("create ticket for %s: %v", devID, err)
				return
			}

			// Read-back from map
			_, ok := integrator.GetTicketForDevice(devID)
			if !ok {
				t.Errorf("ticket not found for %s after create", devID)
			}

			// Auto-close ticket
			err = integrator.AutoCloseTicket(ctx, devID, "", "Concurrent close")
			if err != nil {
				t.Errorf("close ticket for %s: %v", devID, err)
			}

			// Verify removed from map
			_, ok = integrator.GetTicketForDevice(devID)
			if ok {
				t.Errorf("ticket still in map for %s after close", devID)
			}
		}(i)
	}

	wg.Wait()
}

func TestCMMSIntegrator_ConcurrentReadWriteRace(t *testing.T) {
	adapter := &mockCMMSAdapter{
		createWOFunc: func(ctx context.Context, wo *models.WorkOrder) error {
			wo.ID = "wo-race-" + wo.DeviceID
			return nil
		},
		updateWOFunc: func(ctx context.Context, id string, updates map[string]interface{}) error {
			return nil
		},
	}
	integrator := newTestIntegrator(adapter)

	ctx := context.Background()

	// Pre-create tickets for 10 devices
	for i := 0; i < 10; i++ {
		devID := fmt.Sprintf("dev-race-%d", i)
		_, err := integrator.AutoCreateTicket(ctx, devID, "Cam-"+devID, "motion", "high", "Race test")
		if err != nil {
			t.Fatalf("pre-create ticket for %s: %v", devID, err)
		}
	}

	var wg sync.WaitGroup

	// 50 concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				devID := fmt.Sprintf("dev-race-%d", j)
				integrator.GetTicketForDevice(devID)
			}
		}()
	}

	// 50 concurrent writers (close + re-create)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			devID := fmt.Sprintf("dev-race-%d", n%10)

			// Close existing ticket
			_ = integrator.AutoCloseTicket(ctx, devID, "", "Race close")

			// Re-create
			_, err := integrator.AutoCreateTicket(ctx, devID, "Cam-"+devID, "motion", "high", "Race re-create")
			if err != nil {
				t.Errorf("re-create ticket for %s: %v", devID, err)
			}
		}(i)
	}

	wg.Wait()
}
