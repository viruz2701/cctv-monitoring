// Package auth — аутентификация и управление доступом.
//
// Содержит общие вспомогательные функции для работы с JWT_SECRET.
// Соответствует: ISO 27001 A.9.4 (Authentication), OWASP ASVS V2 (Authentication)
package auth

import (
	"errors"
	"os"
)

// ErrJWTSecretMissing возвращается когда JWT_SECRET не установлен.
// Используется для graceful degradation — сервер продолжает работу,
// но /health возвращает 503.
var ErrJWTSecretMissing = errors.New("JWT_SECRET environment variable is required")

// GetJWTSecret возвращает JWT_SECRET из переменных окружения.
// Возвращает error если секрет не задан — никогда не паникует.
//
// Compliance:
//   - ISO 27001 A.9.4.2 (Secure authentication — key management)
//   - OWASP ASVS V2.1 (Secret verification)
//   - Приказ ОАЦ №66 п. 7.18.1 (Unique identification — key material)
//
// Graceful degradation: При отсутствии JWT_SECRET аутентификация недоступна,
// но сервер продолжает обработку запросов не требующих аутентификации.
func GetJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrJWTSecretMissing
	}
	return []byte(secret), nil
}

// IsJWTSecretSet проверяет установлен ли JWT_SECRET.
// Используется для health check — если не установлен, /health возвращает 503.
func IsJWTSecretSet() bool {
	return os.Getenv("JWT_SECRET") != ""
}
