// Package models — domain models для QR-кодов (UX-4.2).
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Queue-based batch generation)
//   - ISO 27001 A.12.4 (Audit trail — каждый QR логируется)
//   - OWASP ASVS V5 (Input validation — whitelist)
//   - Приказ ОАЦ №66 п. 7.18 (Уникальная идентификация устройств)
package models

import "time"

// ── QR Code Types ────────────────────────────────────────────────────────

type QRCodeType string

const (
	QRTypeDevice    QRCodeType = "device"
	QRTypeWorkOrder QRCodeType = "work_order"
	QRTypeSparePart QRCodeType = "spare_part"
	QRTypeTO        QRCodeType = "to"      // Technical Output
	QRTypeOnboard   QRCodeType = "onboard" // Onboarding token
	QRTypeVerify    QRCodeType = "verify"  // Verification link
)

// ── QR Code Data (what's encoded in the QR) ──────────────────────────────

// QRCodePayload — данные, кодируемые в QR-код.
// Версионируется через version для обратной совместимости.
type QRCodePayload struct {
	Version    int        `json:"v"`
	Type       QRCodeType `json:"t"`
	CodeID     string     `json:"cid"` // UUID кода
	EntityID   string     `json:"eid"` // device_id / wo_id / part_id
	EntityName string     `json:"enm,omitempty"`
	SiteID     string     `json:"sid,omitempty"`
	TenantID   string     `json:"tid,omitempty"`
	Timestamp  string     `json:"ts"`            // время генерации (RFC3339)
	BaseURL    string     `json:"url,omitempty"` // base URL для verify
	TOHash     string     `json:"toh,omitempty"` // hash-chain для TO traceability
}

// ── Generate Batch Request ───────────────────────────────────────────────

// QRGenerateBatchRequest — запрос на bulk генерацию QR-кодов.
type QRGenerateBatchRequest struct {
	Type    QRCodeType     `json:"type" validate:"required,oneof=device work_order spare_part to onboard verify"`
	Entries []QRBatchEntry `json:"entries" validate:"required,min=1,max=100"`
}

// QRBatchEntry — одна запись в batch генерации.
type QRBatchEntry struct {
	EntityID   string `json:"entity_id" validate:"required,max=255"`
	EntityName string `json:"entity_name,omitempty" validate:"max=500"`
	SiteID     string `json:"site_id,omitempty" validate:"omitempty,uuid"`
}

// QRGenerateBatchResponse — ответ с batch QR-кодами.
type QRGenerateBatchResponse struct {
	BatchID     string      `json:"batch_id"`
	Total       int         `json:"total"`
	Codes       []QRCodeRef `json:"codes"`
	PDFURL      string      `json:"pdf_url,omitempty"`
	GeneratedAt string      `json:"generated_at"`
}

// QRCodeRef — ссылка на сгенерированный QR-код.
type QRCodeRef struct {
	CodeID   string `json:"code_id"`
	EntityID string `json:"entity_id"`
	QRData   string `json:"qr_data"` // JSON payload для QR
	QRURL    string `json:"qr_url"`  // URL для QR image/PDF
}

// ── Onboarding ───────────────────────────────────────────────────────────

// QROnboardRequest — запрос на onboard устройства через QR.
type QROnboardRequest struct {
	CodeID     string  `json:"code_id" validate:"required,uuid"`
	DeviceID   string  `json:"device_id" validate:"required,uuid"`
	SiteID     string  `json:"site_id" validate:"required,uuid"`
	Name       string  `json:"name" validate:"required,min=1,max=255"`
	Latitude   float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude  float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	VendorType string  `json:"vendor_type,omitempty" validate:"max=100"`
}

// QROnboardResponse — ответ после onboard устройства.
type QROnboardResponse struct {
	CodeID      string `json:"code_id"`
	DeviceID    string `json:"device_id"`
	SiteID      string `json:"site_id"`
	Status      string `json:"status"` // onboarded, already_onboarded
	QRURL       string `json:"qr_url"`
	OnboardedAt string `json:"onboarded_at"`
}

