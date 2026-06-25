// Package compliance — Compliance & Fines Shield (KF-15.1.1).
//
// Конвертирует downtime CCTV-камер в денежный риск ($/час штрафа).
//
// Compliance:
//   - IEC 62443-3-3 SR 7.1 (Resource availability — risk quantification)
//   - ISO 27001 A.12.4 (Audit trail — логирование расчётов)
//   - ISO 27019 PCC.A.13 (ICS asset risk assessment)
//   - СТБ 34.101.27 п. 6.3 (Оценка рисков)
//   - OWASP ASVS V5 (Input validation через whitelist device types)
//   - Приказ ОАЦ № 66 п. 7.18 (Идентификация устройств)
package compliance

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"gb-telemetry-collector/internal/downtime"
)

// ═══════════════════════════════════════════════════════════════════════
// Risk level
// ═══════════════════════════════════════════════════════════════════════

// RiskLevel представляет уровень финансового риска.
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// ═══════════════════════════════════════════════════════════════════════
// Models
// ═══════════════════════════════════════════════════════════════════════

// ComplianceRisk представляет расчёт финансового риска для устройства.
type ComplianceRisk struct {
	DeviceID      string    `json:"device_id" db:"device_id"`
	DeviceName    string    `json:"device_name,omitempty"`
	DeviceType    string    `json:"device_type" db:"device_type"`
	SiteID        string    `json:"site_id,omitempty" db:"site_id"`
	SiteName      string    `json:"site_name,omitempty"`
	DowntimeMin   int64     `json:"total_downtime_min" db:"total_downtime_min"`
	DowntimeHours float64   `json:"downtime_hours"`
	HourlyFine    float64   `json:"hourly_fine" db:"hourly_fine"`
	TotalExposure float64   `json:"total_exposure" db:"total_exposure"`
	RiskLevel     RiskLevel `json:"risk_level" db:"risk_level"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// ComplianceSummary представляет агрегированную сводку рисков.
type ComplianceSummary struct {
	TotalExposure    float64           `json:"total_exposure"`
	AtRiskDevices    int               `json:"at_risk_devices"`
	CompliantDevices int               `json:"compliant_devices"`
	TotalDevices     int               `json:"total_devices"`
	TopRisks         []ComplianceRisk  `json:"top_risks,omitempty"`
	RiskBreakdown    map[RiskLevel]int `json:"risk_breakdown,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Default fine rates per device type ($/hour)
// ═══════════════════════════════════════════════════════════════════════

// DefaultHourlyFines — таблица штрафов по умолчанию в $/час простоя.
//
// Основание:
//   - Кассовые зоны (cash register): $500/ч — критическая инфраструктура
//   - Периметр (perimeter): $200/ч — охраняемая зона
//   - Склад (warehouse): $300/ч — товарно-материальные ценности
//   - Офис (office): $100/ч — стандартная зона
var DefaultHourlyFines = map[string]float64{
	"cash_register": 500.0,
	"perimeter":     200.0,
	"warehouse":     300.0,
	"office":        100.0,

	// Fallback для device_type
	"camera":  100.0,
	"nvr":     250.0,
	"dvr":     200.0,
	"switch":  150.0,
	"server":  400.0,
	"encoder": 180.0,
	"ups":     120.0,
}

// ═══════════════════════════════════════════════════════════════════════
// Risk thresholds (total exposure in $)
// ═══════════════════════════════════════════════════════════════════════

const (
	// ThresholdMedium — начиная с $1000 — средний риск
	ThresholdMedium float64 = 1000.0
	// ThresholdHigh — начиная с $5000 — высокий риск
	ThresholdHigh float64 = 5000.0
	// ThresholdCritical — начиная с $25000 — критический риск
	ThresholdCritical float64 = 25000.0

	// AtRiskDowntimeMin — минимальный downtime для статуса "at risk" (≥ 1 час)
	AtRiskDowntimeMin int64 = 60
)

// ═══════════════════════════════════════════════════════════════════════
// Engine
// ═══════════════════════════════════════════════════════════════════════

// Engine вычисляет compliance риски на основе данных о простоях.
type Engine struct {
	mu          sync.RWMutex
	logger      *slog.Logger
	tracker     *downtime.Tracker
	hourlyFines map[string]float64 // device_type → $/hour
}

// NewEngine создаёт новый Compliance Engine.
// Принимает Tracker из downtime пакета и опциональную карту штрафов.
func NewEngine(tracker *downtime.Tracker, logger *slog.Logger, customFines map[string]float64) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	fines := make(map[string]float64)
	for k, v := range DefaultHourlyFines {
		fines[k] = v
	}
	for k, v := range customFines {
		fines[k] = v
	}
	return &Engine{
		logger:      logger.With("component", "compliance"),
		tracker:     tracker,
		hourlyFines: fines,
	}
}

