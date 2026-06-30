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

// ────────────────────────────────────────────────────────────────────────────
// Region constants — ValidRegions update (P2-CR.3: +CN, P2-CR.4: +US)
// ────────────────────────────────────────────────────────────────────────────

// init обновляет ValidRegions при загрузке пакета.
func init() {
	ValidRegions = []string{RegionBY, RegionEU, RegionINTL, RegionRU, RegionCN, RegionUS, RegionVN, RegionID, RegionNG, RegionKE}
}

// ────────────────────────────────────────────────────────────────────────────
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
// CN Profile — Китай (SM2/SM3/SM4, MLPS 2.0, Cybersecurity Law)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - GM/T 0002-2012 (SM4) — Block cipher
//   - GM/T 0003-2012 (SM2) — Public key cryptography
//   - GM/T 0004-2012 (SM3) — Hash function
//   - MLPS 2.0 (GB/T 22239-2019) — Multi-Level Protection Scheme
//   - China Cybersecurity Law (网络安全法) — Data localization
//   - PIPL (个人信息保护法) — Personal Information Protection
//   - DSL (数据安全法) — Data Security Law
//   - ISO 27001 A.5.1 — Information security policies

// CNProfile implements ComplianceProfile для Китая.
type CNProfile struct {
	*BaseProfile
}

// NewCNProfile создаёт CN Compliance Profile.
func NewCNProfile() *CNProfile {
	return &CNProfile{
		BaseProfile: NewBaseProfile(RegionCN,
			"SM (Китайская Народная Республика)",
			"Профиль соответствия для КНР. SM2/SM3/SM4, MLPS 2.0, Cybersecurity Law, PIPL.",
		),
	}
}

func (p *CNProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoSM4, // SM4 (GM/T 0002-2012)
		KeySize:       128,       // SM4 128-bit ключ
		AADRequired:   true,
		TLSMinVersion: "1.3",
	}
}

func (p *CNProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSM3, // SM3 (GM/T 0004-2012)
		SaltRequired:   true,
		OutputSizeBits: 256,
	}
}

func (p *CNProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureSM2, // SM2 (GM/T 0003-2012)
		Curve:       "sm2p256v1",
		HashForSign: HashSM3,
	}
}

func (p *CNProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              10, // MLPS 2.0: минимум 10 символов
		RequireMFA:             true,
		MFATypes:               []MFAType{MFATOTP, MFASMS},
		MaxAgeDays:             90, // Ротация 90 дней (MLPS 2.0)
		HistoryCount:           5,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *CNProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionCN},
		CrossBorderTransferAllowed: false, // Cybersecurity Law: data localization
		ColdStorageRegion:          RegionCN,
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *CNProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365, // 1 год (MLPS 2.0 Level 3)
		EventDataDays:      180, // 6 месяцев
		VideoDataDays:      90,  // 90 дней (MLPS 2.0 для video surveillance)
		LegalHoldSupported: true,
		AutoDeleteEnabled:  false, // Ручное удаление (PIPL)
	}
}

func (p *CNProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true,
		ChainHashPrev:   true,
		RetentionYears:  1, // 1 год (MLPS 2.0)
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *CNProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       30, // 30 мин бездействия (MLPS 2.0)
		MaxSessionHours:          12, // 12 часов
		MaxConcurrentSessions:    2,  // 2 сессии (MLPS 2.0)
		FailedLoginLockout:       5,
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
// VN Profile — Вьетнам (TCVN 11930:2017, Camera Standard 2025)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - TCVN 11930:2017 — Information security
//   - Camera Standard 2025 — CCTV-specific regulation
//   - Cyber Information Security Law (2015)
//   - Personal Data Protection (Decree 13/2023/ND-CP)
//   - Data residency requirements (localization)
//   - ISO 27001 A.5.1 — Information security policies
//   - IEC 62443-3-3 — IACS Security
//   - OWASP ASVS L3

// VNProfile implements ComplianceProfile для Вьетнама.
type VNProfile struct {
	*BaseProfile
}

// NewVNProfile создаёт VN Compliance Profile.
func NewVNProfile() *VNProfile {
	return &VNProfile{
		BaseProfile: NewBaseProfile(RegionVN,
			"TCVN 11930:2017 (Vietnam)",
			"Профиль соответствия для Вьетнама. TCVN 11930:2017, Camera Standard 2025, Decree 13/2023/ND-CP.",
		),
	}
}

func (p *VNProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM (международный, compatible)
		KeySize:       256,
		AADRequired:   true,  // TCVN 11930: AAD рекомендуется
		TLSMinVersion: "1.3", // Camera Standard 2025
	}
}

