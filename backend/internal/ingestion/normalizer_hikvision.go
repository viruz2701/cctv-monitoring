// Package ingestion — Unified Ingestion Layer.
//
// ═══════════════════════════════════════════════════════════════════════
// Hikvision Vendor Normalizer
//
// Hikvision использует ISAPI протокол с JSON форматом событий.
// Типы событий: motion, tamper, videoLoss, diskFull, temperature, etc.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5.1: Input validation
//
// ═══════════════════════════════════════════════════════════════════════
package ingestion

import (
	"encoding/json"
	"fmt"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ── Hikvision Types ────────────────────────────────────────────────────

// HikvisionEvent — структура Hikvision-формата событий.
type HikvisionEvent struct {
	EventType     string  `json:"eventType"`
	EventState    string  `json:"eventState"`   // active, inactive
	EventTrigger  string  `json:"eventTrigger"` // timed, alarm, manual
	ChannelID     int     `json:"channelID"`
	DateTime      string  `json:"dateTime"`
	PicName       string  `json:"picName,omitempty"`
	EventPriority int     `json:"eventPriority,omitempty"`
	Description   string  `json:"description,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	CPUUsage      float64 `json:"cpuUsage,omitempty"`
	MemoryUsage   float64 `json:"memoryUsage,omitempty"`
	NetworkUsage  float64 `json:"networkUsage,omitempty"`
	DiskUsage     float64 `json:"diskUsage,omitempty"`
}

// ── Normalizer ─────────────────────────────────────────────────────────

// HikvisionNormalizer implements VendorHandler for Hikvision.
type HikvisionNormalizer struct{}

func (h *HikvisionNormalizer) Normalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var hik HikvisionEvent
	if err := json.Unmarshal(payload, &hik); err != nil {
		return nil, fmt.Errorf("hikvision unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       hik.EventType,
		Timestamp:  parseTimeOrDefault(hik.DateTime, time.Now()),
		Source:     "hikvision",
		Message:    hik.Description,
		ImageURL:   hik.PicName,
		RawPayload: string(payload),
		Metadata: map[string]string{
			"event_state":   hik.EventState,
			"event_trigger": hik.EventTrigger,
			"channel_id":    fmt.Sprintf("%d", hik.ChannelID),
		},
		Metrics: buildHikMetrics(hik),
		Tags: map[string]string{
			"vendor": "hikvision",
			"type":   dataType,
		},
	}

	event.Severity = mapHikPriority(hik.EventPriority)

	if hik.EventState == "active" {
		event.Severity = "high"
	}

	return event, nil
}

// buildHikMetrics собирает метрики из HikvisionEvent.
func buildHikMetrics(hik HikvisionEvent) []models.Metric {
	var metrics []models.Metric
	if hik.Temperature > 0 {
		metrics = append(metrics, models.Metric{Name: "temperature", Value: hik.Temperature, Unit: "celsius"})
	}
	if hik.CPUUsage > 0 {
		metrics = append(metrics, models.Metric{Name: "cpu_usage", Value: hik.CPUUsage, Unit: "percent"})
	}
	if hik.MemoryUsage > 0 {
		metrics = append(metrics, models.Metric{Name: "memory_usage", Value: hik.MemoryUsage, Unit: "percent"})
	}
	if hik.NetworkUsage > 0 {
		metrics = append(metrics, models.Metric{Name: "network_usage", Value: hik.NetworkUsage, Unit: "bps"})
	}
	if hik.DiskUsage > 0 {
		metrics = append(metrics, models.Metric{Name: "disk_usage", Value: hik.DiskUsage, Unit: "percent"})
	}
	return metrics
}

// mapHikPriority маппит Hikvision eventPriority в severity string.
func mapHikPriority(priority int) string {
	switch priority {
	case 1:
		return "low"
	case 2:
		return "medium"
	case 3, 4, 5:
		return "high"
	default:
		return "low"
	}
}
