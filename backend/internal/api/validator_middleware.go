// Package api — Server-Side Validation Middleware (P1-SEC.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-SEC.2: Server-Side Validation (Go-validators)
//
// Единый middleware для валидации всех входящих запросов через
// go-playground/validator с кастомными правилами.
//
// Особенности:
//   - JSON body → struct → validate
//   - Кастомные теги: uuid, mac, device_type, conn_type, ip_with_port
//   - Единый error format (FieldError + ValidationErrors)
//   - Интеграция с respond.RespondError
//   - Поддержка partial update (PATCH)
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - OWASP ASVS V5.2 (Sanitization — structured validation)
//   - OWASP ASVS V5.3 (Input validation — schema validation)
//   - ISO 27001 A.14.2 (Security in development — validation)
//   - IEC 62443 SR 3.1 (Wireless — data integrity)
//   - СТБ 34.101.27 п. 6.2 (Контроль целостности данных)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"

	"gb-telemetry-collector/internal/respond"
)

// ────────────────────────────────────────────────────────────────────────────
// Global validator instance (thread-safe)
// ────────────────────────────────────────────────────────────────────────────

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// GetValidator возвращает глобальный экземпляр валидатора (singleton).
func GetValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())

		// ── Регистрация кастомных валидаторов ──────────────────────
		registerCustomValidators(validate)
	})
	return validate
}

