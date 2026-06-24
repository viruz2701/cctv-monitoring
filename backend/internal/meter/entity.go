// Package meter — CCTV Meter entity + TimescaleDB hypertable для метрик.
//
// AH-5.3.1: Meter entity — определяет тип метрики и её параметры.
// AH-5.3.2: Reading — TimescaleDB hypertable для хранения значений метрик.
//
// CCTV-метры:
//   - bitrate: текущий битрейт видео (kbps)
//   - fps: частота кадров
//   - cpu_temp: температура процессора камеры (°C)
//   - cpu_usage: загрузка CPU (%)
//   - memory_usage: использование памяти (%)
//   - error_count: количество ошибок за период
//   - offline_ratio: процент недоступности
//   - packet_loss: потеря пакетов (%)
//   - signal_strength: уровень сигнала WiFi (dBm)
//   - disk_usage: заполнение диска NVR (%)
//   - recording_duration: длительность записи (часы)
//   - motion_events: количество детекций движения
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — мониторинг метрик)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - Приказ ОАЦ №66 п. 7.18.3 (Edge device monitoring)
package meter

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// AH-5.3.1: Meter entity
// ═══════════════════════════════════════════════════════════════════════

// MeterKind — тип метрики CCTV.
type MeterKind string

const (
	MeterBitrate         MeterKind = "bitrate"          // kbps
	MeterFPS             MeterKind = "fps"              // frames per second
	MeterCPUTemp         MeterKind = "cpu_temp"         // °C
	MeterCPUUsage        MeterKind = "cpu_usage"        // %
	MeterMemoryUsage     MeterKind = "memory_usage"     // %
	MeterErrorCount      MeterKind = "error_count"      // count
	MeterOfflineRatio    MeterKind = "offline_ratio"    // % (24h rolling)
	MeterPacketLoss      MeterKind = "packet_loss"      // %
	MeterSignalStrength  MeterKind = "signal_strength"  // dBm
	MeterDiskUsage       MeterKind = "disk_usage"       // %
	MeterRecordingDuration MeterKind = "recording_duration" // hours
	MeterMotionEvents    MeterKind = "motion_events"    // count/period
)

// ValidMeterKinds для whitelist validation (OWASP ASVS V5.1).
var ValidMeterKinds = []string{
	string(MeterBitrate), string(MeterFPS), string(MeterCPUTemp),
	string(MeterCPUUsage), string(MeterMemoryUsage), string(MeterErrorCount),
	string(MeterOfflineRatio), string(MeterPacketLoss), string(MeterSignalStrength),
	string(MeterDiskUsage), string(MeterRecordingDuration), string(MeterMotionEvents),
}

// MeterUnit — единица измерения.
type MeterUnit string

const (
	UnitKbps       MeterUnit = "kbps"
	UnitFPS        MeterUnit = "fps"
	UnitCelsius    MeterUnit = "celsius"
	UnitPercent    MeterUnit = "percent"
	UnitCount      MeterUnit = "count"
	UnitDBm        MeterUnit = "dBm"
	UnitHours      MeterUnit = "hours"
	UnitRatio      MeterUnit = "ratio"
)

// MeterThreshold — пороговое значение метрики для алерта.
type MeterThreshold struct {
	Warning  float64 `json:"warning"`  // жёлтый уровень
	Critical float64 `json:"critical"` // красный уровень
	Min      float64 `json:"min"`      // минимальное допустимое
	Max      float64 `json:"max"`      // максимальное допустимое
}

