// Package respond — централизованная обработка HTTP-ошибок с traceID.
//
// Единый пакет для JSON-ответов с ошибками во всём проекте.
// Заменяет все http.Error(w, ...) в проекте.
// Все пакеты ДОЛЖНЫ использовать respond.RespondError или respond.Error.
//
// Соответствие:
//   - OWASP ASVS V7.1.1: Стандартизированный формат ответов
//   - ISO 27001 A.12.4.1: Логирование ошибок с trace_id
//   - IEC 62443-3-3 SR 3.1: Отсутствие утечки информации
package respond

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/trace"
)

// ── API Error ──────────────────────────────────────────────────────────

// APIError — типизированная ошибка API с HTTP-статусом и кодом.
type APIError struct {
	Status  int         `json:"-"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Err     error       `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap реализует errors.Unwrap.
func (e *APIError) Unwrap() error { return e.Err }

// Предопределённые коды ошибок.
const (
	ErrCodeValidation      = "VALIDATION_ERROR"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeForbidden       = "FORBIDDEN"
	ErrCodeConflict        = "CONFLICT"
	ErrCodeRateLimit       = "RATE_LIMIT"
	ErrCodeInternal        = "INTERNAL_ERROR"
	ErrCodeBadRequest      = "BAD_REQUEST"
	ErrCodeExternalService = "EXTERNAL_SERVICE_ERROR"
)

// Конструкторы ошибок.
func NewValidationError(msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: ErrCodeValidation, Message: msg}
}

func NewNotFoundError(msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Code: ErrCodeNotFound, Message: msg}
}

func NewUnauthorizedError(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: ErrCodeUnauthorized, Message: msg}
}

func NewForbiddenError(msg string) *APIError {
	return &APIError{Status: http.StatusForbidden, Code: ErrCodeForbidden, Message: msg}
}

func NewConflictError(msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: ErrCodeConflict, Message: msg}
}

func NewBadRequestError(msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: ErrCodeBadRequest, Message: msg}
}

func NewRateLimitError(msg string) *APIError {
	return &APIError{Status: http.StatusTooManyRequests, Code: ErrCodeRateLimit, Message: msg}
}

func NewExternalServiceError(msg string) *APIError {
	return &APIError{Status: http.StatusBadGateway, Code: ErrCodeExternalService, Message: msg}
}

func NewInternalError(msg string, err error) *APIError {
	return &APIError{Status: http.StatusInternalServerError, Code: ErrCodeInternal, Message: msg, Err: err}
}

// ── RespondError ───────────────────────────────────────────────────────

// RespondError отправляет стандартизированный JSON-ответ об ошибке с traceID.
// Заменяет все http.Error(w, ...) в проекте.
//
// Принимает любую ошибку; если она имеет тип *APIError, использует её
// статус и код. Иначе — 500 Internal Server Error.
func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = &APIError{
			Status:  http.StatusInternalServerError,
			Code:    ErrCodeInternal,
			Message: "internal server error",
			Err:     err,
		}
	}

	traceID := trace.FromContext(r.Context())

	errorBody := map[string]interface{}{
		"code":    apiErr.Code,
		"message": apiErr.Message,
	}
	if apiErr.Details != nil {
		errorBody["details"] = apiErr.Details
	}

	resp := map[string]interface{}{
		"error":     errorBody,
		"trace_id":  traceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if apiErr.Status >= 500 {
		slog.Error("respond error",
			"trace_id", traceID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", apiErr.Status,
			"code", apiErr.Code,
			"error", apiErr.Error(),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	_ = json.NewEncoder(w).Encode(resp)
}

// ── Legacy Helpers ─────────────────────────────────────────────────────

// Error отправляет JSON-ответ с ошибкой и логирует её.
// Если status >= 500, ошибка логируется как server error.
//
// Deprecated: используйте RespondError с конструкторами APIError.
func Error(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := map[string]string{"error": msg}
	_ = json.NewEncoder(w).Encode(resp)

	if status >= 500 {
		slog.Error("respond.Error",
			"status", status,
			"error", msg,
		)
	}
}

// Errorf отправляет JSON-ответ с форматированной ошибкой.
//
// Deprecated: используйте RespondError с конструкторами APIError.
func Errorf(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Error(w, status, msg)
}

// OK отправляет JSON-ответ об успехе.
func OK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
