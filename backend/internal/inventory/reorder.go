// Package inventory — P2-INV.4: Reorder Automation — автоматический перезаказ.
//
// Правила: min/max stock, lead time, seasonal adjustments.
// Соответствует:
//   - IEC 62443-3-3 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
package inventory

import (
	"math"
	"sort"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.1: ReorderPoint — точка перезаказа
// ═══════════════════════════════════════════════════════════════════════

// ReorderPoint содержит параметры точки перезаказа для запчасти.
type ReorderPoint struct {
	PartID   string `json:"part_id"`
	PartName string `json:"part_name"`
	PartSKU  string `json:"part_sku"`

	// MinStock — минимальный запас (триггер перезаказа)
	MinStock int `json:"min_stock"`

	// MaxStock — максимальный желаемый запас
	MaxStock int `json:"max_stock"`

	// ReorderQty — количество для заказа
	ReorderQty int `json:"reorder_qty"`

	// LeadTimeDays — время выполнения заказа поставщиком
	LeadTimeDays int `json:"lead_time_days"`

	// SafetyStock — страховой запас (на время задержек)
	SafetyStock int `json:"safety_stock"`

	// LeadTimeDemand — ожидаемый расход за время поставки
	LeadTimeDemand int `json:"lead_time_demand"`
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.2: ReorderRule — правило перезаказа
// ═══════════════════════════════════════════════════════════════════════

// ReorderRule содержит правило для автоматического перезаказа.
type ReorderRule struct {
	ID       string `json:"id"`
	PartID   string `json:"part_id"`
	PartName string `json:"part_name,omitempty"`

	// MinStock — минимальный запас (триггер)
	MinStock int `json:"min_stock"`

	// MaxStock — максимальный желаемый запас
	MaxStock int `json:"max_stock"`

	// ReorderQty — количество для заказа (0 = auto-calculate)
	ReorderQty int `json:"reorder_qty"`

	// LeadTimeDays — время выполнения заказа поставщиком
	LeadTimeDays int `json:"lead_time_days"`

	// SafetyStockDays — дней страхового запаса
	SafetyStockDays int `json:"safety_stock_days"`

	// DailyConsumption — среднесуточное потребление
	DailyConsumption float64 `json:"daily_consumption"`

	// SeasonalMultiplier — сезонный коэффициент (1.0 = норма)
	SeasonalMultiplier float64 `json:"seasonal_multiplier"`

	// AutoApprove — автоматически утверждать заказ (без подтверждения)
	AutoApprove bool `json:"auto_approve"`

	// PreferredVendorID — предпочтительный поставщик
	PreferredVendorID string `json:"preferred_vendor_id,omitempty"`

	// IsActive — правило активно
	IsActive bool `json:"is_active"`

	// Notes — заметки
	Notes string `json:"notes,omitempty"`

	// CreatedAt — время создания
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt — время последнего обновления
	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultReorderRule создаёт правило перезаказа по умолчанию на основе запчасти.
func DefaultReorderRule(part Part) ReorderRule {
	dailyConsumption := 0.0
	if part.LeadTimeDays > 0 {
		// Оценка: min_stock / lead_time = примерное дневное потребление
		dailyConsumption = float64(part.MinStock) / float64(part.LeadTimeDays)
	}

	return ReorderRule{
		PartID:             part.ID,
		PartName:           part.Name,
		MinStock:           part.MinStock,
		MaxStock:           part.MaxStock,
		ReorderQty:         0, // auto-calculate
		LeadTimeDays:       part.LeadTimeDays,
		SafetyStockDays:    7,
		DailyConsumption:   dailyConsumption,
		SeasonalMultiplier: 1.0,
		AutoApprove:        false,
		PreferredVendorID:  part.PreferredVendor,
		IsActive:           true,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.3: ReorderPolicy — политика перезаказа
// ═══════════════════════════════════════════════════════════════════════

// ReorderPolicy определяет стратегию перезаказа.
type ReorderPolicy string

const (
	ReorderPolicyMinMax ReorderPolicy = "min_max" // Min/Max (классический)
	ReorderPolicyFixed  ReorderPolicy = "fixed"   // Фиксированный интервал
	ReorderPolicyKanban ReorderPolicy = "kanban"  // Канбан
	ReorderPolicyDemand ReorderPolicy = "demand"  // По потребности
)

// ReorderPolicyConfig содержит конфигурацию политики перезаказа.
type ReorderPolicyConfig struct {
	Policy             ReorderPolicy `json:"policy"`
	ReviewIntervalDays int           `json:"review_interval_days"` // для fixed policy
	KanbanSize         int           `json:"kanban_size"`          // для kanban policy
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.4: ReorderCalculator — калькулятор перезаказа
// ═══════════════════════════════════════════════════════════════════════

// ReorderCalculator вычисляет параметры перезаказа.
type ReorderCalculator struct {
	SeasonalFactors map[int]float64 // month → factor (1.0 = normal)
}

// NewReorderCalculator создаёт калькулятор с сезонными факторами по умолчанию.
func NewReorderCalculator() *ReorderCalculator {
	return &ReorderCalculator{
		SeasonalFactors: DefaultSeasonalFactors(),
	}
}

// DefaultSeasonalFactors возвращает сезонные коэффициенты по умолчанию для CCTV.
// Зимой выше нагрузка на оборудование (отопление, влажность) → больше запчастей.
func DefaultSeasonalFactors() map[int]float64 {
	return map[int]float64{
		1:  1.3, // Январь — пик
		2:  1.2, // Февраль
		3:  1.1, // Март
		4:  1.0, // Апрель
		5:  0.9, // Май
		6:  0.8, // Июнь — минимум
		7:  0.9, // Июль
		8:  1.0, // Август
		9:  1.1, // Сентябрь
		10: 1.2, // Октябрь
		11: 1.2, // Ноябрь
		12: 1.3, // Декабрь — пик
	}
}

// GetSeasonalFactor возвращает сезонный коэффициент для текущего месяца.
func (rc *ReorderCalculator) GetSeasonalFactor() float64 {
	month := time.Now().Month()
	if factor, ok := rc.SeasonalFactors[int(month)]; ok {
		return factor
	}
	return 1.0
}

// GetSeasonalFactorFor возвращает сезонный коэффициент для указанного месяца.
func (rc *ReorderCalculator) GetSeasonalFactorFor(month time.Month) float64 {
	if factor, ok := rc.SeasonalFactors[int(month)]; ok {
		return factor
	}
	return 1.0
}

// CalculateReorderPoint вычисляет точку перезаказа (ROP).
// ROP = (DailyConsumption × LeadTime) × SeasonalFactor + SafetyStock
func (rc *ReorderCalculator) CalculateReorderPoint(rule ReorderRule) ReorderPoint {
	seasonal := rule.SeasonalMultiplier
	if seasonal <= 0 {
		seasonal = rc.GetSeasonalFactor()
	}

	dailyConsumption := rule.DailyConsumption
	if dailyConsumption <= 0 {
		dailyConsumption = float64(rule.MinStock) / float64(max(rule.LeadTimeDays, 1))
	}

	leadTimeDemand := int(math.Ceil(dailyConsumption * float64(rule.LeadTimeDays) * seasonal))
	safetyStock := int(math.Ceil(dailyConsumption * float64(rule.SafetyStockDays) * seasonal))
	reorderPoint := leadTimeDemand + safetyStock

	// Расчёт количества для заказа
	reorderQty := rule.ReorderQty
	if reorderQty <= 0 {
		reorderQty = max(rule.MaxStock-rule.MinStock, rule.MinStock)
	}
	if reorderQty < 1 {
		reorderQty = 1
	}

	return ReorderPoint{
		PartID:         rule.PartID,
		PartName:       rule.PartName,
		PartSKU:        "",
		MinStock:       reorderPoint,
		MaxStock:       rule.MaxStock,
		ReorderQty:     reorderQty,
		LeadTimeDays:   rule.LeadTimeDays,
		SafetyStock:    safetyStock,
		LeadTimeDemand: leadTimeDemand,
	}
}

// CheckReorder проверяет, нужно ли делать перезаказ.
func (rc *ReorderCalculator) CheckReorder(part Part, rule ReorderRule) *AutoOrderTrigger {
	if !rule.IsActive || !part.IsActive {
		return nil
	}

	rop := rc.CalculateReorderPoint(rule)
	currentStock := part.CurrentStock

	if currentStock > rop.MinStock {
		return nil // выше точки перезаказа
	}

	// Расчёт приоритета
	priority := 3
	reason := "stock below reorder point"
	level := StockLevelLow

	stockRatio := float64(currentStock) / float64(rop.MinStock)
	switch {
	case currentStock <= 0:
		priority = 1
		reason = "out of stock — urgent reorder"
		level = StockLevelOut
	case stockRatio <= 0.25:
		priority = 1
		reason = "critical — stock below 25% of reorder point"
		level = StockLevelCritical
	case stockRatio <= 0.5:
		priority = 2
		reason = "stock below 50% of reorder point"
		level = StockLevelCritical
	}

	return &AutoOrderTrigger{
		PartID:       part.ID,
		PartName:     part.Name,
		PartSKU:      part.SKU,
		CurrentStock: currentStock,
		MinStock:     rop.MinStock,
		Level:        level,
		SuggestedQty: rop.ReorderQty,
		Priority:     priority,
		UnitPrice:    part.UnitPrice,
		VendorID:     rule.PreferredVendorID,
		LeadTimeDays: rop.LeadTimeDays,
		Reason:       reason,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.5: BatchReorder — массовая проверка перезаказов
// ═══════════════════════════════════════════════════════════════════════

// BatchReorderResult содержит результаты массовой проверки перезаказов.
type BatchReorderResult struct {
	Triggers     []AutoOrderTrigger `json:"triggers"`
	TotalChecked int                `json:"total_checked"`
	TotalRules   int                `json:"total_rules"`
	CheckedAt    time.Time          `json:"checked_at"`
}

// BatchCheckReorder проверяет несколько запчастей по их правилам.
func (rc *ReorderCalculator) BatchCheckReorder(parts []Part, rules []ReorderRule) BatchReorderResult {
	// Строим map rule → part
	ruleMap := make(map[string]ReorderRule)
	for _, r := range rules {
		ruleMap[r.PartID] = r
	}

	var triggers []AutoOrderTrigger
	for _, part := range parts {
		rule, ok := ruleMap[part.ID]
		if !ok {
			rule = DefaultReorderRule(part)
		}
		if trigger := rc.CheckReorder(part, rule); trigger != nil {
			triggers = append(triggers, *trigger)
		}
	}

	// Сортировка по приоритету
	sort.Slice(triggers, func(i, j int) bool {
		if triggers[i].Priority != triggers[j].Priority {
			return triggers[i].Priority < triggers[j].Priority
		}
		return triggers[i].SuggestedQty > triggers[j].SuggestedQty
	})

	return BatchReorderResult{
		Triggers:     triggers,
		TotalChecked: len(parts),
		TotalRules:   len(rules),
		CheckedAt:    time.Now().UTC(),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.6: ReorderSuggestion — рекомендация по перезаказу
// ═══════════════════════════════════════════════════════════════════════

// ReorderSuggestion содержит рекомендацию по перезаказу с деталями.
type ReorderSuggestion struct {
	PartID         string  `json:"part_id"`
	PartName       string  `json:"part_name"`
	PartSKU        string  `json:"part_sku"`
	CurrentStock   int     `json:"current_stock"`
	ReorderPoint   int     `json:"reorder_point"`
	RecommendedQty int     `json:"recommended_qty"`
	Priority       int     `json:"priority"`
	EstimatedCost  float64 `json:"estimated_cost"`
	Currency       string  `json:"currency"`
	Reason         string  `json:"reason"`
	AutoApprovable bool    `json:"auto_approvable"`
}

// GenerateSuggestions генерирует рекомендации по перезаказу.
func (rc *ReorderCalculator) GenerateSuggestions(parts []Part, rules []ReorderRule, currency string) []ReorderSuggestion {
	result := rc.BatchCheckReorder(parts, rules)

	var suggestions []ReorderSuggestion
	for _, t := range result.Triggers {
		// Проверяем auto-approve
		autoApprovable := false
		for _, r := range rules {
			if r.PartID == t.PartID {
				autoApprovable = r.AutoApprove
				break
			}
		}

		suggestions = append(suggestions, ReorderSuggestion{
			PartID:         t.PartID,
			PartName:       t.PartName,
			PartSKU:        t.PartSKU,
			CurrentStock:   t.CurrentStock,
			ReorderPoint:   t.MinStock,
			RecommendedQty: t.SuggestedQty,
			Priority:       t.Priority,
			EstimatedCost:  float64(t.SuggestedQty) * t.UnitPrice,
			Currency:       currency,
			Reason:         t.Reason,
			AutoApprovable: autoApprovable,
		})
	}

	return suggestions
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.4.7: ReorderReport — отчёт по перезаказам
// ═══════════════════════════════════════════════════════════════════════

// ReorderReport содержит полный отчёт по перезаказам.
type ReorderReport struct {
	GeneratedAt         time.Time           `json:"generated_at"`
	Policy              ReorderPolicyConfig `json:"policy"`
	TotalParts          int                 `json:"total_parts"`
	PartsBelowROP       int                 `json:"parts_below_rop"`
	Suggestions         []ReorderSuggestion `json:"suggestions"`
	AutoApprovableCount int                 `json:"auto_approvable_count"`
	EstimatedTotalCost  float64             `json:"estimated_total_cost"`
	Currency            string              `json:"currency"`
}

// GenerateReport генерирует полный отчёт по перезаказам.
func (rc *ReorderCalculator) GenerateReport(
	parts []Part,
	rules []ReorderRule,
	policy ReorderPolicyConfig,
	currency string,
) ReorderReport {
	suggestions := rc.GenerateSuggestions(parts, rules, currency)

	autoApprovableCount := 0
	estimatedTotalCost := 0.0
	for _, s := range suggestions {
		if s.AutoApprovable {
			autoApprovableCount++
		}
		estimatedTotalCost += s.EstimatedCost
	}

	return ReorderReport{
		GeneratedAt:         time.Now().UTC(),
		Policy:              policy,
		TotalParts:          len(parts),
		PartsBelowROP:       len(suggestions),
		Suggestions:         suggestions,
		AutoApprovableCount: autoApprovableCount,
		EstimatedTotalCost:  estimatedTotalCost,
		Currency:            currency,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// max возвращает максимальное из двух int.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
