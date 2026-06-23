// Package api — helpers for input validation (OWASP ASVS V5).
//
// Соответствует:
//   - OWASP ASVS V5.1 (Input validation — whitelist approach)
//   - OWASP ASVS V5.2 (Sanitization — parameterized queries in repository layer)
//   - OWASP ASVS V5.3 (Encoding — JSON auto-escaping)
//   - СТБ 34.101.27 п. 6.2 (Контроль целостности входных данных)
package api

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

// ── Whitelist constants ────────────────────────────────────────────────

var (
	validDeviceTypes  = []string{"camera", "nvr", "dvr", "switch"}
	validStatuses     = []string{"ONLINE", "OFFLINE", "WARNING"}
	validConnTypes    = []string{"ip", "p2p", "snmp", "syslog", "alarm", "gb28181", "onvif"}
	validAssetClasses = []string{"critical", "confidential", "internal", "public"}
	validHealthStatus = []string{"healthy", "faulty", "degraded"}
)

// ── Validator ──────────────────────────────────────────────────────────

// Validator provides simple struct field validation.
// Заменяет github.com/go-playground/validator для избежания лишних зависимостей.
type Validator struct {
	errors []string
}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{errors: make([]string, 0)}
}

// Required проверяет, что строка не пуста.
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, fmt.Sprintf("%s: required", field))
	}
	return v
}

// MinLength проверяет минимальную длину строки.
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.errors = append(v.errors, fmt.Sprintf("%s: minimum length %d", field, min))
	}
	return v
}

// MaxLength проверяет максимальную длину строки.
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.errors = append(v.errors, fmt.Sprintf("%s: maximum length %d", field, max))
	}
	return v
}

// OneOf проверяет, что значение входит в список разрешённых (whitelist — OWASP ASVS V5.1).
func (v *Validator) OneOf(field, value string, allowed []string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errors = append(v.errors, fmt.Sprintf("%s: must be one of [%s]", field, strings.Join(allowed, ", ")))
	return v
}

// UUID regex
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// UUID проверяет, что строка является валидным UUID.
func (v *Validator) UUID(field, value string) *Validator {
	if value != "" && !uuidRegex.MatchString(value) {
		v.errors = append(v.errors, fmt.Sprintf("%s: invalid UUID format", field))
	}
	return v
}

// MAC проверяет, что строка является валидным MAC-адресом.
func (v *Validator) MAC(field, value string) *Validator {
	if value != "" {
		if _, err := net.ParseMAC(value); err != nil {
			v.errors = append(v.errors, fmt.Sprintf("%s: invalid MAC address", field))
		}
	}
	return v
}

// RangeFloat проверяет, что число в диапазоне.
func (v *Validator) RangeFloat(field string, value float64, min, max float64) *Validator {
	if value < min || value > max {
		v.errors = append(v.errors, fmt.Sprintf("%s: must be between %.0f and %.0f", field, min, max))
	}
	return v
}

// Valid returns true if no validation errors.
func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

// Errors returns all validation errors.
func (v *Validator) Errors() []string {
	return v.errors
}

// Error returns combined error message.
func (v *Validator) Error() string {
	return strings.Join(v.errors, "; ")
}

// ── Validation functions ───────────────────────────────────────────────

// validateCreateDeviceRequest проверяет CreateDeviceRequest (OWASP ASVS V5 — whitelist).
func validateCreateDeviceRequest(req *createDeviceRequestFields) error {
	v := NewValidator()

	v.Required("device_id", req.DeviceID).
		UUID("device_id", req.DeviceID).
		Required("name", req.Name).
		MinLength("name", req.Name, 1).
		MaxLength("name", req.Name, 255).
		MaxLength("location", req.Location, 500).
		RangeFloat("latitude", req.Latitude, -90, 90).
		RangeFloat("longitude", req.Longitude, -180, 180).
		MaxLength("vendor_type", req.VendorType, 100).
		Required("device_type", req.DeviceType).
		OneOf("device_type", req.DeviceType, validDeviceTypes).
		Required("status", req.Status).
		OneOf("status", req.Status, validStatuses).
		Required("connection_type", req.ConnectionType).
		OneOf("connection_type", req.ConnectionType, validConnTypes).
		Required("asset_class", req.AssetClass).
		OneOf("asset_class", req.AssetClass, validAssetClasses).
		MaxLength("manufacturer", req.Manufacturer, 200).
		MaxLength("serial_number", req.SerialNumber, 200).
		MAC("mac_address", req.MacAddress).
		MaxLength("firmware_version", req.FirmwareVersion, 50).
		MaxLength("p2p_brand", req.P2PBrand, 100).
		MaxLength("p2p_serial", req.P2PSerial, 100).
		MaxLength("user_agent", req.UserAgent, 500)

	if req.SiteID != nil {
		v.UUID("site_id", *req.SiteID)
	}

	if !v.Valid() {
		return errors.New(v.Error())
	}
	return nil
}

