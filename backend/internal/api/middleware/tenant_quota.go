// Package middleware — Tenant Quota Middleware (P1-QUOTA).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-QUOTA: Tenant Quota Middleware
//
// Проверяет квоты на мутирующих запросах (POST, PUT, DELETE, PATCH).
// Пропускает GET/HEAD/OPTIONS запросы (read-only).
//
// Soft limit (80%): только warning header (X-Quota-Warning)
// Hard limit (100%): blocks the request
// Grace period: не блокирует в течение grace периода
//
// Соответствует:
//   - OWASP ASVS V2.2.1 (Rate limiting)
//   - ISO 27001 A.12.1.2 (Capacity management)
//   - IEC 62443-3-3 SR 3.1 (Resource management)
//
// ═══════════════════════════════════════════════════════════════════════════
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/respond"
	"gb-telemetry-collector/internal/tenant"
)

// ── Context keys ─────────────────────────────────────────────────────

// QuotaWarningContextKey — ключ контекста для X-Quota-Warning header.
const QuotaWarningContextKey contextKey = "quota-warning"

// QuotaCheckedContextKey — ключ контекста для флага "quota проверена".
const QuotaCheckedContextKey contextKey = "quota-checked"

// ── Middleware ───────────────────────────────────────────────────────

// QuotaMiddlewareConfig — конфигурация QuotaMiddleware.
type QuotaMiddlewareConfig struct {
	QuotaManager *tenant.QuotaManager
	QuotaType    tenant.QuotaType            // тип квоты для проверки на этом маршруте
	MethodMap    map[string]tenant.QuotaType // HTTP method -> quota type
}

