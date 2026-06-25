// Package compliance — unit tests for Compliance & Fines Shield (KF-15.1.1).
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing — table-driven)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - OWASP ASVS V5 (Input validation testing)
//   - СТБ 34.101.27 п. 7.4 (Тестирование безопасности)
package compliance

import (
	"math"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// CalculateRisk — table-driven tests
// ═══════════════════════════════════════════════════════════════════════

func TestCalculateRisk(t *testing.T) {
	tests := []struct {
		name            string
		downtimeMinutes int64
		deviceType      string
		hourlyRate      float64
		wantExposure    float64
		wantRiskLevel   RiskLevel
	}{
		{
			name:            "zero downtime",
			downtimeMinutes: 0,
			deviceType:      "camera",
			hourlyRate:      0,
			wantExposure:    0,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "negative downtime treated as zero",
			downtimeMinutes: -10,
			deviceType:      "camera",
			hourlyRate:      0,
			wantExposure:    0,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "camera 30min low risk",
			downtimeMinutes: 30,
			deviceType:      "camera",
			hourlyRate:      0,
			wantExposure:    50.0,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "camera 10 hours medium risk",
			downtimeMinutes: 600,
			deviceType:      "camera",
			hourlyRate:      0,
			wantExposure:    1000.0,
			wantRiskLevel:   RiskLevelMedium,
		},
		{
			name:            "cash_register 6 hours medium risk (3000 < 5000)",
			downtimeMinutes: 360,
			deviceType:      "cash_register",
			hourlyRate:      0,
			wantExposure:    3000.0,
			wantRiskLevel:   RiskLevelMedium,
		},
		{
			name:            "cash_register 10 hours high risk (10*500=5000)",
			downtimeMinutes: 600,
			deviceType:      "cash_register",
			hourlyRate:      0,
			wantExposure:    5000.0,
			wantRiskLevel:   RiskLevelHigh,
		},
		{
			name:            "cash_register 50 hours critical risk",
			downtimeMinutes: 3000,
			deviceType:      "cash_register",
			hourlyRate:      0,
			wantExposure:    25000.0,
			wantRiskLevel:   RiskLevelCritical,
		},
		{
			name:            "custom hourly rate overrides default",
			downtimeMinutes: 60,
			deviceType:      "camera",
			hourlyRate:      1000.0,
			wantExposure:    1000.0,
			wantRiskLevel:   RiskLevelMedium,
		},
		{
			name:            "perimeter 2 hours",
			downtimeMinutes: 120,
			deviceType:      "perimeter",
			hourlyRate:      0,
			wantExposure:    400.0,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "warehouse 20 hours high risk",
			downtimeMinutes: 1200,
			deviceType:      "warehouse",
			hourlyRate:      0,
			wantExposure:    6000.0,
			wantRiskLevel:   RiskLevelHigh,
		},
		{
			name:            "office 100 hours critical risk",
			downtimeMinutes: 6000,
			deviceType:      "office",
			hourlyRate:      0,
			wantExposure:    10000.0,
			wantRiskLevel:   RiskLevelHigh,
		},
		{
			name:            "nvr with default fine",
			downtimeMinutes: 60,
			deviceType:      "nvr",
			hourlyRate:      0,
			wantExposure:    250.0,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "unknown device type falls back to camera",
			downtimeMinutes: 120,
			deviceType:      "unknown_type",
			hourlyRate:      0,
			wantExposure:    200.0,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "exact threshold boundary — just below medium ($999.99 < $1000)",
			downtimeMinutes: 600,
			deviceType:      "camera",
			hourlyRate:      99.999, // 10h * 99.999 = 999.99
			wantExposure:    999.99,
			wantRiskLevel:   RiskLevelLow,
		},
		{
			name:            "exact threshold boundary — medium/high at $5000",
			downtimeMinutes: 600,
			deviceType:      "camera",
			hourlyRate:      500.0,
			wantExposure:    5000.0,
			wantRiskLevel:   RiskLevelHigh,
		},
		{
			name:            "exact threshold boundary — high/critical at $25000",
			downtimeMinutes: 3000,
			deviceType:      "cash_register",
			hourlyRate:      500.0,
			wantExposure:    25000.0,
			wantRiskLevel:   RiskLevelCritical,
		},
		{
			name:            "server high hourly rate",
			downtimeMinutes: 60,
			deviceType:      "server",
			hourlyRate:      0,
			wantExposure:    400.0,
			wantRiskLevel:   RiskLevelLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExposure, gotRiskLevel := CalculateRisk(tt.downtimeMinutes, tt.deviceType, tt.hourlyRate)

			if math.Abs(gotExposure-tt.wantExposure) > 0.01 {
				t.Errorf("CalculateRisk() exposure = %v, want %v", gotExposure, tt.wantExposure)
			}
			if gotRiskLevel != tt.wantRiskLevel {
				t.Errorf("CalculateRisk() riskLevel = %v, want %v", gotRiskLevel, tt.wantRiskLevel)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Engine.CalculateRisk — интеграционные тесты с Engine
// ═══════════════════════════════════════════════════════════════════════

func TestEngineCalculateRisk(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	tests := []struct {
		name            string
		downtimeMinutes int64
		deviceType      string
		hourlyRate      float64
		wantExposure    float64
		wantRiskLevel   RiskLevel
	}{
		{
			name:            "engine with default fines",
			downtimeMinutes: 120,
			deviceType:      "cash_register",
			hourlyRate:      0,
			wantExposure:    1000.0,
			wantRiskLevel:   RiskLevelMedium,
		},
		{
			name:            "engine custom rate overrides default",
			downtimeMinutes: 60,
			deviceType:      "camera",
			hourlyRate:      200.0,
			wantExposure:    200.0,
			wantRiskLevel:   RiskLevelLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExposure, gotRiskLevel := engine.CalculateRisk(tt.downtimeMinutes, tt.deviceType, tt.hourlyRate)

			if math.Abs(gotExposure-tt.wantExposure) > 0.01 {
				t.Errorf("Engine.CalculateRisk() exposure = %v, want %v", gotExposure, tt.wantExposure)
			}
			if gotRiskLevel != tt.wantRiskLevel {
				t.Errorf("Engine.CalculateRisk() riskLevel = %v, want %v", gotRiskLevel, tt.wantRiskLevel)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TestGetComplianceSummary
// ═══════════════════════════════════════════════════════════════════════

func TestGetComplianceSummary(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	// Пустой список
	summary := engine.GetComplianceSummary(nil)
	if summary == nil {
		t.Fatal("GetComplianceSummary() returned nil")
	}
	if summary.TotalDevices != 0 {
		t.Errorf("expected 0 devices, got %d", summary.TotalDevices)
	}
	if summary.TotalExposure != 0 {
		t.Errorf("expected 0 exposure, got %f", summary.TotalExposure)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TestDefaultHourlyFines
// ═══════════════════════════════════════════════════════════════════════

func TestDefaultHourlyFines(t *testing.T) {
	expected := map[string]float64{
		"cash_register": 500.0,
		"perimeter":     200.0,
		"warehouse":     300.0,
		"office":        100.0,
		"camera":        100.0,
		"nvr":           250.0,
		"dvr":           200.0,
		"switch":        150.0,
		"server":        400.0,
		"encoder":       180.0,
		"ups":           120.0,
	}

	for k, v := range expected {
		if got, ok := DefaultHourlyFines[k]; !ok {
			t.Errorf("missing default fine for %s", k)
		} else if got != v {
			t.Errorf("DefaultHourlyFines[%s] = %v, want %v", k, got, v)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TestEngineSetHourlyFine
// ═══════════════════════════════════════════════════════════════════════

func TestEngineSetHourlyFine(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	engine.SetHourlyFine("camera", 999.0)
	if got := engine.GetHourlyFine("camera"); got != 999.0 {
		t.Errorf("GetHourlyFine() after Set = %v, want 999.0", got)
	}

	// Проверяем что не сломали другие типы
	if got := engine.GetHourlyFine("cash_register"); got != 500.0 {
		t.Errorf("GetHourlyFine(cash_register) = %v, want 500.0", got)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TestCustomFinesInConstructor
// ═══════════════════════════════════════════════════════════════════════

func TestCustomFinesInConstructor(t *testing.T) {
	custom := map[string]float64{
		"camera":      150.0,
		"custom_type": 999.0,
	}

	engine := NewEngine(nil, nil, custom)

	// Custom override
	if got := engine.GetHourlyFine("camera"); got != 150.0 {
		t.Errorf("expected 150.0, got %v", got)
	}

	// New type from custom
	if got := engine.GetHourlyFine("custom_type"); got != 999.0 {
		t.Errorf("expected 999.0, got %v", got)
	}

	// Default still available
	if got := engine.GetHourlyFine("cash_register"); got != 500.0 {
		t.Errorf("expected 500.0, got %v", got)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Benchmarks
// ═══════════════════════════════════════════════════════════════════════

func BenchmarkCalculateRisk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculateRisk(120, "camera", 0)
	}
}

func BenchmarkEngineCalculateRisk(b *testing.B) {
	engine := NewEngine(nil, nil, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateRisk(120, "camera", 0)
	}
}
