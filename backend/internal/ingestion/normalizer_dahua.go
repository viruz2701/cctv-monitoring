// Package ingestion — Unified Ingestion Layer.
//
// ═══════════════════════════════════════════════════════════════════════
// Dahua Vendor Normalizer
//
// Dahua использует HTTP API с JSON форматом событий.
// Типы событий: VideoMotion, VideoLoss, Temperature, etc.
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

// ── Dahua Types ────────────────────────────────────────────────────────

// DahuaEvent — структура Dahua-формата событий.
type DahuaEvent struct {
	Code      string  `json:"code"`  // VideoMotion, VideoLoss, etc.
	Action    string  `json:"action"` // Start, Stop, Pulse
	Index     int     `json:"index"`
	Data      struct {
		Temperature float64 `json:"temperature,omitempty"`
		CPUUsage    float64 `json:"cpuUsage,omitempty"`
		NetRate     float64 `json:"netRate,omitempty"`
	} `json:"data,omitempty"`
	Time       string `json:"time,omitempty"`
	TemperFlag string `json:"temperFlag,omitempty"` // норма/температура
}

// ── Normalizer ─────────────────────────────────────────────────────────

// DahuaNormalizer implements VendorHandler for Dahua.
type DahuaNormalizer struct{}

func (d *DahuaNormalizer) Normalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var dahua DahuaEvent
	if err := json.Unmarshal(payload, &dahua); err != nil {
		return nil, fmt.Errorf("dahua unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       strings.ToLower(dahua.Code),
		Timestamp:  parseTimeOrDefault(dahua.Time, time.Now()),
		Source:     "dahua",
		RawPayload: string(payload),
		Metadata: map[string]string{
			"action":      dahua.Action,
			"channel":     fmt.Sprintf("%d", dahua.Index),
			"temper_flag": dahua.TemperFlag,
		},
		Tags: map[string]string{
			"vendor": "dahua",
			"type":   dataType,
		},
	}

	// Dahua severity mapping
	switch strings.ToLower(dahua.Action) {
	case "start", "pulse":
		event.Severity = "high"
	case "stop":
		event.Severity = "low"
	default:
		event.Severity = "medium"
	}

	// Dahua event messages
	switch strings.ToLower(dahua.Code) {
	case "videomotion":
		event.Message = fmt.Sprintf("Motion detected on channel %d", dahua.Index)
	case "videoloss":
		event.Message = fmt.Sprintf("Video loss on channel %d", dahua.Index)
	case "temperature":
		event.Message = fmt.Sprintf("Temperature anomaly on channel %d", dahua.Index)
		event.Metrics = []models.Metric{
			{Name: "temperature", Value: dahua.Data.Temperature, Unit: "celsius"},
		}
	default:
		event.Message = fmt.Sprintf("Event %s on channel %d", dahua.Code, dahua.Index)
	}

	// Добавляем метрики если есть
	if dahua.Data.CPUUsage > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "cpu_usage", Value: dahua.Data.CPUUsage, Unit: "percent"})
	}
	if dahua.Data.NetRate > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "network_rate", Value: dahua.Data.NetRate, Unit: "bps"})
	}

	return event, nil
}
