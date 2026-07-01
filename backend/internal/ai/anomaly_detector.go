// Package ai — Anomaly Detection Engine.
//
// P2-AI.4: Статистический анализ метрик устройств:
//   - z-score аномалии
//   - Moving average smoothing
//   - Per-device metric buffers
package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"
)

// ─── Статистические функции ───────────────────────────────────────────────

// MaxFiniteZScore — максимальное конечное значение z-score.
// Используется вместо math.Inf для предотвращения ошибки JSON сериализации
// "json: unsupported value: +Inf" (P1-HI-08).
const MaxFiniteZScore = 100.0

// calculateZScore вычисляет z-score для значения относительно выборки.
// z = (x - mean) / stddev
//
// Защита от division by zero: при stdDev == 0 возвращает конечное значение,
// а не math.Inf, чтобы избежать ошибки JSON сериализации (P1-HI-08).
func calculateZScore(value float64, values []float64) (float64, float64, float64) {
	n := len(values)
	if n == 0 {
		return 0, 0, 0
	}

	// Mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)

	// Standard deviation
	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(n)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		if value == mean {
			return 0, mean, 0
		}
		// Если stdDev = 0, но value != mean — все исторические значения
		// одинаковы, текущее отличается. Возвращаем MaxFiniteZScore вместо
		// math.Inf, т.к. Inf не сериализуется в JSON (P1-HI-08).
		return MaxFiniteZScore, mean, 0
	}

	z := (value - mean) / stdDev
	return z, mean, stdDev
}

// calculateMovingAverage вычисляет скользящее среднее.
func calculateMovingAverage(values []float64, window int) []float64 {
	if len(values) == 0 || window <= 0 {
		return nil
	}
	if window > len(values) {
		window = len(values)
	}

	result := make([]float64, 0, len(values)-window+1)
	for i := 0; i <= len(values)-window; i++ {
		var sum float64
		for j := i; j < i+window; j++ {
			sum += values[j]
		}
		result = append(result, sum/float64(window))
	}
	return result
}

// ─── MetricBuffer ─────────────────────────────────────────────────────────

// MetricBuffer — FIFO буфер метрик для одного типа метрики устройства.
type MetricBuffer struct {
	mu     sync.RWMutex
	points []DeviceMetricPoint
	maxLen int
}

// NewMetricBuffer создаёт буфер метрик с заданным максимальным размером.
func NewMetricBuffer(maxLen int) *MetricBuffer {
	return &MetricBuffer{
		points: make([]DeviceMetricPoint, 0, maxLen),
		maxLen: maxLen,
	}
}

// Push добавляет точку метрики в буфер (FIFO, при переполнении удаляет старые).
func (b *MetricBuffer) Push(p DeviceMetricPoint) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.points) >= b.maxLen {
		// Удаляем первые 10% для амортизации
		trim := b.maxLen / 10
		if trim < 1 {
			trim = 1
		}
		b.points = b.points[trim:]
	}
	b.points = append(b.points, p)
}

// Points возвращает копию текущих точек.
func (b *MetricBuffer) Points() []DeviceMetricPoint {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]DeviceMetricPoint, len(b.points))
	copy(result, b.points)
	return result
}

// Len возвращает количество точек в буфере.
func (b *MetricBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.points)
}

// Values возвращает числовые значения точек.
func (b *MetricBuffer) Values() []float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	vals := make([]float64, len(b.points))
	for i, p := range b.points {
		vals[i] = p.Value
	}
	return vals
}

// Clear очищает буфер.
func (b *MetricBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.points = make([]DeviceMetricPoint, 0, b.maxLen)
}

// ─── AnomalyDetector ──────────────────────────────────────────────────────

// AnomalyDetector — движок обнаружения аномалий на основе статистических методов.
//
// Хранит буферы метрик per-device/per-metric-type и запускает анализ
// по z-score и скользящему среднему.
type AnomalyDetector struct {
	cfg    AnomalyConfig
	logger *slog.Logger

	mu sync.RWMutex
	// metrics[deviceID][metricType] → *MetricBuffer
	metrics map[string]map[string]*MetricBuffer

	// activeAnomalies[deviceID] → []*AnomalyResult
	activeAnomalies map[string][]*AnomalyResult

	// Все аномалии по ID для быстрого доступа
	anomalyIndex map[string]*AnomalyResult

	// Исторические аномалии (для листинга)
	resolvedAnomalies []*AnomalyResult
	maxResolved       int
}

// NewAnomalyDetector создаёт новый AnomalyDetector.
func NewAnomalyDetector(cfg AnomalyConfig, logger *slog.Logger) *AnomalyDetector {
	if logger == nil {
		logger = slog.Default()
	}
	return &AnomalyDetector{
		cfg:               cfg,
		logger:            logger.With("component", "anomaly-detector"),
		metrics:           make(map[string]map[string]*MetricBuffer),
		activeAnomalies:   make(map[string][]*AnomalyResult),
		anomalyIndex:      make(map[string]*AnomalyResult),
		resolvedAnomalies: make([]*AnomalyResult, 0, 1000),
		maxResolved:       10000,
	}
}

