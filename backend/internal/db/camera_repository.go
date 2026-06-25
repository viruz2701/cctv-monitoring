// Package db — Camera Specs Repository (P0-9)
//
// Хранение и поиск технических характеристик камер.
// Соответствует:
//   - ISO 27001 A.8.1.2 (Asset inventory — каталог оборудования)
//   - ISO 27019 PCC.A.8 (Asset management для ICS)
//   - IEC 62443 SR 3.1 (Identification of IACS devices)
//   - OWASP ASVS V5 (Parameterized queries — SQL injection prevention)
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ── Model ────────────────────────────────────────────────────────────────

// CameraSpec представляет технические характеристики модели камеры.
type CameraSpec struct {
	ID                  int       `json:"id"`
	Brand               string    `json:"brand"`
	Model               string    `json:"model"`
	Type                *string   `json:"type,omitempty"`
	Resolution          *string   `json:"resolution,omitempty"`
	MaxFPS              *int      `json:"max_fps,omitempty"`
	LensMM              *string   `json:"lens_mm,omitempty"`
	Infrared            *bool     `json:"infrared,omitempty"`
	PoE                 *bool     `json:"poe,omitempty"`
	PoEClass            *string   `json:"poe_class,omitempty"`
	PowerWatts          *float64  `json:"power_watts,omitempty"`
	StorageDaysEstimate *int      `json:"storage_days_estimate,omitempty"`
	BandwidthMbps       *float64  `json:"bandwidth_mbps,omitempty"`
	Protocols           []string  `json:"protocols,omitempty"`
	ONVIFProfile        *string   `json:"onvif_profile,omitempty"`
	AudioSupport        *bool     `json:"audio_support,omitempty"`
	OutdoorRating       *string   `json:"outdoor_rating,omitempty"`
	WeightGrams         *int      `json:"weight_grams,omitempty"`
	Dimensions          *string   `json:"dimensions,omitempty"`
	Notes               *string   `json:"notes,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// CameraBrand представляет бренд с количеством моделей.
type CameraBrand struct {
	Brand string `json:"brand"`
	Count int    `json:"count"`
}

// CameraModelSummary — краткая информация о модели для списка.
type CameraModelSummary struct {
	ID         int     `json:"id"`
	Brand      string  `json:"brand"`
	Model      string  `json:"model"`
	Type       *string `json:"type,omitempty"`
	Resolution *string `json:"resolution,omitempty"`
}

// ── Repository Methods ───────────────────────────────────────────────────

// GetCameraSpecs возвращает характеристики камеры по brand и model.
// Использует parameterized query (OWASP ASVS V5.2 — SQL injection prevention).
func (db *DB) GetCameraSpecs(ctx context.Context, brand, model string) (*CameraSpec, error) {
	var spec CameraSpec
	var typ, resolution, lensMM, poeClass, onvifProfile, outdoorRating, dimensions, notes *string
	var infrared, poe, audioSupport *bool
	var maxFPS, storageDaysEstimate, weightGrams *int
	var powerWatts, bandwidthMbps *float64
	var protocols []string
	var createdAt time.Time

	err := db.Pool.QueryRow(ctx, `
		SELECT
			id, brand, model, type, resolution, max_fps, lens_mm,
			infrared, poe, poe_class, power_watts, storage_days_estimate,
			bandwidth_mbps, protocols, onvif_profile, audio_support,
			outdoor_rating, weight_grams, dimensions, notes, created_at
		FROM camera_specs
		WHERE brand = $1 AND model = $2
	`, brand, model).Scan(
		&spec.ID, &spec.Brand, &spec.Model, &typ, &resolution, &maxFPS, &lensMM,
		&infrared, &poe, &poeClass, &powerWatts, &storageDaysEstimate,
		&bandwidthMbps, &protocols, &onvifProfile, &audioSupport,
		&outdoorRating, &weightGrams, &dimensions, &notes, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get camera spec %s/%s: %w", brand, model, err)
	}

	spec.Type = typ
	spec.Resolution = resolution
	spec.MaxFPS = maxFPS
	spec.LensMM = lensMM
	spec.Infrared = infrared
	spec.PoE = poe
	spec.PoEClass = poeClass
	spec.PowerWatts = powerWatts
	spec.StorageDaysEstimate = storageDaysEstimate
	spec.BandwidthMbps = bandwidthMbps
	spec.Protocols = protocols
	spec.ONVIFProfile = onvifProfile
	spec.AudioSupport = audioSupport
	spec.OutdoorRating = outdoorRating
	spec.WeightGrams = weightGrams
	spec.Dimensions = dimensions
	spec.Notes = notes
	spec.CreatedAt = createdAt

	return &spec, nil
}

// ListBrands возвращает список всех брендов с количеством моделей.
func (db *DB) ListBrands(ctx context.Context) ([]CameraBrand, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT brand, COUNT(*) AS count
		FROM camera_specs
		GROUP BY brand
		ORDER BY brand ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list camera brands: %w", err)
	}
	defer rows.Close()

	var brands []CameraBrand
	for rows.Next() {
		var b CameraBrand
		if err := rows.Scan(&b.Brand, &b.Count); err != nil {
			return nil, fmt.Errorf("scan camera brand: %w", err)
		}
		brands = append(brands, b)
	}
	return brands, rows.Err()
}

// ListModels возвращает список моделей для указанного бренда.
func (db *DB) ListModels(ctx context.Context, brand string) ([]CameraModelSummary, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, brand, model, type, resolution
		FROM camera_specs
		WHERE brand = $1
		ORDER BY model ASC
	`, brand)
	if err != nil {
		return nil, fmt.Errorf("list models for brand %q: %w", brand, err)
	}
	defer rows.Close()

	var models []CameraModelSummary
	for rows.Next() {
		var m CameraModelSummary
		if err := rows.Scan(&m.ID, &m.Brand, &m.Model, &m.Type, &m.Resolution); err != nil {
			return nil, fmt.Errorf("scan camera model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// SearchModels ищет модели по brand или model (ILIKE поиск).
func (db *DB) SearchModels(ctx context.Context, query string, limit int) ([]CameraModelSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, brand, model, type, resolution
		FROM camera_specs
		WHERE brand ILIKE $1 OR model ILIKE $1
		ORDER BY brand ASC, model ASC
		LIMIT $2
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search camera models: %w", err)
	}
	defer rows.Close()

	var models []CameraModelSummary
	for rows.Next() {
		var m CameraModelSummary
		if err := rows.Scan(&m.ID, &m.Brand, &m.Model, &m.Type, &m.Resolution); err != nil {
			return nil, fmt.Errorf("scan camera model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// ── Import ───────────────────────────────────────────────────────────────

// CameraSpecImport — структура для импорта из JSON.
type CameraSpecImport struct {
	Brand               string   `json:"brand"`
	Model               string   `json:"model"`
	Type                string   `json:"type,omitempty"`
	Resolution          string   `json:"resolution,omitempty"`
	MaxFPS              int      `json:"max_fps,omitempty"`
	LensMM              string   `json:"lens_mm,omitempty"`
	Infrared            *bool    `json:"infrared,omitempty"`
	PoE                 *bool    `json:"poe,omitempty"`
	PoEClass            string   `json:"poe_class,omitempty"`
	PowerWatts          float64  `json:"power_watts,omitempty"`
	StorageDaysEstimate int      `json:"storage_days_estimate,omitempty"`
	BandwidthMbps       float64  `json:"bandwidth_mbps,omitempty"`
	Protocols           []string `json:"protocols,omitempty"`
	ONVIFProfile        string   `json:"onvif_profile,omitempty"`
	AudioSupport        *bool    `json:"audio_support,omitempty"`
	OutdoorRating       string   `json:"outdoor_rating,omitempty"`
	WeightGrams         int      `json:"weight_grams,omitempty"`
	Dimensions          string   `json:"dimensions,omitempty"`
	Notes               string   `json:"notes,omitempty"`
}

// ImportResult содержит статистику импорта.
type ImportResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

// ImportFromJSON импортирует камеры из JSON-массива.
// Использует UPSERT для обновления существующих записей.
// Соответствует: ISO 27001 A.8.1.2 (Asset inventory update)
func (db *DB) ImportFromJSON(ctx context.Context, data []byte) (*ImportResult, error) {
	var cameras []CameraSpecImport
	if err := json.Unmarshal(data, &cameras); err != nil {
		return nil, fmt.Errorf("unmarshal camera specs JSON: %w", err)
	}

	result := &ImportResult{}

	for _, c := range cameras {
		// Пропускаем пустые записи
		if c.Brand == "" || c.Model == "" {
			result.Skipped++
			continue
		}

		_, err := db.Pool.Exec(ctx, `
			INSERT INTO camera_specs (
				brand, model, type, resolution, max_fps, lens_mm,
				infrared, poe, poe_class, power_watts, storage_days_estimate,
				bandwidth_mbps, protocols, onvif_profile, audio_support,
				outdoor_rating, weight_grams, dimensions, notes
			) VALUES (
				$1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, 0), NULLIF($6, ''),
				$7, $8, NULLIF($9, ''), NULLIF($10, 0), NULLIF($11, 0),
				NULLIF($12, 0), $13, NULLIF($14, ''), $15,
				NULLIF($16, ''), NULLIF($17, 0), NULLIF($18, ''), NULLIF($19, '')
			)
			ON CONFLICT (brand, model) DO UPDATE SET
				type            = EXCLUDED.type,
				resolution      = EXCLUDED.resolution,
				max_fps         = EXCLUDED.max_fps,
				lens_mm         = EXCLUDED.lens_mm,
				infrared        = EXCLUDED.infrared,
				poe             = EXCLUDED.poe,
				poe_class       = EXCLUDED.poe_class,
				power_watts     = EXCLUDED.power_watts,
				storage_days_estimate = EXCLUDED.storage_days_estimate,
				bandwidth_mbps  = EXCLUDED.bandwidth_mbps,
				protocols       = EXCLUDED.protocols,
				onvif_profile   = EXCLUDED.onvif_profile,
				audio_support   = EXCLUDED.audio_support,
				outdoor_rating  = EXCLUDED.outdoor_rating,
				weight_grams    = EXCLUDED.weight_grams,
				dimensions      = EXCLUDED.dimensions,
				notes           = EXCLUDED.notes
		`,
			c.Brand, c.Model, c.Type, c.Resolution, c.MaxFPS, c.LensMM,
			c.Infrared, c.PoE, c.PoEClass, c.PowerWatts, c.StorageDaysEstimate,
			c.BandwidthMbps, c.Protocols, c.ONVIFProfile, c.AudioSupport,
			c.OutdoorRating, c.WeightGrams, c.Dimensions, c.Notes,
		)
		if err != nil {
			result.Errors++
			continue
		}
		result.Inserted++
	}

	return result, nil
}

// SeedCameraSpecs вставляет 10 популярных моделей Hikvision/Dahua.
// Используется как fallback если внешний репозиторий недоступен.
func (db *DB) SeedCameraSpecs(ctx context.Context) error {
	seed := []CameraSpecImport{
		// ── Hikvision ──────────────────────────────────────────────────
		{
			Brand: "Hikvision", Model: "DS-2CD2386G2-I",
			Type: "dome", Resolution: "8MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 12, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Hikvision-CGI"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP67", WeightGrams: 500, Dimensions: "Φ127×96mm",
		},
		{
			Brand: "Hikvision", Model: "DS-2CD2086G2-I",
			Type: "bullet", Resolution: "8MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 12, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Hikvision-CGI"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP67", WeightGrams: 600, Dimensions: "170×70×70mm",
		},
		{
			Brand: "Hikvision", Model: "DS-2CD2146G2-IS",
			Type: "dome", Resolution: "4MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 8, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Hikvision-CGI"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP67", WeightGrams: 450, Dimensions: "Φ111×92mm",
		},
		{
			Brand: "Hikvision", Model: "DS-2DE5225IW-AE(T5)",
			Type: "ptz", Resolution: "2MP", MaxFPS: 25, LensMM: "5-50mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3at",
			PowerWatts: 24.0, BandwidthMbps: 6, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Hikvision-CGI"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP66", WeightGrams: 2500, Dimensions: "Φ181×219mm",
		},
		{
			Brand: "Hikvision", Model: "DS-2CD2T46G2-2I",
			Type: "bullet", Resolution: "4MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 8, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Hikvision-CGI"},
			ONVIFProfile: "S", AudioSupport: boolPtr(false),
			OutdoorRating: "IP67", WeightGrams: 680, Dimensions: "186×94×97mm",
		},
		// ── Dahua ──────────────────────────────────────────────────────
		{
			Brand: "Dahua", Model: "IPC-HDW3849H-AS-PV",
			Type: "dome", Resolution: "8MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 12, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Dahua-API"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP67", WeightGrams: 480, Dimensions: "Φ122×95mm",
		},
		{
			Brand: "Dahua", Model: "IPC-HFW3849H-AS-PV",
			Type: "bullet", Resolution: "8MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 12, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Dahua-API"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP67", WeightGrams: 650, Dimensions: "190×79×77mm",
		},
		{
			Brand: "Dahua", Model: "SD2223G1-GB",
			Type: "ptz", Resolution: "2MP", MaxFPS: 30, LensMM: "4-12mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3at",
			PowerWatts: 21.0, BandwidthMbps: 6, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Dahua-API"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP66", WeightGrams: 2000, Dimensions: "Φ155×210mm",
		},
		{
			Brand: "Dahua", Model: "IPC-HDBW3849H-AS-PV",
			Type: "dome", Resolution: "8MP", MaxFPS: 20, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 10.5, BandwidthMbps: 8, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Dahua-API"},
			ONVIFProfile: "S", AudioSupport: boolPtr(false),
			OutdoorRating: "IP67", WeightGrams: 420, Dimensions: "Φ125×87mm",
		},
		{
			Brand: "Dahua", Model: "IPC-HFW3449H-AS-PV",
			Type: "bullet", Resolution: "4MP", MaxFPS: 30, LensMM: "2.8mm",
			Infrared: boolPtr(true), PoE: boolPtr(true), PoEClass: "802.3af",
			PowerWatts: 12.95, BandwidthMbps: 8, StorageDaysEstimate: 30,
			Protocols:    []string{"ONVIF", "RTSP", "Dahua-API"},
			ONVIFProfile: "S", AudioSupport: boolPtr(true),
			OutdoorRating: "IP67", WeightGrams: 580, Dimensions: "190×79×77mm",
		},
	}

	for _, c := range seed {
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO camera_specs (
				brand, model, type, resolution, max_fps, lens_mm,
				infrared, poe, poe_class, power_watts, storage_days_estimate,
				bandwidth_mbps, protocols, onvif_profile, audio_support,
				outdoor_rating, weight_grams, dimensions, notes
			) VALUES (
				$1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, 0), NULLIF($6, ''),
				$7, $8, NULLIF($9, ''), NULLIF($10, 0), NULLIF($11, 0),
				NULLIF($12, 0), $13, NULLIF($14, ''), $15,
				NULLIF($16, ''), NULLIF($17, 0), NULLIF($18, ''), NULLIF($19, '')
			)
			ON CONFLICT (brand, model) DO NOTHING
		`,
			c.Brand, c.Model, c.Type, c.Resolution, c.MaxFPS, c.LensMM,
			c.Infrared, c.PoE, c.PoEClass, c.PowerWatts, c.StorageDaysEstimate,
			c.BandwidthMbps, c.Protocols, c.ONVIFProfile, c.AudioSupport,
			c.OutdoorRating, c.WeightGrams, c.Dimensions, c.Notes,
		)
		if err != nil {
			return fmt.Errorf("seed camera spec %s/%s: %w", c.Brand, c.Model, err)
		}
	}

	return nil
}

// boolPtr — вспомогательная функция для указателей на bool.
func boolPtr(v bool) *bool { return &v }
