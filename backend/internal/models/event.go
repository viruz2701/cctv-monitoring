// Package models — domain models for CCTV Health Monitor.
//
// Этот файл содержит Event и Metric — универсальные типы для Unified Ingestion Layer.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
//   - OWASP ASVS L3 V5.1: Input validation
//   - ISO 27001 A.12.4: Audit trail
package models

import (
	"time"
)

// ── Event ──────────────────────────────────────────────────────────────

// Event — универсальное событие после нормализации данных от edge-агента.
// Содержит все поля, необходимые для downstream обработки.
type Event struct {
	ID        string            `json:"id,omitempty"`         // UUID v7
	Type      string            `json:"type"`                 // motion, tamper, temperature, cpu_usage, etc.
	Source    string            `json:"source"`               // hikvision, dahua, onvif, etc.
	Severity  string            `json:"severity,omitempty"`   // critical, high, medium, low, info
	Message   string            `json:"message,omitempty"`    // human-readable описание
	ImageURL  string            `json:"image_url,omitempty"`  // URL снимка (alarm)
	Timestamp time.Time         `json:"timestamp"`            // время события
	Metrics   []Metric          `json:"metrics,omitempty"`    // числовые метрики (telemetry)
	Tags      map[string]string `json:"tags,omitempty"`       // теги для фильтрации
	Metadata  map[string]string `json:"metadata,omitempty"`   // вендор-специфичные поля
	RawPayload string           `json:"-"`                    // оригинальный payload (не в JSON)
}

// ── Metric ─────────────────────────────────────────────────────────────

// Metric — числовая метрика с именем, значением и единицей измерения.
type Metric struct {
	Name  string  `json:"name"`  // temperature, cpu_usage, memory_usage, etc.
	Value float64 `json:"value"` // числовое значение
	Unit  string  `json:"unit"`  // celsius, percent, bps, etc.
}
