// Package respond — централизованная обработка HTTP-ошибок.
//
// Заменяет все http.Error(w, ...) в проекте.
// Все пакеты ДОЛЖНЫ использовать respond.Error вместо http.Error.
//
// Соответствие:
//   - OWASP ASVS V7.1.1: Стандартизированный формат ответов
//   - ISO 27001 A.12.4.1: Логирование ошибок
//   - IEC 62443-3-3 SR 3.1: Отсутствие утечки информации
package respond

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Error отправляет JSON-ответ с ошибкой и логирует её.
// Если status >= 500, ошибка логируется как server error.
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