// GetHourlyFine возвращает штраф для указанного типа устройства.
func (e *Engine) GetHourlyFine(deviceType string) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if fine, ok := e.hourlyFines[deviceType]; ok {
		return fine
	}
	// Fallback на "camera" для неизвестных типов
	return e.hourlyFines["camera"]
}

// SetHourlyFine устанавливает штраф для типа устройства (thread-safe).
func (e *Engine) SetHourlyFine(deviceType string, fine float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hourlyFines[deviceType] = fine
}

// ═══════════════════════════════════════════════════════════════════════
// CalculateRisk — расчёт финансового риска (KF-15.1.1)
// ═══════════════════════════════════════════════════════════════════════

// CalculateRisk вычисляет финансовый риск на основе времени простоя,
// типа устройства и почасовой ставки штрафа.
//
// Параметры:
//   - downtimeMinutes: общее время простоя в минутах
//   - deviceType: тип устройства (camera, nvr, perimeter, cash_register и т.д.)
//   - hourlyRate: почасовой штраф ($/час), если 0 — используется default
//
// Возвращает:
//   - totalExposure: общий финансовый риск в $
//   - riskLevel: уровень риска (low/medium/high/critical)
func (e *Engine) CalculateRisk(downtimeMinutes int64, deviceType string, hourlyRate float64) (totalExposure float64, riskLevel RiskLevel) {
	if downtimeMinutes <= 0 {
		return 0, RiskLevelLow
	}

	// Используем переданную ставку или default
	rate := hourlyRate
	if rate <= 0 {
		rate = e.GetHourlyFine(deviceType)
	}

	// Расчёт: (downtime_min / 60) * hourlyRate
	hours := float64(downtimeMinutes) / 60.0
	totalExposure = math.Round(hours*rate*100) / 100

	// Определяем уровень риска
	switch {
	case totalExposure >= ThresholdCritical:
		riskLevel = RiskLevelCritical
	case totalExposure >= ThresholdHigh:
		riskLevel = RiskLevelHigh
	case totalExposure >= ThresholdMedium:
		riskLevel = RiskLevelMedium
	default:
		riskLevel = RiskLevelLow
	}

	e.logger.Debug("risk calculated",
		"downtime_min", downtimeMinutes,
		"device_type", deviceType,
		"hourly_rate", rate,
		"total_exposure", totalExposure,
		"risk_level", riskLevel,
	)

	return totalExposure, riskLevel
}