// ─── Public API ───────────────────────────────────────────────────────────

// Feed добавляет метрику в буфер и возвращает аномалии, если обнаружены.
// Если метрика является аномальной, возвращает AnomalyResult.
//
// Warm-up period (P2-MED-27): пока не накоплено WarmUpSamples точек данных,
// используются статические пороги вместо z-score. Это предотвращает ложные
// срабатывания на малых выборках, когда stdDev может быть 0 или нестабильным.
func (d *AnomalyDetector) Feed(ctx context.Context, m DeviceMetricPoint) *AnomalyResult {
	buf := d.getOrCreateBuffer(m.DeviceID, m.MetricType)
	buf.Push(m)

	// Проверяем, достаточно ли данных
	if buf.Len() < d.cfg.MinDataPoints {
		return nil
	}

	// Анализируем
	return d.evaluate(ctx, m.DeviceID, m.MetricType, m.Value)
}

// isWarmUpComplete проверяет, завершён ли разогрев детектора для указанной метрики.
// Возвращает true, когда накоплено WarmUpSamples или больше точек данных.
// P2-MED-27: до 30+ samples используем статические пороги.
func (d *AnomalyDetector) isWarmUpComplete(deviceID, metricType string) bool {
	buf := d.getOrCreateBuffer(deviceID, metricType)
	return buf.Len() >= d.cfg.WarmUpSamples
}

// checkStaticThreshold проверяет значение метрики по статическим порогам для warm-up периода.
// Возвращает true, если значение выходит за пределы статического порога (аномалия).
// Использует DefaultWarmUpThresholds() если кастомные не заданы.
// P2-MED-27: статические пороги до накопления 30+ samples.
func (d *AnomalyDetector) checkStaticThreshold(deviceID, metricType string, value float64) (bool, float64, float64, float64) {
	thresholds := DefaultWarmUpThresholds()
	t, ok := thresholds[metricType]
	if !ok {
		// Если порог для метрики не найден — пропускаем (нет аномалии)
		return false, 0, 0, 0
	}

	buf := d.getOrCreateBuffer(deviceID, metricType)
	values := buf.Values()

	// Вычисляем среднее по имеющимся данным
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Если задан ToleranceMultiplier — используем его для динамического порога
	// на основе среднего, но не выходя за абсолютные границы [Min, Max].
	tolerance := t.ToleranceMultiplier
	if tolerance <= 0 {
		tolerance = 2.0
	}

	upperBound := mean * tolerance
	lowerBound := mean / tolerance

	// Ограничиваем абсолютными границами
	if t.Max > 0 && upperBound > t.Max {
		upperBound = t.Max
	}
	if lowerBound < t.Min {
		lowerBound = t.Min
	}

	// Проверяем выход за границы
	if value > upperBound || value < lowerBound {
		// Вычисляем z-score-like отклонение для консистентности
		stdDev := 0.0
		if len(values) > 1 {
			var varianceSum float64
			for _, v := range values {
				diff := v - mean
				varianceSum += diff * diff
			}
			variance := varianceSum / float64(len(values))
			stdDev = math.Sqrt(variance)
		}

		return true, mean, stdDev, (value - mean) / (stdDev + 0.0001)
	}

	return false, mean, 0, 0
}

// GetActiveAnomalies возвращает активные (не разрешённые) аномалии.
func (d *AnomalyDetector) GetActiveAnomalies(deviceID, metricType, severity string) []AnomalyResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var results []AnomalyResult
	for _, anomalies := range d.activeAnomalies {
		for _, a := range anomalies {
			if a.Status != AnomalyStatusResolved {
				if deviceID != "" && a.DeviceID != deviceID {
					continue
				}
				if metricType != "" && a.MetricType != metricType {
					continue
				}
				if severity != "" && string(a.Severity) != severity {
					continue
				}
				results = append(results, *a)
			}
		}
	}
	return results
}

