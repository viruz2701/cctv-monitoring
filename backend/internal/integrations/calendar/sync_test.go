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