// GetComplianceRisk создаёт ComplianceRisk из AssetDowntime данных.
func (e *Engine) GetComplianceRisk(dt *downtime.AssetDowntime) *ComplianceRisk {
	if dt == nil {
		return nil
	}

	hourlyFine := e.GetHourlyFine(dt.DeviceType)
	totalExposure, riskLevel := e.CalculateRisk(int64(dt.DurationMin), dt.DeviceType, hourlyFine)

	return &ComplianceRisk{
		DeviceID:      dt.DeviceID,
		DeviceName:    dt.DeviceName,
		DeviceType:    dt.DeviceType,
		SiteID:        dt.SiteID,
		DowntimeMin:   int64(dt.DurationMin),
		DowntimeHours: float64(dt.DurationMin) / 60.0,
		HourlyFine:    hourlyFine,
		TotalExposure: totalExposure,
		RiskLevel:     riskLevel,
		UpdatedAt:     time.Now().UTC(),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GetComplianceSummary — агрегированная сводка рисков
// ═══════════════════════════════════════════════════════════════════════

// GetComplianceSummary вычисляет агрегированную сводку рисков.
// Принимает список простоев и возвращает сводку.
func (e *Engine) GetComplianceSummary(downtimes []*downtime.AssetDowntime) *ComplianceSummary {
	summary := &ComplianceSummary{
		RiskBreakdown: make(map[RiskLevel]int),
		TopRisks:      make([]ComplianceRisk, 0),
	}

	deviceMap := make(map[string]*ComplianceRisk)

	for _, dt := range downtimes {
		risk := e.GetComplianceRisk(dt)
		if risk == nil {
			continue
		}

		// Агрегируем по device_id
		existing, ok := deviceMap[dt.DeviceID]
		if ok {
			existing.DowntimeMin += risk.DowntimeMin
			existing.DowntimeHours = float64(existing.DowntimeMin) / 60.0
			existing.TotalExposure += risk.TotalExposure
			existing.RiskLevel = classifyRisk(existing.TotalExposure)
		} else {
			deviceMap[dt.DeviceID] = risk
		}
	}

	for _, risk := range deviceMap {
		summary.TotalExposure += risk.TotalExposure
		summary.TotalDevices++

		if risk.DowntimeMin >= AtRiskDowntimeMin && risk.TotalExposure > 0 {
			summary.AtRiskDevices++
		} else {
			summary.CompliantDevices++
		}

		summary.RiskBreakdown[risk.RiskLevel]++
		summary.TopRisks = append(summary.TopRisks, *risk)
	}

	// Сортируем top risks по убыванию exposure (топ-10)
	sortRisks(summary.TopRisks)
	if len(summary.TopRisks) > 10 {
		summary.TopRisks = summary.TopRisks[:10]
	}

	summary.TotalExposure = math.Round(summary.TotalExposure*100) / 100

	e.logger.Info("compliance summary calculated",
		"total_exposure", summary.TotalExposure,
		"at_risk", summary.AtRiskDevices,
		"compliant", summary.CompliantDevices,
		"total_devices", summary.TotalDevices,
	)

	return summary
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// classifyRisk определяет уровень риска по сумме экспозиции.
func classifyRisk(exposure float64) RiskLevel {
	switch {
	case exposure >= ThresholdCritical:
		return RiskLevelCritical
	case exposure >= ThresholdHigh:
		return RiskLevelHigh
	case exposure >= ThresholdMedium:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// sortRisks сортирует риски по убыванию total_exposure (insertion sort для малых массивов).
func sortRisks(risks []ComplianceRisk) {
	n := len(risks)
	for i := 1; i < n; i++ {
		key := risks[i]
		j := i - 1
		for j >= 0 && risks[j].TotalExposure < key.TotalExposure {
			risks[j+1] = risks[j]
			j--
		}
		risks[j+1] = key
	}
}

// ═══════════════════════════════════════════════════════════════════════
// CalculateRisk — stateless standalone функция
// ═══════════════════════════════════════════════════════════════════════

// CalculateRisk вычисляет финансовый риск без привязки к Engine.
// Удобно для unit-тестов и использования без создания Engine.
func CalculateRisk(downtimeMinutes int64, deviceType string, hourlyRate float64) (totalExposure float64, riskLevel RiskLevel) {
	if downtimeMinutes <= 0 {
		return 0, RiskLevelLow
	}

	rate := hourlyRate
	if rate <= 0 {
		if fine, ok := DefaultHourlyFines[deviceType]; ok {
			rate = fine
		} else {
			rate = DefaultHourlyFines["camera"]
		}
	}

	hours := float64(downtimeMinutes) / 60.0
	totalExposure = math.Round(hours*rate*100) / 100

	switch {
	case totalExposure >= ThresholdCritical:
		riskLevel = RiskLevelCritical
	case totalExposure >= ThresholdHigh:
		riskLevel = RiskLevelHigh
	case totalExposure >= ThresholdMedium:
		riskLevel = RiskLevelMedium
	default:
		riskLevel = RiskLevelLow
	}

	return totalExposure, riskLevel
}

// ═══════════════════════════════════════════════════════════════════════
// Errors
// ═══════════════════════════════════════════════════════════════════════

// ValidationError возвращает ошибку валидации.
func ValidationError(field, msg string) error {
	return fmt.Errorf("compliance: %s: %s", field, msg)
}
