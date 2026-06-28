// Package ai — Anomaly Detection tests.
//
// P2-AI.4: Unit tests for anomaly detection:
//   - Z-score calculation
//   - Moving average
//   - Metric buffer
//   - Detector Feed/Evaluate
//   - Service integration
package ai

import (
	"context"
	"math"
	"testing"
	"time"
)

// ─── Z-Score Calculation ──────────────────────────────────────────────────

func TestCalculateZScore_NormalDistribution(t *testing.T) {
	// Нормальное распределение: mean=0, std=1
	values := []float64{-2, -1, 0, 1, 2}
	z, mean, std := calculateZScore(2, values)

	if math.Abs(mean-0) > 0.01 {
		t.Errorf("expected mean ~0, got %.4f", mean)
	}
	if math.Abs(std-1.4142) > 0.01 {
		t.Errorf("expected std ~1.4142, got %.4f", std)
	}
	// z-score для значения 2 в распределении [-2,-1,0,1,2]
	if math.Abs(z-1.4142) > 0.01 {
		t.Errorf("expected z ~1.4142, got %.4f", z)
	}
}

func TestCalculateZScore_Outlier(t *testing.T) {
	values := []float64{10, 12, 11, 13, 10, 12, 11}
	z, mean, std := calculateZScore(50, values)

	// Z-score должен быть значительно выше 3
	if math.Abs(z) < 3 {
		t.Errorf("expected |z| >= 3 for outlier, got %.4f", z)
	}
	_ = mean
	_ = std
}

func TestCalculateZScore_EmptyValues(t *testing.T) {
	z, mean, std := calculateZScore(10, []float64{})
	if z != 0 || mean != 0 || std != 0 {
		t.Errorf("expected zeros for empty values, got z=%.4f, mean=%.4f, std=%.4f", z, mean, std)
	}
}

func TestCalculateZScore_ZeroStdDev(t *testing.T) {
	// Все значения одинаковые — std=0
	values := []float64{5, 5, 5, 5, 5}
	z, mean, std := calculateZScore(5, values)

	if z != 0 {
		t.Errorf("expected z=0 when value == mean, got %.4f", z)
	}
	if mean != 5 {
		t.Errorf("expected mean=5, got %.4f", mean)
	}
	_ = std
}

func TestCalculateZScore_ZeroStdDevDifferentValue(t *testing.T) {
	values := []float64{5, 5, 5, 5, 5}
	z, _, _ := calculateZScore(10, values)

	// std=0 но value != mean → бесконечность
	if !math.IsInf(z, 1) {
		t.Errorf("expected +Inf for std=0 and different value, got %.4f", z)
	}
}

// ─── Moving Average ───────────────────────────────────────────────────────

func TestMovingAverage_Basic(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	window := 3

	ma := calculateMovingAverage(values, window)

	expected := []float64{2, 3, 4} // (1+2+3)/3=2, (2+3+4)/3=3, (3+4+5)/3=4
	if len(ma) != len(expected) {
		t.Fatalf("expected %d values, got %d", len(expected), len(ma))
	}
	for i, v := range expected {
		if ma[i] != v {
			t.Errorf("ma[%d] = %.2f, expected %.2f", i, ma[i], v)
		}
	}
}

func TestMovingAverage_WindowLargerThanData(t *testing.T) {
	values := []float64{1, 2, 3}
	ma := calculateMovingAverage(values, 10)

	if len(ma) != 1 {
		t.Fatalf("expected 1 value (full average), got %d", len(ma))
	}
	if ma[0] != 2 {
		t.Errorf("expected 2, got %.2f", ma[0])
	}
}

func TestMovingAverage_Empty(t *testing.T) {
	ma := calculateMovingAverage([]float64{}, 5)
	if ma != nil {
		t.Errorf("expected nil for empty input")
	}

	ma = calculateMovingAverage([]float64{1, 2}, 0)
	if ma != nil {
		t.Errorf("expected nil for zero window")
	}
}

// ─── Metric Buffer ────────────────────────────────────────────────────────

