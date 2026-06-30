// Package ingestion — Unified Ingestion Layer.
//
// ═══════════════════════════════════════════════════════════════════════
// ONVIF Vendor Normalizer
//
// ONVIF использует стандартизированный формат WS-Eventing/NotificationMessage.
// Типы событий: Motion, Tamper, VideoAnalytics, VideoLoss, Disconnect.
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

// ── ONVIF Types ────────────────────────────────────────────────────────

// ONVIFEvent — структура ONVIF-формата событий.
type ONVIFEvent struct {
	Topic struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"` // Motion, Tamper, VideoAnalytics, etc.
	} `json:"topic"`
	Message struct {
		Source    string `json:"source,omitempty"`
		Key       string `json:"key,omitempty"`
		Data      string `json:"data,omitempty"`
		UtcTime   string `json:"utcTime,omitempty"`
		Analog    bool   `json:"analog,omitempty"`
		Rule      string `json:"rule,omitempty"`
		Digital   bool   `json:"digital,omitempty"`
	} `json:"message"`
	SubscriptionRef string `json:"subscriptionRef,omitempty"`
}

// ── Normalizer ─────────────────────────────────────────────────────────

// ONVIFNormalizer implements VendorHandler for ONVIF.
type ONVIFNormalizer struct{}

func (o *ONVIFNormalizer) Normalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var onvif ONVIFEvent
	if err := json.Unmarshal(payload, &onvif); err != nil {
		return nil, fmt.Errorf("onvif unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       strings.ToLower(onvif.Topic.Name),
		Timestamp:  parseTimeOrDefault(onvif.Message.UtcTime, time.Now()),
		Source:     "onvif",
		RawPayload: string(payload),
		Message:    onvif.Message.Data,
		Metadata: map[string]string{
			"topic_namespace":  onvif.Topic.Namespace,
			"subscription_ref": onvif.SubscriptionRef,
			"rule":             onvif.Message.Rule,
		},
		Tags: map[string]string{
			"vendor": "onvif",
			"type":   dataType,
		},
	}

	// ONVIF severity based on event type
	switch strings.ToLower(onvif.Topic.Name) {
	case "motion", "tamper":
		event.Severity = "high"
		if onvif.Message.Analog {
			event.Severity = "medium"
		}
	case "videoloss", "disconnect":
		event.Severity = "critical"
	default:
		event.Severity = "info"
	}

	return event, nil
}
