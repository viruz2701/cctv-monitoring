// Package compliance — Baseline Compliance Profiles.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.1: Baseline Compliance Profiles
//
// 3 baseline профиля:
//   - BY  (СТБ): belt-gcm, bash-256, bign-curve256v1 — Республика Беларусь
//   - EU  (GDPR): aes-256-gcm, sha256, es256 — Европейский Союз
//   - INTL (ISO 27001): aes-256-gcm, sha256, es256 — International
//
// Compliance:
//   - IEC 62443-3-3 SR 5.1, ISO 27001 A.5.1
//   - СТБ 34.101.27 п. 6.2, СТБ 34.101.30
//   - GDPR Art. 32, Art. 44-49
//   - Приказ ОАЦ № 66 п. 7.18
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"fmt"
	"log/slog"
)

// ═══════════════════════════════════════════════════════════════════════════
// BY Profile — Республика Беларусь (СТБ 34.101.27, СТБ 34.101.30)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - СТБ 34.101.27 п. 6.2 — Политики безопасности КИИ
//   - СТБ 34.101.30 — Криптография: belt-gcm, bash-256, bign-curve256v1
//   - СТБ 34.101.31 — belt-gcm (симметричное шифрование)
//   - СТБ 34.101.45 — bign-curve256v1 (цифровые подписи)
//   - СТБ 34.101.77 — bash-256 (хеширование)
//   - IEC 62443-3-3 SR 5.1 — Zone access control
//   - Приказ ОАЦ № 66 п. 7.18 — Идентификация узлов
//   - ISO 27001 A.12.4 — Audit trail

// BYProfile implements ComplianceProfile для Республики Беларусь.
type BYProfile struct {
	*BaseProfile
}

// NewBYProfile создаёт BY Compliance Profile.
func NewBYProfile() *BYProfile {
	return &BYProfile{
		BaseProfile: NewBaseProfile(RegionBY,
			"СТБ 34.101 (Республика Беларусь)",
			"Профиль соответствия для КИИ РБ. СТБ 34.101.27, СТБ 34.101.30, Приказ ОАЦ №66.",
		),
	}
}

func (p *BYProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoBeltGCM, // СТБ 34.101.31
		KeySize:       256,           // 256-bit ключ (СТБ 34.101.30)
		AADRequired:   true,          // AAD для КИИ (дополнительная защита)
		TLSMinVersion: "1.3",         // TLS 1.3 (Приказ ОАЦ №66 п. 7.18.2)
	}
}

func (p *BYProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashBash256, // СТБ 34.101.77
		SaltRequired:   true,        // Соль обязательна для КИИ
		OutputSizeBits: 256,
	}
}

func (p *BYProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureBignCurve256, // СТБ 34.101.45
		Curve:       "bign-curve256v1",
		HashForSign: HashBash256,
	}
}

func (p *BYProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordBeltHash, // belt-hash (СТБ) — stub, fallback bcrypt
		MinLength:              12,               // КИИ: минимум 12 символов
		RequireMFA:             true,             // MFA обязателен для КИИ
		MFATypes:               []MFAType{MFATOTP, MFASMS},
		MaxAgeDays:             90, // Ротация каждые 90 дней (СТБ 34.101.30)
		HistoryCount:           5,  // 5 предыдущих паролей
		RequireComplexity:      true,
		LockoutThreshold:       3,  // 3 попытки (КИИ)
		LockoutDurationMinutes: 30, // Блокировка 30 мин
	}
}

func (p *BYProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionBY},
		CrossBorderTransferAllowed: false, // Запрет трансграничной передачи (КИИ)
		ColdStorageRegion:          RegionBY,
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *BYProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       1825,  // 5 лет (КИИ РБ)
		EventDataDays:      365,   // 1 год
		VideoDataDays:      90,    // 90 дней (типовое для КИИ)
		LegalHoldSupported: true,  // Legal hold для расследований
		AutoDeleteEnabled:  false, // Ручное удаление (КИИ)
	}
}

func (p *BYProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true, // HMAC-подпись всех логов (СТБ)
		ChainHashPrev:   true, // Цепочка хешей (tamper detection)
		RetentionYears:  5,    // 5 лет хранения аудита
		LogAllMutations: true, // Логирование всех мутаций
		IncludeTraceID:  true, // TraceID в логах
	}
}

