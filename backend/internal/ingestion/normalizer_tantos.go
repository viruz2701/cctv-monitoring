// Package ingestion — Unified Ingestion Layer.
//
// ═══════════════════════════════════════════════════════════════════════
// Tantos Vendor Normalizer
//
// Tantos использует HTTP API с JSON форматом.
// Типы событий: eventType-based с severity (info, warning, critical).
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

// ── Tantos Types ───────────────────────────────────────────────────────

// TantosEvent — структура Tantos-формата событий.
type TantosEvent struct {
	EventType   string  `json:"eventType"`
	DeviceName  string  `json:"deviceName"`
	DeviceID    string  `json:"deviceId"`
	Severity    string  `json:"severity"`    // info, warning, critical
	Description string  `json:"description"`
	Source      string  `json:"source"`
	Temperature float64 `json:"temperature,omitempty"`
	CPU         float64 `json:"cpu,omitempty"`
	Memory      float64 `json:"memory,omitempty"`
	Network     float64 `json:"network,omitempty"`
	Storage     float64 `json:"storage,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
	ImageURL    string  `json:"imageUrl,omitempty"`
}

// ── Normalizer ─────────────────────────────────────────────────────────

// TantosNormalizer implements VendorHandler for Tantos.
type TantosNormalizer struct{}

func (t *TantosNormalizer) Normalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var tantos TantosEvent
	if err := json.Unmarshal(payload, &tantos); err != nil {
		return nil, fmt.Errorf("tantos unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       strings.ToLower(tantos.EventType),
		Timestamp:  parseTimeOrDefault(tantos.Timestamp, time.Now()),
		Source:     "tantos",
		Severity:   tantos.Severity,
		Message:    tantos.Description,
		ImageURL:   tantos.ImageURL,
		RawPayload: string(payload),
		Metadata: map[string]string{
			"device_name": tantos.DeviceName,
			"device_id":   tantos.DeviceID,
			"source":      tantos.Source,
		},
		Tags: map[string]string{
			"vendor": "tantos",
			"type":   dataType,
		},
	}

	// Метрики
	if tantos.Temperature > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "temperature", Value: tantos.Temperature, Unit: "celsius"})
	}
	if tantos.CPU > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "cpu_usage", Value: tantos.CPU, Unit: "percent"})
	}
	if tantos.Memory > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "memory_usage", Value: tantos.Memory, Unit: "percent"})
	}
	if tantos.Network > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "network_usage", Value: tantos.Network, Unit: "bps"})
	}
	if tantos.Storage > 0 {
		event.Metrics = append(event.Metrics, models.Metric{Name: "storage_usage", Value: tantos.Storage, Unit: "percent"})
	}

	return event, nil
}