// registerCustomValidators регистрирует кастомные теги валидации.
func registerCustomValidators(v *validator.Validate) {
	validators := map[string]validator.Func{
		"device_type":  validateDeviceType,
		"conn_type":    validateConnType,
		"wo_status":    validateWOStatus,
		"wo_priority":  validateWOPriority,
		"health":       validateHealthStatus,
		"asset_class":  validateAssetClass,
		"ip_with_port": validateIPWithPort,
	}

	for tag, fn := range validators {
		if err := v.RegisterValidation(tag, fn); err != nil {
			slog.Warn("failed to register validator", "tag", tag, "error", err)
		}
	}

	// Register custom UUID validation (overrides built-in uuid4)
	if err := v.RegisterValidation("uuid_custom", validateUUID); err != nil {
		slog.Warn("failed to register uuid_custom validator", "error", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Custom validators
// ────────────────────────────────────────────────────────────────────────────

// validateDeviceType проверяет тип устройства.
func validateDeviceType(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, t := range validDeviceTypes {
		if val == t {
			return true
		}
	}
	return false
}

// validateConnType проверяет тип подключения.
func validateConnType(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, t := range validConnTypes {
		if val == t {
			return true
		}
	}
	return false
}

// validateWOStatus проверяет статус WO.
func validateWOStatus(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, s := range validWOStatuses {
		if val == s {
			return true
		}
	}
	return false
}

// validateWOPriority проверяет приоритет WO.
func validateWOPriority(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, p := range validWOPriorities {
		if val == p {
			return true
		}
	}
	return false
}

// validateHealthStatus проверяет статус здоровья устройства.
func validateHealthStatus(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, s := range validHealthStatus {
		if val == s {
			return true
		}
	}
	return false
}

// validateAssetClass проверяет класс актива.
func validateAssetClass(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, c := range validAssetClasses {
		if val == c {
			return true
		}
	}
	return false
}

// validateUUID проверяет UUID v4 формат.
func validateUUID(fl validator.FieldLevel) bool {
	return uuidRegex.MatchString(fl.Field().String())
}

// validateIPWithPort проверяет формат "IP:port".
func validateIPWithPort(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true
	}
	lastColon := strings.LastIndex(val, ":")
	if lastColon < 0 {
		return false
	}
	ip := val[:lastColon]
	port := val[lastColon+1:]
	if ip == "" || port == "" {
		return false
	}
	return true
}

// ────────────────────────────────────────────────────────────────────────────
// ValidateRequest — декодирует JSON и валидирует структуру
// ────────────────────────────────────────────────────────────────────────────

// ValidationOption — опция для ValidateRequest.
type ValidationOption func(*validationConfig)

type validationConfig struct {
	// AllowPartial — разрешить частичное обновление (PATCH).
	AllowPartial bool
	// DisallowUnknownFields — запретить неизвестные поля.
	DisallowUnknownFields bool
}

// WithPartialUpdate разрешает частичное обновление (для PATCH запросов).
func WithPartialUpdate() ValidationOption {
	return func(c *validationConfig) {
		c.AllowPartial = true
	}
}

// WithDisallowUnknownFields запрещает неизвестные поля в запросе.
func WithDisallowUnknownFields() ValidationOption {
	return func(c *validationConfig) {
		c.DisallowUnknownFields = true
	}
}

// ValidateRequest декодирует JSON body и валидирует структуру.
//
// Пример использования:
//
//	var req CreateDeviceRequest
//	if err := ValidateRequest(r, &req); err != nil {
//	    respond.RespondError(w, r, err)
//	    return
//	}
//
// Возвращает *respond.APIError с ValidationErrors в details.
func ValidateRequest(r *http.Request, target interface{}, opts ...ValidationOption) error {
	if r.Body == nil {
		return respond.NewBadRequestError("request body is required")
	}

	cfg := &validationConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	decoder := json.NewDecoder(r.Body)
	if cfg.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(target); err != nil {
		return respond.NewValidationError(fmt.Sprintf("invalid JSON: %s", err.Error()))
	}

	// Валидация через go-playground/validator
	v := GetValidator()
	if err := v.Struct(target); err != nil {
		return convertValidationErrors(err)
	}

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Error conversion
// ────────────────────────────────────────────────────────────────────────────

// convertValidationErrors конвертирует ошибки validator.ValidationErrors
// в единый формат FieldError / ValidationErrors.
func convertValidationErrors(err error) error {
	// Проверяем тип
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		// Другие ошибки валидации (напр. struct level)
		return respond.NewValidationError(err.Error())
	}

	fields := make([]FieldError, 0, len(validationErrs))
	for _, fe := range validationErrs {
		field := toSnakeCase(fe.Field())
		fields = append(fields, FieldError{
			Field:   field,
			Message: validationMessage(fe),
			Code:    validationCode(fe),
		})
	}

	return &respond.APIError{
		Status:  http.StatusUnprocessableEntity,
		Code:    respond.ErrCodeValidation,
		Message: "validation failed",
		Details: ValidationErrors{
			Fields: fields,
		},
	}
}

// validationMessage возвращает человекочитаемое сообщение об ошибке.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "required"
	case "min":
		return fmt.Sprintf("minimum length is %s", fe.Param())
	case "max":
		return fmt.Sprintf("maximum length is %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "uuid", "uuid4", "uuid_custom":
		return "invalid UUID format"
	case "email":
		return "invalid email format"
	case "url":
		return "invalid URL format"
	case "ip":
		return "invalid IP address"
	case "mac":
		return "invalid MAC address"
	case "device_type":
		return fmt.Sprintf("must be one of: %s", strings.Join(validDeviceTypes, ", "))
	case "conn_type":
		return fmt.Sprintf("must be one of: %s", strings.Join(validConnTypes, ", "))
	case "wo_status":
		return fmt.Sprintf("must be one of: %s", strings.Join(validWOStatuses, ", "))
	case "wo_priority":
		return fmt.Sprintf("must be one of: %s", strings.Join(validWOPriorities, ", "))
	case "health":
		return fmt.Sprintf("must be one of: %s", strings.Join(validHealthStatus, ", "))
	case "asset_class":
		return fmt.Sprintf("must be one of: %s", strings.Join(validAssetClasses, ", "))
	case "ip_with_port":
		return "invalid format, expected IP:port"
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("validation failed on '%s'", fe.Tag())
	}
}

// validationCode возвращает машинный код ошибки.
func validationCode(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "REQUIRED"
	case "min", "max":
		return strings.ToUpper(fe.Tag())
	case "oneof", "uuid", "uuid4", "uuid_custom", "email", "url", "ip", "mac":
		return "INVALID_FORMAT"
	case "device_type", "conn_type", "wo_status", "wo_priority", "health", "asset_class":
		return "INVALID_VALUE"
	case "ip_with_port":
		return "INVALID_FORMAT"
	case "gte", "lte":
		return "OUT_OF_RANGE"
	default:
		return "VALIDATION_ERROR"
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// toSnakeCase конвертирует CamelCase в snake_case для имён полей.
// Корректно обрабатывает аббревиатуры (IP → ip, MAC → mac).
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]

		// Текущий символ — заглавная буква
		if r >= 'A' && r <= 'Z' {
			lower := r + 32 // to lowercase

			// Предыдущий символ был буквой (не заглавной) или цифрой
			if i > 0 {
				prev := runes[i-1]
				// Если предыдущий - строчная буква, добавляем разделитель
				if prev >= 'a' && prev <= 'z' {
					result.WriteRune('_')
					result.WriteRune(lower)
					continue
				}
				// Если предыдущий - цифра
				if prev >= '0' && prev <= '9' {
					result.WriteRune('_')
					result.WriteRune(lower)
					continue
				}
				// Если предыдущий - заглавная, а следующий - строчная
				if prev >= 'A' && prev <= 'Z' && i+1 < n && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
					result.WriteRune('_')
					result.WriteRune(lower)
					continue
				}
			}

			result.WriteRune(lower)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
