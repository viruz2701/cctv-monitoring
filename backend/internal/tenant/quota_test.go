// Package tenant — Table-driven tests for QuotaManager (P1-QUOTA).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-QUOTA: Tenant Quota Management
//
// Тесты без Redis: QuotaManager с client = nil (fail-open mode)
// все операции пропускают проверки и возвращают успех.
//
// Compliance:
//   - ISO 27001 A.12.1.2 (Capacity management)
//   - IEC 62443-3-3 SR 3.1 (Resource management)
//   - IEC 62443-3-3 SR 7.1 (Audit trail)
//   - OWASP ASVS V2.2.1 (Rate limiting)
//   - СТБ 34.101.27 п. 6.1 (Защита от DoS)
//
// ═══════════════════════════════════════════════════════════════════════════
package tenant

import (
	"context"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// Fail-Open Tests (ISO 27001 A.12.1.2 — graceful degradation)
// ═══════════════════════════════════════════════════════════════════════════

// TestQuotaManager_FailOpen проверяет, что QuotaManager с client=nil
// работает в режиме fail-open: все операции возвращают успех без ошибок.
//
// Это критично для отказоустойчивости — при недоступности Redis
// система не должна блокировать создание ресурсов.
func TestQuotaManager_FailOpen(t *testing.T) {
	qm := NewQuotaManager(nil, nil)
	ctx := context.Background()

	// Current — возвращает 0 без ошибки
	val, err := qm.Current(ctx, "tenant-1", QuotaDevices)
	if err != nil {
		t.Errorf("Current failed: %v", err)
	}
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}

	// Increment — возвращает true (разрешено) без ошибки
	ok, err := qm.Increment(ctx, "tenant-1", QuotaDevices)
	if err != nil {
		t.Errorf("Increment failed: %v", err)
	}
	if !ok {
		t.Error("expected Increment to return true (fail-open)")
	}

	// Decrement — не возвращает ошибку
	if err := qm.Decrement(ctx, "tenant-1", QuotaDevices); err != nil {
		t.Errorf("Decrement failed: %v", err)
	}

	// Check — возвращает статус с нулями без ошибки
	status, err := qm.Check(ctx, "tenant-1", QuotaDevices)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Current != 0 || status.SoftLimit != 0 || status.HardLimit != 0 {
		t.Errorf("expected zero limits, got %+v", status)
	}

	// SetLimits — не возвращает ошибку
	if err := qm.SetLimits(ctx, "tenant-1", QuotaDevices, 200); err != nil {
		t.Errorf("SetLimits failed: %v", err)
	}

	// SetGraceUntil — не возвращает ошибку
	if err := qm.SetGraceUntil(ctx, "tenant-1", time.Now().Add(24*time.Hour)); err != nil {
		t.Errorf("SetGraceUntil failed: %v", err)
	}

}

// ═══════════════════════════════════════════════════════════════════════════
// Default Configs Tests
// ═══════════════════════════════════════════════════════════════════════════

// TestQuotaManager_DefaultConfigs проверяет, что DefaultQuotaConfigs
// возвращает все 5 типов квот с корректными значениями.
func TestQuotaManager_DefaultConfigs(t *testing.T) {
	cfgs := DefaultQuotaConfigs()

	tests := []struct {
		name      string
		qtype     QuotaType
		wantUnit  string
		wantGrace int
	}{
		{name: "devices", qtype: QuotaDevices, wantUnit: "count", wantGrace: 7},
		{name: "users", qtype: QuotaUsers, wantUnit: "count", wantGrace: 7},
		{name: "storage", qtype: QuotaStorage, wantUnit: "gb", wantGrace: 7},
		{name: "api_calls", qtype: QuotaAPICalls, wantUnit: "req/h", wantGrace: 0},
		{name: "work_orders", qtype: QuotaWorkOrders, wantUnit: "count", wantGrace: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, ok := cfgs[tt.qtype]
			if !ok {
				t.Fatalf("quota type %q not found in DefaultQuotaConfigs", tt.qtype)
			}
			if cfg.Type != tt.qtype {
				t.Errorf("expected type %q, got %q", tt.qtype, cfg.Type)
			}
			if cfg.Unit != tt.wantUnit {
				t.Errorf("expected unit %q, got %q", tt.wantUnit, cfg.Unit)
			}
			if cfg.GraceDays != tt.wantGrace {
				t.Errorf("expected grace days %d, got %d", tt.wantGrace, cfg.GraceDays)
			}
		})
	}
}

// TestQuotaManager_DefaultConfigs_HardLimits проверяет hard limit значения
// для всех типов квот.
func TestQuotaManager_DefaultConfigs_HardLimits(t *testing.T) {
	cfgs := DefaultQuotaConfigs()

	tests := []struct {
		name  string
		qtype QuotaType
		want  int64
	}{
		{name: "devices", qtype: QuotaDevices, want: 100},
		{name: "users", qtype: QuotaUsers, want: 10},
		{name: "storage", qtype: QuotaStorage, want: 1000},
		{name: "api_calls", qtype: QuotaAPICalls, want: 10000},
		{name: "work_orders", qtype: QuotaWorkOrders, want: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, ok := cfgs[tt.qtype]
			if !ok {
				t.Fatalf("quota type %q not found", tt.qtype)
			}
			if cfg.HardLimit != tt.want {
				t.Errorf("hard_limit = %d, want %d", cfg.HardLimit, tt.want)
			}
		})
	}
}

