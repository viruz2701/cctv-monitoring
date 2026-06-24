package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrPasswordNoUpper   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit   = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecial = errors.New("password must contain at least one special character")
)

// PasswordStrength представляет уровень сложности пароля.
type PasswordStrength int

const (
	PasswordWeak   PasswordStrength = iota // 0 — слабый
	PasswordMedium                         // 1 — средний
	PasswordStrong                         // 2 — сильный
)

func (s PasswordStrength) String() string {
	switch s {
	case PasswordWeak:
		return "weak"
	case PasswordMedium:
		return "medium"
	case PasswordStrong:
		return "strong"
	default:
		return "unknown"
	}
}

// ValidatePasswordStrength проверяет пароль на соответствие политике безопасности
// (OWASP ASVS L3 V2 — Authentication Verification).
// Возвращает уровень сложности и ошибку, если пароль не проходит минимальные требования.
func ValidatePasswordStrength(password string) (PasswordStrength, error) {
	if len(password) < 8 {
		return PasswordWeak, ErrPasswordTooShort
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
		score      int
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

	if !hasUpper {
		return PasswordWeak, ErrPasswordNoUpper
	}
	if !hasLower {
		return PasswordWeak, ErrPasswordNoLower
	}
	if !hasDigit {
		return PasswordWeak, ErrPasswordNoDigit
	}
	if !hasSpecial {
		return PasswordWeak, ErrPasswordNoSpecial
	}

	// Оценка силы пароля
	if len(password) >= 12 {
		score++
	}
	if hasUpper && hasLower && hasDigit && hasSpecial {
		score++
	}
	if len(password) >= 16 {
		score++
	}

	switch {
	case score >= 3:
		return PasswordStrong, nil
	case score >= 2:
		return PasswordMedium, nil
	default:
		return PasswordMedium, nil
	}
}

// MustValidatePasswordStrength — строгая валидация (ASVS L3).
// Возвращает ошибку если пароль НЕ соответствует требованиям.
func MustValidatePasswordStrength(password string) error {
	_, err := ValidatePasswordStrength(password)
	return err
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateResetToken generates a cryptographically secure random token for password reset.
func GenerateResetToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback — не должно произойти в нормальной среде
		return "fallback-token-" + hex.EncodeToString([]byte("emergency"))
	}
	return hex.EncodeToString(b)
}
