package cmms

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"gb-telemetry-collector/internal/events"
	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Mock CMMSAdapter
// ═══════════════════════════════════════════════════════════════════════

type mockAdapter struct {
	createWOCalled    bool
	completeWOCalled  bool
	updateWOCalled    bool
	getWOCalled       bool
	lastWorkOrder     *models.WorkOrder
	lastUpdateID      string
	lastUpdateUpdates map[string]interface{}
	shouldFail        bool
	failCount         int
}

func (m *mockAdapter) CreateWorkOrder(_ context.Context, wo *models.WorkOrder) error {
	m.createWOCalled = true
	m.lastWorkOrder = wo
	if m.shouldFail {
		m.failCount++
		return modelError("mock: create work order failed")
	}
	return nil
}

func (m *mockAdapter) GetWorkOrders(_ context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	m.getWOCalled = true
	if m.shouldFail {
		return nil, modelError("mock: get work orders failed")
	}
	return []models.WorkOrder{
		{ID: "wo-1", Status: "in_progress"},
	}, nil
}

func (m *mockAdapter) GetWorkOrder(_ context.Context, id string) (*models.WorkOrder, error) {
	return &models.WorkOrder{ID: id, Status: "requested"}, nil
}

func (m *mockAdapter) UpdateWorkOrder(_ context.Context, id string, updates map[string]interface{}) error {
	m.updateWOCalled = true
	m.lastUpdateID = id
	m.lastUpdateUpdates = updates
	if m.shouldFail {
		return modelError("mock: update work order failed")
	}
	return nil
}

func (m *mockAdapter) AssignWorkOrder(_ context.Context, id, userID string) error {
	return nil
}

func (m *mockAdapter) StartWorkOrder(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) CompleteWorkOrder(_ context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	m.completeWOCalled = true
	if m.shouldFail {
		return modelError("mock: complete work order failed")
	}
	return nil
}

func (m *mockAdapter) CancelWorkOrder(_ context.Context, id, reason string) error {
	return nil
}

func (m *mockAdapter) UsePartInWorkOrder(_ context.Context, workOrderID, partID string, quantity int, userID string) error {
	return nil
}

func (m *mockAdapter) CreateSparePart(_ context.Context, part *models.SparePart) error {
	return nil
}

func (m *mockAdapter) GetSpareParts(_ context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	return nil, nil
}

