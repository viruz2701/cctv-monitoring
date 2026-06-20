package gatekeeper

import (
	"time"
)

// EXIFMetadata представляет извлечённые из фото EXIF-данные.
type EXIFMetadata struct {
	GPSLatitude      float64   `json:"gps_latitude"`
	GPSLongitude     float64   `json:"gps_longitude"`
	DateTimeOriginal time.Time `json:"date_time_original"`
	Make             string    `json:"make"`  // производитель устройства (Apple, Samsung)
	Model            string    `json:"model"` // модель устройства (iPhone 15 Pro)
}

// EXIFResult содержит результат EXIF-верификации.
type EXIFResult struct {
	Passed         bool   `json:"passed"`
	GPSMatch       bool   `json:"gps_match"`       // GPS в EXIF совпадает с координатами объекта
	TimestampValid bool   `json:"timestamp_valid"` // время съёмки в пределах ±1 час
	HasEXIF        bool   `json:"has_exif"`        // EXIF не пустой (фото не из галереи)
	Error          string `json:"error,omitempty"`
}

const (
	// MaxEXIFAge — максимальный допустимый возраст фото относительно текущего времени.
	MaxEXIFAge = 1 * time.Hour
	// EXIFGPSThreshold — допустимое отклонение GPS-координат в EXIF от координат объекта (метры).
	EXIFGPSThreshold = 500.0
)

// VerifyEXIF проверяет EXIF-метаданные фото:
//   - EXIF не пустой (фото сделано через камеру, не из галереи)
//   - GPS-координаты в EXIF соответствуют координатам объекта
//   - Время съёмки в пределах ±1 часа от текущего
func VerifyEXIF(exif EXIFMetadata, sitePoint GPSPoint) EXIFResult {
	result := EXIFResult{}

	// EXIF должен быть не пустым — это гарантирует, что фото сделано через камеру
	if exif.Make == "" && exif.Model == "" && exif.DateTimeOriginal.IsZero() {
		result.Error = "exif metadata is empty: photo must be taken with camera, not from gallery"
		return result
	}
	result.HasEXIF = true

	// Проверка времени съёмки
	if !exif.DateTimeOriginal.IsZero() {
		age := time.Since(exif.DateTimeOriginal)
		if age < 0 {
			age = -age
		}
		if age <= MaxEXIFAge {
			result.TimestampValid = true
		}
	}

	if !result.TimestampValid {
		result.Error = "photo timestamp is too old or missing"
		return result
	}

	// Проверка GPS-координат в EXIF
	if exif.GPSLatitude != 0 || exif.GPSLongitude != 0 {
		distance := haversineDistance(exif.GPSLatitude, exif.GPSLongitude, sitePoint.Latitude, sitePoint.Longitude)
		if distance <= EXIFGPSThreshold {
			result.GPSMatch = true
		}
	}

	if !result.GPSMatch {
		result.Error = "exif gps does not match site location"
		return result
	}

	result.Passed = true
	return result
}

// IsEmptyEXIF возвращает true, если EXIF-метаданные практически пустые
// (признак загрузки фото из галереи, а не съёмки через камеру).
func IsEmptyEXIF(exif EXIFMetadata) bool {
	return exif.Make == "" && exif.Model == "" && exif.DateTimeOriginal.IsZero() &&
		exif.GPSLatitude == 0 && exif.GPSLongitude == 0
}
