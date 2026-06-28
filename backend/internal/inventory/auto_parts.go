// Package inventory — управление запчастями, поставщиками и перезаказом.
//
// P2-INV.1: Auto Parts — автоматический заказ запчастей при low stock.
// P2-INV.2: Vendor Scorecards — рейтинг поставщиков.
// P2-INV.3: Lifecycle Cost — расчёт стоимости владения.
// P2-INV.4: Reorder Automation — автоматический перезаказ.
//
// Compliance:
//   - IEC 62443-3-3 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
//   - СТБ 34.101.27 (Защита информации — учёт активов)
//   - OWASP ASVS V5.1 (Input validation)
package inventory

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.1: Auto Parts — автоматический заказ запчастей при low stock
// ═══════════════════════════════════════════════════════════════════════

// PartType — категория запчасти для классификации.
type PartType string

const (
	PartTypeConsumable PartType = "consumable" // Расходные материалы
	PartTypeComponent  PartType = "component"  // Компоненты (PCB, чипы)
	PartTypeModule     PartType = "module"     // Модули (блоки питания, платы)
	PartTypeCable      PartType = "cable"      // Кабели и разъёмы
	PartTypeHousing    PartType = "housing"    // Корпуса и крепления
	PartTypeTool       PartType = "tool"       // Инструменты
	PartTypeOther      PartType = "other"      // Прочее
)