func (p *VNProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   true, // TCVN 11930: соль обязательна
		OutputSizeBits: 256,
	}
}

func (p *VNProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256, // ECDSA P-256
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *VNProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              8,
		RequireMFA:             true, // Camera Standard 2025: MFA required
		MFATypes:               []MFAType{MFATOTP, MFASMS},
		MaxAgeDays:             90, // Ротация 90 дней
		HistoryCount:           3,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *VNProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionVN},
		CrossBorderTransferAllowed: true, // С разрешения Субъекта (Decree 13)
		ColdStorageRegion:          RegionVN,
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true, // TCVN 11930
	}
}

func (p *VNProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       730, // 2 года (TCVN 11930)
		EventDataDays:      365, // 1 год
		VideoDataDays:      90,  // 90 дней (Camera Standard 2025)
		LegalHoldSupported: true,
		AutoDeleteEnabled:  true,
	}
}

func (p *VNProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true, // TCVN 11930 A.12.4
		ChainHashPrev:   false,
		RetentionYears:  2,
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *VNProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       30, // 30 мин (Camera Standard 2025)
		MaxSessionHours:          12,
		MaxConcurrentSessions:    3,
		FailedLoginLockout:       5,
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ID Profile — Индонезия (SNI 27001, UU PDP)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - SNI ISO/IEC 27001 — National ISMS standard (ISO 27001 equivalent)
//   - UU PDP (Law No. 27 of 2022) — Personal Data Protection
//   - Permenkominfo No. 20/2016 — Personal Data Protection
//   - ISO 27001 A.5.1 — Information security policies
//   - IEC 62443-3-3 — IACS Security

// IDProfile implements ComplianceProfile для Индонезии.
type IDProfile struct {
	*BaseProfile
}

// NewIDProfile создаёт ID Compliance Profile.
func NewIDProfile() *IDProfile {
	return &IDProfile{
		BaseProfile: NewBaseProfile(RegionID,
			"SNI 27001 / UU PDP (Indonesia)",
			"Профиль соответствия для Индонезии. SNI ISO/IEC 27001, UU PDP, Permenkominfo 20/2016.",
		),
	}
}

func (p *IDProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM
		KeySize:       256,
		AADRequired:   false,
		TLSMinVersion: "1.2", // SNI ISO/IEC 27001
	}
}

func (p *IDProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   false,
		OutputSizeBits: 256,
	}
}

func (p *IDProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256,
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *IDProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              8,
		RequireMFA:             false, // UU PDP не требует MFA
		MFATypes:               []MFAType{MFATOTP},
		MaxAgeDays:             0, // Без принудительной ротации
		HistoryCount:           3,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *IDProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionID},
		CrossBorderTransferAllowed: true, // UU PDP Art. 55: с согласия субъекта
		ColdStorageRegion:          RegionID,
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *IDProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365, // 1 год (UU PDP)
		EventDataDays:      180,
		VideoDataDays:      30,
		LegalHoldSupported: true,
		AutoDeleteEnabled:  true,
	}
}

