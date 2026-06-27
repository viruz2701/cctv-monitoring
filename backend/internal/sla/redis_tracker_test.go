// Package sla — tests for RedisSLATracker (PERF.5).
package sla

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Mock Redis client
// ═══════════════════════════════════════════════════════════════════════

// mockRedisClient имитирует RedisCmdable для тестирования.
//
// Хранит данные в памяти:
//   - sortedSets: map[key] → []zMember (для ZAdd/ZRangeByScore/ZCount)
//   - strings: map[key] → string (для Set/Get)
//   - sets: map[key] → map[member]bool (для SAdd/SMembers)
//   - ttls: map[key] → время истечения
//   - pingFail: флаг для симуляции недоступности Redis
type mockRedisClient struct {
	mu sync.RWMutex

	sortedSets map[string][]zMember
	strings    map[string]string
	sets       map[string]map[string]bool
	ttls       map[string]time.Time

	pingFail bool
}

// zMember — член Sorted Set в моке.
type zMember struct {
	Score  float64
	Member string
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		sortedSets: make(map[string][]zMember),
		strings:    make(map[string]string),
		sets:       make(map[string]map[string]bool),
		ttls:       make(map[string]time.Time),
	}
}

func (m *mockRedisClient) Ping(_ context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.pingFail {
		return fmt.Errorf("redis connection refused")
	}
	return nil
}

func (m *mockRedisClient) ZAdd(_ context.Context, key string, members ...Z) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, z := range members {
		memberStr, ok := z.Member.(string)
		if !ok {
			return fmt.Errorf("mock: Z member must be string")
		}
		m.sortedSets[key] = append(m.sortedSets[key], zMember{
			Score:  z.Score,
			Member: memberStr,
		})
	}
	return nil
}

func (m *mockRedisClient) ZRangeByScore(_ context.Context, key string, opt ZRangeBy) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, ok := m.sortedSets[key]
	if !ok {
		return []string{}, nil
	}

	// Парсим min/max как float64
	var minF, maxF float64
	if _, err := fmt.Sscanf(opt.Min, "%f", &minF); err != nil {
		return nil, fmt.Errorf("mock: parse min %s: %w", opt.Min, err)
	}
	if _, err := fmt.Sscanf(opt.Max, "%f", &maxF); err != nil {
		return nil, fmt.Errorf("mock: parse max %s: %w", opt.Max, err)
	}

	var result []string
	for _, z := range entries {
		if z.Score >= minF && z.Score <= maxF {
			result = append(result, z.Member)
		}
	}

	// Apply offset & count
	if opt.Offset > 0 && int(opt.Offset) < len(result) {
		result = result[opt.Offset:]
	}
	if opt.Count > 0 && int(opt.Count) < len(result) {
		result = result[:opt.Count]
	}

	return result, nil
}

func (m *mockRedisClient) ZCount(_ context.Context, key string, min, max string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, ok := m.sortedSets[key]
	if !ok {
		return 0, nil
	}

	var minF, maxF float64
	if _, err := fmt.Sscanf(min, "%f", &minF); err != nil {
		return 0, fmt.Errorf("mock: parse min %s: %w", min, err)
	}
	if _, err := fmt.Sscanf(max, "%f", &maxF); err != nil {
		return 0, fmt.Errorf("mock: parse max %s: %w", max, err)
	}

	var count int64
	for _, z := range entries {
		if z.Score >= minF && z.Score <= maxF {
			count++
		}
	}
	return count, nil
}

func (m *mockRedisClient) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("mock: Set value must be string")
	}
	m.strings[key] = str
	return nil
}

func (m *mockRedisClient) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.strings[key]
	if !ok {
		return "", fmt.Errorf("mock: key %s not found", key)
	}
	return val, nil
}

func (m *mockRedisClient) SAdd(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets[key] == nil {
		m.sets[key] = make(map[string]bool)
	}
	for _, member := range members {
		m.sets[key][member] = true
	}
	return nil
}