// Part — запчасть с полной информацией для инвентаризации.
type Part struct {
	ID              string    `json:"id"`
	SKU             string    `json:"sku"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	Type            PartType  `json:"type"`
	CategoryID      string    `json:"category_id,omitempty"`
	CurrentStock    int       `json:"current_stock"`
	MinStock        int       `json:"min_stock"`
	MaxStock        int       `json:"max_stock"`
	UnitPrice       float64   `json:"unit_price"`
	PreferredVendor string    `json:"preferred_vendor,omitempty"`
	LeadTimeDays    int       `json:"lead_time_days"` // Среднее время поставки
	Location        string    `json:"location,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// AutoOrderConfig — конфигурация авто-заказа (P2-INV.1.1)
// ═══════════════════════════════════════════════════════════════════════

// AutoOrderConfig содержит пороги и настройки для автоматического заказа.
type AutoOrderConfig struct {
	// LowStockThreshold — % от min_stock, при котором срабатывает триггер (по умолч. 50)
	LowStockThreshold int `json:"low_stock_threshold"`
	// CriticalThreshold — % от min_stock для критического уровня (по умолч. 25)
	CriticalThreshold int `json:"critical_threshold"`
	// AutoOrderThreshold — % от min_stock для авто-заказа (по умолч. 40)
	AutoOrderThreshold int `json:"auto_order_threshold"`
	// DefaultOrderQty — кол-во по умолчанию для заказа
	DefaultOrderQty int `json:"default_order_qty"`
	// MaxOrderQty — максимальное кол-во в одном заказе
	MaxOrderQty int `json:"max_order_qty"`
	// PreferPreferredVendor — предпочитать основного поставщика
	PreferPreferredVendor bool `json:"prefer_preferred_vendor"`
	// Currency — валюта заказа (ISO 4217)
	Currency string `json:"currency"`
}

// DefaultAutoOrderConfig возвращает конфигурацию по умолчанию.
func DefaultAutoOrderConfig() AutoOrderConfig {
	return AutoOrderConfig{
		LowStockThreshold:     50,
		CriticalThreshold:     25,
		AutoOrderThreshold:    40,
		DefaultOrderQty:       10,
		MaxOrderQty:           100,
		PreferPreferredVendor: true,
		Currency:              "USD",
	}
}

// ═══════════════════════════════════════════════════════════════════════
// StockLevel — уровень запаса (P2-INV.1.2)
// ═══════════════════════════════════════════════════════════════════════

// StockLevel представляет текущий уровень запаса запчасти.
type StockLevel int

const (
	StockLevelUnknown  StockLevel = iota // Неизвестно
	StockLevelOK                         // Достаточно (> min_stock*2)
	StockLevelLow                        // Мало (> min_stock, <= min_stock*2)
	StockLevelCritical                   // Критично (> 0, <= min_stock)
	StockLevelOut                        // Нет в наличии (= 0)
)

func (s StockLevel) String() string {
	switch s {
	case StockLevelOK:
		return "ok"
	case StockLevelLow:
		return "low"
	case StockLevelCritical:
		return "critical"
	case StockLevelOut:
		return "out"
	default:
		return "unknown"
	}
}

// EvaluateStockLevel оценивает уровень запаса запчасти.
func EvaluateStockLevel(currentStock, minStock int) StockLevel {
	switch {
	case currentStock <= 0:
		return StockLevelOut
	case currentStock <= minStock:
		return StockLevelCritical
	case currentStock <= minStock*2:
		return StockLevelLow
	default:
		return StockLevelOK
	}
}

// ═══════════════════════════════════════════════════════════════════════
// AutoOrderTrigger — результат проверки необходимости заказа (P2-INV.1.3)
// ═══════════════════════════════════════════════════════════════════════

// AutoOrderTrigger содержит информацию для создания заказа.
type AutoOrderTrigger struct {
	PartID       string     `json:"part_id"`
	PartName     string     `json:"part_name"`
	PartSKU      string     `json:"part_sku"`
	CurrentStock int        `json:"current_stock"`
	MinStock     int        `json:"min_stock"`
	Level        StockLevel `json:"level"`
	SuggestedQty int        `json:"suggested_quantity"`
	Priority     int        `json:"priority"` // 1=critical, 2=high, 3=normal
	UnitPrice    float64    `json:"unit_price"`
	VendorID     string     `json:"vendor_id,omitempty"`
	LeadTimeDays int        `json:"lead_time_days"`
	Reason       string     `json:"reason"`
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.1.3: CheckAutoOrder — проверка необходимости авто-заказа
// ═══════════════════════════════════════════════════════════════════════

// CheckAutoOrder проверяет, нужно ли создавать заказ для запчасти.
// Возвращает nil, если заказ не требуется.
func CheckAutoOrder(part Part, cfg AutoOrderConfig) *AutoOrderTrigger {
	if !part.IsActive || part.MinStock <= 0 {
		return nil
	}

	level := EvaluateStockLevel(part.CurrentStock, part.MinStock)
	if level == StockLevelOK {
		return nil
	}

	// Порог срабатывания
	threshold := int(math.Ceil(float64(part.MinStock) * float64(cfg.AutoOrderThreshold) / 100.0))
	if part.CurrentStock > threshold && level != StockLevelOut {
		return nil
	}

	// Расчёт рекомендуемого количества
	suggestedQty := part.MaxStock - part.CurrentStock
	if suggestedQty <= 0 {
		suggestedQty = cfg.DefaultOrderQty
	}
	if suggestedQty > cfg.MaxOrderQty {
		suggestedQty = cfg.MaxOrderQty
	}
	if suggestedQty < 1 {
		suggestedQty = 1
	}

	// Приоритет
	priority := 3 // normal
	reason := "low stock"
	switch level {
	case StockLevelCritical:
		priority = 1
		reason = "critical low stock — immediate reorder required"
	case StockLevelOut:
		priority = 1
		reason = "out of stock — urgent reorder required"
	case StockLevelLow:
		priority = 2
		reason = "stock below auto-order threshold"
	}

	// Выбор поставщика
	vendorID := part.PreferredVendor

	return &AutoOrderTrigger{
		PartID:       part.ID,
		PartName:     part.Name,
		PartSKU:      part.SKU,
		CurrentStock: part.CurrentStock,
		MinStock:     part.MinStock,
		Level:        level,
		SuggestedQty: suggestedQty,
		Priority:     priority,
		UnitPrice:    part.UnitPrice,
		VendorID:     vendorID,
		LeadTimeDays: part.LeadTimeDays,
		Reason:       reason,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.1.4: BatchCheck — массовая проверка запчастей
// ═══════════════════════════════════════════════════════════════════════

// BatchCheckResult содержит результат массовой проверки запчастей.
type BatchCheckResult struct {
	Triggers   []AutoOrderTrigger `json:"triggers"`
	TotalParts int                `json:"total_parts"`
	CheckedAt  time.Time          `json:"checked_at"`
}

// BatchCheckAutoOrder проверяет список запчастей и возвращает те,
// которые требуют заказа, отсортированные по приоритету.
func BatchCheckAutoOrder(parts []Part, cfg AutoOrderConfig) BatchCheckResult {
	var triggers []AutoOrderTrigger
	for _, part := range parts {
		if trigger := CheckAutoOrder(part, cfg); trigger != nil {
			triggers = append(triggers, *trigger)
		}
	}

	// Сортировка: сначала критические, потом по suggestedQty (убывание)
	sort.Slice(triggers, func(i, j int) bool {
		if triggers[i].Priority != triggers[j].Priority {
			return triggers[i].Priority < triggers[j].Priority
		}
		return triggers[i].SuggestedQty > triggers[j].SuggestedQty
	})

	return BatchCheckResult{
		Triggers:   triggers,
		TotalParts: len(parts),
		CheckedAt:  time.Now().UTC(),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.1.5: VendorSelector — выбор поставщика для запчасти
// ═══════════════════════════════════════════════════════════════════════

// VendorSelector выбирает лучшего поставщика на основе скор-карты.
type VendorSelector struct {
	Vendors []VendorScore `json:"vendors"`
}

// SelectBestVendor выбирает лучшего поставщика для данной запчасти.
// Учитывает: рейтинг, цену, время поставки, предпочтения.
func (vs *VendorSelector) SelectBestVendor(part Part, preferPreferred bool) *VendorScore {
	if len(vs.Vendors) == 0 {
		return nil
	}

	// Фильтруем активных поставщиков, у которых есть эта запчасть
	var candidates []VendorScore
	for _, v := range vs.Vendors {
		if !v.IsActive {
			continue
		}
		candidates = append(candidates, v)
	}

	if len(candidates) == 0 {
		return nil
	}

	// Если есть предпочтительный поставщик и он активен
	if preferPreferred && part.PreferredVendor != "" {
		for _, v := range candidates {
			if v.ID == part.PreferredVendor {
				return &v
			}
		}
	}

	// Выбираем поставщика с наивысшим общим рейтингом
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].OverallScore() > candidates[j].OverallScore()
	})

	return &candidates[0]
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.1.6: AutoOrderSummary — сводка по авто-заказам
// ═══════════════════════════════════════════════════════════════════════

// AutoOrderSummary представляет сводку по автоматическим заказам.
type AutoOrderSummary struct {
	TotalTriggers      int       `json:"total_triggers"`
	CriticalCount      int       `json:"critical_count"`
	HighCount          int       `json:"high_count"`
	NormalCount        int       `json:"normal_count"`
	EstimatedTotalCost float64   `json:"estimated_total_cost"`
	Currency           string    `json:"currency"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// Summarize создаёт сводку по триггерам.
func Summarize(triggers []AutoOrderTrigger, currency string) AutoOrderSummary {
	summary := AutoOrderSummary{
		Currency:    currency,
		GeneratedAt: time.Now().UTC(),
	}

	for _, t := range triggers {
		summary.TotalTriggers++
		summary.EstimatedTotalCost += float64(t.SuggestedQty) * t.UnitPrice
		switch t.Priority {
		case 1:
			summary.CriticalCount++
		case 2:
			summary.HighCount++
		default:
			summary.NormalCount++
		}
	}

	return summary
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.1.7: GenerateOrderRef — генерация номера заказа
// ═══════════════════════════════════════════════════════════════════════

// GenerateOrderRef генерирует номер заказа в формате PO-YYYY-NNNNNN.
func GenerateOrderRef(seqNumber int) string {
	year := time.Now().Year()
	return fmt.Sprintf("PO-%d-%06d", year, seqNumber)
}
