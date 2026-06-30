// Package ingestion — Unified Ingestion Layer.
//
// ═══════════════════════════════════════════════════════════════════════
// Uniview Vendor Normalizer
//
// Uniview использует HTTP API с JSON форматом.
// Типы событий: eventCode-based с метриками cpuLoad, memLoad, netLoad.
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

// ── Uniview Types ──────────────────────────────────────────────────────

// UniviewEvent — структура Uniview-формата событий.
type UniviewEvent struct {
	EventCode   string  `json:"eventCode"`
	EventDesc   string  `json:"eventDesc"`
	ChannelID   int     `json:"channelId"`
	AlarmInput  int     `json:"alarmInput,omitempty"`
	StartTime   string  `json:"startTime"`
	EndTime     string  `json:"endTime,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	CPULoad     float64 `json:"cpuLoad,omitempty"`
	MemLoad     float64 `json:"memLoad,omitempty"`
	NetLoad     float64 `json:"netLoad,omitempty"`
	DiskLoad    float64 `json:"diskLoad,omitempty"`
}

// ── Normalizer ─────────────────────────────────────────────────────────

// UniviewNormalizer implements VendorHandler for Uniview.
type UniviewNormalizer struct{}

func (u *UniviewNormalizer) Normalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var uniview UniviewEvent
	if err := json.Unmarshal(payload, &uniview); err != nil {
		return nil, fmt.Errorf("uniview unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       strings.ToLower(uniview.EventCode),
		Timestamp:  parseTimeOrDefault(uniview.StartTime, time.Now()),
		Source:     "uniview",
		Message:    uniview.EventDesc,
		RawPayload: string(payload),
		Metadata: map[string]string{
			"channel_id":  fmt.Sprintf("%d", uniview.ChannelID),
			"alarm_input": fmt.Sprintf("%d", uniview.AlarmInput),
		},
		Tags: map[string]string{
			"vendor": "uniview",
			"type":   dataType,
		},
		Severity: "medium",
	}

	// Метрики
	if uniview.Temperature > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "temperature", Value: uniview.Temperature, Unit: "celsius"})
	}
	if uniview.CPULoad > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "cpu_usage", Value: uniview.CPULoad, Unit: "percent"})
	}
	if uniview.MemLoad > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "memory_usage", Value: uniview.MemLoad, Unit: "percent"})
	}
	if uniview.NetLoad > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "network_load", Value: uniview.NetLoad, Unit: "bps"})
	}
	if uniview.DiskLoad > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "disk_usage", Value: uniview.DiskLoad, Unit: "percent"})
	}

	return event, nil
}
