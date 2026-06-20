// Package gatekeeper реализует сервис верификации для закрытия нарядов.
// Проверяет GPS-координаты, EXIF-метаданные фото и AI-сравнение снимков ДО/ПОСЛЕ.
package gatekeeper

import (
	"math"
	"time"
)

// GPSPoint представляет географическую точку с координатами и метаданными.
type GPSPoint struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Accuracy  float64   `json:"accuracy"`  // метры, точность GPS-сигнала
	Timestamp time.Time `json:"timestamp"` // время получения координат
}

// GPSResult содержит результат GPS-верификации.
type GPSResult struct {
	Passed         bool    `json:"passed"`
	DistanceMeters float64 `json:"distance_meters"` // расстояние до объекта
	AccuracyMeters float64 `json:"accuracy_meters"` // точность сигнала
	WithinGeofence bool    `json:"within_geofence"` // внутри геозоны
	TimestampValid bool    `json:"timestamp_valid"` // метка времени актуальна
	Error          string  `json:"error,omitempty"`
}

const (
	// MaxDistanceMeters — максимальное допустимое расстояние от техника до объекта.
	MaxDistanceMeters = 500.0
	// MaxAccuracyMeters — максимальная допустимая погрешность GPS.
	MaxAccuracyMeters = 50.0
	// MaxTimestampAge — максимальный возраст GPS-координат.
	MaxTimestampAge = 5 * time.Minute
)

// VerifyGPS проверяет, что техник находится в пределах допустимого расстояния от объекта.
// Использует формулу гаверсинусов для вычисления расстояния между двумя точками на сфере.
func VerifyGPS(techPoint, sitePoint GPSPoint, geofenceRadiusMeters float64) GPSResult {
	result := GPSResult{
		AccuracyMeters: techPoint.Accuracy,
	}

	// Проверка точности сигнала
	if techPoint.Accuracy > MaxAccuracyMeters {
		result.Error = "gps accuracy too low"
		return result
	}

	// Проверка актуальности временной метки
	age := time.Since(techPoint.Timestamp)
	if age < 0 {
		age = -age
	}
	if age > MaxTimestampAge {
		result.Error = "gps timestamp too old"
		return result
	}
	result.TimestampValid = true

	// Вычисление расстояния по формуле гаверсинусов
	distance := haversineDistance(techPoint.Latitude, techPoint.Longitude, sitePoint.Latitude, sitePoint.Longitude)
	result.DistanceMeters = distance

	// Проверка геозоны: если задан кастомный радиус, используем его, иначе дефолтный
	effectiveRadius := geofenceRadiusMeters
	if effectiveRadius <= 0 {
		effectiveRadius = MaxDistanceMeters
	}
	result.WithinGeofence = distance <= effectiveRadius

	if distance > MaxDistanceMeters {
		result.Error = "technician too far from site"
		return result
	}

	result.Passed = true
	return result
}

// haversineDistance вычисляет расстояние между двумя точками в метрах по формуле гаверсинусов.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	phi1 := lat1 * math.Pi / 180.0
	phi2 := lat2 * math.Pi / 180.0
	deltaPhi := (lat2 - lat1) * math.Pi / 180.0
	deltaLambda := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}
