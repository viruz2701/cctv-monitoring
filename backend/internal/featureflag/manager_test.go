package featureflag

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"
)

// mockDB реализует интерфейс DB для тестов.
type mockDB struct {
	mu    sync.Mutex
	flags map[string]models.FeatureFlag
}

func newMockDB() *mockDB {
	return &mockDB{
		flags: make(map[string]models.FeatureFlag),
	}
}

func (m *mockDB) GetAllFeatureFlags(ctx context.Context) ([]models.FeatureFlag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]models.FeatureFlag, 0, len(m.flags))
	for _, f := range m.flags {
		result = append(result, f)
	}
	return result, nil
}

func (m *mockDB) SetFeatureFlagEnabled(ctx context.Context, key string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if f, ok := m.flags[key]; ok {
		f.Enabled = enabled
		f.UpdatedAt = time.Now()
		m.flags[key] = f
	}
	return nil
}

func TestNewManager(t *testing.T) {
	mock := newMockDB()
	mock.flags["test_flag"] = models.FeatureFlag{
		Key:     "test_flag",
		Enabled: true,
	}

	mgr := NewManager(mock, testLogger())
	defer mgr.Stop()

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestIsEnabled(t *testing.T) {
	mock := newMockDB()
	mock.flags["flag_on"] = models.FeatureFlag{Key: "flag_on", Enabled: true}
	mock.flags["flag_off"] = models.FeatureFlag{Key: "flag_off", Enabled: false}

	mgr := NewManager(mock, testLogger())
	defer mgr.Stop()

	// Даём время на загрузку
	time.Sleep(50 * time.Millisecond)

	tests := []struct {
		key      string
		expected bool
	}{
		{"flag_on", true},
		{"flag_off", false},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := mgr.IsEnabled(tt.key)
			if got != tt.expected {
				t.Errorf("IsEnabled(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestSetEnabled(t *testing.T) {
	mock := newMockDB()
	mock.flags["test_flag"] = models.FeatureFlag{Key: "test_flag", Enabled: false}

	mgr := NewManager(mock, testLogger())
	defer mgr.Stop()

	time.Sleep(50 * time.Millisecond)

	if mgr.IsEnabled("test_flag") {
		t.Error("expected flag to be initially disabled")
	}

	if err := mgr.SetEnabled("test_flag", true); err != nil {
		t.Fatalf("SetEnabled failed: %v", err)
	}

	if !mgr.IsEnabled("test_flag") {
		t.Error("expected flag to be enabled after SetEnabled")
	}
}

func TestFailSecure(t *testing.T) {
	mgr := NewManager(newMockDB(), testLogger())
	defer mgr.Stop()

	time.Sleep(50 * time.Millisecond)

	// Несуществующий флаг должен возвращать false (fail-secure)
	if mgr.IsEnabled("nonexistent") {
		t.Error("expected nonexistent flag to be disabled (fail-secure)")
	}
}

func TestGetAll(t *testing.T) {
	mock := newMockDB()
	mock.flags["a"] = models.FeatureFlag{Key: "a", Enabled: true}
	mock.flags["b"] = models.FeatureFlag{Key: "b", Enabled: false}

	mgr := NewManager(mock, testLogger())
	defer mgr.Stop()

	time.Sleep(50 * time.Millisecond)

	all := mgr.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 flags, got %d", len(all))
	}
}

func TestConcurrentAccess(t *testing.T) {
	mock := newMockDB()
	for i := 0; i < 20; i++ {
		key := string(rune('a' + i))
		mock.flags[key] = models.FeatureFlag{Key: key, Enabled: i%2 == 0}
	}

	mgr := NewManager(mock, testLogger())
	defer mgr.Stop()

	time.Sleep(50 * time.Millisecond)

	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func() {
			mgr.IsEnabled("a")
			mgr.GetAll()
			_ = mgr.SetEnabled("a", true)
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestManagerStop(t *testing.T) {
	mgr := NewManager(newMockDB(), testLogger())

	// Stop не должен паниковать
	mgr.Stop()
	// Повторный Stop тоже не должен паниковать
	mgr.Stop()
}

// testLogger создаёт тестовый логгер.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))
}