func (p *BYProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       30, // 30 мин бездействия (КИИ)
		MaxSessionHours:          8,  // 8 часов макс сессия
		MaxConcurrentSessions:    1,  // 1 одновременная сессия (КИИ)
		FailedLoginLockout:       3,  // 3 неудачных попытки
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5, // Предупреждение за 5 мин
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// EU Profile — Европейский Союз (GDPR, NIS2, eIDAS)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - GDPR Art. 32 — Security of processing
//   - GDPR Art. 44-49 — Data transfers
//   - NIS2 Directive — Incident reporting
//   - eIDAS — Electronic signatures
//   - ISO 27001 A.5.1 — Information security policies
//   - IEC 62443-3-3 — IACS security

// EUProfile implements ComplianceProfile для Европейского Союза.
type EUProfile struct {
	*BaseProfile
}

// NewEUProfile создаёт EU Compliance Profile.
func NewEUProfile() *EUProfile {
	return &EUProfile{
		BaseProfile: NewBaseProfile(RegionEU,
			"GDPR / NIS2 (European Union)",
			"Профиль соответствия для ЕС. GDPR, NIS2, eIDAS, ISO 27001.",
		),
	}
}

func (p *EUProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM (международный)
		KeySize:       256,
		AADRequired:   false,
		TLSMinVersion: "1.2", // TLS 1.2+ (GDPR Art. 32)
	}
}

func (p *EUProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   false,
		OutputSizeBits: 256,
	}
}

func (p *EUProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256, // ECDSA P-256 (eIDAS совместимый)
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *EUProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID, // Argon2id (OWASP рекомендованный)
		MinLength:              8,                // NIST SP 800-63B
		RequireMFA:             false,            // Опционально (GDPR не требует)
		MFATypes:               []MFAType{MFATOTP, MFAFIDO2},
		MaxAgeDays:             0, // NIST: без принудительной ротации
		HistoryCount:           3,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *EUProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionEU},
		CrossBorderTransferAllowed: true, // С SCC (Schrems II)
		ColdStorageRegion:          RegionEU,
		StorageTiers:               []StorageTier{StorageHot, StorageCold, StorageArchive},
		RequireEncryptionAtRest:    true, // GDPR Art. 32
	}
}

func (p *EUProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       730,  // 2 года (GDPR minimum)
		EventDataDays:      180,  // 6 месяцев
		VideoDataDays:      30,   // 30 дней
		LegalHoldSupported: true, // GDPR Art. 17 right to erasure
		AutoDeleteEnabled:  true, // Автоудаление (GDPR data minimisation)
	}
}

func (p *EUProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true,  // ISO 27001 A.12.4
		ChainHashPrev:   false, // Не требуется для EU
		RetentionYears:  2,     // 2 года (GDPR)
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *EUProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       60, // 1 час
		MaxSessionHours:          24, // 24 часа
		MaxConcurrentSessions:    5,  // 5 сессий
		FailedLoginLockout:       5,  // 5 попыток
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// RU Profile — Российская Федерация (ГОСТ, 152-ФЗ, ФСТЭК)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - ГОСТ 28147-89 (Магма) — Симметричное шифрование
//   - ГОСТ Р 34.10-2012 — Цифровые подписи (256/512)
//   - ГОСТ Р 34.11-2012 (Стрибог) — Хеширование 256/512
//   - 152-ФЗ — Персональные данные РФ
//   - Приказ ФСТЭК № 17 — Защита информации
//   - IEC 62443-3-3 — IACS Security
//   - ISO 27001 — Information Security

// RUProfile implements ComplianceProfile для Российской Федерации.
type RUProfile struct {
	*BaseProfile
}

// NewRUProfile создаёт RU Compliance Profile.
func NewRUProfile() *RUProfile {
	return &RUProfile{
		BaseProfile: NewBaseProfile(RegionRU,
			"ГОСТ (Российская Федерация)",
			"Профиль соответствия для РФ. ГОСТ Р 34.10-2012, ГОСТ Р 34.11-2012, 152-ФЗ, ФСТЭК.",
		),
	}
}

func (p *RUProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoGOST, // ГОСТ 28147-89 (Магма)
		KeySize:       256,
		AADRequired:   true,
		TLSMinVersion: "1.3", // ГОСТ TLS 1.3
	}
}

func (p *RUProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashStribog, // ГОСТ Р 34.11-2012 (Стрибог 256)
		SaltRequired:   true,
		OutputSizeBits: 256,
	}
}

func (p *RUProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureGOST3410, // ГОСТ Р 34.10-2012
		Curve:       "GOST-256",
		HashForSign: HashStribog,
	}
}

