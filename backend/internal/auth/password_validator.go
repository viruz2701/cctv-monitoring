// Package auth — Regional Password Validator (P2-CR.3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.3: Regional Password Validator
//
// Валидация паролей на основе региональной PasswordPolicy.
// Заменяет жестко зашитую валидацию в password.go на политику,
// определяемую регионом тенанта.
//
// Compliance:
//   - OWASP ASVS V2.1.1 (Password length)
//   - OWASP ASVS V2.1.2 (Password character set)
//   - OWASP ASVS V2.1.7 (Password history)
//   - NIST SP 800-63B (Verifier-requested rotation)
//   - СТБ 34.101.27 п. 6.2 (Политика паролей)
//
// ═══════════════════════════════════════════════════════════════════════════
package auth

import (
	"fmt"
	"unicode"
)

// ErrPasswordTooLong — пароль длиннее максимальной длины.
var ErrPasswordTooLong = fmt.Errorf("password exceeds maximum length")

// ═══════════════════════════════════════════════════════════════════════════
// ValidatePassword — региональная валидация пароля
// ═══════════════════════════════════════════════════════════════════════════

// ValidatePassword проверяет пароль на соответствие региональной политике.
//
// Параметры:
//   - password: проверяемый пароль
//   - policy: региональная PasswordPolicy
//
// Возвращает:
//   - nil если пароль соответствует политике
//   - ошибку с описанием первого нарушения
//
// Порядок проверок:
//  1. Минимальная длина
//  2. Максимальная длина
//  3. Наличие заглавных букв
//  4. Наличие строчных букв
//  5. Наличие цифр
//  6. Наличие спецсимволов
func ValidatePassword(password string, policy PasswordPolicy) error {
	if password == "" {
		return ErrPasswordTooShort
	}

	// 1. Минимальная длина
	if len(password) < policy.MinLength {
		return fmt.Errorf("%w: minimum %d characters, got %d",
			ErrPasswordTooShort, policy.MinLength, len(password))
	}

	// 2. Максимальная длина
	if policy.MaxLength > 0 && len(password) > policy.MaxLength {
		return fmt.Errorf("%w: maximum %d characters, got %d",
			ErrPasswordTooLong, policy.MaxLength, len(password))
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	// 3. Заглавные буквы
	if policy.RequireUpper && !hasUpper {
		return ErrPasswordNoUpper
	}

	// 4. Строчные буквы
	if policy.RequireLower && !hasLower {
		return ErrPasswordNoLower
	}

	// 5. Цифры
	if policy.RequireDigit && !hasDigit {
		return ErrPasswordNoDigit
	}

	// 6. Спецсимволы
	if policy.RequireSpecial && !hasSpecial {
		return ErrPasswordNoSpecial
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ValidatePasswordForRegion — удобная обёртка
// ═══════════════════════════════════════════════════════════════════════════

// ValidatePasswordForRegion проверяет пароль для указанного региона.
//
// Пример:
//
//	err := ValidatePasswordForRegion("MyP@ssw0rd", RegionEU)
//	if err != nil {
//	    log.Error("password validation failed", "error", err)
//	}
func ValidatePasswordForRegion(password string, region Region) error {
	policy := GetPasswordPolicy(region)
	return ValidatePassword(password, policy)
}
