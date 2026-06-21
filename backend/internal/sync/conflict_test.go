package sync

import (
	"log/slog"
	"testing"
)

func TestNewConflictResolver(t *testing.T) {
	cr := NewConflictResolver(nil)
	if cr == nil {
		t.Fatal("NewConflictResolver returned nil")
	}
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// open variants
		{"open", "open"},
		{"new", "open"},
		{"pending", "open"},
		{"submitted", "open"},
		{"queued", "open"},
		// in_progress variants
		{"in_progress", "in_progress"},
		{"in progress", "in_progress"},
		{"progress", "in_progress"},
		{"active", "in_progress"},
		{"assigned", "in_progress"},
		{"working", "in_progress"},
		// completed variants
		{"completed", "completed"},
		{"complete", "completed"},
		{"done", "completed"},
		{"resolved", "completed"},
		{"closed", "completed"},
		{"finished", "completed"},
		// cancelled variants
		{"cancelled", "cancelled"},
		{"canceled", "cancelled"},
		{"rejected", "cancelled"},
		{"void", "cancelled"},
		// unknown passthrough
		{"custom_status", "custom_status"},
	}

	for _, tt := range tests {
		result := normalizeStatus(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"completed", true},
		{"cancelled", true},
		{"open", false},
		{"in_progress", false},
		{"custom", false},
	}

	for _, tt := range tests {
		result := isTerminalStatus(tt.status)
		if result != tt.expected {
			t.Errorf("isTerminalStatus(%q) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestConflictResolutionLogic(t *testing.T) {
	// Проверяем логику разрешения на уровне чистых функций (без БД):
	// external-wins, auto-reopen, no-conflict

	cr := NewConflictResolver(slog.Default())

	// Сценарий: одинаковые статусы → нет конфликта
	// (тестируем только чистые функции, т.к. ResolveWorkOrder требует БД)
	t.Run("no_conflict_same_status", func(t *testing.T) {
		localNorm := normalizeStatus("open")
		extNorm := normalizeStatus("open")
		if localNorm != extNorm {
			t.Error("same statuses should be normalized to same value")
		}
	})

	// Сценарий: external-wins — статусы различаются, не терминальные
	t.Run("external_wins_diff_status", func(t *testing.T) {
		localNorm := normalizeStatus("open")
		extNorm := normalizeStatus("completed")
		if localNorm == extNorm {
			t.Error("different statuses should be normalized differently")
		}
		if isTerminalStatus(localNorm) {
			t.Error("open should not be terminal")
		}
		if !isTerminalStatus(extNorm) {
			t.Error("completed should be terminal")
		}
	})

	// Сценарий: auto-reopen — локальный completed, внешний in_progress
	t.Run("auto_reopen", func(t *testing.T) {
		localNorm := normalizeStatus("completed")
		extNorm := normalizeStatus("in_progress")
		if !isTerminalStatus(localNorm) {
			t.Error("completed should be terminal")
		}
		if isTerminalStatus(extNorm) {
			t.Error("in_progress should not be terminal")
		}
		// auto-reopen condition: local terminal, external not terminal
		if !(isTerminalStatus(localNorm) && !isTerminalStatus(extNorm)) {
			t.Error("auto-reopen condition should be true")
		}
	})

	// Сценарий: external-wins (не auto-reopen) — оба не терминальные
	t.Run("external_wins_non_terminal", func(t *testing.T) {
		localNorm := normalizeStatus("in_progress")
		extNorm := normalizeStatus("completed")
		if isTerminalStatus(localNorm) {
			t.Error("in_progress should not be terminal")
		}
		if !isTerminalStatus(extNorm) {
			t.Error("completed should be terminal")
		}
		// Not auto-reopen because external is terminal
		if isTerminalStatus(localNorm) && !isTerminalStatus(extNorm) {
			t.Error("auto-reopen condition should be false")
		}
	})

	_ = cr // cr используется для ссылки на пакет
}

func TestNormalizeStatusEdgeCases(t *testing.T) {
	// Пустая строка
	if normalizeStatus("") != "" {
		t.Error("empty status should remain empty")
	}
	// Статус с пробелами
	if normalizeStatus("  open  ") != "  open  " {
		t.Error("status with spaces should be passed through")
	}
}

func TestIsTerminalStatusEdgeCases(t *testing.T) {
	if isTerminalStatus("") {
		t.Error("empty status should not be terminal")
	}
	if isTerminalStatus("COMPLETED") {
		t.Error("COMPLETED (uppercase) should not be terminal (exact match only)")
	}
}