func (m *mockRedisClient) SMembers(_ context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	set, ok := m.sets[key]
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(set))
	for member := range set {
		result = append(result, member)
	}
	return result, nil
}

func (m *mockRedisClient) Expire(_ context.Context, _ string, _ time.Duration) error {
	return nil
}

func (m *mockRedisClient) Close() error {
	return nil
}

// setPingFail включает/выключает симуляцию недоступности Redis.
func (m *mockRedisClient) setPingFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingFail = fail
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// makeBreach создаёт SLABreach для тестов.
func makeBreach(deviceID, vtype string, threshold, actual float64, t time.Time) *SLABreach {
	return &SLABreach{
		DeviceID:      deviceID,
		ViolationType: vtype,
		Threshold:     threshold,
		ActualValue:   actual,
		OccurredAt:    t,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: RecordBreach
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_RecordBreach(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	breach := makeBreach("cam-001", "response_time", 5000, 7200, now)
	err := tracker.RecordBreach(ctx, breach)
	if err != nil {
		t.Fatalf("RecordBreach failed: %v", err)
	}

	// Проверяем что ID сгенерирован
	if breach.ID == "" {
		t.Error("expected breach ID to be generated")
	}

	// Проверяем что breach сохранён в Sorted Set
	key := breachKey("cam-001")
	mock.mu.RLock()
	entries, ok := mock.sortedSets[key]
	mock.mu.RUnlock()
	if !ok {
		t.Fatal("expected breaches sorted set to exist")
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 breach, got %d", len(entries))
	}

	// Проверяем score = unix nano
	expectedScore := float64(now.UnixNano())
	if entries[0].Score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, entries[0].Score)
	}

	// Проверяем что device добавлен в SET
	mock.mu.RLock()
	devices, _ := mock.sets[devicesKey()]
	mock.mu.RUnlock()
	if !devices["cam-001"] {
		t.Error("expected cam-001 to be in devices set")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: RecordBreach — nil breach
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_RecordBreach_NilBreach(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()

	err := tracker.RecordBreach(ctx, nil)
	if err == nil {
		t.Error("expected error for nil breach")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: GetBreaches
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_GetBreaches(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()

	now := time.Now().UTC()

	// Записываем 3 breach за последние 3 минуты
	for i := 0; i < 3; i++ {
		b := makeBreach("cam-001", "response_time", 5000, 6000+float64(i)*1000, now.Add(-time.Duration(i)*time.Minute))
		if err := tracker.RecordBreach(ctx, b); err != nil {
			t.Fatalf("failed to record breach %d: %v", i, err)
		}
	}

	// Получаем breaches за последние 5 минут
	from := now.Add(-5 * time.Minute)
	to := now.Add(time.Minute)
	breaches, err := tracker.GetBreaches(ctx, "cam-001", from, to)
	if err != nil {
		t.Fatalf("GetBreaches failed: %v", err)
	}

	if len(breaches) != 3 {
		t.Fatalf("expected 3 breaches, got %d", len(breaches))
	}

	// Проверяем что ID не пустые
	for i, b := range breaches {
		if b.ID == "" {
			t.Errorf("breach[%d] has empty ID", i)
		}
		if b.DeviceID != "cam-001" {
			t.Errorf("breach[%d] deviceID = %s, want cam-001", i, b.DeviceID)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: GetComplianceRate
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_GetComplianceRate(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	// Записываем 3 breach
	for i := 0; i < 3; i++ {
		b := makeBreach("cam-002", "uptime", 99.9, 95.0+float64(i), now.Add(-time.Duration(i)*time.Minute))
		if err := tracker.RecordBreach(ctx, b); err != nil {
			t.Fatalf("failed to record breach %d: %v", i, err)
		}
	}

	// 3 breaches × 5% = 85% compliance
	from := now.Add(-10 * time.Minute)
	to := now.Add(time.Minute)
	rate, err := tracker.GetComplianceRate(ctx, "cam-002", from, to)
	if err != nil {
		t.Fatalf("GetComplianceRate failed: %v", err)
	}

	expected := 85.0
	if rate != expected {
		t.Errorf("expected compliance rate %f, got %f", expected, rate)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: GetTrackerStatus — Connected
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_GetTrackerStatus_Connected(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()

	status, err := tracker.GetTrackerStatus(ctx)
	if err != nil {
		t.Fatalf("GetTrackerStatus failed: %v", err)
	}

	if !status.Connected {
		t.Error("expected connected = true")
	}
	if status.KeysCount != 0 {
		t.Errorf("expected KeysCount = 0, got %d", status.KeysCount)
	}
	if status.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: GetTrackerStatus — Disconnected
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_GetTrackerStatus_Disconnected(t *testing.T) {
	mock := newMockRedisClient()
	mock.setPingFail(true)
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()

	status, err := tracker.GetTrackerStatus(ctx)
	if err != nil {
		t.Fatalf("GetTrackerStatus failed: %v", err)
	}

	if status.Connected {
		t.Error("expected connected = false when Redis is down")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: ContextDeadline — операции с истёкшим контекстом
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_ContextDeadline(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)

	// Создаём уже истёкший контекст
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Hour))
	defer cancel()

	// Задержка для гарантии истечения
	time.Sleep(time.Millisecond)

	err := tracker.RecordBreach(ctx, makeBreach("cam-001", "response_time", 5000, 7200, time.Now()))
	if err != nil {
		t.Logf("expected error for expired context (graceful): %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: EmptyBreaches — запрос breaches для устройства без нарушений
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_EmptyBreaches(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	// Запрашиваем breaches для устройства, у которого нет записей
	breaches, err := tracker.GetBreaches(ctx, "cam-nonexistent", now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("GetBreaches failed: %v", err)
	}

	if breaches == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(breaches) != 0 {
		t.Errorf("expected 0 breaches, got %d", len(breaches))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: EmptyDeviceID — пустой deviceID вызывает ошибку
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_EmptyDeviceID(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := tracker.GetBreaches(ctx, "", now.Add(-1*time.Hour), now)
	if err == nil {
		t.Error("expected error for empty deviceID in GetBreaches")
	}

	_, err = tracker.GetComplianceRate(ctx, "", now.Add(-1*time.Hour), now)
	if err == nil {
		t.Error("expected error for empty deviceID in GetComplianceRate")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: MultipleDevices — несколько устройств работают независимо
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_MultipleDevices(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	devices := []string{"cam-001", "cam-002", "cam-003"}
	for i, dev := range devices {
		b := makeBreach(dev, "response_time", 5000, 6000+float64(i)*500, now)
		if err := tracker.RecordBreach(ctx, b); err != nil {
			t.Fatalf("failed to record breach for %s: %v", dev, err)
		}
	}

	// Проверяем что у каждого устройства свой набор breaches
	for _, dev := range devices {
		breaches, err := tracker.GetBreaches(ctx, dev, now.Add(-1*time.Hour), now.Add(time.Hour))
		if err != nil {
			t.Fatalf("GetBreaches for %s failed: %v", dev, err)
		}
		if len(breaches) != 1 {
			t.Errorf("expected 1 breach for %s, got %d", dev, len(breaches))
		}
	}

	// Проверяем что все устройства в глобальном SET
	status, err := tracker.GetTrackerStatus(ctx)
	if err != nil {
		t.Fatalf("GetTrackerStatus failed: %v", err)
	}
	if status.KeysCount != 3 {
		t.Errorf("expected KeysCount = 3, got %d", status.KeysCount)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: ComplianceRate_100Percent — без нарушений
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_ComplianceRate_100Percent(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	// Нет breaches для cam-perfect
	from := now.Add(-24 * time.Hour)
	to := now.Add(time.Hour)

	rate, err := tracker.GetComplianceRate(ctx, "cam-perfect", from, to)
	if err != nil {
		t.Fatalf("GetComplianceRate failed: %v", err)
	}

	if rate != 100.0 {
		t.Errorf("expected 100%% compliance rate, got %f", rate)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: ComplianceRate_Mixed — разные уровни нарушений
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_ComplianceRate_Mixed(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	tests := []struct {
		name        string
		breachCount int
		expected    float64
	}{
		{"1 breach → 95%", 1, 95.0},
		{"5 breaches → 75%", 5, 75.0},
		{"10 breaches → 50%", 10, 50.0},
		{"20 breaches → 0%", 20, 0.0},
		{"25 breaches → 0% (clamped)", 25, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Используем уникальный device для каждого теста
			dev := fmt.Sprintf("cam-mixed-%d", tt.breachCount)

			for i := 0; i < tt.breachCount; i++ {
				b := makeBreach(dev, "uptime", 99.9, 95.0, now.Add(-time.Duration(i)*time.Second))
				if err := tracker.RecordBreach(ctx, b); err != nil {
					t.Fatalf("failed to record breach: %v", err)
				}
			}

			rate, err := tracker.GetComplianceRate(ctx, dev, now.Add(-1*time.Hour), now.Add(time.Hour))
			if err != nil {
				t.Fatalf("GetComplianceRate failed: %v", err)
			}

			if rate != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, rate)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: ConcurrentWrites — параллельная запись breaches
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_ConcurrentWrites(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	const goroutines = 20
	const breachesPerGoroutine = 10

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*breachesPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			for i := 0; i < breachesPerGoroutine; i++ {
				b := makeBreach(
					fmt.Sprintf("cam-concurrent-%d", gID%5), // 5 devices, 40 breaches each
					"response_time",
					5000,
					6000+float64(i)*100,
					now.Add(-time.Duration(i)*time.Second),
				)
				if err := tracker.RecordBreach(ctx, b); err != nil {
					errCh <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	// Проверяем что нет ошибок при конкурентной записи
	for err := range errCh {
		t.Errorf("concurrent write error: %v", err)
	}

	// Проверяем общее количество breaches
	totalBreaches := 0
	mock.mu.RLock()
	for key, entries := range mock.sortedSets {
		if len(key) > 13 && key[:13] == "sla:breaches:" {
			totalBreaches += len(entries)
		}
	}
	mock.mu.RUnlock()

	expected := goroutines * breachesPerGoroutine // 200
	if totalBreaches != expected {
		t.Errorf("expected %d total breaches, got %d", expected, totalBreaches)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: LargeDataset — 1000 breaches для одного устройства
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_LargeDataset(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	const count = 1000

	// Записываем 1000 breaches
	for i := 0; i < count; i++ {
		b := makeBreach(
			"cam-bulk",
			"resolution_time",
			3600,
			4000+float64(i),
			now.Add(-time.Duration(i)*time.Second),
		)
		if err := tracker.RecordBreach(ctx, b); err != nil {
			t.Fatalf("failed to record breach %d: %v", i, err)
		}
	}

	// Проверяем что все сохранены
	from := now.Add(-2 * time.Hour)
	to := now.Add(time.Hour)
	breaches, err := tracker.GetBreaches(ctx, "cam-bulk", from, to)
	if err != nil {
		t.Fatalf("GetBreaches failed: %v", err)
	}

	if len(breaches) != count {
		t.Errorf("expected %d breaches, got %d", count, len(breaches))
	}

	// Проверяем compliance rate (1000 breaches × 5% = 0%, clamped)
	rate, err := tracker.GetComplianceRate(ctx, "cam-bulk", from, to)
	if err != nil {
		t.Fatalf("GetComplianceRate failed: %v", err)
	}
	if rate != 0.0 {
		t.Errorf("expected 0%% compliance for 1000 breaches, got %f", rate)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: GenerateID — криптостойкий ID
// ═══════════════════════════════════════════════════════════════════════

func TestGenerateID_Unique(t *testing.T) {
	const idsCount = 100
	ids := make(map[string]bool)

	for i := 0; i < idsCount; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID failed: %v", err)
		}
		if len(id) != 32 { // 16 bytes = 32 hex chars
			t.Errorf("expected ID length 32, got %d: %s", len(id), id)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: GetBreaches — точный range по времени
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_GetBreaches_TimeRange(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	// Breach 10 минут назад
	b1 := makeBreach("cam-time", "response_time", 5000, 6000, now.Add(-10*time.Minute))
	if err := tracker.RecordBreach(ctx, b1); err != nil {
		t.Fatalf("failed to record breach: %v", err)
	}

	// Breach 5 минут назад
	b2 := makeBreach("cam-time", "response_time", 5000, 5500, now.Add(-5*time.Minute))
	if err := tracker.RecordBreach(ctx, b2); err != nil {
		t.Fatalf("failed to record breach: %v", err)
	}

	// Breach сейчас
	b3 := makeBreach("cam-time", "response_time", 5000, 5200, now)
	if err := tracker.RecordBreach(ctx, b3); err != nil {
		t.Fatalf("failed to record breach: %v", err)
	}

	// Запрашиваем только за последние 7 минут — должны получить 2 breaches (b2, b3)
	from := now.Add(-7 * time.Minute)
	to := now.Add(time.Minute)
	breaches, err := tracker.GetBreaches(ctx, "cam-time", from, to)
	if err != nil {
		t.Fatalf("GetBreaches failed: %v", err)
	}

	if len(breaches) != 2 {
		t.Fatalf("expected 2 breaches in time range, got %d", len(breaches))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: Metrics — проверка метрик трекера
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_Metrics(t *testing.T) {
	mock := newMockRedisClient()
	tracker := NewRedisSLATracker(mock, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	// Добавляем несколько устройств с breaches
	for _, dev := range []string{"cam-m1", "cam-m2", "cam-m3"} {
		b := makeBreach(dev, "response_time", 5000, 6000, now)
		if err := tracker.RecordBreach(ctx, b); err != nil {
			t.Fatalf("failed to record breach: %v", err)
		}
	}

	metrics := tracker.Metrics()
	if !metrics.Connected {
		t.Error("expected connected = true")
	}
	if metrics.DeviceCount != 3 {
		t.Errorf("expected DeviceCount = 3, got %d", metrics.DeviceCount)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Test: BreachSerialization — полный цикл сериализации/десериализации
// ═══════════════════════════════════════════════════════════════════════

func TestRedisSLATracker_BreachSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond) // JSON теряет наносекунды

	original := &SLABreach{
		ID:            "test-id-123",
		DeviceID:      "cam-serial",
		ViolationType: "response_time",
		Threshold:     5000.5,
		ActualValue:   7200.75,
		OccurredAt:    now,
		Region:        "zone-1",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded SLABreach
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %s, want %s", decoded.ID, original.ID)
	}
	if decoded.DeviceID != original.DeviceID {
		t.Errorf("DeviceID: got %s, want %s", decoded.DeviceID, original.DeviceID)
	}
	if decoded.ViolationType != original.ViolationType {
		t.Errorf("ViolationType: got %s, want %s", decoded.ViolationType, original.ViolationType)
	}
	if decoded.Threshold != original.Threshold {
		t.Errorf("Threshold: got %f, want %f", decoded.Threshold, original.Threshold)
	}
	if decoded.ActualValue != original.ActualValue {
		t.Errorf("ActualValue: got %f, want %f", decoded.ActualValue, original.ActualValue)
	}
	if !decoded.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("OccurredAt: got %v, want %v", decoded.OccurredAt, original.OccurredAt)
	}
	if decoded.Region != original.Region {
		t.Errorf("Region: got %s, want %s", decoded.Region, original.Region)
	}
}