func TestMetricBuffer_PushAndLen(t *testing.T) {
	buf := NewMetricBuffer(10)

	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got len=%d", buf.Len())
	}

	buf.Push(DeviceMetricPoint{DeviceID: "dev1", MetricType: "cpu_usage", Value: 50})
	buf.Push(DeviceMetricPoint{DeviceID: "dev1", MetricType: "cpu_usage", Value: 60})

	if buf.Len() != 2 {
		t.Errorf("expected len=2, got %d", buf.Len())
	}
}

func TestMetricBuffer_FIFOOverflow(t *testing.T) {
	buf := NewMetricBuffer(5)

	for i := 0; i < 10; i++ {
		buf.Push(DeviceMetricPoint{DeviceID: "dev1", MetricType: "cpu_usage", Value: float64(i)})
	}

	// После переполнения должно остаться 5 элементов (последние)
	if buf.Len() != 5 {
		t.Errorf("expected len=5 after overflow, got %d", buf.Len())
	}

	values := buf.Values()
	if values[0] != 5 {
		t.Errorf("expected first value 5 after overflow, got %.0f", values[0])
	}
}

func TestMetricBuffer_Values(t *testing.T) {
	buf := NewMetricBuffer(10)
	buf.Push(DeviceMetricPoint{Value: 1.5})
	buf.Push(DeviceMetricPoint{Value: 2.5})

	values := buf.Values()
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if values[0] != 1.5 || values[1] != 2.5 {
		t.Errorf("values mismatch: %v", values)
	}
}

func TestMetricBuffer_Clear(t *testing.T) {
	buf := NewMetricBuffer(10)
	buf.Push(DeviceMetricPoint{Value: 1})
	buf.Clear()

	if buf.Len() != 0 {
		t.Errorf("expected len=0 after clear, got %d", buf.Len())
	}
}

func TestMetricBuffer_ConcurrentSafety(t *testing.T) {
	buf := NewMetricBuffer(100)
	done := make(chan bool)

	// Параллельная запись
	go func() {
		for i := 0; i < 50; i++ {
			buf.Push(DeviceMetricPoint{Value: float64(i)})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			buf.Push(DeviceMetricPoint{Value: float64(i * 2)})
		}
		done <- true
	}()

	// Параллельное чтение
	go func() {
		for i := 0; i < 20; i++ {
			_ = buf.Values()
			_ = buf.Len()
			_ = buf.Points()
		}
		done <- true
	}()

	// Ждём все горутины
	for i := 0; i < 3; i++ {
		<-done
	}

	if buf.Len() == 0 {
		t.Error("buffer should not be empty after concurrent operations")
	}
}

// ─── Anomaly Detector ─────────────────────────────────────────────────────

func TestDetector_Feed_NotEnoughData(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.MinDataPoints = 5
	detector := NewAnomalyDetector(cfg, nil)

	// Меньше чем MinDataPoints — не должно быть аномалий
	for i := 0; i < 4; i++ {
		m := DeviceMetricPoint{
			DeviceID:   "dev-1",
			MetricType: "cpu_usage",
			Value:      50,
			Timestamp:  time.Now(),
		}
		result := detector.Feed(context.Background(), m)
		if result != nil {
			t.Fatalf("expected no anomaly with < min data points, got anomaly at point %d", i)
		}
	}
}

func TestDetector_Feed_DetectsAnomaly(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.ZScoreThreshold = 2.0
	cfg.MinDataPoints = 5
	detector := NewAnomalyDetector(cfg, nil)

	ctx := context.Background()

	// Нормальные значения
	for i := 0; i < 10; i++ {
		m := DeviceMetricPoint{
			DeviceID:   "dev-1",
			MetricType: "cpu_usage",
			Value:      50,
			Timestamp:  time.Now(),
		}
		detector.Feed(ctx, m)
	}

	// Аномалия — значение 100 при норме 50
	anomaly := detector.Feed(ctx, DeviceMetricPoint{
		DeviceID:   "dev-1",
		MetricType: "cpu_usage",
		Value:      100,
		Timestamp:  time.Now(),
	})

	if anomaly == nil {
		t.Fatal("expected anomaly to be detected")
	}
	if anomaly.DeviceID != "dev-1" {
		t.Errorf("expected device dev-1, got %s", anomaly.DeviceID)
	}
	if anomaly.MetricType != "cpu_usage" {
		t.Errorf("expected metric cpu_usage, got %s", anomaly.MetricType)
	}
	if anomaly.Status != AnomalyStatusNew {
		t.Errorf("expected status new, got %s", anomaly.Status)
	}
}

