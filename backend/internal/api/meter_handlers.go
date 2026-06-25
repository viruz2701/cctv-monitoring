// Package api — Meter Readings handlers (AH-5.3.4).
//
// GET /api/v1/meters/{deviceId}/readings — возвращает time-series readings.
// GET /api/v1/meters/{deviceId}/stats — агрегированная статистика.
//
// Соответствует:
//   - IEC 62443 SR 7.1 (Resource availability — мониторинг метрик)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - Приказ ОАЦ №66 п. 7.18.3 (Edge device monitoring)
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/meter"
)

// ── Types ────────────────────────────────────────────────────────────

// MeterReadingResponse — ответ с показаниями метрик.
type MeterReadingResponse struct {
	DeviceID string          `json:"device_id"`
	Meters   []MeterWithData `json:"meters"`
}

// MeterWithData — метрика с её показаниями.
type MeterWithData struct {
	ID       string              `json:"id"`
	Kind     meter.MeterKind     `json:"kind"`
	Name     string              `json:"name"`
	Unit     meter.MeterUnit     `json:"unit"`
	Readings []meter.Reading     `json:"readings"`
	Stats    *meter.ReadingStats `json:"stats,omitempty"`
}

// ── Handler: GET /api/v1/meters/{deviceId}/readings ────────────────

func (s *Server) handleMeterReadings(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}

	// Парсим period (по умолчанию 24h)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}
	duration, err := time.ParseDuration(period)
	if err != nil {
		duration = 24 * time.Hour
	}

	limit := 200 // макс 200 точек на график
	since := time.Now().Add(-duration)

	// Получаем метрики устройства из БД
	readings, err := s.db.GetMeterReadings(r.Context(), deviceID, since, limit)
	if err != nil {
		s.logger.Error("failed to get meter readings", "device_id", deviceID, "error", err)
		respondError(w, r, NewInternalError("failed to get meter readings", err))
		return
	}

	// Группируем по kind
	type meterGroup struct {
		id   string
		kind meter.MeterKind
		name string
		unit meter.MeterUnit
		vals []meter.Reading
	}

	groups := make(map[string]*meterGroup)
	for _, rd := range readings {
		key := string(rd.Kind)
		if _, ok := groups[key]; !ok {
			groups[key] = &meterGroup{
				id:   rd.MeterID,
				kind: rd.Kind,
				name: string(rd.Kind),
				unit: meterReadingUnit(rd.Kind),
			}
		}
		groups[key].vals = append(groups[key].vals, rd)
	}

	// Собираем ответ
	meters := make([]MeterWithData, 0, len(groups))
	for _, g := range groups {
		meters = append(meters, MeterWithData{
			ID:       g.id,
			Kind:     g.kind,
			Name:     g.name,
			Unit:     g.unit,
			Readings: g.vals,
		})
	}

	jsonResponse(w, http.StatusOK, MeterReadingResponse{
		DeviceID: deviceID,
		Meters:   meters,
	})
}

// ── Helpers ──────────────────────────────────────────────────────────

func meterReadingUnit(kind meter.MeterKind) meter.MeterUnit {
	units := map[meter.MeterKind]meter.MeterUnit{
		meter.MeterBitrate:           "kbps",
		meter.MeterFPS:               "fps",
		meter.MeterCPUTemp:           "celsius",
		meter.MeterCPUUsage:          "%",
		meter.MeterMemoryUsage:       "%",
		meter.MeterErrorCount:        "count",
		meter.MeterOfflineRatio:      "%",
		meter.MeterPacketLoss:        "%",
		meter.MeterSignalStrength:    "dBm",
		meter.MeterDiskUsage:         "%",
		meter.MeterRecordingDuration: "hours",
		meter.MeterMotionEvents:      "count",
	}

	if u, ok := units[kind]; ok {
		return u
	}
	return ""
}

// ── DB Interface ─────────────────────────────────────────────────────

// GetMeterReadings возвращает показания метрик устройства за период.
func (db *DB) GetMeterReadings(ctx context.Context, deviceID string, since time.Time, limit int) ([]meter.Reading, error) {
	// TODO: реализовать SQL запрос к timescaledb hypertable
	// Пока возвращаем заглушку для разработки
	_ = ctx
	_ = deviceID
	_ = since
	_ = limit
	return nil, fmt.Errorf("GetMeterReadings not implemented: add migration 011_meter_tables")
}
