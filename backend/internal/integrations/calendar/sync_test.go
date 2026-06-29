package calendar

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

// ── Mock Store ────────────────────────────────────────────────────────

type mockStore struct {
	conns    []Connection
	mappings map[string]*EventMapping // key: woID:provider
	syncLogs []SyncLogEntry
}

func newMockStore() *mockStore {
	return &mockStore{
		mappings: make(map[string]*EventMapping),
	}
}

func (m *mockStore) ListConnections(_ context.Context) ([]Connection, error) {
	return m.conns, nil
}

func (m *mockStore) GetConnection(_ context.Context, provider, userID string) (*Connection, error) {
	for _, c := range m.conns {
		if c.Provider == provider && c.UserID == userID {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockStore) SaveConnection(_ context.Context, conn *Connection) error {
	for i, c := range m.conns {
		if c.UserID == conn.UserID && c.Provider == conn.Provider {
			m.conns[i] = *conn
			return nil
		}
	}
	m.conns = append(m.conns, *conn)
	return nil
}

func (m *mockStore) DeleteConnection(_ context.Context, id string) error {
	for i, c := range m.conns {
		if c.ID == id {
			m.conns = append(m.conns[:i], m.conns[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockStore) GetEventMapping(_ context.Context, woID, provider string) (*EventMapping, error) {
	key := woID + ":" + provider
	em, ok := m.mappings[key]
	if !ok {
		return nil, nil
	}
	return em, nil
}

func (m *mockStore) SaveEventMapping(_ context.Context, mapping *EventMapping) error {
	key := mapping.WOID + ":" + mapping.Provider
	m.mappings[key] = mapping
	return nil
}

func (m *mockStore) DeleteEventMapping(_ context.Context, woID, provider string) error {
	key := woID + ":" + provider
	delete(m.mappings, key)
	return nil
}

func (m *mockStore) ListEventMappingsByProvider(_ context.Context, provider string) ([]EventMapping, error) {
	var result []EventMapping
	for _, em := range m.mappings {
		if em.Provider == provider {
			result = append(result, *em)
		}
	}
	return result, nil
}

func (m *mockStore) LogSync(_ context.Context, entry *SyncLogEntry) error {
	m.syncLogs = append(m.syncLogs, *entry)
	return nil
}

// ── Mock Provider ─────────────────────────────────────────────────────

type mockProvider struct {
	name       string
	events     map[string]string // woID → externalID
	createErr  error
	updateErr  error
	deleteErr  error
	syncResult []CalendarChange
	syncErr    error
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:   name,
		events: make(map[string]string),
	}
}

func (m *mockProvider) CreateEvent(_ context.Context, wo WorkOrderEvent) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	extID := "ext-" + wo.ID + "-" + m.name
	m.events[wo.ID] = extID
	return extID, nil
}

func (m *mockProvider) UpdateEvent(_ context.Context, eventID string, wo WorkOrderEvent) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.events[wo.ID] = eventID
	return nil
}

func (m *mockProvider) DeleteEvent(_ context.Context, eventID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for woID, eid := range m.events {
		if eid == eventID {
			delete(m.events, woID)
			break
		}
	}
	return nil
}

func (m *mockProvider) SyncChanges(_ context.Context, _ time.Time) ([]CalendarChange, error) {
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	return m.syncResult, nil
}

// ── Tests ─────────────────────────────────────────────────────────────

func TestSyncEngine_PushCreate(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
		{Provider: "outlook", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	engine.RegisterProvider("google", newMockProvider("google"))
	engine.RegisterProvider("outlook", newMockProvider("outlook"))

	wo := WorkOrderEvent{
		ID:          "wo-1",
		Title:       "Test Work Order",
		Description: "Test description",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(1 * time.Hour),
		Status:      "scheduled",
	}

	err := engine.PushCreate(context.Background(), wo)
	if err != nil {
		t.Fatalf("PushCreate failed: %v", err)
	}

	// Проверяем, что маппинги созданы
	for _, provider := range []string{"google", "outlook"} {
		mapping, err := store.GetEventMapping(context.Background(), wo.ID, provider)
		if err != nil {
			t.Fatalf("GetEventMapping(%s) failed: %v", provider, err)
		}
		if mapping == nil {
			t.Fatalf("mapping for %s should not be nil", provider)
		}
		if mapping.ExternalID == "" {
			t.Errorf("ExternalID for %s should not be empty", provider)
		}
		if mapping.Status != "active" {
			t.Errorf("expected status 'active', got %q", mapping.Status)
		}
	}

	// Проверяем, что sync logs созданы
	if len(store.syncLogs) != 2 {
		t.Errorf("expected 2 sync logs, got %d", len(store.syncLogs))
	}
}

func TestSyncEngine_PushUpdate(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
	}
	store.mappings["wo-1:google"] = &EventMapping{
		WOID:       "wo-1",
		Provider:   "google",
		ExternalID: "ext-wo-1-google",
		Status:     "active",
		LastSynced: time.Now(),
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	mock := newMockProvider("google")
	mock.events["wo-1"] = "ext-wo-1-google"
	engine.RegisterProvider("google", mock)

	wo := WorkOrderEvent{
		ID:     "wo-1",
		Title:  "Updated Work Order",
		Status: "in_progress",
	}

	err := engine.PushUpdate(context.Background(), wo)
	if err != nil {
		t.Fatalf("PushUpdate failed: %v", err)
	}

	mapping, _ := store.GetEventMapping(context.Background(), "wo-1", "google")
	if mapping == nil {
		t.Fatal("mapping should exist")
	}
	if mapping.Status != "updated" {
		t.Errorf("expected status 'updated', got %q", mapping.Status)
	}
}

func TestSyncEngine_PushDelete(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
	}
	store.mappings["wo-1:google"] = &EventMapping{
		WOID:       "wo-1",
		Provider:   "google",
		ExternalID: "ext-wo-1-google",
		Status:     "active",
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	mock := newMockProvider("google")
	mock.events["wo-1"] = "ext-wo-1-google"
	engine.RegisterProvider("google", mock)

	err := engine.PushDelete(context.Background(), "wo-1")
	if err != nil {
		t.Fatalf("PushDelete failed: %v", err)
	}

	// Проверяем, что маппинг удалён
	mapping, _ := store.GetEventMapping(context.Background(), "wo-1", "google")
	if mapping != nil {
		t.Error("mapping should be deleted after PushDelete")
	}
}

func TestSyncEngine_PullChanges(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	mock := newMockProvider("google")
	mock.syncResult = []CalendarChange{
		{EventID: "ext-1", Type: "created", Provider: "google", ChangedAt: time.Now()},
		{EventID: "ext-2", Type: "updated", Provider: "google", ChangedAt: time.Now()},
	}
	engine.RegisterProvider("google", mock)

	changes, err := engine.PullChanges(context.Background())
	if err != nil {
		t.Fatalf("PullChanges failed: %v", err)
	}

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Provider != "google" {
		t.Errorf("expected provider 'google', got %q", changes[0].Provider)
	}

	// Проверяем sync logs
	if len(store.syncLogs) != 2 {
		t.Errorf("expected 2 sync logs, got %d", len(store.syncLogs))
	}
}

func TestSyncEngine_DisabledConnection(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: false, UserID: "user1"}, // disabled!
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	engine.RegisterProvider("google", newMockProvider("google"))

	wo := WorkOrderEvent{ID: "wo-1", Title: "Test"}
	err := engine.PushCreate(context.Background(), wo)
	if err != nil {
		t.Fatalf("PushCreate failed: %v", err)
	}

	// Проверяем, что маппинг НЕ создан для disabled connection
	mapping, _ := store.GetEventMapping(context.Background(), "wo-1", "google")
	if mapping != nil {
		t.Error("mapping should not exist for disabled connection")
	}
}

func TestSyncEngine_ProviderError(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	mock := newMockProvider("google")
	mock.createErr = errors.New("API unavailable")
	engine.RegisterProvider("google", mock)

	wo := WorkOrderEvent{ID: "wo-1", Title: "Test"}
	err := engine.PushCreate(context.Background(), wo)
	if err != nil {
		t.Fatalf("PushCreate should not return error on provider failure: %v", err)
	}

	// Проверяем, что sync log содержит ошибку
	if len(store.syncLogs) != 1 {
		t.Fatalf("expected 1 sync log, got %d", len(store.syncLogs))
	}
	if store.syncLogs[0].Status != "error" {
		t.Errorf("expected status 'error', got %q", store.syncLogs[0].Status)
	}
}

func TestSyncEngine_StartStop(t *testing.T) {
	store := newMockStore()
	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	engine.Stop()

	// Повторный Start не должен паниковать
	engine.Start(ctx)
	engine.Stop()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SyncInterval != 5*time.Minute {
		t.Errorf("expected SyncInterval 5m, got %v", cfg.SyncInterval)
	}
	if cfg.ConflictStrategy != "wo_wins" {
		t.Errorf("expected ConflictStrategy 'wo_wins', got %q", cfg.ConflictStrategy)
	}
	if cfg.SyncWindow != 30*24*time.Hour {
		t.Errorf("expected SyncWindow 720h, got %v", cfg.SyncWindow)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P1-CALENDAR: Extended Test Suite (ISO 27001 A.12.4 — audit trail)
// ═══════════════════════════════════════════════════════════════════════

// TestSyncEngine_PushCreate_MultipleProviders проверяет создание событий
// в 2+ провайдерах (Google + Outlook) одной операцией PushCreate.
//
// Ожидается:
//   - Маппинги созданы для обоих провайдеров
//   - Sync logs содержат 2 записи (success)
func TestSyncEngine_PushCreate_MultipleProviders(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
		{Provider: "outlook", Enabled: true, UserID: "user1"},
		{Provider: "yahoo", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	engine.RegisterProvider("google", newMockProvider("google"))
	engine.RegisterProvider("outlook", newMockProvider("outlook"))
	// yahoo не регистрируем — должен быть пропущен без ошибки

	wo := WorkOrderEvent{
		ID:          "wo-multi-1",
		Title:       "Multi Provider Test",
		Description: "Testing multiple providers",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(1 * time.Hour),
		Status:      "scheduled",
	}

	if err := engine.PushCreate(context.Background(), wo); err != nil {
		t.Fatalf("PushCreate failed: %v", err)
	}

	// Проверяем маппинги для зарегистрированных провайдеров
	for _, provider := range []string{"google", "outlook"} {
		mapping, err := store.GetEventMapping(context.Background(), wo.ID, provider)
		if err != nil {
			t.Fatalf("GetEventMapping(%s) failed: %v", provider, err)
		}
		if mapping == nil {
			t.Fatalf("mapping for %s should exist", provider)
		}
		if mapping.ExternalID == "" {
			t.Errorf("ExternalID for %s should not be empty", provider)
		}
		if mapping.Status != "active" {
			t.Errorf("expected status 'active' for %s, got %q", provider, mapping.Status)
		}
	}

	// Yahoo не зарегистрирован — маппинга быть не должно
	mapping, _ := store.GetEventMapping(context.Background(), wo.ID, "yahoo")
	if mapping != nil {
		t.Error("mapping for unregistered provider 'yahoo' should not exist")
	}

	// Проверяем sync logs — должно быть 2 успешных (google + outlook)
	if len(store.syncLogs) != 2 {
		t.Errorf("expected 2 sync logs, got %d", len(store.syncLogs))
	}
	for _, log := range store.syncLogs {
		if log.Status != "success" {
			t.Errorf("expected success status, got %q for provider %s", log.Status, log.Provider)
		}
	}
}

// TestSyncEngine_PushUpdate_MissingMapping проверяет fallback на PushCreate,
// когда маппинг отсутствует (например, после сброса БД).
//
// PushUpdate → GetEventMapping возвращает nil → PushCreate → маппинг создан.
func TestSyncEngine_PushUpdate_MissingMapping(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	engine.RegisterProvider("google", newMockProvider("google"))

	// PushUpdate без предварительного маппинга (wo-1:google отсутствует)
	wo := WorkOrderEvent{
		ID:          "wo-missing-map",
		Title:       "Missing Mapping Test",
		Description: "Should fallback to create",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(1 * time.Hour),
		Status:      "scheduled",
	}

	if err := engine.PushUpdate(context.Background(), wo); err != nil {
		t.Fatalf("PushUpdate failed: %v", err)
	}

	// Проверяем, что маппинг был создан (fallback create)
	mapping, err := store.GetEventMapping(context.Background(), wo.ID, "google")
	if err != nil {
		t.Fatalf("GetEventMapping failed: %v", err)
	}
	if mapping == nil {
		t.Fatal("expected mapping to be created via fallback PushCreate")
	}
	if mapping.ExternalID == "" {
		t.Error("ExternalID should not be empty after fallback create")
	}
	if mapping.Status != "active" {
		t.Errorf("expected status 'active' after fallback, got %q", mapping.Status)
	}
}

// TestSyncEngine_PushCreate_EmptyConnections проверяет PushCreate
// при отсутствии активных подключений (no-op).
//
// Не должно быть ошибок, маппингов или sync logs.
func TestSyncEngine_PushCreate_EmptyConnections(t *testing.T) {
	store := newMockStore()
	// Нет подключений
	store.conns = []Connection{}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	engine.RegisterProvider("google", newMockProvider("google"))

	wo := WorkOrderEvent{
		ID:          "wo-empty-1",
		Title:       "Empty Connections Test",
		Description: "No connections should not cause errors",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(1 * time.Hour),
		Status:      "scheduled",
	}

	if err := engine.PushCreate(context.Background(), wo); err != nil {
		t.Fatalf("PushCreate with empty connections failed: %v", err)
	}

	// Проверяем, что маппинги не созданы
	mapping, _ := store.GetEventMapping(context.Background(), wo.ID, "google")
	if mapping != nil {
		t.Error("mapping should not exist when there are no connections")
	}

	// Sync logs не должны быть созданы
	if len(store.syncLogs) != 0 {
		t.Errorf("expected 0 sync logs, got %d", len(store.syncLogs))
	}
}

// TestSyncEngine_PullChanges_Empty проверяет PullChanges при пустых
// изменениях от всех провайдеров (no-op).
func TestSyncEngine_PullChanges_Empty(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
		{Provider: "outlook", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	mockGoogle := newMockProvider("google")
	mockGoogle.syncResult = []CalendarChange{} // пустой результат
	mockOutlook := newMockProvider("outlook")
	mockOutlook.syncResult = []CalendarChange{} // пустой результат
	engine.RegisterProvider("google", mockGoogle)
	engine.RegisterProvider("outlook", mockOutlook)

	changes, err := engine.PullChanges(context.Background())
	if err != nil {
		t.Fatalf("PullChanges failed: %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}

	// Sync logs не должны быть созданы (нет изменений для логирования)
	if len(store.syncLogs) != 0 {
		t.Errorf("expected 0 sync logs for empty changes, got %d", len(store.syncLogs))
	}
}

// TestSyncEngine_PullChanges_MultipleProviders проверяет получение
// изменений от 2+ провайдеров одновременно.
//
// Ожидается:
//   - Все изменения агрегированы в один срез
//   - Sync logs для каждого изменения
func TestSyncEngine_PullChanges_MultipleProviders(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
		{Provider: "outlook", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())

	mockGoogle := newMockProvider("google")
	mockGoogle.syncResult = []CalendarChange{
		{EventID: "g-evt-1", Type: "created", ExternalID: "g-ext-1", Provider: "google", ChangedAt: time.Now()},
		{EventID: "g-evt-2", Type: "updated", ExternalID: "g-ext-2", Provider: "google", ChangedAt: time.Now()},
	}

	mockOutlook := newMockProvider("outlook")
	mockOutlook.syncResult = []CalendarChange{
		{EventID: "o-evt-1", Type: "created", ExternalID: "o-ext-1", Provider: "outlook", ChangedAt: time.Now()},
		{EventID: "o-evt-2", Type: "deleted", ExternalID: "o-ext-2", Provider: "outlook", ChangedAt: time.Now()},
	}

	engine.RegisterProvider("google", mockGoogle)
	engine.RegisterProvider("outlook", mockOutlook)

	changes, err := engine.PullChanges(context.Background())
	if err != nil {
		t.Fatalf("PullChanges failed: %v", err)
	}

	// Всего должно быть 4 изменения (2 от google + 2 от outlook)
	if len(changes) != 4 {
		t.Fatalf("expected 4 changes (2 google + 2 outlook), got %d", len(changes))
	}

	// Проверяем, что изменения от обоих провайдеров присутствуют
	providerCount := make(map[string]int)
	for _, ch := range changes {
		providerCount[ch.Provider]++
	}
	if providerCount["google"] != 2 {
		t.Errorf("expected 2 google changes, got %d", providerCount["google"])
	}
	if providerCount["outlook"] != 2 {
		t.Errorf("expected 2 outlook changes, got %d", providerCount["outlook"])
	}

	// Проверяем sync logs — должно быть 4 записи
	if len(store.syncLogs) != 4 {
		t.Errorf("expected 4 sync logs, got %d", len(store.syncLogs))
	}
}

// TestSyncEngine_PushDelete_NoMapping проверяет PushDelete без маппинга
// (no-op — не должно быть ошибок или попыток удаления).
func TestSyncEngine_PushDelete_NoMapping(t *testing.T) {
	store := newMockStore()
	store.conns = []Connection{
		{Provider: "google", Enabled: true, UserID: "user1"},
	}

	engine := NewSyncEngine(store, DefaultConfig(), slog.Default())
	mock := newMockProvider("google")
	engine.RegisterProvider("google", mock)

	// PushDelete для WO без маппинга
	err := engine.PushDelete(context.Background(), "wo-no-mapping")
	if err != nil {
		t.Fatalf("PushDelete without mapping failed: %v", err)
	}

	// Маппингов не должно быть создано
	mappings, _ := store.ListEventMappingsByProvider(context.Background(), "google")
	if len(mappings) != 0 {
		t.Errorf("expected 0 mappings, got %d", len(mappings))
	}

	// Sync logs не должны быть созданы (нет операций)
	if len(store.syncLogs) != 0 {
		t.Errorf("expected 0 sync logs for no-op delete, got %d", len(store.syncLogs))
	}
}