func (p *RUProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              8, // ФСТЭК: минимум 8 символов
		RequireMFA:             true,
		MFATypes:               []MFAType{MFATOTP, MFASMS},
		MaxAgeDays:             90, // Ротация 90 дней (ФСТЭК)
		HistoryCount:           5,  // 5 предыдущих паролей
		RequireComplexity:      true,
		LockoutThreshold:       3,  // 3 попытки (ФСТЭК)
		LockoutDurationMinutes: 30, // Блокировка 30 мин
	}
}

func (p *RUProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionRU},
		CrossBorderTransferAllowed: false, // Запрет трансграничной передачи (152-ФЗ)
		ColdStorageRegion:          RegionRU,
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *RUProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       1095, // 3 года (152-ФЗ)
		EventDataDays:      365,  // 1 год
		VideoDataDays:      30,   // 30 дней
		LegalHoldSupported: true,
		AutoDeleteEnabled:  false, // Ручное удаление (ФСТЭК)
	}
}

func (p *RUProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true,
		ChainHashPrev:   true,
		RetentionYears:  3, // 3 года (152-ФЗ)
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *RUProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       30, // 30 мин бездействия (ФСТЭК)
		MaxSessionHours:          8,  // 8 часов
		MaxConcurrentSessions:    1,  // 1 сессия
		FailedLoginLockout:       3,  // 3 попытки
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// INTL Profile — International (ISO 27001, ISO 27019, IEC 62443)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - ISO 27001:2022 — Information Security Management
//   - ISO 27019 — ICS/SCADA Security
//   - IEC 62443-3-3 — IACS Security
//   - OWASP ASVS L3 — Application Security
//
// INTL является fallback профилем для регионов без специфичных требований.

// INTLProfile implements ComplianceProfile для International.
type INTLProfile struct {
	*BaseProfile
}

// NewINTLProfile создаёт INTL Compliance Profile.
func NewINTLProfile() *INTLProfile {
	return &INTLProfile{
		BaseProfile: NewBaseProfile(RegionINTL,
			"ISO 27001 (International)",
			"Профиль соответствия International. ISO 27001, ISO 27019, IEC 62443, OWASP ASVS L3.",
		),
	}
}

func (p *INTLProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM
		KeySize:       256,
		AADRequired:   false,
		TLSMinVersion: "1.2",
	}
}

func (p *INTLProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   false,
		OutputSizeBits: 256,
	}
}

func (p *INTLProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256, // ECDSA P-256
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *INTLProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              8,
		RequireMFA:             false,
		MFATypes:               []MFAType{MFATOTP},
		MaxAgeDays:             0, // Без принудительной ротации (NIST)
		HistoryCount:           3,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *INTLProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             ValidRegions,
		CrossBorderTransferAllowed: true,
		ColdStorageRegion:          "",
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *INTLProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365, // 1 год (ISO 27001 A.12.4 minimum)
		EventDataDays:      90,  // 90 дней
		VideoDataDays:      30,  // 30 дней
		LegalHoldSupported: false,
		AutoDeleteEnabled:  true,
	}
}

func (p *INTLProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    false, // Опционально для INTL
		ChainHashPrev:   false,
		RetentionYears:  1, // 1 год
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *INTLProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       120, // 2 часа
		MaxSessionHours:          24,  // 24 часа
		MaxConcurrentSessions:    10,  // 10 сессий
		FailedLoginLockout:       5,
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// RegisterBaselineProfiles — регистрация всех baseline профилей
// ═══════════════════════════════════════════════════════════════════════════

// RegisterBaselineProfiles создаёт и регистрирует все baseline профили.
// Используется при startup для инициализации реестра.
//
// Возвращает настроенный ProfileRegistry с BY, EU и INTL профилями.
// INTL является обязательным профилем (startup фейлится если не зарегистрирован).
func RegisterBaselineProfiles(logger *slog.Logger) *ProfileRegistry {
	if logger == nil {
		logger = slog.Default()
	}

	registry := NewProfileRegistry(
		WithRequiredRegions(RegionINTL),
		WithLogger(logger),
		WithDefaultProfile(RegionINTL),
		WithProfile(NewBYProfile()),
		WithProfile(NewRUProfile()),
		WithProfile(NewEUProfile()),
		WithProfile(NewINTLProfile()),
	)

	// Валидация startup
	if err := registry.Validate(); err != nil {
		// Паникуем только если INTL не зарегистрирован
		logger.Error("compliance registry validation failed",
			"error", err,
		)
		panic(fmt.Sprintf("compliance registry: %v", err))
	}

	logger.Info("baseline compliance profiles registered",
		"profiles", registry.List(),
		"count", registry.Count(),
	)

	return registry
}
