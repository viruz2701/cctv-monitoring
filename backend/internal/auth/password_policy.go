// Package auth — Regional Password Policies (P2-CR.3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.3: Regional Password Policies
//
// Проблема: Единая password policy для всех регионов не соответствует
// локальным требованиям (СТБ 34.101.27, ФСТЭК, NIST SP 800-63B, GDPR).
//
// Решение:
//   - PasswordPolicy struct с character-based требованиями
//   - GetPasswordPolicy(region) — региональная политика
//   - 5 profiles: BY, RU, EU, US, CN
//
// Compliance:
//   - СТБ 34.101.27 п. 6.2 (BY — 12 символов, ротация 90d)
//   - ФСТЭК Приказ №17 (RU — 8 символов, ротация 90d)
//   - NIST SP 800-63B (EU — 8 символов, без принудительной ротации)
//   - NIST SP 800-63B (US — 8 символов, ротация 90d)
//   - MLPS 2.0 / GB/T 22239 (CN — 8 символов, ротация 90d)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import "fmt"

// ═══════════════════════════════════════════════════════════════════════════
// Region constants
// ═══════════════════════════════════════════════════════════════════════════

// Region represents a geographic/political region for password policy.
type Region string

const (
	RegionBY Region = "BY" // Республика Беларусь (СТБ 34.101.27)
	RegionRU Region = "RU" // Российская Федерация (ФСТЭК, 152-ФЗ)
	RegionEU Region = "EU" // Европейский Союз (GDPR, NIST SP 800-63B)
	RegionUS Region = "US" // США (NIST SP 800-63B, FedRAMP)
	RegionCN Region = "CN" // Китай (MLPS 2.0, GB/T 22239)
)

// ValidRegions — список всех поддерживаемых регионов.
var ValidRegions = []Region{RegionBY, RegionRU, RegionEU, RegionUS, RegionCN}

// ═══════════════════════════════════════════════════════════════════════════
// PasswordPolicy
// ═══════════════════════════════════════════════════════════════════════════

// PasswordPolicy defines character-based password requirements for a region.
//
// Fields:
//   - MinLength: минимальная длина пароля
//   - MaxLength: максимальная длина пароля (0 = без ограничения)
//   - RequireUpper: требовать заглавные буквы
//   - RequireLower: требовать строчные буквы
//   - RequireDigit: требовать цифры
//   - RequireSpecial: требовать спецсимволы
//   - RotationDays: дни до принудительной ротации (0 = без ротации)
//   - HistoryLength: количество предыдущих паролей для проверки (0 = без проверки)
type PasswordPolicy struct {
	MinLength      int  `json:"min_length"`
	MaxLength      int  `json:"max_length"`
	RequireUpper   bool `json:"require_upper"`
	RequireLower   bool `json:"require_lower"`
	RequireDigit   bool `json:"require_digit"`
	RequireSpecial bool `json:"require_special"`
	RotationDays   int  `json:"rotation_days"`
	HistoryLength  int  `json:"history_length"`
}

// ═══════════════════════════════════════════════════════════════════════════
// DefaultPasswordPolicy — политика по умолчанию (NIST SP 800-63B)
// ═══════════════════════════════════════════════════════════════════════════

// DefaultPasswordPolicy возвращает политику по умолчанию.
// Основание: NIST SP 800-63B (8 символов, без ротации).
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      8,
		MaxLength:      0, // без ограничения
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		RotationDays:   0, // NIST: без принудительной ротации
		HistoryLength:  3,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Regional policies
// ═══════════════════════════════════════════════════════════════════════════

// byPasswordPolicy — Республика Беларусь.
//
// Соответствие:
//   - СТБ 34.101.27 п. 6.2: минимум 12 символов
//   - СТБ 34.101.30: ротация 90 дней, история 5 паролей
//   - Приказ ОАЦ №66 п. 7.18: сложные пароли
func byPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      12,
		MaxLength:      128,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		RotationDays:   90,
		HistoryLength:  5,
	}
}

// ruPasswordPolicy — Российская Федерация.
//
// Соответствие:
//   - ФСТЭК Приказ №17: минимум 8 символов
//   - 152-ФЗ: ротация 90 дней
//   - Методика ФСТЭК: история 5 паролей
func ruPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      8,
		MaxLength:      128,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		RotationDays:   90,
		HistoryLength:  5,
	}
}

// euPasswordPolicy — Европейский Союз.
//
// Соответствие:
//   - NIST SP 800-63B: минимум 8 символов, без принудительной ротации
//   - GDPR Art. 32: безопасное хранение паролей
//   - ENISA: recommended 12+ для администраторов
func euPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      8,
		MaxLength:      0, // без ограничения
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: false, // NIST: спецсимволы опциональны
		RotationDays:   0,     // NIST: без принудительной ротации
		HistoryLength:  3,
	}
}

// usPasswordPolicy — США.
//
// Соответствие:
//   - NIST SP 800-63B: минимум 8 символов
//   - FedRAMP: ротация 90 дней для privileged
//   - OMB M-21-07: история 3 пароля
func usPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      8,
		MaxLength:      0, // без ограничения
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: false, // NIST: спецсимволы опциональны
		RotationDays:   90,    // Для privileged аккаунтов
		HistoryLength:  3,
	}
}

// cnPasswordPolicy — Китай.
//
// Соответствие:
//   - MLPS 2.0 (GB/T 22239): минимум 8 символов
//   - GM/T 0006: ротация 90 дней
//   - Cryptography Law: сложные пароли
func cnPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:      8,
		MaxLength:      64,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		RotationDays:   90,
		HistoryLength:  5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetPasswordPolicy — региональная политика паролей
// ═══════════════════════════════════════════════════════════════════════════

// GetPasswordPolicy возвращает PasswordPolicy для указанного региона.
//
// Если регион не найден, возвращает DefaultPasswordPolicy (NIST SP 800-63B).
func GetPasswordPolicy(region Region) PasswordPolicy {
	switch region {
	case RegionBY:
		return byPasswordPolicy()
	case RegionRU:
		return ruPasswordPolicy()
	case RegionEU:
		return euPasswordPolicy()
	case RegionUS:
		return usPasswordPolicy()
	case RegionCN:
		return cnPasswordPolicy()
	default:
		return DefaultPasswordPolicy()
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// ParseRegion парсит строку в Region.
// Возвращает ошибку для неизвестного региона.
func ParseRegion(s string) (Region, error) {
	switch Region(s) {
	case RegionBY, RegionRU, RegionEU, RegionUS, RegionCN:
		return Region(s), nil
	default:
		return "", fmt.Errorf("auth: unknown password policy region: %s", s)
	}
}

// String возвращает человекочитаемое название региона.
func (r Region) String() string {
	switch r {
	case RegionBY:
		return "Belarus (СТБ 34.101.27)"
	case RegionRU:
		return "Russia (ФСТЭК, 152-ФЗ)"
	case RegionEU:
		return "European Union (GDPR, NIST)"
	case RegionUS:
		return "United States (NIST SP 800-63B)"
	case RegionCN:
		return "China (MLPS 2.0)"
	default:
		return fmt.Sprintf("Unknown (%s)", string(r))
	}
}
