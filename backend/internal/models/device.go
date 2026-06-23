// Package models — domain models for CCTV Health Monitor.
// Соответствует: СТБ 34.101.27, ISO 27001 A.8 (Asset Management), OWASP ASVS V5 (Validation)
package models

import (
	"net"
	"time"
)

// ── Device Status ──────────────────────────────────────────────────────

type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "ONLINE"
	StatusOffline DeviceStatus = "OFFLINE"
	StatusWarning DeviceStatus = "WARNING"
)

// ── Device Type ────────────────────────────────────────────────────────

type DeviceType string

const (
	DeviceTypeCamera DeviceType = "camera"
	DeviceTypeNVR    DeviceType = "nvr"
	DeviceTypeDVR    DeviceType = "dvr"
	DeviceTypeSwitch DeviceType = "switch"
)

// ValidDeviceTypes для whitelist validation (OWASP ASVS V5.1)
var ValidDeviceTypes = []string{
	string(DeviceTypeCamera),
	string(DeviceTypeNVR),
	string(DeviceTypeDVR),
	string(DeviceTypeSwitch),
}

// ── Connection Type ────────────────────────────────────────────────────

type ConnectionType string

const (
	ConnIP       ConnectionType = "ip"
	ConnP2P      ConnectionType = "p2p"
	ConnSNMP     ConnectionType = "snmp"
	ConnSyslog   ConnectionType = "syslog"
	ConnAlarm    ConnectionType = "alarm"
	ConnGB28181  ConnectionType = "gb28181"
	ConnONVIF    ConnectionType = "onvif"
)

// ── Asset Class ────────────────────────────────────────────────────────

type AssetClass string

const (
	AssetCritical     AssetClass = "critical"
	AssetConfidential AssetClass = "confidential"
	AssetInternal     AssetClass = "internal"
	AssetPublic       AssetClass = "public"
)

// ValidAssetClasses для whitelist validation (OWASP ASVS V5.1)
var ValidAssetClasses = []string{
	string(AssetCritical),
	string(AssetConfidential),
	string(AssetInternal),
	string(AssetPublic),
}

// ── Health Status ──────────────────────────────────────────────────────

type HealthStatus string

const (
	HealthHealthy  HealthStatus = "healthy"
	HealthFaulty   HealthStatus = "faulty"
	HealthDegraded HealthStatus = "degraded"
)

// ── Device (полная модель) ─────────────────────────────────────────────

// Device представляет устройство видеонаблюдения.
// JSON-теги для API, validate теги для OWASP ASVS V5 (whitelist validation).
type Device struct {
	DeviceID    string       `json:"device_id" validate:"required,uuid"`
	OwnerID     *string      `json:"owner_id,omitempty" validate:"omitempty,uuid"`
	SiteID      *string      `json:"site_id,omitempty" validate:"omitempty,uuid"`
	Name        string       `json:"name" validate:"required,min=1,max=255"`
	Location    string       `json:"location" validate:"max=500"`
	Latitude    float64      `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude   float64      `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	VendorType  string       `json:"vendor_type" validate:"max=100"`
	DeviceType  DeviceType   `json:"device_type" validate:"required,oneof=camera nvr dvr switch"`
	Status      DeviceStatus `json:"status" validate:"required,oneof=ONLINE OFFLINE WARNING"`
	Health      HealthStatus `json:"health" validate:"required,oneof=healthy faulty degraded"`
	AssetClass  AssetClass   `json:"asset_class" validate:"required,oneof=critical confidential internal public"`

	// Timestamps
	LastSeen     time.Time `json:"last_seen" validate:"required"`
	RegisteredAt time.Time `json:"registered_at" validate:"required"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"` // soft delete

	// Connectivity
	ContactAddr       *net.UDPAddr   `json:"-" validate:"-"`
	HeartbeatInterval int            `json:"heartbeat_interval" validate:"min=0,max=86400"`
	UserAgent         string         `json:"user_agent" validate:"max=500"`
	ConnectionType    ConnectionType `json:"connection_type" validate:"required,oneof=ip p2p snmp syslog alarm gb28181 onvif"`

	// P2P
	P2PBrand    string `json:"p2p_brand,omitempty" validate:"max=100"`
	P2PSerial   string `json:"p2p_serial,omitempty" validate:"max=100"`
	CloudStatus string `json:"cloud_status,omitempty" validate:"max=50"`

	// Manufacturer info
	Manufacturer    string `json:"manufacturer,omitempty" validate:"max=200"`
	SerialNumber    string `json:"serial_number,omitempty" validate:"max=200"`
	MacAddress      string `json:"mac_address,omitempty" validate:"omitempty,mac"`
	FirmwareVersion string `json:"firmware_version,omitempty" validate:"max=50"`

	// Last alarm (for quick status)
	LastAlarm *Alarm `json:"last_alarm,omitempty"`
	LastError string `json:"last_error,omitempty" validate:"max=1000"`
}