func (p *IDProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true, // SNI ISO/IEC 27001 A.12.4
		ChainHashPrev:   false,
		RetentionYears:  1,
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *IDProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       60,
		MaxSessionHours:          24,
		MaxConcurrentSessions:    5,
		FailedLoginLockout:       5,
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// NG Profile — Нигерия (NDPR — Nigeria Data Protection Regulation)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - NDPR 2019 — Nigeria Data Protection Regulation
//   - NDPR 2022 — Implementation Framework
//   - NITDA Guidelines — National IT Development Agency
//   - ISO 27001 A.5.1 — Information security policies
//   - IEC 62443-3-3 — IACS Security
//   - Использует INTL baseline с NDPR-specific настройками

// NGProfile implements ComplianceProfile для Нигерии.
type NGProfile struct {
	*BaseProfile
}

// NewNGProfile создаёт NG Compliance Profile.
func NewNGProfile() *NGProfile {
	return &NGProfile{
		BaseProfile: NewBaseProfile(RegionNG,
			"NDPR (Nigeria)",
			"Профиль соответствия для Нигерии. NDPR 2019/2022, NITDA Guidelines.",
		),
	}
}

func (p *NGProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM
		KeySize:       256,
		AADRequired:   false,
		TLSMinVersion: "1.2",
	}
}

func (p *NGProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   false,
		OutputSizeBits: 256,
	}
}

func (p *NGProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256,
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *NGProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              8,
		RequireMFA:             false,
		MFATypes:               []MFAType{MFATOTP},
		MaxAgeDays:             0,
		HistoryCount:           3,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *NGProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             ValidRegions,
		CrossBorderTransferAllowed: true, // NDPR: с адекватной защитой
		ColdStorageRegion:          "",
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *NGProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365, // 1 год (NDPR)
		EventDataDays:      90,
		VideoDataDays:      30,
		LegalHoldSupported: false,
		AutoDeleteEnabled:  true,
	}
}

func (p *NGProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    false,
		ChainHashPrev:   false,
		RetentionYears:  1,
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *NGProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       120,
		MaxSessionHours:          24,
		MaxConcurrentSessions:    10,
		FailedLoginLockout:       5,
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// KE Profile — Кения (DPA 2019 — Data Protection Act)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - DPA 2019 — Kenya Data Protection Act
//   - Data Protection Regulations 2021
//   - Digital Health Act specified requirements
//   - ISO 27001 A.5.1 — Information security policies
//   - IEC 62443-3-3 — IACS Security
//   - M-Pesa integration security requirements (fintech adjacent)

// KEProfile implements ComplianceProfile для Кении.
type KEProfile struct {
	*BaseProfile
}

// NewKEProfile создаёт KE Compliance Profile.
func NewKEProfile() *KEProfile {
	return &KEProfile{
		BaseProfile: NewBaseProfile(RegionKE,
			"DPA 2019 (Kenya)",
			"Профиль соответствия для Кении. Data Protection Act 2019, Regulations 2021.",
		),
	}
}

func (p *KEProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM
		KeySize:       256,
		AADRequired:   true, // DPA 2019: дополнительная защита
		TLSMinVersion: "1.2",
	}
}

func (p *KEProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   true, // DPA 2019: соль для хешей
		OutputSizeBits: 256,
	}
}

func (p *KEProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256,
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *KEProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID,
		MinLength:              8,
		RequireMFA:             true,                       // DPA 2019: MPA рекомендуется
		MFATypes:               []MFAType{MFATOTP, MFASMS}, // SMS для M-Pesa регионов
		MaxAgeDays:             90,                         // Ротация 90 дней (DPA Regulations)
		HistoryCount:           3,
		RequireComplexity:      true,
		LockoutThreshold:       5,
		LockoutDurationMinutes: 15,
	}
}

func (p *KEProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionKE},
		CrossBorderTransferAllowed: true, // DPA 2019 Art. 51: с согласия
		ColdStorageRegion:          RegionKE,
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

func (p *KEProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365, // 1 год (DPA 2019)
		EventDataDays:      180,
		VideoDataDays:      30,
		LegalHoldSupported: true, // DPA 2019 Art. 26: legal hold
		AutoDeleteEnabled:  true,
	}
}

