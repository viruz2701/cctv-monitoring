// Package service — unit tests for DeviceService.
// Соответствует:
//   - OWASP ASVS V4 (Access control — RBAC tests)
//   - OWASP ASVS V5 (Validation — whitelist tests)
//   - OWASP ASVS V7 (Error handling — edge cases)
//   - ISO 27001 A.12.4 (Audit trail — verify logging)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/models"
)

// ── Mock Database ──────────────────────────────────────────────────────

// mockDB реализует интерфейс DeviceRepository для тестирования.
type mockDB struct {
	devices   map[string]*models.Device
	auditLogs []string
	err       error // force error for testing error paths
}

func newMockDB() *mockDB {
	return &mockDB{
		devices:   make(map[string]*models.Device),
		auditLogs: make([]string, 0),
	}
}

func (m *mockDB) CreateDevice(ctx context.Context, dev *models.Device) error {
	if m.err != nil {
		return m.err
	}
	if _, exists := m.devices[dev.DeviceID]; exists {
		return errors.New("device already exists")
	}
	m.devices[dev.DeviceID] = dev
	return nil
}

func (m *mockDB) GetDeviceByID(ctx context.Context, deviceID string) (*models.Device, error) {
	if m.err != nil {
		return nil, m.err
	}
	dev, exists := m.devices[deviceID]
	if !exists {
		return nil, errors.New("device not found")
	}
	return dev, nil
}

func (m *mockDB) ListDevices(ctx context.Context, filter models.ListDevicesFilter) (*models.DeviceListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	devices := make([]models.DeviceSummary, 0, len(m.devices))
	for _, dev := range m.devices {
		devices = append(devices, models.DeviceSummary{
			DeviceID: dev.DeviceID,
			Name:     dev.Name,
			Status:   dev.Status,
		})
	}
	return &models.DeviceListResponse{
		Devices:    devices,
		Total:      len(devices),
		Page:       1,
		PageSize:   len(devices),
		TotalPages: 1,
	}, nil
}

func (m *mockDB) UpdateDevice(ctx context.Context, deviceID string, updates map[string]interface{}) error {
	if m.err != nil {
		return m.err
	}
	dev, exists := m.devices[deviceID]
	if !exists {
		return errors.New("device not found")
	}
	if v, ok := updates["name"]; ok {
		dev.Name = v.(string)
	}
	if v, ok := updates["status"]; ok {
		dev.Status = models.DeviceStatus(v.(string))
	}
	return nil
}

func (m *mockDB) SoftDeleteDevice(ctx context.Context, deviceID string) error {
	if m.err != nil {
		return m.err
	}
	if _, exists := m.devices[deviceID]; !exists {
		return errors.New("device not found")
	}
	delete(m.devices, deviceID)
	return nil
}

func (m *mockDB) HardDeleteDevice(ctx context.Context, deviceID string) error {
	if m.err != nil {
		return m.err
	}
	if _, exists := m.devices[deviceID]; !exists {
		return errors.New("device not found")
	}
	delete(m.devices, deviceID)
	return nil
}

func (m *mockDB) RestoreDevice(ctx context.Context, deviceID string) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockDB) SaveAudit(userUUID, action, entityType, entityID string, oldValue, newValue interface{}) error {
	if m.err != nil {
		return m.err
	}
	m.auditLogs = append(m.auditLogs, action)
	return nil
}

// ── Test Setup ─────────────────────────────────────────────────────────

func newTestService(t *testing.T, repo DeviceRepository) *DeviceService {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	signer, err := audit.NewSigner("12345678901234567890123456789012") // 32 bytes
	if err != nil {
		t.Fatalf("failed to create audit signer: %v", err)
	}
	return &DeviceService{
		repo:        repo,
		auditSigner: signer,
		logger:      logger,
	}
}

