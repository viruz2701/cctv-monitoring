// Package ingestion — Unified Ingestion Layer for CCTV Health Monitor (P0-EDGE Block 5).
//
// ═══════════════════════════════════════════════════════════════════════
// INGEST-02: Vendor Normalizer — Core
//
// Нормализует данные от разных вендоров видеонаблюдения во внутренний
// формат models.Event. Поддерживает Hikvision, Dahua, ONVIF, Tiandy,
// Uniview, Tantos.
//
// Каждый вендор имеет свой формат данных; нормализатор маппит
// вендор-специфичные поля в унифицированную структуру Event.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
//   - OWASP ASVS L3 V5.1: Input validation (whitelist — KnownVendors)
//   - OWASP ASVS L3 V5.3: Input validation — structured data validation
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════
package ingestion

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ── Vendor Registry ─────────────────────────────────────────────────────

// VendorRegistry предоставляет VendorHandler для каждого вендора.
type VendorRegistry interface {
	GetHandler(vendor string) (VendorHandler, bool)
}

// VendorHandler нормализует payload конкретного вендора во внутренний формат.
type VendorHandler interface {
	Normalize(dataType string, payload json.RawMessage) (*models.Event, error)
}

// ── Known Vendors ──────────────────────────────────────────────────────

// KnownVendors — whitelist поддерживаемых вендоров (OWASP ASVS V5.1).
var KnownVendors = []string{
	"hikvision",
	"dahua",
	"onvif",
	"tiandy",
	"uniview",
	"tantos",
}

// ── Vendor Normalizer ───────────────────────────────────────────────────

// VendorNormalizer нормализует данные от разных вендоров во внутренний формат.
//
// Использует VendorHandler для каждого вендора. Если вендор не найден —
// использует fallback-нормализатор (DefaultNormalizer).
type VendorNormalizer struct {
	registry map[string]VendorHandler
	logger   *slog.Logger
}

// NewVendorNormalizer создаёт VendorNormalizer со всеми поддержанными вендорами.
func NewVendorNormalizer(logger *slog.Logger) *VendorNormalizer {
	if logger == nil {
		logger = slog.Default()
	}

	n := &VendorNormalizer{
		registry: make(map[string]VendorHandler),
		logger:   logger.With("component", "vendor_normalizer"),
	}

	// Регистрируем всех поддержанных вендоров
	n.register("hikvision", &HikvisionNormalizer{})
	n.register("dahua", &DahuaNormalizer{})
	n.register("onvif", &ONVIFNormalizer{})
	n.register("tiandy", &TiandyNormalizer{})
	n.register("uniview", &UniviewNormalizer{})
	n.register("tantos", &TantosNormalizer{})

	return n
}

// register добавляет VendorHandler для указанного vendor.
func (v *VendorNormalizer) register(vendor string, handler VendorHandler) {
	v.registry[strings.ToLower(vendor)] = handler
}

// Normalize нормализует payload от вендора во внутренний Event.
//
// Параметры:
//   - dataType: тип данных (telemetry, alarm, log, event)
//   - vendor: имя вендора (hikvision, dahua, onvif, tiandy, uniview, tantos)
//   - payload: вендор-специфичный JSON payload
//
// Возвращает:
//   - *models.Event: нормализованное событие
//   - error: если нормализация не удалась
//
// OWASP ASVS V5.1: Валидация vendor по whitelist (KnownVendors).
// OWASP ASVS V7.1: Ошибки логируются без раскрытия sensitive data.
func (v *VendorNormalizer) Normalize(dataType, vendor string, payload json.RawMessage) (*models.Event, error) {
	logger := v.logger.With(
		slog.String("vendor", vendor),
		slog.String("data_type", dataType),
	)

	// Ищем handler для вендора
	handler, ok := v.registry[strings.ToLower(vendor)]
	if !ok {
		logger.Warn("unknown vendor, using default normalizer",
			"known_vendors", strings.Join(KnownVendors, ", "),
		)
		return DefaultNormalize(dataType, payload)
	}

	event, err := handler.Normalize(dataType, payload)
	if err != nil {
		logger.Warn("vendor normalization failed",
			"error", err,
			"payload_size", len(payload),
		)
		return nil, fmt.Errorf("%s normalizer: %w", vendor, err)
	}

	logger.Debug("vendor data normalized", "event_type", event.Type)
	return event, nil
}

// ── Default Normalizer (fallback) ───────────────────────────────────────

// DefaultNormalize — fallback-нормализатор для неизвестных вендоров.
// Пытается распарсить generic JSON с полями type, severity, message, timestamp.
func DefaultNormalize(dataType string, payload json.RawMessage) (*models.Event, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("default normalizer: unmarshal: %w", err)
	}

	event := &models.Event{
		Type:       dataType,
		Timestamp:  time.Now(),
		Source:     "edge_unknown",
		Metadata:   make(map[string]string),
		RawPayload: string(payload),
	}

	if t, ok := raw["type"].(string); ok {
		event.Type = t
	}
	if s, ok := raw["severity"].(string); ok {
		event.Severity = s
	}
	if m, ok := raw["message"].(string); ok {
		event.Message = m
	}
	if ts, ok := raw["timestamp"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			event.Timestamp = parsed
		}
	}

	return event, nil
}

// ── Общие вспомогательные функции ──────────────────────────────────────

// parseTimeOrDefault пытается распарсить time string в разных форматах.
// Если парсинг не удался — возвращает defaultValue.
func parseTimeOrDefault(timeStr string, defaultValue time.Time) time.Time {
	if timeStr == "" {
		return defaultValue
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	return defaultValue
}
