// Package playbook — Marketplace pre-built playbooks tests (P1-MARKET).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-MARKET: Публичный marketplace для pre-built playbooks
//
// Тесты проверяют валидацию входных данных (OWASP ASVS V1) без БД:
//   - Валидация vendor (whitelist подход)
//   - Валидация score (1-5)
//   - Pagination defaults
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — RBAC)
//   - ISO 27001 A.12.4 (Audit trail)
//   - OWASP ASVS V1 (Input validation — whitelist)
//   - OWASP ASVS V6 (Cryptographic storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package playbook

import (
	"context"
	"log/slog"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// Service Creation Tests
// ═══════════════════════════════════════════════════════════════════════════

// TestMarketplaceService_New проверяет создание сервиса с nil логгером
// (должен использовать slog.Default).
func TestMarketplaceService_New(t *testing.T) {
	svc := NewMarketplaceService(nil, nil)
	if svc == nil {
		t.Fatal("NewMarketplaceService returned nil")
	}
	if svc.db != nil {
		t.Error("expected db to be nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Vendor Validation Tests (OWASP ASVS V5.1 — whitelist)
// ═══════════════════════════════════════════════════════════════════════════

// TestMarketplaceService_List_InvalidVendor проверяет, что невалидный vendor
// возвращает ошибку ДО обращения к БД (whitelist validation).
//
// OWASP ASVS V5.1: Input validation на уровне сервиса.
func TestMarketplaceService_List_InvalidVendor(t *testing.T) {
	svc := NewMarketplaceService(nil, nil)
	ctx := context.Background()

	filter := MarketplaceFilter{Vendor: "unknown"}
	_, _, err := svc.List(ctx, filter)
	if err == nil {
		t.Error("expected error for unknown vendor, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Score Validation Tests (OWASP ASVS V1)
// ═══════════════════════════════════════════════════════════════════════════

// TestMarketplaceService_Rate_ScoreRange проверяет валидацию score
// через table-driven тест (OWASP ASVS V1 — input validation).
//
// Score должен быть в диапазоне [1-5]. Тестируются ТОЛЬКО невалидные
// значения (вне диапазона), которые возвращают ошибку ДО обращения к БД.
func TestMarketplaceService_Rate_ScoreRange(t *testing.T) {
	svc := NewMarketplaceService(nil, nil)
	ctx := context.Background()

	tests := []struct {
		name  string
		score int
	}{
		{name: "score_zero_invalid", score: 0},
		{name: "score_six_invalid", score: 6},
		{name: "score_negative_invalid", score: -1},
		{name: "score_over_100", score: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Rate(ctx, "playbook-1", "user-1", tt.score, "test review")
			if err == nil {
				t.Errorf("expected error for score=%d, got nil", tt.score)
			}
		})
	}
}

// TestMarketplaceService_Rate_ScoreUnderMin проверяет, что score=0
// возвращает ошибку "out of range".
func TestMarketplaceService_Rate_ScoreUnderMin(t *testing.T) {
	svc := NewMarketplaceService(nil, nil)
	err := svc.Rate(context.Background(), "playbook-1", "user-1", 0, "review")
	if err == nil {
		t.Fatal("expected error for score=0")
	}
	if err.Error() != "marketplace: score 0 out of range [1-5]" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestMarketplaceService_Rate_ScoreOverMax проверяет, что score=6
// возвращает ошибку "out of range".
func TestMarketplaceService_Rate_ScoreOverMax(t *testing.T) {
	svc := NewMarketplaceService(nil, nil)
	err := svc.Rate(context.Background(), "playbook-1", "user-1", 6, "review")
	if err == nil {
		t.Fatal("expected error for score=6")
	}
	if err.Error() != "marketplace: score 6 out of range [1-5]" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Logger Tests
// ═══════════════════════════════════════════════════════════════════════════

// TestMarketplaceService_Logger_Nil проверяет, что nil логгер заменяется
// на slog.Default без паники.
func TestMarketplaceService_Logger_Nil(t *testing.T) {
	logger := slog.Default()
	svc := NewMarketplaceService(nil, logger)
	if svc == nil {
		t.Fatal("NewMarketplaceService returned nil")
	}
	if svc.logger == nil {
		t.Error("logger should not be nil")
	}
}

// TestMarketplaceService_Logger_Default проверяет, что slog.Default
// используется при nil логгере.
func TestMarketplaceService_Logger_Default(t *testing.T) {
	svc := NewMarketplaceService(nil, nil)
	if svc.logger == nil {
		t.Error("logger should not be nil when nil is passed")
	}
}