// GetAllAnomalies возвращает все аномалии (активные + исторические).
func (d *AnomalyDetector) GetAllAnomalies(deviceID, metricType, severity, status string, limit int) []AnomalyResult {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var results []AnomalyResult

	// Собираем активные
	for _, anomalies := range d.activeAnomalies {
		for _, a := range anomalies {
			if d.matchAnomaly(a, deviceID, metricType, severity, status) {
				results = append(results, *a)
			}
		}
	}

	// Собираем разрешённые
	for _, a := range d.resolvedAnomalies {
		if d.matchAnomaly(a, deviceID, metricType, severity, status) {
			results = append(results, *a)
		}
	}

	// Сортируем по времени убывания (новые сверху)
	sort.Slice(results, func(i, j int) bool {
		return results[i].DetectedAt.After(results[j].DetectedAt)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// AcknowledgeAnomaly подтверждает аномалию.
func (d *AnomalyDetector) AcknowledgeAnomaly(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	a, ok := d.anomalyIndex[id]
	if !ok {
		return fmt.Errorf("anomaly not found: %s", id)
	}
	if a.Status == AnomalyStatusResolved {
		return fmt.Errorf("anomaly already resolved: %s", id)
	}
	a.Status = AnomalyStatusAcknowledged
	return nil
}

// ResolveAnomaly разрешает аномалию (переносит в историю).
func (d *AnomalyDetector) ResolveAnomaly(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	a, ok := d.anomalyIndex[id]
	if !ok {
		return fmt.Errorf("anomaly not found: %s", id)
	}
	if a.Status == AnomalyStatusResolved {
		return nil // уже resolved
	}

	now := time.Now()
	a.Status = AnomalyStatusResolved
	a.ResolvedAt = &now

	// Удаляем из activeAnomalies
	deviceAnomalies := d.activeAnomalies[a.DeviceID]
	filtered := make([]*AnomalyResult, 0, len(deviceAnomalies))
	for _, item := range deviceAnomalies {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		delete(d.activeAnomalies, a.DeviceID)
	} else {
		d.activeAnomalies[a.DeviceID] = filtered
	}

	// Добавляем в историю
	d.resolvedAnomalies = append(d.resolvedAnomalies, a)
	if len(d.resolvedAnomalies) > d.maxResolved {
		d.resolvedAnomalies = d.resolvedAnomalies[len(d.resolvedAnomalies)-d.maxResolved:]
	}

	return nil
}

// GetBufferStats возвращает статистику по буферам метрик.
func (d *AnomalyDetector) GetBufferStats() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]int)
	for devID, types := range d.metrics {
		for metricType, buf := range types {
			key := devID + ":" + metricType
			stats[key] = buf.Len()
		}
	}
	return stats
}

// ─── Internal ─────────────────────────────────────────────────────────────

// getOrCreateBuffer возвращает или создаёт буфер для deviceID + metricType.
func (d *AnomalyDetector) getOrCreateBuffer(deviceID, metricType string) *MetricBuffer {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.metrics[deviceID] == nil {
		d.metrics[deviceID] = make(map[string]*MetricBuffer)
	}
	if d.metrics[deviceID][metricType] == nil {
		d.metrics[deviceID][metricType] = NewMetricBuffer(d.cfg.MetricBufferSize)
	}
	return d.metrics[deviceID][metricType]
}

// evaluate анализирует метрику на аномалию.
//
// Если разогрев не завершён (P2-MED-27), использует статические пороги
// вместо z-score, который ненадёжен на выборках < 30 элементов.
func (d *AnomalyDetector) evaluate(ctx context.Context, deviceID, metricType string, currentValue float64) *AnomalyResult {
	buf := d.getOrCreateBuffer(deviceID, metricType)
	values := buf.Values()

	var zScore, mean, stdDev float64
	var isAnomaly bool

	// P2-MED-27: Warm-up period — статические пороги до 30+ samples
	if !d.isWarmUpComplete(deviceID, metricType) {
		isAnomaly, mean, stdDev, zScore = d.checkStaticThreshold(deviceID, metricType, currentValue)
		if !isAnomaly {
			d.logger.DebugContext(ctx, "metric within static threshold (warm-up)",
				"device_id", deviceID,
				"metric_type", metricType,
				"value", currentValue,
				"samples", len(values),
			)
			return nil
		}
		d.logger.WarnContext(ctx, "metric exceeded static threshold (warm-up)",
			"device_id", deviceID,
			"metric_type", metricType,
			"value", currentValue,
			"samples", len(values),
		)
	} else {
		// Нормальный режим: z-score анализ
		// Берём только нужное окно для moving average
		window := d.cfg.MovingAverageWindow
		if window > len(values) {
			window = len(values)
		}
		windowValues := values[len(values)-window:]

		zScore, mean, stdDev = calculateZScore(currentValue, windowValues)

		// Проверяем порог
		if math.Abs(zScore) < d.cfg.ZScoreThreshold {
			return nil
		}
	}

	// Проверяем, нет ли уже активной аномалии для этого типа метрики
	d.mu.RLock()
	deviceAnomalies := d.activeAnomalies[deviceID]
	for _, existing := range deviceAnomalies {
		if existing.MetricType == metricType && existing.Status != AnomalyStatusResolved {
			// Обновляем существующую
			d.mu.RUnlock()
			d.updateExistingAnomaly(existing, currentValue, mean, stdDev, zScore)
			return nil
		}
	}
	d.mu.RUnlock()

	// Создаём новую аномалию
	severity := GetSeverityFromZScore(math.Abs(zScore))
	now := time.Now()

	result := &AnomalyResult{
		ID:           newAnomalyID(),
		DeviceID:     deviceID,
		MetricType:   metricType,
		CurrentValue: currentValue,
		MeanValue:    mean,
		StdDev:       stdDev,
		ZScore:       zScore,
		Severity:     severity,
		Status:       AnomalyStatusNew,
		Description:  buildDescription(deviceID, metricType, currentValue, mean, zScore, severity),
		DetectedAt:   now,
		TraceID:      traceIDFromContext(ctx),
	}

	d.mu.Lock()
	d.activeAnomalies[deviceID] = append(d.activeAnomalies[deviceID], result)
	d.anomalyIndex[result.ID] = result

	// Ограничиваем количество аномалий на устройство
	if len(d.activeAnomalies[deviceID]) > d.cfg.MaxAnomaliesPerDevice {
		// Удаляем самую старую
		oldest := d.activeAnomalies[deviceID][0]
		d.activeAnomalies[deviceID] = d.activeAnomalies[deviceID][1:]
		delete(d.anomalyIndex, oldest.ID)
	}
	d.mu.Unlock()

	d.logger.WarnContext(ctx, "anomaly detected",
		"device_id", deviceID,
		"metric_type", metricType,
		"z_score", fmt.Sprintf("%.2f", zScore),
		"current_value", fmt.Sprintf("%.2f", currentValue),
		"mean", fmt.Sprintf("%.2f", mean),
		"severity", severity,
		"anomaly_id", result.ID,
	)

	return result
}