// validateUpdateDeviceRequest проверяет UpdateDeviceRequest (частичное обновление).
func validateUpdateDeviceRequest(req *updateDeviceRequestFields) error {
	v := NewValidator()

	if req.Name != nil {
		v.MinLength("name", *req.Name, 1).MaxLength("name", *req.Name, 255)
	}
	if req.Location != nil {
		v.MaxLength("location", *req.Location, 500)
	}
	if req.Latitude != nil {
		v.RangeFloat("latitude", *req.Latitude, -90, 90)
	}
	if req.Longitude != nil {
		v.RangeFloat("longitude", *req.Longitude, -180, 180)
	}
	if req.VendorType != nil {
		v.MaxLength("vendor_type", *req.VendorType, 100)
	}
	if req.DeviceType != nil {
		v.OneOf("device_type", *req.DeviceType, validDeviceTypes)
	}
	if req.Status != nil {
		v.OneOf("status", *req.Status, validStatuses)
	}
	if req.ConnectionType != nil {
		v.OneOf("connection_type", *req.ConnectionType, validConnTypes)
	}
	if req.AssetClass != nil {
		v.OneOf("asset_class", *req.AssetClass, validAssetClasses)
	}
	if req.Health != nil {
		v.OneOf("health", *req.Health, validHealthStatus)
	}
	if req.Manufacturer != nil {
		v.MaxLength("manufacturer", *req.Manufacturer, 200)
	}
	if req.SerialNumber != nil {
		v.MaxLength("serial_number", *req.SerialNumber, 200)
	}
	if req.MacAddress != nil {
		v.MAC("mac_address", *req.MacAddress)
	}
	if req.FirmwareVersion != nil {
		v.MaxLength("firmware_version", *req.FirmwareVersion, 50)
	}
	if req.SiteID != nil {
		v.UUID("site_id", *req.SiteID)
	}
	if req.P2PBrand != nil {
		v.MaxLength("p2p_brand", *req.P2PBrand, 100)
	}
	if req.P2PSerial != nil {
		v.MaxLength("p2p_serial", *req.P2PSerial, 100)
	}
	if req.UserAgent != nil {
		v.MaxLength("user_agent", *req.UserAgent, 500)
	}

	if !v.Valid() {
		return errors.New(v.Error())
	}
	return nil
}

// formatValidationError преобразует ошибку в читаемое сообщение.
func formatValidationError(err error) string {
	if err == nil {
		return "validation failed"
	}
	return err.Error()
}

// ── Internal request field structs (для валидации) ─────────────────────

type createDeviceRequestFields struct {
	DeviceID       string
	Name           string
	Location       string
	Latitude       float64
	Longitude      float64
	VendorType     string
	DeviceType     string
	Status         string
	ConnectionType string
	AssetClass     string
	Manufacturer   string
	SerialNumber   string
	MacAddress     string
	FirmwareVersion string
	SiteID         *string
	P2PBrand       string
	P2PSerial      string
	UserAgent      string
}

type updateDeviceRequestFields struct {
	Name           *string
	Location       *string
	Latitude       *float64
	Longitude      *float64
	VendorType     *string
	DeviceType     *string
	Status         *string
	ConnectionType *string
	AssetClass     *string
	Manufacturer   *string
	SerialNumber   *string
	MacAddress     *string
	FirmwareVersion *string
	SiteID         *string
	P2PBrand       *string
	P2PSerial      *string
	UserAgent      *string
	Health         *string
}
