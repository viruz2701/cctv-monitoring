// Package api — helpers for input validation (OWASP ASVS V5).
//
// Соответствует:
//   - OWASP ASVS V5.1 (Input validation — whitelist approach)
//   - OWASP ASVS V5.2 (Sanitization — parameterized queries in repository layer)
//   - OWASP ASVS V5.3 (Encoding — JSON auto-escaping)
//   - СТБ 34.101.27 п. 6.2 (Контроль целостности входных данных)
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"gb-telemetry-collector/internal/auth"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ── Whitelist constants ────────────────────────────────────────────────

var (
	validDeviceTypes  = []string{"camera", "nvr", "dvr", "switch"}
	validStatuses     = []string{"ONLINE", "OFFLINE", "WARNING"}
	validConnTypes    = []string{"ip", "p2p", "snmp", "syslog", "alarm", "gb28181", "onvif"}
	validAssetClasses = []string{"critical", "confidential", "internal", "public"}
	validHealthStatus = []string{"healthy", "faulty", "degraded"}
	validWorkTypes    = []string{"installation", "maintenance", "repair", "inspection"}
	validPriorities   = []string{"low", "medium", "high", "critical"}
	validWOPriorities = []string{"low", "medium", "high", "critical"}
	validWOStatuses   = []string{"open", "assigned", "in_progress", "completed", "cancelled", "on_hold"}
)

// ── Structured Field Error (P1-SEC.3) ───────────────────────────────────

// FieldError представляет ошибку валидации для конкретного поля.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationErrors — коллекция ошибок валидации с поддержкой JSON.
type ValidationErrors struct {
	Fields []FieldError `json:"fields"`
}

func (ve *ValidationErrors) Error() string {
	msgs := make([]string, len(ve.Fields))
	for i, fe := range ve.Fields {
		msgs[i] = fe.Field + ": " + fe.Message
	}
	return strings.Join(msgs, "; ")
}

// Add добавляет ошибку поля.
func (ve *ValidationErrors) Add(field, message, code string) {
	ve.Fields = append(ve.Fields, FieldError{Field: field, Message: message, Code: code})
}

// Valid возвращает true если нет ошибок.
func (ve *ValidationErrors) Valid() bool {
	return len(ve.Fields) == 0
}

// ── Validator ──────────────────────────────────────────────────────────

// Validator provides simple struct field validation with structured errors.
// P1-SEC.3: Все ошибки содержат field-level информацию для inline error mapping.
type Validator struct {
	fieldErrors []FieldError
}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{fieldErrors: make([]FieldError, 0)}
}

// Required проверяет, что строка не пуста.
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: "required", Code: "REQUIRED",
		})
	}
	return v
}

// MinLength проверяет минимальную длину строки.
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: fmt.Sprintf("minimum length %d", min), Code: "MIN_LENGTH",
		})
	}
	return v
}

// MaxLength проверяет максимальную длину строки.
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: fmt.Sprintf("maximum length %d", max), Code: "MAX_LENGTH",
		})
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
	v.fieldErrors = append(v.fieldErrors, FieldError{
		Field: field, Message: fmt.Sprintf("must be one of [%s]", strings.Join(allowed, ", ")), Code: "INVALID_VALUE",
	})
	return v
}

// UUID regex
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// UUID проверяет, что строка является валидным UUID.
func (v *Validator) UUID(field, value string) *Validator {
	if value != "" && !uuidRegex.MatchString(value) {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: "invalid UUID format", Code: "INVALID_FORMAT",
		})
	}
	return v
}

// MAC проверяет, что строка является валидным MAC-адресом.
func (v *Validator) MAC(field, value string) *Validator {
	if value != "" {
		if _, err := net.ParseMAC(value); err != nil {
			v.fieldErrors = append(v.fieldErrors, FieldError{
				Field: field, Message: "invalid MAC address", Code: "INVALID_FORMAT",
			})
		}
	}
	return v
}

// RangeFloat проверяет, что число в диапазоне.
func (v *Validator) RangeFloat(field string, value float64, min, max float64) *Validator {
	if value < min || value > max {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: fmt.Sprintf("must be between %.0f and %.0f", min, max), Code: "OUT_OF_RANGE",
		})
	}
	return v
}

// Email проверяет формат email (простая проверка).
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func (v *Validator) Email(field, value string) *Validator {
	if value != "" && !emailRegex.MatchString(value) {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: "invalid email format", Code: "INVALID_FORMAT",
		})
	}
	return v
}

// IP проверяет формат IP-адреса.
func (v *Validator) IP(field, value string) *Validator {
	if value != "" && net.ParseIP(value) == nil {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: "invalid IP address", Code: "INVALID_FORMAT",
		})
	}
	return v
}

// Port проверяет, что порт в допустимом диапазоне.
func (v *Validator) Port(field string, value int) *Validator {
	if value < 1 || value > 65535 {
		v.fieldErrors = append(v.fieldErrors, FieldError{
			Field: field, Message: "port must be between 1 and 65535", Code: "OUT_OF_RANGE",
		})
	}
	return v
}

// Valid returns true if no validation errors.
func (v *Validator) Valid() bool {
	return len(v.fieldErrors) == 0
}

// Errors возвращает все ошибки полей.
func (v *Validator) Errors() []FieldError {
	return v.fieldErrors
}

// Error возвращает объединённое сообщение об ошибке.
func (v *Validator) Error() string {
	ve := &ValidationErrors{Fields: v.fieldErrors}
	return ve.Error()
}

// ToValidationErrors преобразует ошибки валидатора в ValidationErrors.
func (v *Validator) ToValidationErrors() *ValidationErrors {
	return &ValidationErrors{Fields: v.fieldErrors}
}

// ── respondValidationError — отправка структурированной ошибки валидации ──