// updateExistingAnomaly обновляет значения существующей аномалии.
func (d *AnomalyDetector) updateExistingAnomaly(a *AnomalyResult, currentValue, mean, stdDev, zScore float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	a.CurrentValue = currentValue
	a.MeanValue = mean
	a.StdDev = stdDev
	a.ZScore = zScore
	a.Severity = GetSeverityFromZScore(math.Abs(zScore))
	a.Description = buildDescription(a.DeviceID, a.MetricType, currentValue, mean, zScore, a.Severity)
}

// matchAnomaly проверяет, соответствует ли аномалия фильтрам.
func (d *AnomalyDetector) matchAnomaly(a *AnomalyResult, deviceID, metricType, severity, status string) bool {
	if deviceID != "" && a.DeviceID != deviceID {
		return false
	}
	if metricType != "" && a.MetricType != metricType {
		return false
	}
	if severity != "" && string(a.Severity) != severity {
		return false
	}
	if status != "" && string(a.Status) != status {
		return false
	}
	return true
}

// ─── Helpers ──────────────────────────────────────────────────────────────

// newAnomalyID генерирует уникальный ID для аномалии.
func newAnomalyID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "anm-" + hex.EncodeToString(b)
}

// traceIDFromContext извлекает traceID из контекста или генерирует новый.
func traceIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value("trace_id").(string)
	if !ok || id == "" {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		return hex.EncodeToString(b)
	}
	return id
}

// buildDescription формирует человекочитаемое описание аномалии.
func buildDescription(deviceID, metricType string, currentValue, mean, zScore float64, severity Severity) string {
	direction := "выше"
	if currentValue < mean {
		direction = "ниже"
	}

	metricNames := map[string]string{
		"heartbeat_latency": "Задержка heartbeat",
		"error_rate":        "Частота ошибок",
		"packet_loss":       "Потеря пакетов",
		"cpu_usage":         "Загрузка CPU",
		"memory_usage":      "Использование памяти",
		"disk_usage":        "Использование диска",
		"video_bitrate":     "Битрейт видео",
		"fps":               "FPS",
		"connection_jitter": "Джиттер соединения",
		"temperature":       "Температура",
	}

	metricName := metricType
	if name, ok := metricNames[metricType]; ok {
		metricName = name
	}

	return fmt.Sprintf("[%s] %s устройства %s: текущее значение %.1f %s среднего %.1f (z-score: %.2f)",
		severity, metricName, deviceID, currentValue, direction, mean, zScore,
	)
}

// ─── Ensure context interface ─────────────────────────────────────────────

// contextTraceID — ключ для traceID в контексте.
type contextTraceID struct{}

// WithTraceID добавляет traceID в контекст.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, contextTraceID{}, traceID)
}

// TraceIDFromContext извлекает traceID из контекста (re-export для удобства).
func TraceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextTraceID{}).(string)
	if id == "" {
		// Пробуем из стандартного middleware
		if stdID, ok := ctx.Value("trace_id").(string); ok {
			return stdID
		}
		return "unknown"
	}
	return id
}

// init — регистрируем контекстный ключ
func init() {
	_ = contextTraceID{}
}
