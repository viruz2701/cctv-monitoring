// Package api — централизованная обработка HTTP-ошибок с traceID.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ── TraceID Middleware ──────────────────────────────────────────────────

type ctxKeyTraceID struct{}

// TraceIDMiddleware генерирует X-Request-ID для каждого запроса (ISO 27001 A.12.4).
// Принимает входящий X-Request-ID или создаёт новый.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Request-ID")
		if traceID == "" {
			traceID = generateTraceID()
		}
		w.Header().Set("X-Request-ID", traceID)
		ctx := context.WithValue(r.Context(), ctxKeyTraceID{}, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceIDFromContext извлекает traceID из контекста.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTraceID{}).(string); ok {
		return v
	}
	return "unknown"
}

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ── API Error ──────────────────────────────────────────────────────────

// APIError — типизированная ошибка API с HTTP-статусом и кодом.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
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

// ── respondError ───────────────────────────────────────────────────────

// respondError отправляет стандартизированный JSON-ответ об ошибке с traceID.
// Заменяет все http.Error(w, ...) в проекте.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr, ok := err.(*APIError)
	if !ok {
		apiErr = &APIError{
			Status:  http.StatusInternalServerError,
			Code:    ErrCodeInternal,
			Message: "internal server error",
			Err:     err,
		}
	}

	traceID := TraceIDFromContext(r.Context())
	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    apiErr.Code,
			"message": apiErr.Message,
		},
		"trace_id":  traceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if apiErr.Status >= 500 {
		slog.Error("api error",
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