// TestQuotaManager_DefaultConfigs_SoftLimits проверяет, что soft limit
// равен 80% от hard limit для всех типов квот.
func TestQuotaManager_DefaultConfigs_SoftLimits(t *testing.T) {
	cfgs := DefaultQuotaConfigs()

	tests := []struct {
		name     string
		qtype    QuotaType
		hard     int64
		softWant int64 // 80% от hard
	}{
		{name: "devices", qtype: QuotaDevices, hard: 100, softWant: 80},
		{name: "users", qtype: QuotaUsers, hard: 10, softWant: 8},
		{name: "storage", qtype: QuotaStorage, hard: 1000, softWant: 800},
		{name: "api_calls", qtype: QuotaAPICalls, hard: 10000, softWant: 8000},
		{name: "work_orders", qtype: QuotaWorkOrders, hard: 500, softWant: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, ok := cfgs[tt.qtype]
			if !ok {
				t.Fatalf("quota type %q not found", tt.qtype)
			}
			if cfg.HardLimit != tt.hard {
				t.Errorf("hard_limit = %d, want %d", cfg.HardLimit, tt.hard)
			}
			if cfg.SoftLimit != tt.softWant {
				t.Errorf("soft_limit = %d, want %d (80%% of %d)",
					cfg.SoftLimit, tt.softWant, tt.hard)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// QuotaTypeFromString Tests
// ═══════════════════════════════════════════════════════════════════════════

// TestQuotaManager_QuotaTypeFromString_Valid проверяет преобразование
// валидных строк в QuotaType (table-driven, 5 кейсов).
func TestQuotaManager_QuotaTypeFromString_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected QuotaType
	}{
		{name: "devices lowercase", input: "devices", expected: QuotaDevices},
		{name: "users lowercase", input: "users", expected: QuotaUsers},
		{name: "storage_gb full", input: "storage_gb", expected: QuotaStorage},
		{name: "storage alias", input: "storage", expected: QuotaStorage},
		{name: "api_calls full", input: "api_calls", expected: QuotaAPICalls},
		{name: "api alias", input: "api", expected: QuotaAPICalls},
		{name: "work_orders full", input: "work_orders", expected: QuotaWorkOrders},
		{name: "wo alias", input: "wo", expected: QuotaWorkOrders},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QuotaTypeFromString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("QuotaTypeFromString(%q) = %q, want %q",
					tt.input, got, tt.expected)
			}
		})
	}
}

// TestQuotaManager_QuotaTypeFromString_Invalid проверяет, что невалидные
// строки возвращают ошибку (table-driven, 3 кейса).
func TestQuotaManager_QuotaTypeFromString_Invalid(t *testing.T) {
	invalidInputs := []string{
		"unknown",
		"cameras",
		"",
		"sensors",
		"INVALID_TYPE",
	}

	for _, input := range invalidInputs {
		t.Run("invalid_"+input, func(t *testing.T) {
			_, err := QuotaTypeFromString(input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// AllQuotaTypes Tests
// ═══════════════════════════════════════════════════════════════════════════

// TestQuotaManager_AllQuotaTypes_Count проверяет, что AllQuotaTypes
// возвращает ровно 5 типов квот.
func TestQuotaManager_AllQuotaTypes_Count(t *testing.T) {
	types := AllQuotaTypes()
	if len(types) != 5 {
		t.Errorf("expected 5 quota types, got %d: %v", len(types), types)
	}

	// Проверяем, что все ожидаемые типы присутствуют
	expected := map[QuotaType]bool{
		QuotaDevices:    false,
		QuotaUsers:      false,
		QuotaStorage:    false,
		QuotaAPICalls:   false,
		QuotaWorkOrders: false,
	}

	for _, qt := range types {
		if _, ok := expected[qt]; ok {
			expected[qt] = true
		} else {
			t.Errorf("unexpected quota type %q in AllQuotaTypes()", qt)
		}
	}

	for qt, found := range expected {
		if !found {
			t.Errorf("quota type %q missing from AllQuotaTypes()", qt)
		}
	}
}

// TestQuotaManager_AllQuotaTypes_NoDuplicates проверяет, что AllQuotaTypes
// не содержит дубликатов.
func TestQuotaManager_AllQuotaTypes_NoDuplicates(t *testing.T) {
	types := AllQuotaTypes()
	seen := make(map[QuotaType]int)

	for _, qt := range types {
		seen[qt]++
	}

	for qt, count := range seen {
		if count > 1 {
			t.Errorf("quota type %q appears %d times (duplicate)", qt, count)
		}
	}
}