// ── CreateDeviceRequest (для POST /api/v1/devices) ─────────────────────

// CreateDeviceRequest — структура запроса на создание устройства.
// Whitelist validation через validate теги (OWASP ASVS V5.1).
type CreateDeviceRequest struct {
	DeviceID       string         `json:"device_id" validate:"required,uuid"`
	Name           string         `json:"name" validate:"required,min=1,max=255"`
	Location       string         `json:"location,omitempty" validate:"max=500"`
	Latitude       float64        `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude      float64        `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	VendorType     string         `json:"vendor_type,omitempty" validate:"max=100"`
	DeviceType     string         `json:"device_type" validate:"required,oneof=camera nvr dvr switch"`
	Status         string         `json:"status" validate:"required,oneof=ONLINE OFFLINE WARNING"`
	ConnectionType string         `json:"connection_type" validate:"required,oneof=ip p2p snmp syslog alarm gb28181 onvif"`
	AssetClass     string         `json:"asset_class" validate:"required,oneof=critical confidential internal public"`
	Manufacturer   string         `json:"manufacturer,omitempty" validate:"max=200"`
	SerialNumber   string         `json:"serial_number,omitempty" validate:"max=200"`
	MacAddress     string         `json:"mac_address,omitempty" validate:"omitempty,mac"`
	FirmwareVersion string        `json:"firmware_version,omitempty" validate:"max=50"`
	SiteID         *string        `json:"site_id,omitempty" validate:"omitempty,uuid"`
	P2PBrand       string         `json:"p2p_brand,omitempty" validate:"max=100"`
	P2PSerial      string         `json:"p2p_serial,omitempty" validate:"max=100"`
	UserAgent      string         `json:"user_agent,omitempty" validate:"max=500"`
}

// ── UpdateDeviceRequest (для PUT /api/v1/devices/{id}) ─────────────────