func (p *KEProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true, // DPA 2019: audit trail
		ChainHashPrev:   false,
		RetentionYears:  1,
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

func (p *KEProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       60,
		MaxSessionHours:          24,
		MaxConcurrentSessions:    5,
		FailedLoginLockout:       5,
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// US Profile — США (NIST SP 800-53, FedRAMP, FIPS 140-3)
// ═══════════════════════════════════════════════════════════════════════════
//
// Соответствие стандартам:
//   - NIST SP 800-53 Rev. 5 — Security and Privacy Controls
//   - FedRAMP Rev. 5 — Federal Risk and Authorization Management Program
//   - FIPS 140-3 — Cryptographic Module Validation
//   - FIPS 199 — Security Categorization
//   - HIPAA Security Rule — Healthcare (if applicable)
//   - SOC 2 Type II — Service Organization Controls
//   - ISO 27001 A.5.1 — Information security policies
//   - IEC 62443-3-3 — IACS Security

// USProfile implements ComplianceProfile для США.
type USProfile struct {
	*BaseProfile
}

// NewUSProfile создаёт US Compliance Profile.
func NewUSProfile() *USProfile {
	return &USProfile{
		BaseProfile: NewBaseProfile(RegionUS,
			"NIST SP 800-53 (United States)",
			"Профиль соответствия для США. NIST SP 800-53, FedRAMP, FIPS 140-3, SOC 2.",
		),
	}
}

func (p *USProfile) Crypto() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM, // AES-256-GCM (FIPS 140-3 validated)
		KeySize:       256,
		AADRequired:   false,
		TLSMinVersion: "1.2", // NIST SP 800-52 Rev. 2 (TLS 1.2+)
	}
}

func (p *USProfile) Hash() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256, // SHA-256 (FIPS 180-4)
		SaltRequired:   false,
		OutputSizeBits: 256,
	}
}

func (p *USProfile) Signature() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256, // ECDSA P-256 (FIPS 186-5)
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

func (p *USProfile) Password() PasswordPolicy {
	return PasswordPolicy{
		HashProvider:           PasswordArgon2ID, // NIST SP 800-63B
		MinLength:              8,                // NIST SP 800-63B A.3
		RequireMFA:             true,             // FedRAMP: MFA required
		MFATypes:               []MFAType{MFATOTP, MFAFIDO2},
		MaxAgeDays:             0, // NIST SP 800-63B: no forced rotation
		HistoryCount:           3,
		RequireComplexity:      false, // NIST SP 800-63B
		LockoutThreshold:       5,     // FedRAMP: 5 attempts
		LockoutDurationMinutes: 30,    // FedRAMP: 30 min lockout
	}
}

func (p *USProfile) DataResidency() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{RegionUS},
		CrossBorderTransferAllowed: true, // With adequate safeguards
		ColdStorageRegion:          RegionUS,
		StorageTiers:               []StorageTier{StorageHot, StorageCold, StorageArchive},
		RequireEncryptionAtRest:    true, // FedRAMP SC-13
	}
}

func (p *USProfile) Retention() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365,  // 1 год (NIST SP 800-53 AU-11)
		EventDataDays:      365,  // 1 год (FedRAMP AU-4)
		VideoDataDays:      90,   // 90 дней
		LegalHoldSupported: true, // FedRAMP: legal hold
		AutoDeleteEnabled:  true,
	}
}

func (p *USProfile) Audit() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true, // NIST SP 800-53 AU-3
		ChainHashPrev:   false,
		RetentionYears:  1,    // FedRAMP AU-11
		LogAllMutations: true, // FedRAMP AU-2
		IncludeTraceID:  true,
	}
}

func (p *USProfile) Session() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       30, // FedRAMP AC-12
		MaxSessionHours:          24,
		MaxConcurrentSessions:    3, // FedRAMP AC-10
		FailedLoginLockout:       5, // FedRAMP AC-7
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
// Возвращает настроенный ProfileRegistry с BY, EU, RU, CN, US и INTL профилями.
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
		WithProfile(NewCNProfile()),
		WithProfile(NewUSProfile()),
		WithProfile(NewVNProfile()),
		WithProfile(NewIDProfile()),
		WithProfile(NewNGProfile()),
		WithProfile(NewKEProfile()),
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