// Meter — метрика CCTV-устройства.
//
// Определяет тип измерения, единицы, пороги и частоту сбора.
type Meter struct {
	ID             string         `json:"id" db:"id"`
	DeviceID       string         `json:"device_id" db:"device_id" validate:"required"`
	Kind           MeterKind      `json:"kind" db:"kind" validate:"required"`
	Name           string         `json:"name" db:"name" validate:"required,max=100"`
	Unit           MeterUnit      `json:"unit" db:"unit"`
	Interval       int            `json:"interval_seconds" db:"interval_seconds"` // частота сбора (сек)
	RetentionDays  int            `json:"retention_days" db:"retention_days"`     // срок хранения
	Thresholds     MeterThreshold `json:"thresholds" db:"thresholds"`
	Enabled        bool           `json:"enabled" db:"enabled"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// DefaultMeters возвращает стандартный набор метрик для CCTV-камеры.
func DefaultMeters(deviceID string) []*Meter {
	defaults := []struct {
		kind           MeterKind
		name           string
		unit           MeterUnit
		interval       int
		retention      int
		warn, crit, min, max float64
	}{
		{MeterBitrate, "Video Bitrate", UnitKbps, 60, 90, 8000, 12000, 100, 0},
		{MeterFPS, "Frame Rate", UnitFPS, 60, 90, 15, 10, 5, 30},
		{MeterCPUTemp, "CPU Temperature", UnitCelsius, 300, 90, 75, 85, -20, 100},
		{MeterCPUUsage, "CPU Usage", UnitPercent, 300, 90, 70, 90, 0, 100},
		{MeterMemoryUsage, "Memory Usage", UnitPercent, 300, 90, 80, 95, 0, 100},
		{MeterErrorCount, "Error Count", UnitCount, 60, 30, 10, 50, 0, 0},
		{MeterOfflineRatio, "Offline Ratio (24h)", UnitPercent, 3600, 30, 5, 15, 0, 100},
		{MeterPacketLoss, "Packet Loss", UnitPercent, 60, 30, 2, 5, 0, 100},
		{MeterDiskUsage, "NVR Disk Usage", UnitPercent, 600, 90, 80, 95, 0, 100},
		{MeterMotionEvents, "Motion Events", UnitCount, 3600, 30, 50, 100, 0, 0},
	}

	meters := make([]*Meter, 0, len(defaults))
	for _, d := range defaults {
		meters = append(meters, &Meter{
			DeviceID:      deviceID,
			Kind:          d.kind,
			Name:          d.name,
			Unit:          d.unit,
			Interval:      d.interval,
			RetentionDays: d.retention,
			Thresholds: MeterThreshold{
				Warning:  d.warn,
				Critical: d.crit,
				Min:      d.min,
				Max:      d.max,
			},
			Enabled: true,
		})
	}
	return meters
}

// ═══════════════════════════════════════════════════════════════════════
// AH-5.3.2: Reading (TimescaleDB hypertable)
// ═══════════════════════════════════════════════════════════════════════

// Reading — одно значение метрики в момент времени.
//
// Хранится в TimescaleDB hypertable для эффективной работы с time-series.
// Retention: настраиваемый (1-12 месяцев), через drop_chunks.
type Reading struct {
	Time        time.Time `json:"time" db:"time"`                  // время измерения ( hypertable column)
	MeterID     string    `json:"meter_id" db:"meter_id"`           // ссылка на Meter.ID
	DeviceID    string    `json:"device_id" db:"device_id"`         // denormalized для фильтрации
	Kind        MeterKind `json:"kind" db:"kind"`                  // тип метрики (denormalized)
	Value       float64   `json:"value" db:"value"`                // значение
	Tags        []ReadingTag `json:"tags,omitempty" db:"tags"`     // дополнительные теги
}

// ReadingTag — тег для метрики.
type ReadingTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ReadingStats — агрегированная статистика по метрике за период.
type ReadingStats struct {
	Kind      MeterKind `json:"kind"`
	MeterID   string    `json:"meter_id"`
	DeviceID  string    `json:"device_id"`
	Min       float64   `json:"min"`
	Max       float64   `json:"max"`
	Avg       float64   `json:"avg"`
	Median    float64   `json:"median"`
	P95       float64   `json:"p95"`
	P99       float64   `json:"p99"`
	Count     int       `json:"count"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// IsWithinThreshold проверяет, находится ли значение в пределах порогов.
//
// Для метрик где higher=worse (temp, disk): critical > warning
// Для метрик где lower=worse (fps): critical < warning
//
// Возвращает: "ok", "warning", "critical"
func IsWithinThreshold(value float64, threshold MeterThreshold) string {
	// Проверка выхода за абсолютные границы
	if value < threshold.Min || value > threshold.Max {
		return "critical"
	}

	// Higher-is-worse: critical > warning (например, cpu_temp: 85 > 75)
	if threshold.Critical > threshold.Warning {
		if value >= threshold.Critical {
			return "critical"
		}
		if value >= threshold.Warning {
			return "warning"
		}
		return "ok"
	}

	// Lower-is-worse: critical < warning (например, fps: 10 < 15)
	if value <= threshold.Critical {
		return "critical"
	}
	if value <= threshold.Warning {
		return "warning"
	}
	return "ok"
}

// ValidateMeterKind проверяет тип метрики.
func ValidateMeterKind(kind string) bool {
	for _, v := range ValidMeterKinds {
		if kind == v {
			return true
		}
	}
	return false
}