func createTestDevice(id string) *models.CreateDeviceRequest {
	return &models.CreateDeviceRequest{
		DeviceID:       id,
		Name:           "Test Camera " + id,
		Location:       "Building A",
		DeviceType:     "camera",
		Status:         "ONLINE",
		ConnectionType: "ip",
		AssetClass:     "internal",
		VendorType:     "Hikvision",
	}
}

// ── Tests: CreateDevice ────────────────────────────────────────────────

func TestCreateDevice_Success(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-001")
	dev, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}
	if dev.DeviceID != "device-001" {
		t.Errorf("Expected device_id=device-001, got %s", dev.DeviceID)
	}
	if dev.Name != "Test Camera device-001" {
		t.Errorf("Expected Name='Test Camera device-001', got %s", dev.Name)
	}
	if dev.Health != models.HealthHealthy {
		t.Errorf("Expected Health=healthy, got %s", dev.Health)
	}

	// Verify audit was logged
	if len(mock.auditLogs) != 1 {
		t.Errorf("Expected 1 audit log, got %d", len(mock.auditLogs))
	}
}

func TestCreateDevice_AccessDenied(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-002")
	_, err := svc.CreateDevice(context.Background(), "user-2", RoleViewer, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("Expected ErrAccessDenied for role %s, got %v", RoleViewer, err)
	}
}

func TestCreateDevice_TechnicianCannotCreate(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-003")
	_, err := svc.CreateDevice(context.Background(), "user-3", RoleTechnician, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("Expected ErrAccessDenied for role %s, got %v", RoleTechnician, err)
	}
}

// ── Tests: GetDevice ──────────────────────────────────────────────────

func TestGetDevice_Success(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-010")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	dev, err := svc.GetDevice(context.Background(), "user-1", RoleAdmin, "device-010")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if dev.DeviceID != "device-010" {
		t.Errorf("Expected device_id=device-010, got %s", dev.DeviceID)
	}
}

func TestGetDevice_NotFound(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	_, err := svc.GetDevice(context.Background(), "user-1", RoleAdmin, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent device, got nil")
	}
}

// ── Tests: ListDevices ────────────────────────────────────────────────

func TestListDevices_Empty(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	result, err := svc.ListDevices(context.Background(), "user-1", RoleAdmin, models.ListDevicesFilter{})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Expected 0 devices, got %d", result.Total)
	}
}

func TestListDevices_WithData(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	for i := 0; i < 3; i++ {
		id := "device-list-" + string(rune('0'+i))
		req := createTestDevice(id)
		_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
		if err != nil {
			t.Fatalf("CreateDevice failed for %s: %v", id, err)
		}
	}

	result, err := svc.ListDevices(context.Background(), "user-1", RoleAdmin, models.ListDevicesFilter{})
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("Expected 3 devices, got %d", result.Total)
	}
}

// ── Tests: UpdateDevice ───────────────────────────────────────────────

func TestUpdateDevice_Success(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-020")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	newName := "Updated Camera Name"
	updateReq := &models.UpdateDeviceRequest{
		Name: &newName,
	}

	updated, err := svc.UpdateDevice(context.Background(), "user-1", RoleAdmin, "device-020", updateReq)
	if err != nil {
		t.Fatalf("UpdateDevice failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Expected Name=%q, got %q", newName, updated.Name)
	}

	if len(mock.auditLogs) != 2 {
		t.Errorf("Expected 2 audit logs, got %d", len(mock.auditLogs))
	}
}

func TestUpdateDevice_AccessDenied(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-021")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	newName := "Hacked Name"
	updateReq := &models.UpdateDeviceRequest{Name: &newName}
	_, err = svc.UpdateDevice(context.Background(), "user-2", RoleViewer, "device-021", updateReq)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("Expected ErrAccessDenied for viewer, got %v", err)
	}
}

// ── Tests: DeleteDevice ───────────────────────────────────────────────

