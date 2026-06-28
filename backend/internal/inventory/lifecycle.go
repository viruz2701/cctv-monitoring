// Package inventory — P2-INV.3: Lifecycle Cost — расчёт стоимости владения.
//
// TCO = Purchase + Maintenance + Energy + Disposal.
// Соответствует:
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
//   - IEC 62443 SR 7.1 (Resource availability — asset TCO)
//   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
package inventory

import (
	"fmt"
	"math"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.3.1: LifecycleCost — стоимость владения запчастью/активом
// ═══════════════════════════════════════════════════════════════════════

// LifecycleCost содержит все компоненты стоимости владения.
type LifecycleCost struct {
	PartID   string `json:"part_id"`
	PartName string `json:"part_name"`
	PartSKU  string `json:"part_sku"`
	Currency string `json:"currency"`

	// ═══ Компоненты TCO ═══

	// PurchaseCost — стоимость приобретения
	PurchaseCost float64 `json:"purchase_cost"`

	// MaintenanceCost — стоимость обслуживания (за всё время)
	MaintenanceCost float64 `json:"maintenance_cost"`

	// EnergyCost — стоимость энергопотребления
	EnergyCost float64 `json:"energy_cost"`

	// DisposalCost — стоимость утилизации
	DisposalCost float64 `json:"disposal_cost"`

	// InstallationCost — стоимость установки
	InstallationCost float64 `json:"installation_cost,omitempty"`

	// TrainingCost — стоимость обучения персонала
	TrainingCost float64 `json:"training_cost,omitempty"`

	// TransportCost — стоимость транспортировки
	TransportCost float64 `json:"transport_cost,omitempty"`

	// ═══ Временные параметры ═══

	// ExpectedLifespanDays — ожидаемый срок службы в днях
	ExpectedLifespanDays int `json:"expected_lifespan_days"`

	// OperationalDays — количество дней в эксплуатации
	OperationalDays int `json:"operational_days"`

	// PurchaseDate — дата приобретения
	PurchaseDate *time.Time `json:"purchase_date,omitempty"`

	// LastMaintenanceDate — дата последнего обслуживания
	LastMaintenanceDate *time.Time `json:"last_maintenance_date,omitempty"`

	// CalculatedAt — время расчёта
	CalculatedAt time.Time `json:"calculated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.3.2: TCO calculations
// ═══════════════════════════════════════════════════════════════════════

// TotalCostOfOwnership возвращает полную стоимость владения.
func (lc *LifecycleCost) TotalCostOfOwnership() float64 {
	return lc.PurchaseCost +
		lc.MaintenanceCost +
		lc.EnergyCost +
		lc.DisposalCost +
		lc.InstallationCost +
		lc.TrainingCost +
		lc.TransportCost
}

// AnnualCost возвращает среднегодовую стоимость владения.
func (lc *LifecycleCost) AnnualCost() float64 {
	if lc.ExpectedLifespanDays <= 0 || lc.OperationalDays <= 0 {
		return lc.TotalCostOfOwnership()
	}
	years := float64(lc.OperationalDays) / 365.0
	if years < 0.01 {
		return lc.TotalCostOfOwnership()
	}
	return lc.TotalCostOfOwnership() / years
}

// CostPerDay возвращает стоимость владения в день.
func (lc *LifecycleCost) CostPerDay() float64 {
	if lc.OperationalDays <= 0 {
		return lc.TotalCostOfOwnership()
	}
	return lc.TotalCostOfOwnership() / float64(lc.OperationalDays)
}

// RemainingValue возвращает остаточную стоимость (линейная амортизация).
// Если актив полностью самортизирован, возвращает disposal cost.
func (lc *LifecycleCost) RemainingValue() float64 {
	if lc.ExpectedLifespanDays <= 0 {
		return lc.PurchaseCost
	}

	remainingDays := lc.ExpectedLifespanDays - lc.OperationalDays
	if remainingDays <= 0 {
		return lc.DisposalCost
	}

	depreciationRate := float64(lc.OperationalDays) / float64(lc.ExpectedLifespanDays)
	remainingValue := lc.PurchaseCost * (1.0 - depreciationRate)

	// Не ниже стоимости утилизации
	if remainingValue < lc.DisposalCost {
		return lc.DisposalCost
	}
	return math.Round(remainingValue*100) / 100
}

// Summary возвращает краткую сводку TCO.
func (lc *LifecycleCost) Summary() string {
	tco := lc.TotalCostOfOwnership()
	annual := lc.AnnualCost()
	perDay := lc.CostPerDay()

	return fmt.Sprintf("TCO: %.2f %s | Annual: %.2f %s | Per day: %.4f %s",
		tco, lc.Currency, annual, lc.Currency, perDay, lc.Currency)
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.3.3: TCOBreakdown — детализация TCO
// ═══════════════════════════════════════════════════════════════════════

// TCOComponent представляет один компонент TCO.
type TCOComponent struct {
	Name string  `json:"name"`
	Cost float64 `json:"cost"`
	Pct  float64 `json:"percentage"`
}

// TCOBreakdown содержит детализацию TCO по компонентам.
type TCOBreakdown struct {
	Components []TCOComponent `json:"components"`
	Total      float64        `json:"total"`
	Currency   string         `json:"currency"`
}

// Breakdown возвращает детализацию TCO по компонентам с процентами.
func (lc *LifecycleCost) Breakdown() TCOBreakdown {
	total := lc.TotalCostOfOwnership()
	components := []TCOComponent{
		{Name: "Purchase", Cost: lc.PurchaseCost},
		{Name: "Maintenance", Cost: lc.MaintenanceCost},
		{Name: "Energy", Cost: lc.EnergyCost},
		{Name: "Disposal", Cost: lc.DisposalCost},
		{Name: "Installation", Cost: lc.InstallationCost},
		{Name: "Training", Cost: lc.TrainingCost},
		{Name: "Transport", Cost: lc.TransportCost},
	}

	var breakdown []TCOComponent
	for _, c := range components {
		if c.Cost > 0 {
			pct := 0.0
			if total > 0 {
				pct = math.Round(c.Cost/total*1000) / 10 // 1 decimal
			}
			breakdown = append(breakdown, TCOComponent{
				Name: c.Name,
				Cost: math.Round(c.Cost*100) / 100,
				Pct:  pct,
			})
		}
	}

	return TCOBreakdown{
		Components: breakdown,
		Total:      math.Round(total*100) / 100,
		Currency:   lc.Currency,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.3.4: LifecycleCalculator — калькулятор TCO
// ═══════════════════════════════════════════════════════════════════════

// LifecycleCalculatorConfig содержит настройки калькулятора TCO.
type LifecycleCalculatorConfig struct {
	// AverageEnergyPrice — средняя цена электроэнергии (за кВт·ч)
	AverageEnergyPrice float64 `json:"average_energy_price"`
	// PowerConsumptionWatts — энергопотребление в ваттах
	PowerConsumptionWatts float64 `json:"power_consumption_watts"`
	// DailyOperatingHours — часов работы в день
	DailyOperatingHours float64 `json:"daily_operating_hours"`
	// DisposalCostPct — % от purchase cost для утилизации
	DisposalCostPct float64 `json:"disposal_cost_pct"`
	// AnnualMaintenancePct — % от purchase cost на ежегодное обслуживание
	AnnualMaintenancePct float64 `json:"annual_maintenance_pct"`
	// Currency — валюта расчётов
	Currency string `json:"currency"`
}

// DefaultLifecycleConfig возвращает конфигурацию по умолчанию.
func DefaultLifecycleConfig() LifecycleCalculatorConfig {
	return LifecycleCalculatorConfig{
		AverageEnergyPrice:    0.12, // $0.12/кВт·ч
		PowerConsumptionWatts: 50,   // 50 Вт (типовое CCTV-устройство)
		DailyOperatingHours:   24,   // круглосуточно
		DisposalCostPct:       0.05, // 5% от purchase
		AnnualMaintenancePct:  0.10, // 10% от purchase в год
		Currency:              "USD",
	}
}

// LifecycleCalculator вычисляет TCO для запчастей и активов.
type LifecycleCalculator struct {
	Config LifecycleCalculatorConfig
}

// NewLifecycleCalculator создаёт новый калькулятор TCO.
func NewLifecycleCalculator(cfg LifecycleCalculatorConfig) *LifecycleCalculator {
	return &LifecycleCalculator{Config: cfg}
}

// Calculate вычисляет полный TCO для запчасти.
func (lc *LifecycleCalculator) Calculate(
	part Part,
	purchaseCost float64,
	operationalDays int,
) LifecycleCost {
	// Стоимость обслуживания
	maintenanceYears := float64(operationalDays) / 365.0
	maintenanceCost := purchaseCost * lc.Config.AnnualMaintenancePct * maintenanceYears

	// Стоимость энергопотребления
	hoursOperational := float64(operationalDays) * lc.Config.DailyOperatingHours
	kwh := (lc.Config.PowerConsumptionWatts / 1000.0) * hoursOperational
	energyCost := kwh * lc.Config.AverageEnergyPrice

	// Стоимость утилизации
	disposalCost := purchaseCost * lc.Config.DisposalCostPct

	now := time.Now().UTC()
	purchaseDate := now.AddDate(0, 0, -operationalDays)

	return LifecycleCost{
		PartID:               part.ID,
		PartName:             part.Name,
		PartSKU:              part.SKU,
		Currency:             lc.Config.Currency,
		PurchaseCost:         math.Round(purchaseCost*100) / 100,
		MaintenanceCost:      math.Round(maintenanceCost*100) / 100,
		EnergyCost:           math.Round(energyCost*100) / 100,
		DisposalCost:         math.Round(disposalCost*100) / 100,
		ExpectedLifespanDays: 3650, // 10 лет по умолчанию
		OperationalDays:      operationalDays,
		PurchaseDate:         &purchaseDate,
		CalculatedAt:         now,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// LifecycleComparison — сравнение TCO нескольких запчастей/активов
// ═══════════════════════════════════════════════════════════════════════

// LifecycleComparison сравнивает TCO нескольких запчастей.
type LifecycleComparison struct {
	Items       []LifecycleCost `json:"items"`
	BestValueID string          `json:"best_value_id"` // запчасть с лучшим TCO/день
	GeneratedAt time.Time       `json:"generated_at"`
}

// Compare возвращает сравнение TCO для нескольких запчастей.
func Compare(costs []LifecycleCost) LifecycleComparison {
	if len(costs) == 0 {
		return LifecycleComparison{GeneratedAt: time.Now().UTC()}
	}

	bestValueID := costs[0].PartID
	bestCostPerDay := costs[0].CostPerDay()

	for _, c := range costs[1:] {
		cpd := c.CostPerDay()
		if cpd < bestCostPerDay {
			bestCostPerDay = cpd
			bestValueID = c.PartID
		}
	}

	return LifecycleComparison{
		Items:       costs,
		BestValueID: bestValueID,
		GeneratedAt: time.Now().UTC(),
	}
}
