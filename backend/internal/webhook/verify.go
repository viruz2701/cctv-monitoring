// Package webhook предоставляет единый механизм HMAC-верификации вебхуков.
//
// SEC-05: Унификация проверки подписей вебхуков для всех CMMS адаптеров
// (ServiceNow, TOIR, Jira).
//
// Compliance:
//   - OWASP ASVS V6.3 (Integrity verification — HMAC for webhooks)
//   - ISO 27001 A.12.4.2 (Protection of log information)
//   - IEC 62443 SR 3.1 (Communication integrity — HMAC)
//   - СТБ 34.101.27 п. 7.5 (Audit trail integrity)
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// ═══════════════════════════════════════════════════════════════════════
// VerifyOptions — опции верификации вебхука.
// ═══════════════════════════════════════════════════════════════════════

// VerifyOptions настраивает процесс верификации HMAC подписи.
type VerifyOptions struct {
	// SignaturePrefix — префикс, который нужно удалить из заголовка перед
	// сравнением (например, "sha256=" для Jira). Если пустой — не удаляется.
	SignaturePrefix string

	// SignatureHeader — имя HTTP-заголовка с подписью (используется middleware).
	SignatureHeader string

	// Logger — логгер для предупреждений. Если nil — используется slog.Default().
	Logger *slog.Logger
}

// VerifyOption — функциональный параметр для VerifyOptions.
type VerifyOption func(*VerifyOptions)

// WithSignaturePrefix устанавливает префикс подписи (например, "sha256=").
func WithSignaturePrefix(prefix string) VerifyOption {
	return func(o *VerifyOptions) {
		o.SignaturePrefix = prefix
	}
}

// WithSignatureHeader устанавливает имя заголовка с подписью.
func WithSignatureHeader(header string) VerifyOption {
	return func(o *VerifyOptions) {
		o.SignatureHeader = header
	}
}

// WithLogger устанавливает логгер.
func WithLogger(logger *slog.Logger) VerifyOption {
	return func(o *VerifyOptions) {
		o.Logger = logger
	}
}

// ═══════════════════════════════════════════════════════════════════════
// VerifyHMAC — единая HMAC-верификация вебхуков.
// ═══════════════════════════════════════════════════════════════════════

// VerifyHMAC проверяет HMAC-SHA256 подпись тела запроса.
//
// Параметры:
//   - secret: секретный ключ для HMAC
//   - sigHeader: значение заголовка с подписью (может содержать префикс)
//   - body: тело запроса (сырые байты)
//   - opts: опции (например, WithSignaturePrefix("sha256="))
//
// Возвращает true если подпись валидна.
//
// Security:
//   - OWASP ASVS V6.3: пустой secret — SECURITY VIOLATION, reject
//   - IEC 62443 SR 3.1: Communication integrity — HMAC обязателен
//   - ISO 27001 A.12.4.2: Protection of log information
//
// Graceful degradation: НЕТ. Пустой secret = reject all (fail secure).
//
// ⚠ СТБ COMPLIANCE: После добавления bp2012/crypto заменить на bash-256:
//
//	import "github.com/bp2012/crypto/bash"
//	mac := bash.NewHmac([]byte(secret), bash.Size256)
func VerifyHMAC(secret string, sigHeader string, body []byte, opts ...VerifyOption) bool {
	options := &VerifyOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	// IEC 62443 SR 7.1: Fail Secure — при отсутствии секрета reject, не allow
	if secret == "" {
		options.Logger.Warn("webhook: HMAC secret is empty, rejecting request (IEC 62443 SR 7.1)")
		return false
	}

	// Если подпись не предоставлена — отклоняем
	if sigHeader == "" {
		return false
	}

	// Удаляем префикс если указан (например, "sha256=" для Jira)
	sig := sigHeader
	if options.SignaturePrefix != "" && len(sigHeader) > len(options.SignaturePrefix) {
		if sigHeader[:len(options.SignaturePrefix)] == options.SignaturePrefix {
			sig = sigHeader[len(options.SignaturePrefix):]
		}
	}

	// Вычисляем ожидаемую подпись
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
}

// ═══════════════════════════════════════════════════════════════════════
// VerifyMiddleware — chi middleware для HMAC-верификации вебхуков.
// ═══════════════════════════════════════════════════════════════════════

// VerifyMiddleware создаёт chi middleware для HMAC-верификации входящих вебхуков.
//
// Пример:
//
//	r.With(webhook.VerifyMiddleware(secret, webhook.WithSignatureHeader("X-SN-Signature"))).
//	    Post("/webhook/servicenow", handler)
func VerifyMiddleware(secret string, opts ...VerifyOption) func(http.Handler) http.Handler {
	options := &VerifyOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// IEC 62443 SR 7.1: Fail Secure — reject при отсутствии secret
			if secret == "" {
				options.Logger.Warn("webhook middleware: HMAC secret empty, rejecting (IEC 62443 SR 7.1)")
				http.Error(w, `{"error":"webhook not configured"}`, http.StatusInternalServerError)
				return
			}

			headerName := options.SignatureHeader
			if headerName == "" {
				headerName = "X-Signature-256"
			}

			sig := r.Header.Get(headerName)
			if sig == "" {
				options.Logger.Warn("webhook: missing signature header",
					"header", headerName,
				)
				http.Error(w, `{"error":"missing signature"}`, http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				options.Logger.Error("webhook: read body", "error", err)
				http.Error(w, `{"error":"read error"}`, http.StatusBadRequest)
				return
			}
			r.Body.Close()

			if !VerifyHMAC(secret, sig, body, opts...) {
				options.Logger.Warn("webhook: invalid signature",
					"header", headerName,
				)
				http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
				return
			}

			// Восстанавливаем тело для последующих handler'ов
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helper — удобная обёртка для inline-верификации внутри http.Handler.
// ═══════════════════════════════════════════════════════════════════════

// ServeHTTPWithVerify — обёртка для http.Handler, которая проверяет HMAC
// подпись перед вызовом handler. Удобно когда middleware не подходит
// (например, нужно логировать body после верификации).
//
// Пример:
//
//	handler := webhook.ServeHTTPWithVerify(secret, func(w, r, body) {
//	    // тело уже верифицировано
//	}, opts...)
func ServeHTTPWithVerify(secret string, next func(w http.ResponseWriter, r *http.Request, body []byte), opts ...VerifyOption) http.Handler {
	options := &VerifyOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
		if err != nil {
			options.Logger.Error("webhook: read body", "error", err)
			http.Error(w, `{"error":"read error"}`, http.StatusBadRequest)
			return
		}

		headerName := options.SignatureHeader
		if headerName == "" {
			headerName = "X-Signature-256"
		}

		if !VerifyHMAC(secret, r.Header.Get(headerName), body, opts...) {
			options.Logger.Warn("webhook: invalid signature",
				"header", headerName,
			)
			http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
			return
		}

		next(w, r, body)
	})
}

// ═══════════════════════════════════════════════════════════════════════
// JSON helpers
// ═══════════════════════════════════════════════════════════════════════

// JSONError отправляет JSON-ответ с ошибкой.
func JSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// JSONOK отправляет JSON-ответ об успехе.
func JSONOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
