// Package providers — Password Migration Service.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.3: Password Migration
//
// Обеспечивает read-old, write-new миграцию при смене compliance profile.
//
// Сценарий:
//  1. Пользователь имеет пароль, захешированный старым алгоритмом (например, bcrypt)
//  2. После смены региона (BY→EU) новый пароль хешируется новым алгоритмом (Argon2id)
//  3. Старый пароль остаётся читаемым через старый провайдер
//  4. При успешной верификации — пароль перехешируется новым алгоритмом
//
// Compliance:
//   - OWASP ASVS V2.4.2 (Password migration)
//   - NIST SP 800-63B (Password storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"fmt"
	"log/slog"
)

// ────────────────────────────────────────────────────────────────────────────
// PasswordMigrator
// ────────────────────────────────────────────────────────────────────────────

// PasswordMigrator handles password hash migration between providers.
//
// Read-old, write-new стратегия:
//   - При логине проверяем пароль через current provider
//   - Если не подходит — пробуем old providers
//   - Если подошёл через old — перехешируем через current
type PasswordMigrator struct {
	current  PasswordHashProvider
	fallback []PasswordHashProvider
	logger   *slog.Logger
}

// NewPasswordMigrator создаёт новый мигратор паролей.
//
// current — текущий провайдер (согласно compliance profile).
// fallback — предыдущие провайдеры (для старых хешей).
func NewPasswordMigrator(current PasswordHashProvider, fallback []PasswordHashProvider) *PasswordMigrator {
	return &PasswordMigrator{
		current:  current,
		fallback: fallback,
		logger:   slog.Default().With("component", "password.migration"),
	}
}

// Verify проверяет пароль, пробуя все провайдеры.
//
// Возвращает:
//   - true, "" — пароль верный, миграция не требуется
//   - true, newHash — пароль верный, требуется перехеширование (возвращает newHash)
//   - false, "" — пароль неверный
func (m *PasswordMigrator) Verify(password, currentHash string) (bool, string) {
	// 1. Пробуем current provider
	if ok, _ := m.current.Verify(password, currentHash); ok {
		return true, "" // Достаточно свежий хеш
	}

	// 2. Пробуем fallback providers
	for _, old := range m.fallback {
		if ok, _ := old.Verify(password, currentHash); ok {
			// Пароль подошёл через старый провайдер — перехешируем
			newHash, err := m.current.Hash(password)
			if err != nil {
				m.logger.Error("password rehash failed",
					"old_provider", old.Name(),
					"new_provider", m.current.Name(),
					"error", err,
				)
				return true, "" // Возвращаем успех без нового хеша
			}

			m.logger.Info("password rehashed",
				"old_provider", old.Name(),
				"new_provider", m.current.Name(),
			)

			return true, newHash
		}
	}

	return false, ""
}

// Current возвращает текущий провайдер.
func (m *PasswordMigrator) Current() PasswordHashProvider {
	return m.current
}

// ────────────────────────────────────────────────────────────────────────────
// Convenience
// ────────────────────────────────────────────────────────────────────────────

// MigratorFromProfile создаёт PasswordMigrator на основе compliance профиля
// и опционального списка fallback провайдеров.
//
// Пример:
//
//	migrator := MigratorFromProfile("belt-hash", "bcrypt", "argon2id")
//	// current: belt-hash, fallback: [bcrypt, argon2id]
func MigratorFromProfile(currentProfile string, fallbackProfiles ...string) (*PasswordMigrator, error) {
	current, err := PasswordHashFromProfile(currentProfile)
	if err != nil {
		return nil, fmt.Errorf("password migrator: current provider: %w", err)
	}

	var fallbacks []PasswordHashProvider
	for _, fp := range fallbackProfiles {
		p, err := PasswordHashFromProfile(fp)
		if err != nil {
			return nil, fmt.Errorf("password migrator: fallback %s: %w", fp, err)
		}
		fallbacks = append(fallbacks, p)
	}

	return NewPasswordMigrator(current, fallbacks), nil
}
