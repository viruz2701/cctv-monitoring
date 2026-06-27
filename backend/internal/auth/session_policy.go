// Package auth — Regional session policies for CCTV Health Monitor.
//
// P2-CR.4: Session & Auth Regional Policies
//   - 5 regional profiles (BY/RU/EU/US/CN)
//   - Session timeout enforcement, concurrent session limits, lockout policy
//   - Admin override for emergency cases
//
// Compliance:
//   - IEC 62443-3-3 SR 2.1 (Account management — session timeout)
//   - ISO 27001 A.9.4 (Access control — session management)
//   - ISO 27019 PCC.A.9 (ICS session management)
//   - СТБ 34.101.27 п. 6.1 (Аутентификация — таймауты сессий)
//   - OWASP ASVS V3 (Session Management)
//   - Приказ ОАЦ №66 п. 7.18.2 (Защита сетей — управление сессиями)
package auth

import (
	"time"
)

// SessionPolicy определяет политику управления сессиями для региона.
//
// Fields:
//   - IdleTimeout: максимальное время бездействия (IEC 62443 SR 2.1)
//   - AbsoluteTimeout: максимальная длительность сессии (ISO 27001 A.9.4)
//   - MaxConcurrentSessions: макс. одновременных сессий (OWASP ASVS V3.1)
//   - FailedLoginLockout: кол-во неудачных попыток до блокировки
//   - LockoutDuration: длительность блокировки
type SessionPolicy struct {
	IdleTimeout           time.Duration
	AbsoluteTimeout       time.Duration
	MaxConcurrentSessions int
	FailedLoginLockout    int
	LockoutDuration       time.Duration
}

// DefaultSessionPolicy возвращает политику по умолчанию (наиболее строгая — BY).
func DefaultSessionPolicy() SessionPolicy {
	return GetSessionPolicy(RegionBY)
}

// GetSessionPolicy возвращает политику сессий для указанного региона.
// Если регион неизвестен, возвращает политику BY (наиболее строгие требования КИИ).
//
// Compliance:
//   - IEC 62443 SR 2.1: timeout values aligned with zone SL-3
//   - ISO 27001 A.9.4: session timeout enforcement
//   - СТБ 34.101.27 п. 6.1: аутентификация с таймаутами
func GetSessionPolicy(region Region) SessionPolicy {
	switch region {
	case RegionBY:
		return SessionPolicy{
			IdleTimeout:           30 * time.Minute,
			AbsoluteTimeout:       8 * time.Hour,
			MaxConcurrentSessions: 3,
			FailedLoginLockout:    5,
			LockoutDuration:       15 * time.Minute,
		}
	case RegionRU:
		return SessionPolicy{
			IdleTimeout:           15 * time.Minute,
			AbsoluteTimeout:       4 * time.Hour,
			MaxConcurrentSessions: 2,
			FailedLoginLockout:    5,
			LockoutDuration:       30 * time.Minute,
		}
	case RegionEU:
		return SessionPolicy{
			IdleTimeout:           8 * time.Hour,
			AbsoluteTimeout:       24 * time.Hour,
			MaxConcurrentSessions: 5,
			FailedLoginLockout:    10,
			LockoutDuration:       15 * time.Minute,
		}
	case RegionUS:
		return SessionPolicy{
			IdleTimeout:           30 * time.Minute,
			AbsoluteTimeout:       8 * time.Hour,
			MaxConcurrentSessions: 3,
			FailedLoginLockout:    5,
			LockoutDuration:       15 * time.Minute,
		}
	case RegionCN:
		return SessionPolicy{
			IdleTimeout:           15 * time.Minute,
			AbsoluteTimeout:       4 * time.Hour,
			MaxConcurrentSessions: 2,
			FailedLoginLockout:    5,
			LockoutDuration:       15 * time.Minute,
		}
	default:
		// Fail secure: неизвестный регион → BY (максимально строгие требования КИИ)
		return GetSessionPolicy(RegionBY)
	}
}

// WarningThreshold возвращает длительность до таймаута, за которую нужно
// отправить предупреждение. Составляет 10% от IdleTimeout, минимум 1 минута.
func (p SessionPolicy) WarningThreshold() time.Duration {
	warning := p.IdleTimeout / 10
	if warning < 1*time.Minute {
		warning = 1 * time.Minute
	}
	return warning
}

// IsAdminOverride проверяет, является ли роль административной для целей
// экстренного обхода политик сессий (ISO 27001 A.9.2.3 — Privilege management).
func IsAdminOverride(role string) bool {
	return role == "admin" || role == "superadmin"
}