func TestDetector_GetActiveAnomalies(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.ZScoreThreshold = 1.5
	cfg.MinDataPoints = 3
	detector := NewAnomalyDetector(cfg, nil)

	ctx := context.Background()

	// Норма для dev-1
	for i := 0; i < 5; i++ {
		detector.Feed(ctx, DeviceMetricPoint{
			DeviceID: "dev-1", MetricType: "cpu_usage", Value: 50,
		})
		detector.Feed(ctx, DeviceMetricPoint{
			DeviceID: "dev-2", MetricType: "cpu_usage", Value: 30,
		})
	}

	// Аномалии
	detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 95})
	detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-2", MetricType: "cpu_usage", Value: 80})

	active := detector.GetActiveAnomalies("", "", "")
	if len(active) != 2 {
		t.Fatalf("expected 2 active anomalies, got %d", len(active))
	}

	// Фильтр по устройству
	dev1 := detector.GetActiveAnomalies("dev-1", "", "")
	if len(dev1) != 1 {
		t.Fatalf("expected 1 anomaly for dev-1, got %d", len(dev1))
	}
}

func TestDetector_GetAllAnomalies(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.ZScoreThreshold = 1.5
	cfg.MinDataPoints = 3
	detector := NewAnomalyDetector(cfg, nil)

	ctx := context.Background()

	// Норма
	for i := 0; i < 5; i++ {
		detector.Feed(ctx, DeviceMetricPoint{
			DeviceID: "dev-1", MetricType: "cpu_usage", Value: 50,
		})
	}

	// Создаём аномалию
	anomaly := detector.Feed(ctx, DeviceMetricPoint{
		DeviceID: "dev-1", MetricType: "cpu_usage", Value: 95,
	})
	if anomaly == nil {
		t.Fatal("expected anomaly")
	}

	// Resolve
	if err := detector.ResolveAnomaly(anomaly.ID); err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	// Проверяем, что resolved аномалия всё ещё возвращается
	all := detector.GetAllAnomalies("", "", "", "", 0)
	if len(all) != 1 {
		t.Fatalf("expected 1 anomaly total, got %d", len(all))
	}

	// Но не в active
	active := detector.GetActiveAnomalies("", "", "")
	if len(active) != 0 {
		t.Fatalf("expected 0 active anomalies, got %d", len(active))
	}
}

func TestDetector_AcknowledgeAndResolve(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.ZScoreThreshold = 1.5
	cfg.MinDataPoints = 3
	detector := NewAnomalyDetector(cfg, nil)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		detector.Feed(ctx, DeviceMetricPoint{
			DeviceID: "dev-1", MetricType: "cpu_usage", Value: 50,
		})
	}

	anomaly := detector.Feed(ctx, DeviceMetricPoint{
		DeviceID: "dev-1", MetricType: "cpu_usage", Value: 95,
	})
	if anomaly == nil {
		t.Fatal("expected anomaly")
	}

	// Acknowledge
	if err := detector.AcknowledgeAnomaly(anomaly.ID); err != nil {
		t.Fatalf("acknowledge failed: %v", err)
	}

	updated := detector.GetAllAnomalies("", "", "", "", 0)
	if len(updated) != 1 || updated[0].Status != AnomalyStatusAcknowledged {
		t.Errorf("expected acknowledged status, got %s", updated[0].Status)
	}

	// Resolve
	if err := detector.ResolveAnomaly(anomaly.ID); err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	resolved := detector.GetAllAnomalies("", "", "", "resolved", 0)
	if len(resolved) != 1 {
		t.Errorf("expected 1 resolved anomaly, got %d", len(resolved))
	}
}