func (m *mockAdapter) GetSparePart(_ context.Context, id string) (*models.SparePart, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateSparePart(_ context.Context, id string, updates map[string]interface{}) error {
	return nil
}

func (m *mockAdapter) DeleteSparePart(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) GetLowStockParts(_ context.Context) ([]models.SparePart, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateSparePartStock(_ context.Context, id string, quantity int) error {
	return nil
}

func (m *mockAdapter) CreateMaintenanceSchedule(_ context.Context, schedule *models.MaintenanceSchedule) error {
	return nil
}

func (m *mockAdapter) GetMaintenanceSchedules(_ context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	return nil, nil
}

func (m *mockAdapter) GetMaintenanceSchedule(_ context.Context, id string) (*models.MaintenanceSchedule, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateMaintenanceSchedule(_ context.Context, id string, updates map[string]interface{}) error {
	return nil
}

func (m *mockAdapter) DeleteMaintenanceSchedule(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) GetDueSchedules(_ context.Context) ([]models.MaintenanceSchedule, error) {
	return nil, nil
}

func (m *mockAdapter) CompleteMaintenanceSchedule(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) GetSLAConfig(_ context.Context, priority string) (*models.SLAConfig, error) {
	return nil, nil
}

func (m *mockAdapter) GetAllSLAConfigs(_ context.Context) ([]models.SLAConfig, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateSLAConfig(_ context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	return nil
}

func (m *mockAdapter) GetTechnicianWorkload(_ context.Context, userID string) (*models.TechnicianWorkload, error) {
	return nil, nil
}

func (m *mockAdapter) GetAllTechnicianWorkloads(_ context.Context) ([]models.TechnicianWorkload, error) {
	return nil, nil
}

func (m *mockAdapter) GetTechnicianMonthlyStats(_ context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateTechnicianSkills(_ context.Context, userID string, skills []string, certifications []string) error {
	return nil
}

func (m *mockAdapter) GetMaintenanceReport(_ context.Context) ([]models.MaintenanceReport, error) {
	return nil, nil
}

func (m *mockAdapter) GetSLAComplianceReport(_ context.Context) ([]models.SLAComplianceReport, error) {
	return nil, nil
}

func (m *mockAdapter) CreateTechnicianSiteAssignment(_ context.Context, assignment *models.TechnicianSiteAssignment) error {
	return nil
}

func (m *mockAdapter) GetTechnicianSiteAssignments(_ context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateTechnicianSiteAssignment(_ context.Context, id string, updates map[string]interface{}) error {
	return nil
}

func (m *mockAdapter) DeleteTechnicianSiteAssignment(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) GetSites(_ context.Context, _ map[string]interface{}) ([]models.Site, error) {
	return nil, nil
}

func (m *mockAdapter) GetSite(_ context.Context, id string) (*models.Site, error) {
	return nil, nil
}

func (m *mockAdapter) CreateSite(_ context.Context, site *models.Site) error {
	return nil
}

func (m *mockAdapter) UpdateSite(_ context.Context, id string, updates map[string]interface{}) error {
	return nil
}

func (m *mockAdapter) DeleteSite(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) GetCategories(_ context.Context) ([]models.SparePartCategory, error) {
	return nil, nil
}

func (m *mockAdapter) CreateCategory(_ context.Context, cat *models.SparePartCategory) error {
	return nil
}

func (m *mockAdapter) UpdateCategory(_ context.Context, id string, updates map[string]interface{}) error {
	return nil
}

func (m *mockAdapter) DeleteCategory(_ context.Context, id string) error {
	return nil
}

func (m *mockAdapter) CreateWorkRequest(_ context.Context, _ *models.WorkRequest) error {
	return nil
}

func (m *mockAdapter) GetWorkRequests(_ context.Context, _ map[string]interface{}) ([]models.WorkRequest, error) {
	return nil, nil
}

func (m *mockAdapter) GetWorkRequest(_ context.Context, _ string) (*models.WorkRequest, error) {
	return nil, nil
}

func (m *mockAdapter) ApproveWorkRequest(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockAdapter) RejectWorkRequest(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockAdapter) ConvertWorkRequestToWO(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockAdapter) SavePushToken(_ context.Context, userID, token, platform string) error {
	return nil
}

// ── Vendors (INV-7.2.1) ──────────────────────────────────────────

func (m *mockAdapter) CreateVendor(_ context.Context, _ *models.Vendor) error {
	return nil
}

func (m *mockAdapter) GetVendors(_ context.Context, _ map[string]interface{}) ([]models.Vendor, error) {
	return nil, nil
}

func (m *mockAdapter) GetVendor(_ context.Context, _ string) (*models.Vendor, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateVendor(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}

func (m *mockAdapter) DeleteVendor(_ context.Context, _ string) error {
	return nil
}

// ── WorkOrder ↔ Alert (DM-1.3.1) ────────────────────────────────

func (m *mockAdapter) LinkAlertToWorkOrder(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockAdapter) UnlinkAlertFromWorkOrder(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockAdapter) GetAlertsForWorkOrder(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, nil
}

func (m *mockAdapter) GetWorkOrdersForAlert(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, nil
}

type modelError string

func (e modelError) Error() string { return string(e) }

// ═══════════════════════════════════════════════════════════════════════
// Mock AuditLogger
// ═══════════════════════════════════════════════════════════════════════

type mockAuditLogger struct {
	events []DispatcherEvent
}

func (m *mockAuditLogger) LogDispatcherEvent(_ context.Context, event DispatcherEvent) error {
	m.events = append(m.events, event)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Tests
// ═══════════════════════════════════════════════════════════════════════

func setupTestDispatcher(t *testing.T, adapter *mockAdapter) (*EventDispatcher, string) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg := DispatcherConfig{
		CircuitBreakerThreshold: 5,
		CircuitBreakerResetTime: 100 * time.Millisecond,
		FallbackMaxRetries:      3,
		WorkerPoolSize:          2,
		AuditLogEnabled:         true,
	}

	// Создаём NATS subscriber (без подключения — для теста используем nil)
	// В тестах мы не запускаем реальный NATS, а тестируем dispatchBySource напрямую.
	_ = logger
	_ = cfg
	_ = adapter

	return nil, fallbackDir
}

// TestCircuitBreaker проверяет базовую работу circuit breaker.
func TestCircuitBreaker(t *testing.T) {
	cb := newCircuitBreaker(3, 50*time.Millisecond)

	// Начальное состояние — closed
	if !cb.allow() {
		t.Error("expected circuit breaker to allow initially")
	}

	// 3 ошибки → open
	cb.failure()
	cb.failure()
	cb.failure()

	if cb.allow() {
		t.Error("expected circuit breaker to be open after 3 failures")
	}

	// Ждём reset
	time.Sleep(60 * time.Millisecond)

	if !cb.allow() {
		t.Error("expected circuit breaker to allow after reset timeout")
	}

	// success → closed
	cb.success()

	// После success должно быть снова closed
	if !cb.allow() {
		t.Error("expected circuit breaker to be closed after success")
	}
}

// TestCircuitBreakerHalfOpen проверяет half-open → closed transition.
func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(2, 50*time.Millisecond)

	cb.failure()
	cb.failure()

	// Open
	if cb.allow() {
		t.Error("expected open")
	}

	// Ждём reset
	time.Sleep(60 * time.Millisecond)

	// Half-open — allow one
	if !cb.allow() {
		t.Error("expected half-open to allow")
	}

	// Success → closed
	cb.success()

	// After success, closed again — should allow
	if !cb.allow() {
		t.Error("expected closed after success in half-open")
	}
}

// TestMapSeverityToPriority проверяет маппинг severity → priority.
func TestMapSeverityToPriority(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}
	auditLogger := &mockAuditLogger{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, auditLogger, DispatcherConfig{
		CircuitBreakerThreshold: 5,
		CircuitBreakerResetTime: time.Second,
		FallbackMaxRetries:      3,
		WorkerPoolSize:          2,
		AuditLogEnabled:         false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	tests := []struct {
		severity string
		expected string
	}{
		{"critical", "critical"},
		{"high", "high"},
		{"medium", "medium"},
		{"low", "low"},
		{"unknown", "medium"},
	}

	for _, tt := range tests {
		result := d.mapSeverityToPriority(tt.severity)
		if result != tt.expected {
			t.Errorf("mapSeverityToPriority(%q) = %q, want %q", tt.severity, result, tt.expected)
		}
	}
}

// TestMapProbabilityToPriority проверяет маппинг probability → priority.
func TestMapProbabilityToPriority(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	d, err := NewEventDispatcher(&mockAdapter{}, nil, os.TempDir(), nil, DispatcherConfig{
		AuditLogEnabled: false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	tests := []struct {
		prob     float64
		expected string
	}{
		{0.99, "critical"},
		{0.95, "critical"},
		{0.90, "high"},
		{0.85, "high"},
		{0.82, "medium"},
		{0.80, "medium"},
		{0.70, "low"},
	}

	for _, tt := range tests {
		result := d.mapProbabilityToPriority(tt.prob)
		if result != tt.expected {
			t.Errorf("mapProbabilityToPriority(%f) = %q, want %q", tt.prob, result, tt.expected)
		}
	}
}

// TestHandleAlarmCreated проверяет создание WorkOrder из AlarmEvent.
func TestHandleAlarmCreated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}
	auditLogger := &mockAuditLogger{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, auditLogger, DispatcherConfig{
		CircuitBreakerThreshold: 5,
		CircuitBreakerResetTime: time.Second,
		FallbackMaxRetries:      3,
		WorkerPoolSize:          2,
		AuditLogEnabled:         false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	alarmEvent := events.AlarmEvent{
		DeviceID:   "device-1",
		DeviceName: "Camera-101",
		Type:       "motion",
		Severity:   "critical",
		Message:    "Motion detected in Zone A",
		Timestamp:  time.Now(),
	}

	data, _ := json.Marshal(alarmEvent)

	err = d.handleAlarmCreated(context.Background(), adapter, data)
	if err != nil {
		t.Fatalf("handleAlarmCreated failed: %v", err)
	}

	if !adapter.createWOCalled {
		t.Fatal("expected CreateWorkOrder to be called")
	}

	if adapter.lastWorkOrder == nil {
		t.Fatal("expected work order to be created")
	}

	if adapter.lastWorkOrder.Type != "corrective" {
		t.Errorf("expected type 'corrective', got %q", adapter.lastWorkOrder.Type)
	}
	if adapter.lastWorkOrder.Priority != "critical" {
		t.Errorf("expected priority 'critical', got %q", adapter.lastWorkOrder.Priority)
	}
	if adapter.lastWorkOrder.DeviceID != "device-1" {
		t.Errorf("expected device_id 'device-1', got %q", adapter.lastWorkOrder.DeviceID)
	}
	if *adapter.lastWorkOrder.CreatedBy != "system:alarm" {
		t.Errorf("expected created_by 'system:alarm', got %q", *adapter.lastWorkOrder.CreatedBy)
	}
}

// TestHandleAlarmResolved проверяет закрытие WO при resolve alarm.
func TestHandleAlarmResolved(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}
	auditLogger := &mockAuditLogger{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, auditLogger, DispatcherConfig{
		AuditLogEnabled: false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	payload := map[string]interface{}{
		"alarm_id":      "alarm-1",
		"resolved_by":   "tech-1",
		"resolution":    "False alarm",
		"auto_resolved": false,
	}
	data, _ := json.Marshal(payload)

	err = d.handleAlarmResolved(context.Background(), adapter, data)
	if err != nil {
		t.Fatalf("handleAlarmResolved failed: %v", err)
	}

	if !adapter.getWOCalled {
		t.Error("expected GetWorkOrders to be called")
	}
}

// TestHandlePredictionCreatedLowProb проверяет, что при низкой вероятности
// preventive WO не создаётся.
func TestHandlePredictionCreatedLowProb(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}
	auditLogger := &mockAuditLogger{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, auditLogger, DispatcherConfig{
		AuditLogEnabled: false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	// Низкая вероятность — WO не создаётся
	predictionEvent := events.PredictionEvent{
		DeviceID:    "device-1",
		FailureMode: "HDD Failure",
		Probability: 0.5,
	}

	data, _ := json.Marshal(predictionEvent)
	err = d.handlePredictionCreated(context.Background(), adapter, data)
	if err != nil {
		t.Fatalf("handlePredictionCreated failed: %v", err)
	}

	if adapter.createWOCalled {
		t.Error("expected NO CreateWorkOrder for probability < 0.8")
	}
}

// TestHandlePredictionCreatedHighProb проверяет создание preventive WO
// при высокой вероятности отказа.
func TestHandlePredictionCreatedHighProb(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}
	auditLogger := &mockAuditLogger{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, auditLogger, DispatcherConfig{
		AuditLogEnabled: false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	predictionEvent := events.PredictionEvent{
		DeviceID:       "device-1",
		DeviceName:     "Camera-101",
		FailureMode:    "HDD Failure",
		Probability:    0.92,
		EstimatedDays:  14,
		Recommendation: "Replace HDD within 2 weeks",
		Timestamp:      time.Now(),
	}

	data, _ := json.Marshal(predictionEvent)
	err = d.handlePredictionCreated(context.Background(), adapter, data)
	if err != nil {
		t.Fatalf("handlePredictionCreated failed: %v", err)
	}

	if !adapter.createWOCalled {
		t.Fatal("expected CreateWorkOrder to be called for probability >= 0.8")
	}

	if adapter.lastWorkOrder.Type != "preventive" {
		t.Errorf("expected type 'preventive', got %q", adapter.lastWorkOrder.Type)
	}
	if adapter.lastWorkOrder.Priority != "high" {
		t.Errorf("expected priority 'high' for p=0.92, got %q", adapter.lastWorkOrder.Priority)
	}
	if *adapter.lastWorkOrder.CreatedBy != "system:prediction" {
		t.Errorf("expected created_by 'system:prediction', got %q", *adapter.lastWorkOrder.CreatedBy)
	}
}

// TestHandleCMMSWOStatusChanged проверяет обновление статуса WO.
func TestHandleCMMSWOStatusChanged(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}
	auditLogger := &mockAuditLogger{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, auditLogger, DispatcherConfig{
		AuditLogEnabled: false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	cmmsEvent := events.CMMSEvent{
		Event:       "status_changed",
		WorkOrderID: "wo-123",
		Status:      "in_progress",
	}

	data, _ := json.Marshal(cmmsEvent)
	err = d.handleCMMSWOStatusChanged(context.Background(), adapter, data)
	if err != nil {
		t.Fatalf("handleCMMSWOStatusChanged failed: %v", err)
	}

	if !adapter.updateWOCalled {
		t.Fatal("expected UpdateWorkOrder to be called")
	}
	if adapter.lastUpdateID != "wo-123" {
		t.Errorf("expected work order id 'wo-123', got %q", adapter.lastUpdateID)
	}
	if adapter.lastUpdateUpdates["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got %v", adapter.lastUpdateUpdates["status"])
	}
}

// TestFallbackQueueIntegration проверяет интеграцию с FallbackQueue.
func TestFallbackQueueIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	// Создаём FallbackQueue напрямую
	fq, err := NewFallbackQueue(fallbackDir, 3, logger)
	if err != nil {
		t.Fatalf("failed to create fallback queue: %v", err)
	}

	// Энкьюируем тестовое событие
	payload := map[string]interface{}{
		"source":     string(events.SourceAlarms),
		"event_type": "alarm.created",
		"data":       json.RawMessage(`{"device_id":"test-1","type":"motion","severity":"critical","message":"test"}`),
	}
	err = fq.Enqueue("dispatch_alarm.created", payload)
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Проверяем что запись есть
	entries, err := fq.Pending()
	if err != nil {
		t.Fatalf("failed to list pending: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 pending entry, got %d", len(entries))
	}

	// Проверяем удаление
	err = fq.Remove(entries[0].ID)
	if err != nil {
		t.Fatalf("failed to remove entry: %v", err)
	}

	entries, err = fq.Pending()
	if err != nil {
		t.Fatalf("failed to list pending after remove: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 pending entries after remove, got %d", len(entries))
	}
}

// TestHealthMetrics проверяет Health() и Metrics().
func TestHealthMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	adapter := &mockAdapter{}

	fallbackDir, err := os.MkdirTemp("", "cmms-dispatcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(adapter, nil, fallbackDir, nil, DispatcherConfig{
		CircuitBreakerThreshold: 5,
		CircuitBreakerResetTime: time.Second,
		FallbackMaxRetries:      3,
		WorkerPoolSize:          2,
		AuditLogEnabled:         false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	health := d.Health()
	if health["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", health["status"])
	}
	if health["circuit_breaker"] != "closed" {
		t.Errorf("expected circuit_breaker 'closed', got %v", health["circuit_breaker"])
	}

	metrics := d.Metrics()
	if metrics["events_processed_total"] != 0 {
		t.Errorf("expected 0 processed, got %d", metrics["events_processed_total"])
	}
}

// TestDispatcherConfigDefaults проверяет значения по умолчанию.
func TestDispatcherConfigDefaults(t *testing.T) {
	cfg := DispatcherConfig{}
	cfg.validate()

	if cfg.CircuitBreakerThreshold != 5 {
		t.Errorf("expected CircuitBreakerThreshold=5, got %d", cfg.CircuitBreakerThreshold)
	}
	if cfg.CircuitBreakerResetTime != 30*time.Second {
		t.Errorf("expected CircuitBreakerResetTime=30s, got %v", cfg.CircuitBreakerResetTime)
	}
	if cfg.FallbackMaxRetries != 10 {
		t.Errorf("expected FallbackMaxRetries=10, got %d", cfg.FallbackMaxRetries)
	}
	if cfg.WorkerPoolSize != 4 {
		t.Errorf("expected WorkerPoolSize=4, got %d", cfg.WorkerPoolSize)
	}
}

// TestAdapterName проверяет определение имени адаптера.
func TestAdapterName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	internalAdapter := &mockAdapter{}
	fallbackDir, _ := os.MkdirTemp("", "cmms-dispatcher-test-*")
	defer os.RemoveAll(fallbackDir)

	d, err := NewEventDispatcher(internalAdapter, nil, fallbackDir, nil, DispatcherConfig{
		AuditLogEnabled: false,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create dispatcher: %v", err)
	}

	// Для mockAdapter имя будет "cmms.mockAdapter" (Go type name)
	name := d.adapterName()
	if name == "" {
		t.Error("expected non-empty adapter name")
	}
}