// respondValidationError отправляет 400 с структурированными field-level ошибками.
// P1-SEC.3: Формат позволяет фронтенду мапить ошибки к полям формы (inline error mapping).
func respondValidationError(w http.ResponseWriter, r *http.Request, ve *ValidationErrors) {
	traceID := traceFromRequest(r)

	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    ErrCodeValidation,
			"message": "validation failed",
			"fields":  ve.Fields,
		},
		"trace_id":  traceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(resp)
}

// traceFromRequest извлекает traceID из контекста запроса.
func traceFromRequest(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	return TraceIDFromContext(r.Context())
}

// ── Rate Limiting для failed validations (P1-SEC.3) ─────────────────────

// validationRateLimiter отслеживает failed validation попытки по IP.
// Блокирует IP после N failed validations за период.
type validationRateLimiter struct {
	mu        sync.Mutex
	attempts  map[string]*validationAttempt
	threshold int           // максимальное число failed validations
	window    time.Duration // временное окно
	banTime   time.Duration // время блокировки
}

type validationAttempt struct {
	count    int
	firstTry time.Time
	banUntil time.Time
}

var defaultValidationRateLimiter = &validationRateLimiter{
	attempts:  make(map[string]*validationAttempt),
	threshold: 20,               // 20 failed validations
	window:    5 * time.Minute,  // за 5 минут
	banTime:   15 * time.Minute, // блокировка на 15 минут
}

// checkValidationRateLimit проверяет, не превышен ли лимит failed validations для IP.
func (rl *validationRateLimiter) check(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.attempts[ip]

	if !exists {
		rl.attempts[ip] = &validationAttempt{count: 1, firstTry: now}
		return true
	}

	// Проверяем, не забанен ли IP
	if !entry.banUntil.IsZero() && now.Before(entry.banUntil) {
		return false
	}

	// Сброс если окно истекло
	if now.Sub(entry.firstTry) > rl.window {
		entry.count = 1
		entry.firstTry = now
		entry.banUntil = time.Time{}
		return true
	}

	entry.count++

	if entry.count > rl.threshold {
		entry.banUntil = now.Add(rl.banTime)
		return false
	}

	return true
}

// ValidationRateLimitMiddleware — middleware для rate limiting failed validations.
func ValidationRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Пропускаем safe методы
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		ip := auth.ClientIP(r)
		if !defaultValidationRateLimiter.check(ip) {
			RespondError(w, r, NewRateLimitError("too many invalid requests. Try again later."))
			return
		}
		next.ServeHTTP(w, r)
	})
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
	if req.ParentDeviceID != nil {
		v.UUID("parent_device_id", *req.ParentDeviceID)
	}

	if !v.Valid() {
		return v.ToValidationErrors()
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
	if req.ParentDeviceID != nil {
		v.UUID("parent_device_id", *req.ParentDeviceID)
	}

	if !v.Valid() {
		return v.ToValidationErrors()
	}
	return nil
}

// ── Domain Validators (P1-SEC.3) ────────────────────────────────────────

// validateWorkOrderRequest проверяет запрос на создание/обновление Work Order.
func validateWorkOrderRequest(title, workType, priority, description string) error {
	v := NewValidator()
	v.Required("title", title).
		MinLength("title", title, 3).
		MaxLength("title", title, 200).
		Required("work_type", workType).
		OneOf("work_type", workType, validWorkTypes).
		Required("priority", priority).
		OneOf("priority", priority, validPriorities).
		Required("description", description).
		MinLength("description", description, 10).
		MaxLength("description", description, 5000)

	if !v.Valid() {
		return v.ToValidationErrors()
	}
	return nil
}

// validateSiteRequest проверяет запрос на создание/обновление Site.
func validateSiteRequest(name, address, city string) error {
	v := NewValidator()
	v.Required("name", name).
		MinLength("name", name, 2).
		MaxLength("name", name, 200).
		Required("address", address).
		MaxLength("address", address, 500).
		Required("city", city).
		MaxLength("city", city, 100)

	if !v.Valid() {
		return v.ToValidationErrors()
	}
	return nil
}

// validateLoginRequest проверяет запрос логина.
func validateLoginRequest(username, password string) error {
	v := NewValidator()
	v.Required("username", username).
		MaxLength("username", username, 100).
		Required("password", password).
		MaxLength("password", password, 256)

	if !v.Valid() {
		return v.ToValidationErrors()
	}
	return nil
}

// formatValidationError преобразует ошибку в читаемое сообщение.
func formatValidationError(err error) string {
	if err == nil {
		return "validation failed"
	}
	var ve *ValidationErrors
	if errors.As(err, &ve) {
		return ve.Error()
	}
	return err.Error()
}

// ── Internal request field structs (для валидации) ─────────────────────

type createDeviceRequestFields struct {
	DeviceID        string
	Name            string
	Location        string
	Latitude        float64
	Longitude       float64
	VendorType      string
	DeviceType      string
	Status          string
	ConnectionType  string
	AssetClass      string
	Manufacturer    string
	SerialNumber    string
	MacAddress      string
	FirmwareVersion string
	SiteID          *string
	P2PBrand        string
	P2PSerial       string
	UserAgent       string
	ParentDeviceID  *string
}

type updateDeviceRequestFields struct {
	Name            *string
	Location        *string
	Latitude        *float64
	Longitude       *float64
	VendorType      *string
	DeviceType      *string
	Status          *string
	ConnectionType  *string
	AssetClass      *string
	Manufacturer    *string
	SerialNumber    *string
	MacAddress      *string
	FirmwareVersion *string
	SiteID          *string
	P2PBrand        *string
	P2PSerial       *string
	UserAgent       *string
	Health          *string
	ParentDeviceID  *string
}
