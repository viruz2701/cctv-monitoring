package models

import (
	"net"
	"time"
)

type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "ONLINE"
	StatusOffline DeviceStatus = "OFFLINE"
)

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
	// другие можно добавить
)

type Device struct {
	DeviceID          string       `json:"device_id"`
	Status            DeviceStatus `json:"status"`
	LastSeen          time.Time    `json:"last_seen"`
	RegisteredAt      time.Time    `json:"registered_at"`
	ContactAddr       *net.UDPAddr `json:"-"` // NAT-адрес для ответов
	HeartbeatInterval int          `json:"heartbeat_interval"`
	UserAgent         string       `json:"user_agent"`
	LastAlarm         *Alarm       `json:"last_alarm,omitempty"`
	LastError         string       `json:"last_error,omitempty"`
	OwnerID           *string      `json:"owner_id,omitempty"` // для будущей ролевой модели
	Name              string       `json:"name"`
	Location          string       `json:"location"`
	VendorType        string       `json:"vendor_type"`
	P2PBrand          string       `json:"p2p_brand,omitempty"`
	P2PSerial         string       `json:"p2p_serial,omitempty"`
	CloudStatus       string       `json:"cloud_status,omitempty"`
}

type Alarm struct {
	DeviceID    string        `json:"device_id"`
	Priority    AlarmPriority `json:"priority"`
	Method      AlarmMethod   `json:"method"`
	Timestamp   time.Time     `json:"timestamp"`
	Description string        `json:"description,omitempty"`
	ImagePath   string        `json:"image_path,omitempty"` // путь к сохранённому изображению (если есть)
}

type Keepalive struct {
	DeviceID string
	Status   string // "OK" или другое
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
