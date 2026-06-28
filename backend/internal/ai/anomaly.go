// Package ai — Anomaly Detection Engine для CCTV устройств.
//
// P2-AI.4: Anomaly Detection
//   - Сбор метрик устройств (heartbeat, ошибки, лаги)
//   - Статистические методы (z-score, moving average)
//   - API endpoint GET /api/v1/ai/anomalies
//   - WebSocket уведомления при обнаружении
//
// Compliance:
//   - IEC 62443 SR 3.3 (Security monitoring — anomaly detection)
//   - ISO 27001 A.12.4.1 (Event logging — anomaly events)
//   - ISO 27001 A.12.6.1 (Capacity management — metric trends)
//   - СТБ 34.101.27 п. 7.3 (Анализ защищённости)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
package ai

import (
	"time"
)

// ─── Конфигурация ─────────────────────────────────────────────────────────

// AnomalyConfig — конфигурация движка обнаружения аномалий.
type AnomalyConfig struct {
	// ZScoreThreshold — порог z-score для классификации аномалии (default: 3.0).
	ZScoreThreshold float64 `mapstructure:"z_score_threshold"`

	// MovingAverageWindow — размер окна скользящего среднего (default: 10).
	MovingAverageWindow int `mapstructure:"moving_average_window"`

	// MinDataPoints — минимальное количество точек данных для анализа (default: 5).
	MinDataPoints int `mapstructure:"min_data_points"`

	// MetricBufferSize — размер буфера метрик на устройство (default: 1000).
	MetricBufferSize int `mapstructure:"metric_buffer_size"`

	// EvaluationInterval — интервал автоматической оценки (default: "5m").
	EvaluationInterval string `mapstructure:"evaluation_interval"`

	// NATSTopicPrefix — префикс NATS топика для событий (default: "ai.anomaly").
	NATSTopicPrefix string `mapstructure:"nats_topic_prefix"`

	// AnomalyRetentionHours — время хранения аномалий в памяти (default: 168 = 7 дней).
	AnomalyRetentionHours int `mapstructure:"anomaly_retention_hours"`

	// MaxAnomaliesPerDevice — максимум активных аномалий на устройство (default: 50).
	MaxAnomaliesPerDevice int `mapstructure:"max_anomalies_per_device"`
}

// DefaultAnomalyConfig возвращает конфигурацию по умолчанию.
func DefaultAnomalyConfig() AnomalyConfig {
	return AnomalyConfig{
		ZScoreThreshold:       3.0,
		MovingAverageWindow:   10,
		MinDataPoints:         5,
		MetricBufferSize:      1000,
		EvaluationInterval:    "5m",
		NATSTopicPrefix:       "ai.anomaly",
		AnomalyRetentionHours: 168, // 7 days
		MaxAnomaliesPerDevice: 50,
	}
}

// ─── Типы метрик ──────────────────────────────────────────────────────────

// MetricType — тип метрики устройства.
type MetricType string

const (
	MetricHeartbeatLatency MetricType = "heartbeat_latency"
	MetricErrorRate        MetricType = "error_rate"
	MetricPacketLoss       MetricType = "packet_loss"
	MetricCPUUsage         MetricType = "cpu_usage"
	MetricMemoryUsage      MetricType = "memory_usage"
	MetricDiskUsage        MetricType = "disk_usage"
	MetricVideoBitrate     MetricType = "video_bitrate"
	MetricFPS              MetricType = "fps"
	MetricConnectionJitter MetricType = "connection_jitter"
	MetricTemperature      MetricType = "temperature"
)

// ValidMetricTypes — список допустимых типов метрик (OWASP ASVS V5.1 whitelist).
var ValidMetricTypes = []string{
	string(MetricHeartbeatLatency),
	string(MetricErrorRate),
	string(MetricPacketLoss),
	string(MetricCPUUsage),
	string(MetricMemoryUsage),
	string(MetricDiskUsage),
	string(MetricVideoBitrate),
	string(MetricFPS),
	string(MetricConnectionJitter),
	string(MetricTemperature),
}

// ─── DeviceMetricPoint ────────────────────────────────────────────────────

// DeviceMetricPoint — одна точка метрики устройства.
type DeviceMetricPoint struct {
	DeviceID   string    `json:"device_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

// ─── Severity ─────────────────────────────────────────────────────────────

// Severity — уровень серьёзности аномалии.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ValidSeverities — whitelist для валидации.
var ValidSeverities = []string{
	string(SeverityLow),
	string(SeverityMedium),
	string(SeverityHigh),
	string(SeverityCritical),
}

// ─── AnomalyStatus ────────────────────────────────────────────────────────

// AnomalyStatus — статус аномалии.
type AnomalyStatus string

const (
	AnomalyStatusNew          AnomalyStatus = "new"
	AnomalyStatusAcknowledged AnomalyStatus = "acknowledged"
	AnomalyStatusResolved     AnomalyStatus = "resolved"
)

// ValidAnomalyStatuses — whitelist для валидации.
var ValidAnomalyStatuses = []string{
	string(AnomalyStatusNew),
	string(AnomalyStatusAcknowledged),
	string(AnomalyStatusResolved),
}

// ─── AnomalyResult ────────────────────────────────────────────────────────

// AnomalyResult — результат обнаружения аномалии.
type AnomalyResult struct {
	ID           string        `json:"id"`
	DeviceID     string        `json:"device_id"`
	MetricType   string        `json:"metric_type"`
	CurrentValue float64       `json:"current_value"`
	MeanValue    float64       `json:"mean_value"`
	StdDev       float64       `json:"std_dev"`
	ZScore       float64       `json:"z_score"`
	Severity     Severity      `json:"severity"`
	Status       AnomalyStatus `json:"status"`
	Description  string        `json:"description"`
	DetectedAt   time.Time     `json:"detected_at"`
	ResolvedAt   *time.Time    `json:"resolved_at,omitempty"`
	TraceID      string        `json:"trace_id"`
}

// ─── AnomalyEvent ─────────────────────────────────────────────────────────

// AnomalyEventType — тип события аномалии.
type AnomalyEventType string

const (
	AnomalyEventDetected AnomalyEventType = "anomaly_detected"
	AnomalyEventResolved AnomalyEventType = "anomaly_resolved"
)

// AnomalyEvent — событие аномалии для NATS / WebSocket.
type AnomalyEvent struct {
	Type    AnomalyEventType `json:"type"`
	Payload AnomalyResult    `json:"payload"`
}

// ─── GetSeverityFromZScore ────────────────────────────────────────────────

// GetSeverityFromZScore определяет уровень серьёзности по z-score.
//
// Пороги:
//   - 3.0–4.0: low
//   - 4.0–5.0: medium
//   - 5.0–6.0: high
//   - > 6.0:   critical
func GetSeverityFromZScore(zScore float64) Severity {
	switch {
	case zScore >= 6.0:
		return SeverityCritical
	case zScore >= 5.0:
		return SeverityHigh
	case zScore >= 4.0:
		return SeverityMedium
	case zScore >= 3.0:
		return SeverityLow
	default:
		return SeverityLow
	}
}