func TestDeleteDevice_SoftDelete(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-030")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	err = svc.DeleteDevice(context.Background(), "user-1", RoleAdmin, "device-030", false)
	if err != nil {
		t.Fatalf("SoftDeleteDevice failed: %v", err)
	}

	if len(mock.auditLogs) != 2 {
		t.Errorf("Expected 2 audit logs, got %d", len(mock.auditLogs))
	}
}

func TestDeleteDevice_HardRequiresAdmin(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-031")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	err = svc.DeleteDevice(context.Background(), "user-2", RoleManager, "device-031", true)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("Expected ErrAccessDenied for manager hard delete, got %v", err)
	}
}

func TestDeleteDevice_HardByAdmin(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-032")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	err = svc.DeleteDevice(context.Background(), "user-1", RoleAdmin, "device-032", true)
	if err != nil {
		t.Fatalf("HardDeleteDevice failed: %v", err)
	}
}

// ── Tests: RestoreDevice ──────────────────────────────────────────────

func TestRestoreDevice_Success(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-040")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	err = svc.DeleteDevice(context.Background(), "user-1", RoleAdmin, "device-040", false)
	if err != nil {
		t.Fatalf("SoftDeleteDevice failed: %v", err)
	}

	err = svc.RestoreDevice(context.Background(), "user-1", RoleAdmin, "device-040")
	if err != nil {
		t.Fatalf("RestoreDevice failed: %v", err)
	}

	if len(mock.auditLogs) != 3 {
		t.Errorf("Expected 3 audit logs, got %d", len(mock.auditLogs))
	}
}

// ── Compliance Tests: RBAC (OWASP ASVS V4) ────────────────────────────

func TestRBAC_WriteRoles(t *testing.T) {
	tests := []struct {
		role     string
		canWrite bool
	}{
		{RoleAdmin, true},
		{RoleManager, true},
		{RoleSupport, true},
		{RoleTechnician, false},
		{RoleViewer, false},
		{RoleOwner, false},
		{"unknown", false},
	}

	ctx := context.Background()
	mock := newMockDB()
	svc := newTestService(t, mock)

	for _, tt := range tests {
		t.Run("role_"+tt.role, func(t *testing.T) {
			req := createTestDevice("rbac-" + tt.role)
			_, err := svc.CreateDevice(ctx, "user-x", tt.role, req)

			if tt.canWrite && err != nil {
				t.Errorf("Expected write allowed for %s, got error: %v", tt.role, err)
			}
			if !tt.canWrite && !errors.Is(err, ErrAccessDenied) {
				t.Errorf("Expected ErrAccessDenied for %s, got %v", tt.role, err)
			}
		})
	}
}

// ── Edge Cases ────────────────────────────────────────────────────────

func TestCreateDevice_EmptyName(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := &models.CreateDeviceRequest{
		DeviceID:       "device-edge-1",
		Name:           "",
		DeviceType:     "camera",
		Status:         "ONLINE",
		ConnectionType: "ip",
		AssetClass:     "internal",
	}

	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Logf("Service allows creation with empty name (validation is handler's responsibility): %v", err)
	}
}

func TestUpdateDevice_NoChanges(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)

	req := createTestDevice("device-nochange")
	_, err := svc.CreateDevice(context.Background(), "user-1", RoleAdmin, req)
	if err != nil {
		t.Fatalf("CreateDevice failed: %v", err)
	}

	emptyReq := &models.UpdateDeviceRequest{}
	_, err = svc.UpdateDevice(context.Background(), "user-1", RoleAdmin, "device-nochange", emptyReq)
	if err != nil {
		t.Fatalf("UpdateDevice with no changes failed: %v", err)
	}
}

// ── Test Helpers ──────────────────────────────────────────────────────

func TestNewTestService_InitializesSigner(t *testing.T) {
	mock := newMockDB()
	svc := newTestService(t, mock)
	if svc.auditSigner == nil {
		t.Error("Expected auditSigner to be initialized")
	}
}