// UpdateDeviceRequest — структура для частичного обновления устройства.
// Все поля опциональны (omitvalidate для частичного обновления).
type UpdateDeviceRequest struct {
	Name           *string  `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Location       *string  `json:"location,omitempty" validate:"omitempty,max=500"`
	Latitude       *float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude      *float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	VendorType     *string  `json:"vendor_type,omitempty" validate:"omitempty,max=100"`
	DeviceType     *string  `json:"device_type,omitempty" validate:"omitempty,oneof=camera nvr dvr switch"`
	Status         *string  `json:"status,omitempty" validate:"omitempty,oneof=ONLINE OFFLINE WARNING"`
	ConnectionType *string  `json:"connection_type,omitempty" validate:"omitempty,oneof=ip p2p snmp syslog alarm gb28181 onvif"`
	AssetClass     *string  `json:"asset_class,omitempty" validate:"omitempty,oneof=critical confidential internal public"`
	Manufacturer   *string  `json:"manufacturer,omitempty" validate:"omitempty,max=200"`
	SerialNumber   *string  `json:"serial_number,omitempty" validate:"omitempty,max=200"`
	MacAddress     *string  `json:"mac_address,omitempty" validate:"omitempty,mac"`
	FirmwareVersion *string `json:"firmware_version,omitempty" validate:"omitempty,max=50"`
	SiteID         *string  `json:"site_id,omitempty" validate:"omitempty,uuid"`
	P2PBrand       *string  `json:"p2p_brand,omitempty" validate:"omitempty,max=100"`
	P2PSerial      *string  `json:"p2p_serial,omitempty" validate:"omitempty,max=100"`
	UserAgent      *string  `json:"user_agent,omitempty" validate:"omitempty,max=500"`
	Health         *string  `json:"health,omitempty" validate:"omitempty,oneof=healthy faulty degraded"`
}

// ── DeviceListResponse (для GET /api/v1/devices) ───────────────────────

// DeviceListResponse — ответ со списком устройств и метаданными пагинации.
type DeviceListResponse struct {
	Devices      []DeviceSummary `json:"devices"`
	Total        int             `json:"total"`
	Page         int             `json:"page"`
	PageSize     int             `json:"page_size"`
	TotalPages   int             `json:"total_pages"`
}

// DeviceSummary — краткая информация об устройстве для списка.
// Не включает sensitive поля (OWASP ASVS V8 — Data Protection).
type DeviceSummary struct {
	DeviceID   string       `json:"device_id"`
	Name       string       `json:"name"`
	Location   string       `json:"location"`
	VendorType string       `json:"vendor_type"`
	DeviceType DeviceType   `json:"device_type"`
	Status     DeviceStatus `json:"status"`
	Health     HealthStatus `json:"health"`
	LastSeen   time.Time    `json:"last_seen"`
	SiteID     *string      `json:"site_id,omitempty"`
}

// ── ListDevicesFilter (для фильтрации и пагинации) ─────────────────────

type ListDevicesFilter struct {
	Page         int
	PageSize     int
	Status       string
	DeviceType   string
	VendorType   string
	SiteID       string
	Search       string // поиск по имени или device_id
	AssetClass   string
	WithDeleted  bool // включать soft-deleted записи
}

// DefaultPageSize — размер страницы по умолчанию.
const DefaultPageSize = 20

// MaxPageSize — максимальный размер страницы.
const MaxPageSize = 100

// ── Существующие модели (Alarm, Keepalive, ParsedLog, User, Prediction) ─

type AlarmPriority int

const (
	AlarmPriorityLow    AlarmPriority = 1
	AlarmPriorityMedium AlarmPriority = 2
	AlarmPriorityHigh   AlarmPriority = 3
)

type AlarmMethod int

const (
	AlarmMethodMotionDetection AlarmMethod = 1
	AlarmMethodVideoLoss       AlarmMethod = 5
	AlarmMethodEquipmentFault  AlarmMethod = 6
)

type Alarm struct {
	DeviceID    string        `json:"device_id"`
	Priority    AlarmPriority `json:"priority"`
	Method      AlarmMethod   `json:"method"`
	Timestamp   time.Time     `json:"timestamp"`
	Description string        `json:"description,omitempty"`
	ImagePath   string        `json:"image_path,omitempty"`
}

type Keepalive struct {
	DeviceID string
	Status   string
}

type ParsedLog struct {
	Time      time.Time `json:"time"`
	DeviceID  string    `json:"device_id"`
	LogLevel  string    `json:"log_level"`
	EventCode int       `json:"event_code"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	Raw       string    `json:"raw"`
}

type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	PasswordHash   string     `json:"-"`
	Role           string     `json:"role"`
	OwnerID        *string    `json:"owner_id,omitempty"`
	Email          string     `json:"email,omitempty"`
	Avatar         string     `json:"avatar,omitempty"`
	Sites          []string   `json:"sites,omitempty"`
	Status         string     `json:"status,omitempty"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	TOTPSecret     string     `json:"-"`
	TOTPEnabled    bool       `json:"totp_enabled"`
	TelegramChatID string     `json:"telegram_chat_id,omitempty"`
	TelegramAlerts bool       `json:"telegram_alerts"`
	Telegram2FA    bool       `json:"telegram_2fa"`
}

type Prediction struct {
	DeviceID           string    `json:"device_id"`
	PredictionDate     time.Time `json:"prediction_date"`
	FailureProbability float64   `json:"failure_probability"`
	Explanation        string    `json:"explanation"`
}