// ── Verification ─────────────────────────────────────────────────────────

// QRVerifyRequest — запрос на верификацию по QR (TO initiation).
type QRVerifyRequest struct {
	CodeID string  `json:"code_id" validate:"required,uuid"`
	WOID   string  `json:"wo_id" validate:"required,uuid"`
	GPSLat float64 `json:"gps_lat" validate:"required,min=-90,max=90"`
	GPSLng float64 `json:"gps_lng" validate:"required,min=-180,max=180"`
	GPSAcc float64 `json:"gps_acc" validate:"min=0"`
}

// QRVerifyResponse — ответ с результатом верификации и TO data.
type QRVerifyResponse struct {
	CodeID       string           `json:"code_id"`
	Verified     bool             `json:"verified"`
	DeviceID     string           `json:"device_id"`
	SiteID       string           `json:"site_id,omitempty"`
	GPSDistance  float64          `json:"gps_distance_m"`
	GPSPassed    bool             `json:"gps_passed"`
	TOInitiated  bool             `json:"to_initiated"`
	TOJournalRef *TOJournalRef    `json:"to_journal,omitempty"`
	History      []QRHistoryEntry `json:"history,omitempty"`
}

// TOJournalRef — ссылка на сгенерированный TO journal.
type TOJournalRef struct {
	JournalID   string `json:"journal_id"`
	WOID        string `json:"wo_id"`
	Status      string `json:"status"`
	GeneratedAt string `json:"generated_at"`
	HashChain   string `json:"hash_chain"`
}

// QRHistoryEntry — запись в истории QR-сканирований.
type QRHistoryEntry struct {
	ScannedAt   string  `json:"scanned_at"`
	Action      string  `json:"action"` // onboard, verify, maintenance
	UserID      string  `json:"user_id,omitempty"`
	GPSLat      float64 `json:"gps_lat,omitempty"`
	GPSLng      float64 `json:"gps_lng,omitempty"`
	GPSDistance float64 `json:"gps_distance_m,omitempty"`
	HashBlock   string  `json:"hash_block,omitempty"`
}

// ── QR Code Record (DB) ─────────────────────────────────────────────────

// QRCodeRecord — запись QR-кода в БД.
type QRCodeRecord struct {
	CodeID    string     `json:"code_id"`
	Type      QRCodeType `json:"type"`
	EntityID  string     `json:"entity_id"`
	SiteID    *string    `json:"site_id,omitempty"`
	TenantID  string     `json:"tenant_id"`
	Payload   string     `json:"payload"` // сериализованный QRCodePayload
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	Status    string     `json:"status"` // active, used, expired, revoked
}

// QRScanLog — лог сканирования QR-кода (audit trail).
type QRScanLog struct {
	ID        string    `json:"id"`
	CodeID    string    `json:"code_id"`
	Action    string    `json:"action"` // onboard, verify, maintenance
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id,omitempty"`
	WOID      string    `json:"wo_id,omitempty"`
	GPSLat    float64   `json:"gps_lat,omitempty"`
	GPSLng    float64   `json:"gps_lng,omitempty"`
	GPSAcc    float64   `json:"gps_acc,omitempty"`
	Result    string    `json:"result"` // success, failed
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	PrevHash  string    `json:"prev_hash"` // hash-chain (ISO 27001 A.12.4)
	Hash      string    `json:"hash"`
}

// ── GPS Constants ───────────────────────────────────────────────────────

const (
	// GPSMaxDistanceMeters — максимальное расстояние от устройства (50м).
	GPSMaxDistanceMeters = 50.0

	// GPSMaxAccuracyMeters — максимальная погрешность GPS (25м).
	GPSMaxAccuracyMeters = 25.0
)