func TestDetector_UnknownID(t *testing.T) {
	detector := NewAnomalyDetector(DefaultAnomalyConfig(), nil)

	if err := detector.AcknowledgeAnomaly("nonexistent"); err == nil {
		t.Error("expected error for unknown anomaly")
	}

	if err := detector.ResolveAnomaly("nonexistent"); err == nil {
		t.Error("expected error for unknown anomaly")
	}
}

func TestDetector_MultipleMetricsPerDevice(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.ZScoreThreshold = 2.0
	cfg.MinDataPoints = 3
	detector := NewAnomalyDetector(cfg, nil)

	ctx := context.Background()

	// Норма для cpu и memory
	for i := 0; i < 5; i++ {
		detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 50})
		detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "memory_usage", Value: 60})
	}

	// Аномалия cpu
	cpuAnomaly := detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 95})
	if cpuAnomaly == nil {
		t.Fatal("expected cpu anomaly")
	}

	// Аномалия memory
	memAnomaly := detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "memory_usage", Value: 90})
	if memAnomaly == nil {
		t.Fatal("expected memory anomaly")
	}

	// Проверяем фильтр по типу метрики
	cpuOnly := detector.GetActiveAnomalies("", "cpu_usage", "")
	if len(cpuOnly) != 1 {
		t.Fatalf("expected 1 cpu anomaly, got %d", len(cpuOnly))
	}
	if cpuOnly[0].MetricType != "cpu_usage" {
		t.Errorf("expected cpu_usage, got %s", cpuOnly[0].MetricType)
	}

	_ = memAnomaly
}

func TestDetector_SeverityLevels(t *testing.T) {
	tests := []struct {
		zScore   float64
		expected Severity
	}{
		{3.5, SeverityLow},
		{4.5, SeverityMedium},
		{5.5, SeverityHigh},
		{7.0, SeverityCritical},
		{2.0, SeverityLow}, // ниже порога
	}

	for _, tt := range tests {
		result := GetSeverityFromZScore(tt.zScore)
		if result != tt.expected {
			t.Errorf("GetSeverityFromZScore(%.1f) = %s, expected %s", tt.zScore, result, tt.expected)
		}
	}
}

func TestDetector_MaxAnomaliesPerDevice(t *testing.T) {
	cfg := DefaultAnomalyConfig()
	cfg.ZScoreThreshold = 1.5
	cfg.MinDataPoints = 2
	cfg.MaxAnomaliesPerDevice = 3
	detector := NewAnomalyDetector(cfg, nil)

	ctx := context.Background()

	// Норма
	for i := 0; i < 3; i++ {
		detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 50})
	}

	// Создаём 4 аномалии разных метрик (но dev-1 max=3)
	for i := 0; i < 5; i++ {
		detector.Feed(ctx, DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 100})
	}

	// Должно быть не более 3 аномалий на dev-1
	anomalies := detector.GetActiveAnomalies("dev-1", "", "")
	if len(anomalies) > 3 {
		t.Errorf("expected max 3 anomalies per device, got %d", len(anomalies))
	}
}

// ─── GetBufferStats ───────────────────────────────────────────────────────

func TestDetector_GetBufferStats(t *testing.T) {
	detector := NewAnomalyDetector(DefaultAnomalyConfig(), nil)

	detector.Feed(context.Background(), DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 50})
	detector.Feed(context.Background(), DeviceMetricPoint{DeviceID: "dev-1", MetricType: "cpu_usage", Value: 60})
	detector.Feed(context.Background(), DeviceMetricPoint{DeviceID: "dev-2", MetricType: "memory_usage", Value: 70})

	stats := detector.GetBufferStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 buffer stats entries, got %d", len(stats))
	}
	if stats["dev-1:cpu_usage"] != 2 {
		t.Errorf("expected 2 points for dev-1:cpu_usage, got %d", stats["dev-1:cpu_usage"])
	}
}
