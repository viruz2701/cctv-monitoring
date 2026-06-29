// Package api — централизованная обработка HTTP-ошибок с traceID.
//
// Все типы и функции ошибок ре-экспортируются из пакета respond.
// Для новых пакетов используйте respond.RespondError напрямую.
package api

import (
	"context"

	"gb-telemetry-collector/internal/respond"
	"gb-telemetry-collector/internal/trace"
)

// ── TraceID Middleware (bridge to internal/trace) ───────────────────────
//
// Deprecated: новые пакеты должны использовать trace.Middleware.
var TraceIDMiddleware = trace.Middleware

// TraceIDFromContext извлекает traceID из контекста.
// Deprecated: новые пакеты должны использовать trace.FromContext.
func TraceIDFromContext(ctx context.Context) string {
	id := trace.FromContext(ctx)
	if id == "" {
		return "unknown"
	}
	return id
}

// ── API Error (ре-экспорт из respond) ──────────────────────────────────

// APIError — типизированная ошибка API с HTTP-статусом и кодом.
type APIError = respond.APIError

// Предопределённые коды ошибок.
const (
	ErrCodeValidation      = respond.ErrCodeValidation
	ErrCodeNotFound        = respond.ErrCodeNotFound
	ErrCodeUnauthorized    = respond.ErrCodeUnauthorized
	ErrCodeForbidden       = respond.ErrCodeForbidden
	ErrCodeConflict        = respond.ErrCodeConflict
	ErrCodeRateLimit       = respond.ErrCodeRateLimit
	ErrCodeInternal        = respond.ErrCodeInternal
	ErrCodeBadRequest      = respond.ErrCodeBadRequest
	ErrCodeExternalService = respond.ErrCodeExternalService
	ErrCodeQuotaExceeded   = respond.ErrCodeQuotaExceeded
)

// Конструкторы ошибок.
var NewValidationError = respond.NewValidationError
var NewNotFoundError = respond.NewNotFoundError
var NewUnauthorizedError = respond.NewUnauthorizedError
var NewForbiddenError = respond.NewForbiddenError
var NewConflictError = respond.NewConflictError
var NewBadRequestError = respond.NewBadRequestError
var NewRateLimitError = respond.NewRateLimitError
var NewExternalServiceError = respond.NewExternalServiceError
var NewInternalError = respond.NewInternalError

// RespondError отправляет стандартизированный JSON-ответ об ошибке с traceID.
// Заменяет все http.Error(w, ...) в проекте.
//
// Deprecated: новые пакеты должны использовать respond.RespondError.
var RespondError = respond.RespondError
