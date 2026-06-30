// Package ingestion — Unified Ingestion Layer.
//
// ═══════════════════════════════════════════════════════════════════════
// Tiandy Vendor Normalizer
//
// Tiandy использует HTTP API с JSON форматом.
// Типы событий: alarm, motion, temperature, cpuRate, etc.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5.1: Input validation
// ═══════════════════════════════════════════════════════════════════════
package ingestion

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ── Tiandy Types ───────────────────────────────────────────────────────

// TiandyEvent — структура Tiandy-формата событий.
type TiandyEvent struct {
	EventName   string  `json:"eventName"`
	Channel     int     `json:"channel"`
	Status      string  `json:"status"`           // alarm, normal
	StartTime   string  `json:"startTime,omitempty"`
	EndTime     string  `json:"endTime,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	DeviceTemp  float64 `json:"deviceTemp,omitempty"`
	CPURate     float64 `json:"cpuRate,omitempty"`
	MemoryRate  float64 `json:"memoryRate,omitempty"`
	NetSpeed    float64 `json:"netSpeed,omitempty"`
	Description string  `json:"description,omitempty"`
}

// ── Normalizer ─────────────────────────────────────────────────────────

// TiandyNormalizer implements VendorHandler for Tiandy.
type TiandyNormalizer struct{}

func (t *TiandyNormalizer) Normalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var tiandy TiandyEvent
	if err := json.Unmarshal(payload, &tiandy); err != nil {
		return nil, fmt.Errorf("tiandy unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       strings.ToLower(tiandy.EventName),
		Timestamp:  parseTimeOrDefault(tiandy.StartTime, time.Now()),
		Source:     "tiandy",
		Message:    tiandy.Description,
		RawPayload: string(payload),
		Metadata: map[string]string{
			"channel": fmt.Sprintf("%d", tiandy.Channel),
			"status":  tiandy.Status,
		},
		Tags: map[string]string{
			"vendor": "tiandy",
			"type":   dataType,
		},
	}

	// Tiandy severity
	switch strings.ToLower(tiandy.Status) {
	case "alarm":
		event.Severity = "high"
	default:
		event.Severity = "info"
	}

	// Метрики
	if tiandy.Temperature > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "temperature", Value: tiandy.Temperature, Unit: "celsius"})
	}
	if tiandy.DeviceTemp > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "device_temp", Value: tiandy.DeviceTemp, Unit: "celsius"})
	}
	if tiandy.CPURate > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "cpu_usage", Value: tiandy.CPURate, Unit: "percent"})
	}
	if tiandy.MemoryRate > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "memory_usage", Value: tiandy.MemoryRate, Unit: "percent"})
	}
	if tiandy.NetSpeed > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "network_speed", Value: tiandy.NetSpeed, Unit: "bps"})
	}

	return event, nil
}
