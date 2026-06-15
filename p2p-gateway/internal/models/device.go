package models

type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "online"
	StatusOffline DeviceStatus = "offline"
	StatusUnknown DeviceStatus = "unknown"
)

type Device struct {
	ID           string       `json:"id"`
	Brand        string       `json:"brand"`
	Serial       string       `json:"serial"`
	Username     string       `json:"username,omitempty"`
	Password     string       `json:"password,omitempty"`
	SecurityCode string       `json:"security_code,omitempty"`
	ProxyPort    int          `json:"proxy_port"`
	RTSPURL      string       `json:"rtsp_url"`
	Status       DeviceStatus `json:"status"`
	LastSeen     string       `json:"last_seen,omitempty"`
}
