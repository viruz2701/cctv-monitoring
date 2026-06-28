// Package inventory — P2-INV.2: Vendor Scorecards — рейтинг поставщиков.
//
// Метрики: delivery time, quality, price, reliability.
// Соответствует ISO 27001 A.15 (Supplier relationships), IEC 62443 SL-3.
package inventory

import (
	"math"
	"sort"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.2.1: VendorScore — скор-карта поставщика
// ═══════════════════════════════════════════════════════════════════════

// VendorScore содержит метрики оценки поставщика.
type VendorScore struct {
	ID         string `json:"id"`
	VendorID   string `json:"vendor_id"`
	VendorName string `json:"vendor_name"`
	IsActive   bool   `json:"is_active"`

	// ═══ Метрики (0.0 – 100.0) ═══

	// DeliveryScore — своевременность поставок (P2-INV.2.2)
	// 100 = всегда вовремя, 0 = никогда
	DeliveryScore float64 `json:"delivery_score"`

	// QualityScore — качество продукции (P2-INV.2.3)
	// 100 = 0% брака, 0 = >50% брака
	QualityScore float64 `json:"quality_score"`

	// PriceScore — ценовая конкурентоспособность (P2-INV.2.4)
	// 100 = ниже рынка, 50 = рынок, 0 = значительно выше
	PriceScore float64 `json:"price_score"`

	// ReliabilityScore — надёжность поставщика (P2-INV.2.5)
	// 100 = идеально, 0 = ненадёжен
	ReliabilityScore float64 `json:"reliability_score"`

	// TotalOrders — общее количество заказов у этого поставщика
	TotalOrders int `json:"total_orders"`

	// CompletedOrders — успешно выполненные заказы
	CompletedOrders int `json:"completed_orders"`

	// AvgLeadTimeDays — среднее время поставки в днях
	AvgLeadTimeDays float64 `json:"avg_lead_time_days"`

	// PriceCompetitiveness — ценовая конкурентоспособность (0.0–1.0)
	PriceCompetitiveness float64 `json:"price_competitiveness"`

	// DefectRate — процент брака (0.0–1.0)
	DefectRate float64 `json:"defect_rate"`

	// OnTimeDeliveryRate — процент своевременных поставок (0.0–1.0)
	OnTimeDeliveryRate float64 `json:"on_time_delivery_rate"`

	// ContractComplianceRate — соблюдение контрактных условий (0.0–1.0)
	ContractComplianceRate float64 `json:"contract_compliance_rate"`

	// LastOrderDate — дата последнего заказа
	LastOrderDate *time.Time `json:"last_order_date,omitempty"`

	// Categories — категории запчастей, которые поставляет
	Categories []string `json:"categories,omitempty"`

	// Notes — заметки
	Notes string `json:"notes,omitempty"`

	// UpdatedAt — время последнего обновления
	UpdatedAt time.Time `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.2.6: WeightConfig — веса для расчёта общего рейтинга
// ═══════════════════════════════════════════════════════════════════════

// WeightConfig содержит веса метрик для расчёта общего рейтинга.
// Сумма весов должна быть равна 1.0.
type WeightConfig struct {
	DeliveryWeight    float64 `json:"delivery_weight"`
	QualityWeight     float64 `json:"quality_weight"`
	PriceWeight       float64 `json:"price_weight"`
	ReliabilityWeight float64 `json:"reliability_weight"`
}

// DefaultWeights возвращает веса по умолчанию.
func DefaultWeights() WeightConfig {
	return WeightConfig{
		DeliveryWeight:    0.30,
		QualityWeight:     0.30,
		PriceWeight:       0.20,
		ReliabilityWeight: 0.20,
	}
}

// Validate проверяет, что сумма весов равна 1.0.
func (w WeightConfig) Validate() bool {
	sum := w.DeliveryWeight + w.QualityWeight + w.PriceWeight + w.ReliabilityWeight
	return math.Abs(sum-1.0) < 0.001
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.2.7: OverallScore — общий рейтинг поставщика
// ═══════════════════════════════════════════════════════════════════════

// OverallScore возвращает взвешенный общий рейтинг поставщика (0–100).
func (vs *VendorScore) OverallScore() float64 {
	w := DefaultWeights()
	return vs.DeliveryScore*w.DeliveryWeight +
		vs.QualityScore*w.QualityWeight +
		vs.PriceScore*w.PriceWeight +
		vs.ReliabilityScore*w.ReliabilityWeight
}

// OverallScoreWithWeights возвращает общий рейтинг с пользовательскими весами.
func (vs *VendorScore) OverallScoreWithWeights(w WeightConfig) float64 {
	return vs.DeliveryScore*w.DeliveryWeight +
		vs.QualityScore*w.QualityWeight +
		vs.PriceScore*w.PriceWeight +
		vs.ReliabilityScore*w.ReliabilityWeight
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.2.8: Rating — текстовый рейтинг
// ═══════════════════════════════════════════════════════════════════════

// VendorRating представляет текстовую оценку поставщика.
type VendorRating string

const (
	VendorRatingExcellent VendorRating = "excellent" // ≥ 90
	VendorRatingGood      VendorRating = "good"      // ≥ 75
	VendorRatingAverage   VendorRating = "average"   // ≥ 50
	VendorRatingPoor      VendorRating = "poor"      // ≥ 25
	VendorRatingBad       VendorRating = "bad"       // < 25
)

// Rating возвращает текстовую оценку на основе общего рейтинга.
func (vs *VendorScore) Rating() VendorRating {
	score := vs.OverallScore()
	switch {
	case score >= 90:
		return VendorRatingExcellent
	case score >= 75:
		return VendorRatingGood
	case score >= 50:
		return VendorRatingAverage
	case score >= 25:
		return VendorRatingPoor
	default:
		return VendorRatingBad
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.2.9: ScoreCalculator — расчёт метрик
// ═══════════════════════════════════════════════════════════════════════

// ScoreCalculator вычисляет метрики поставщика на основе данных.
type ScoreCalculator struct {
	Config WeightConfig
}

// NewScoreCalculator создаёт новый калькулятор с весами по умолчанию.
func NewScoreCalculator() *ScoreCalculator {
	return &ScoreCalculator{
		Config: DefaultWeights(),
	}
}

// CalculateDeliveryScore вычисляет оценку своевременности поставок.
// onTime: кол-во своевременных поставок, total: общее кол-во.
func (sc *ScoreCalculator) CalculateDeliveryScore(onTime, total int) float64 {
	if total <= 0 {
		return 0
	}
	rate := float64(onTime) / float64(total)

	// onTimeRate = 1.0 → 100, 0.5 → 50, 0.0 → 0
	score := rate * 100.0

	// Штраф за задержки: каждый % задержек = -1.5 балла
	lateRate := 1.0 - rate
	penalty := lateRate * 50.0

	result := score - penalty
	if result < 0 {
		return 0
	}
	return math.Round(result*100) / 100
}

// CalculateQualityScore вычисляет оценку качества.
// defectQty: кол-во бракованных, totalQty: общее кол-во.
func (sc *ScoreCalculator) CalculateQualityScore(defectQty, totalQty int) float64 {
	if totalQty <= 0 {
		return 0
	}
	defectRate := float64(defectQty) / float64(totalQty)

	// defectRate = 0.0 → 100, 0.05 → 75, 0.10 → 50, 0.25 → 0
	score := (1.0 - defectRate*4.0) * 100.0
	if score < 0 {
		return 0
	}
	return math.Round(score*100) / 100
}

// CalculatePriceScore вычисляет ценовую оценку.
// ourPrice: цена поставщика, marketAvgPrice: среднерыночная цена.
// Если ourPrice > marketAvgPrice — выше рынка (штраф).
func (sc *ScoreCalculator) CalculatePriceScore(ourPrice, marketAvgPrice float64) float64 {
	if marketAvgPrice <= 0 || ourPrice <= 0 {
		return 50 // нейтрально
	}

	ratio := ourPrice / marketAvgPrice
	var score float64

	switch {
	case ratio <= 0.8:
		score = 100 // значительно ниже рынка
	case ratio <= 0.95:
		score = 85 // ниже рынка
	case ratio <= 1.05:
		score = 65 // рыночная цена
	case ratio <= 1.20:
		score = 40 // выше рынка
	default:
		score = 20 // значительно выше рынка
	}

	return score
}

// CalculateReliabilityScore вычисляет оценку надёжности.
// completed: успешно выполненные, total: общее кол-во заказов.
func (sc *ScoreCalculator) CalculateReliabilityScore(completed, total int) float64 {
	if total <= 0 {
		return 0
	}
	completionRate := float64(completed) / float64(total)
	return math.Round(completionRate*100*100) / 100
}

// CalculateAllScores вычисляет все метрики для поставщика.
func (sc *ScoreCalculator) CalculateAllScores(
	onTime, totalDeliveries int,
	defectQty, totalQty int,
	ourPrice, marketAvgPrice float64,
	completedOrders, totalOrders int,
) (delivery, quality, price, reliability float64) {
	delivery = sc.CalculateDeliveryScore(onTime, totalDeliveries)
	quality = sc.CalculateQualityScore(defectQty, totalQty)
	price = sc.CalculatePriceScore(ourPrice, marketAvgPrice)
	reliability = sc.CalculateReliabilityScore(completedOrders, totalOrders)
	return
}

// UpdateScore обновляет все метрики VendorScore на основе сырых данных.
func (sc *ScoreCalculator) UpdateScore(vs *VendorScore) {
	onTime := int(vs.OnTimeDeliveryRate * float64(vs.TotalOrders))
	vs.DeliveryScore = sc.CalculateDeliveryScore(onTime, vs.TotalOrders)
	vs.QualityScore = sc.CalculateQualityScore(int(vs.DefectRate*float64(vs.TotalOrders)), vs.TotalOrders)
	vs.PriceScore = sc.CalculatePriceScore(1.0, vs.PriceCompetitiveness)
	vs.ReliabilityScore = sc.CalculateReliabilityScore(vs.CompletedOrders, vs.TotalOrders)
	vs.UpdatedAt = time.Now().UTC()
}

// ═══════════════════════════════════════════════════════════════════════
// P2-INV.2.10: VendorRanking — ранжирование поставщиков
// ═══════════════════════════════════════════════════════════════════════

// VendorRanking содержит ранжированный список поставщиков.
type VendorRanking struct {
	RankedVendors []RankedVendor `json:"ranked_vendors"`
	GeneratedAt   time.Time      `json:"generated_at"`
}

// RankedVendor — поставщик с рангом.
type RankedVendor struct {
	Rank         int          `json:"rank"`
	Vendor       VendorScore  `json:"vendor"`
	OverallScore float64      `json:"overall_score"`
	Rating       VendorRating `json:"rating"`
}

// RankVendors ранжирует поставщиков по общему рейтингу.
func RankVendors(vendors []VendorScore) VendorRanking {
	sorted := make([]VendorScore, len(vendors))
	copy(sorted, vendors)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OverallScore() > sorted[j].OverallScore()
	})

	ranking := VendorRanking{
		GeneratedAt: time.Now().UTC(),
	}

	prevScore := -1.0
	currentRank := 0
	for i, v := range sorted {
		score := v.OverallScore()
		if i == 0 || score < prevScore {
			currentRank = i + 1
		}
		ranking.RankedVendors = append(ranking.RankedVendors, RankedVendor{
			Rank:         currentRank,
			Vendor:       v,
			OverallScore: score,
			Rating:       v.Rating(),
		})
		prevScore = score
	}

	return ranking
}