// QuotaMiddleware создаёт middleware для проверки квот.
//
// Параметры:
//   - qm: QuotaManager (nil-safe — пропускает все запросы)
//   - qt: тип квоты для проверки на этом маршруте
//
// Если QuotaManager == nil, middleware пропускает все запросы (fail-open).
//
// Параметр qt:
//   - Указать конкретный тип (tenant.QuotaDevices) для явной проверки
//   - Оставить пустым ("") для авто-определения из HTTP метода и пути
func QuotaMiddleware(qm *tenant.QuotaManager, qt tenant.QuotaType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fail-open: если QuotaManager не настроен, пропускаем
			if qm == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Read-only методы не проверяем
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// ── V2: Authentication ──
			tenantID := auth.GetTenantID(r)
			if tenantID == "" {
				next.ServeHTTP(w, r) // middleware не блокирует — TenantMiddleware должен быть включён
				return
			}

			// Авто-определение типа квоты, если не задан явно
			quotaType := qt
			if quotaType == "" {
				quotaType = QuotaTypeFromMethod(r.Method, r.URL.Path)
				if quotaType == "" {
					// Не удалось определить тип — проверяем api_calls (универсальная квота)
					quotaType = tenant.QuotaAPICalls
				}
			}

			// Проверяем квоту
			status, err := qm.Check(r.Context(), tenantID, quotaType)
			if err != nil {
				// Fail-open: при ошибке Redis пропускаем
				next.ServeHTTP(w, r)
				return
			}

			// ── Soft limit warning ──
			if status.IsSoft && !status.IsHard {
				w.Header().Set("X-Quota-Warning", fmt.Sprintf(
					"%s quota at %.0f%% (soft limit: %d)",
					qt, status.Usage, status.SoftLimit,
				))
				// Сохраняем warning в контекст
				ctx := context.WithValue(r.Context(), QuotaWarningContextKey, status)
				ctx = context.WithValue(ctx, QuotaCheckedContextKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// ── Hard limit — блокируем (если не на grace period) ──
			if status.IsHard && !status.OnGrace {
				w.Header().Set("X-Quota-Blocked", "true")
				w.Header().Set("Retry-After", "3600") // предлагаем повторить через час

				respond.RespondError(w, r, respond.NewQuotaExceededError(
					fmt.Sprintf("%s quota exceeded: %d/%d (hard limit)", quotaType, status.Current, status.HardLimit),
				))
				return
			}

			// ── Hard limit, но на grace period ──
			if status != nil && status.IsHard && status.OnGrace {
				w.Header().Set("X-Quota-Grace", "true")
				ctx := context.WithValue(r.Context(), QuotaCheckedContextKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// ── Всё в порядке ──
			ctx := context.WithValue(r.Context(), QuotaCheckedContextKey, true)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// QuotaWarningFromContext извлекает X-Quota-Warning из контекста.
func QuotaWarningFromContext(ctx context.Context) *tenant.QuotaStatus {
	status, _ := ctx.Value(QuotaWarningContextKey).(*tenant.QuotaStatus)
	return status
}

// QuotaCheckedFromContext проверяет, была ли проведена проверка квоты.
func QuotaCheckedFromContext(ctx context.Context) bool {
	checked, _ := ctx.Value(QuotaCheckedContextKey).(bool)
	return checked
}

// ── Utilities ────────────────────────────────────────────────────────

// QuotaTypeFromMethod определяет тип квоты по HTTP методу и пути.
//
// Пример:
//
//	POST /api/v1/devices → QuotaDevices
//	POST /api/v1/users → QuotaUsers
//	POST /api/v1/work-orders → QuotaWorkOrders
//	любой запрос → QuotaAPICalls (учитывается отдельно)
func QuotaTypeFromMethod(method, path string) tenant.QuotaType {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return "" // read-only, не проверяем
	}

	// Определяем тип ресурса из пути
	switch {
	case strings.Contains(path, "/devices"):
		return tenant.QuotaDevices
	case strings.Contains(path, "/users"):
		return tenant.QuotaUsers
	case strings.Contains(path, "/work-orders"), strings.Contains(path, "/work_orders"):
		return tenant.QuotaWorkOrders
	case strings.Contains(path, "/storage"):
		return tenant.QuotaStorage
	default:
		return "" // не удалось определить
	}
}

// ── Response helpers ─────────────────────────────────────────────────

// QuotaExceededErrorResponse — стандартный ответ при превышении квоты.
type QuotaExceededErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	RetryIn int    `json:"retry_in_seconds"`
}

// QuotaWarningResponse — стандартный ответ при приближении к лимиту.
type QuotaWarningResponse struct {
	Warning     string  `json:"warning"`
	QuotaType   string  `json:"quota_type"`
	Usage       float64 `json:"usage_percent"`
	Current     int64   `json:"current"`
	SoftLimit   int64   `json:"soft_limit"`
	HardLimit   int64   `json:"hard_limit"`
	Recommended string  `json:"recommended_action"`
}

// NewQuotaWarning создаёт предупреждение о soft limit.
func NewQuotaWarning(status *tenant.QuotaStatus) QuotaWarningResponse {
	return QuotaWarningResponse{
		Warning:     fmt.Sprintf("%s quota is near limit", status.Type),
		QuotaType:   string(status.Type),
		Usage:       status.Usage,
		Current:     status.Current,
		SoftLimit:   status.SoftLimit,
		HardLimit:   status.HardLimit,
		Recommended: "Consider requesting a quota increase",
	}
}

// ── Header constants ─────────────────────────────────────────────────

const (
	// HeaderQuotaWarning — warning header для soft limit.
	HeaderQuotaWarning = "X-Quota-Warning"
	// HeaderQuotaBlocked — header для hard limit.
	HeaderQuotaBlocked = "X-Quota-Blocked"
	// HeaderQuotaGrace — header для grace period.
	HeaderQuotaGrace = "X-Quota-Grace"
	// HeaderQuotaCurrent — header с текущим использованием.
	HeaderQuotaCurrent = "X-Quota-Current"
	// HeaderQuotaLimit — header с лимитом.
	HeaderQuotaLimit = "X-Quota-Limit"
	// HeaderQuotaUsage — header с процентом использования.
	HeaderQuotaUsage = "X-Quota-Usage"
)

// SetQuotaHeaders устанавливает quota headers в ответ.
func SetQuotaHeaders(w http.ResponseWriter, status *tenant.QuotaStatus) {
	w.Header().Set(HeaderQuotaCurrent, strconv.FormatInt(status.Current, 10))
	w.Header().Set(HeaderQuotaLimit, strconv.FormatInt(status.HardLimit, 10))
	w.Header().Set(HeaderQuotaUsage, strconv.FormatFloat(status.Usage, 'f', 1, 64))
}
