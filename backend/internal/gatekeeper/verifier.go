package gatekeeper

import (
	"context"
	"fmt"
	"time"

	"gb-telemetry-collector/internal/models"
)

// VerificationRequest — входящий запрос на верификацию от мобильного клиента.
type VerificationRequest struct {
	GPS                GPSPoint     `json:"gps"`
	PhotoEXIF          EXIFMetadata `json:"photo_exif"`
	PhotoBeforeURL     string       `json:"photo_before_url"`
	PhotoAfterURL      string       `json:"photo_after_url"`
	ChecklistCompleted bool         `json:"checklist_completed"`
	Signature          string       `json:"signature"`
	// GPSSkipReason — если техник хочет пропустить GPS (подвал, плохая погода).
	GPSSkipReason string `json:"gps_skip_reason,omitempty"`
}

// VerificationResponse — результат полной верификации.
type VerificationResponse struct {
	Passed      bool       `json:"passed"`
	Token       string     `json:"token,omitempty"` // verification JWT (если passed)
	GPS         GPSResult  `json:"gps"`
	EXIF        EXIFResult `json:"exif"`
	AI          AIResult   `json:"ai"`
	Message     string     `json:"message,omitempty"`
	FailReasons []string   `json:"fail_reasons,omitempty"`
}

// SiteInfo содержит информацию об объекте, необходимую для верификации.
type SiteInfo struct {
	SiteID               string  `json:"site_id"`
	SiteName             string  `json:"site_name"`
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	GeofenceRadiusMeters float64 `json:"geofence_radius_meters"` // 0 = использовать дефолт
}

// SiteProvider — интерфейс для получения информации об объекте.
// Позволяет gatekeeper не зависеть от конкретной реализации БД.
type SiteProvider interface {
	GetSiteInfo(ctx context.Context, workOrderID string) (*SiteInfo, error)
}

// Verifier — главный компонент Gatekeeper Service.
// Оркестрирует GPS, EXIF и AI проверки.
type Verifier struct {
	siteProvider SiteProvider
}

// NewVerifier создаёт новый экземпляр Verifier.
func NewVerifier(provider SiteProvider) *Verifier {
	return &Verifier{siteProvider: provider}
}

// Verify выполняет полную верификацию наряда:
//  1. Загружает информацию об объекте
//  2. Проверяет GPS (если не skipped)
//  3. Проверяет EXIF
//  4. Проверяет AI (если есть фото ДО/ПОСЛЕ)
//  5. Выпускает verification token при успехе
func (v *Verifier) Verify(ctx context.Context, req VerificationRequest, workOrderID, technicianID string) (*VerificationResponse, error) {
	response := &VerificationResponse{}

	// 1. Загружаем информацию об объекте
	site, err := v.siteProvider.GetSiteInfo(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("get site info: %w", err)
	}

	sitePoint := GPSPoint{
		Latitude:  site.Latitude,
		Longitude: site.Longitude,
	}

	// 2. Проверка чек-листа
	if !req.ChecklistCompleted {
		response.FailReasons = append(response.FailReasons, "checklist not completed")
	}

	// 3. GPS Verification
	if req.GPSSkipReason != "" {
		// Graceful degradation: GPS пропущен с обоснованием
		response.GPS = GPSResult{
			Passed: true,
			Error:  fmt.Sprintf("skipped: %s", req.GPSSkipReason),
		}
		response.GPS.Passed = true // считается passed, но с пометкой skipped
	} else {
		response.GPS = VerifyGPS(req.GPS, sitePoint, site.GeofenceRadiusMeters)
		if !response.GPS.Passed {
			response.FailReasons = append(response.FailReasons, "gps: "+response.GPS.Error)
		}
	}

	// 4. EXIF Verification
	response.EXIF = VerifyEXIF(req.PhotoEXIF, sitePoint)
	if !response.EXIF.Passed {
		response.FailReasons = append(response.FailReasons, "exif: "+response.EXIF.Error)
	}

	// 5. AI Verification (Phase 2)
	if req.PhotoBeforeURL != "" && req.PhotoAfterURL != "" {
		response.AI = VerifyAI(ctx, AIVerifyRequest{
			PhotoBeforeURL: req.PhotoBeforeURL,
			PhotoAfterURL:  req.PhotoAfterURL,
		})
		if !response.AI.Passed && !response.AI.Skipped {
			response.FailReasons = append(response.FailReasons, "ai: "+response.AI.Error)
		}
	} else {
		response.AI = AIResult{Skipped: true, Error: "no before/after photos provided"}
	}

	// 6. Итоговое решение
	// EXIF обязателен всегда. GPS можно пропустить с обоснованием. AI опционален.
	exifOK := response.EXIF.Passed
	gpsOK := response.GPS.Passed
	checklistOK := req.ChecklistCompleted

	response.Passed = exifOK && gpsOK && checklistOK

	if response.Passed {
		// Выпускаем verification token
		gpsSkipped := req.GPSSkipReason != ""
		token, err := GenerateVerificationToken(workOrderID, technicianID, gpsOK, exifOK, response.AI.Passed, gpsSkipped)
		if err != nil {
			return nil, fmt.Errorf("generate verification token: %w", err)
		}
		response.Token = token
		response.Message = "verification passed"
	} else {
		response.Message = "verification failed"
	}

	return response, nil
}

// DBToSiteInfo преобразует models.WorkOrder в SiteInfo через запрос к БД.
// Используется адаптером SiteProvider.
func DBToSiteInfo(wo models.WorkOrder) *SiteInfo {
	// Координаты объекта извлекаются из work order metadata
	// В текущей реализации координаты хранятся в device metadata
	return &SiteInfo{
		SiteID:   wo.DeviceID,
		SiteName: wo.DeviceName,
		// Latitude/Longitude будут заполнены из таблицы devices
	}
}

// VerifyChecklist проверяет, что все элементы чек-листа выполнены.
func VerifyChecklist(items []models.BasicChecklistItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !item.Completed {
			return false
		}
	}
	return true
}

// FormatTimestamp форматирует time.Time для логов.
func FormatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}
