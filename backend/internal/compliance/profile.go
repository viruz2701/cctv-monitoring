// Package compliance — Compliance Profile Abstraction Layer (P0-CE.1).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.1: ComplianceProfile Abstraction Layer
//
// Проблема: Криптография и security policies захардкожены под РБ,
// блокирует выход на другие рынки.
//
// Решение:
//   - ComplianceProfile интерфейс с 8 policy методами
//   - Provider Registry для runtime-загрузки провайдеров по региону
//   - Inject через DI container на основе tenant/instance config
//   - 3 baseline профиля: BY (СТБ), EU (GDPR), INTL (ISO 27001)
//
// Compliance:
//   - IEC 62443-3-3 SR 5.1 (Zone-based access — региональные политики)
//   - ISO 27001 A.5.1 (Information security policies — региональные)
//   - ISO 27019 PCC.A.5 (ICS security policies)
//   - СТБ 34.101.27 п. 6.2 (Политики безопасности)
//   - СТБ 34.101.30 (Криптографические алгоритмы)
//   - OWASP ASVS V2 (Authentication), V3 (Session), V6 (Storage)
//   - Приказ ОАЦ № 66 п. 7.18 (Идентификация и защита узлов)
//   - GDPR Art. 44-49 (Data transfer), Art. 32 (Security)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"errors"
	"fmt"
)

// ────────────────────────────────────────────────────────────────────────────
// Region constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// RegionBY — Республика Беларусь (СТБ 34.101.27, СТБ 34.101.30, Приказ ОАЦ №66)
	RegionBY = "BY"
	// RegionEU — Европейский Союз (GDPR, NIS2, eIDAS)
	RegionEU = "EU"
	// RegionINTL — International (ISO 27001, ISO 27019, IEC 62443)
	RegionINTL = "INTL"
	// RegionRU — Российская Федерация (ГОСТ, 152-ФЗ, ФСТЭК) — stub
	RegionRU = "RU"
	// RegionCN — Китай (SM2/SM3/SM4, MLPS 2.0) — stub
	RegionCN = "CN"
	// RegionUS — США (FIPS 140-3, HIPAA, SOC 2) — stub
	RegionUS = "US"
)

// ValidRegions — список поддерживаемых регионов на текущем этапе.
var ValidRegions = []string{RegionBY, RegionEU, RegionINTL}

// ────────────────────────────────────────────────────────────────────────────
// CryptoPolicy — политика шифрования
// ────────────────────────────────────────────────────────────────────────────

// CryptoProviderType — тип криптографического провайдера.
type CryptoProviderType string

const (
	CryptoAES256GCM CryptoProviderType = "aes-256-gcm"   // Международный стандарт
	CryptoBeltGCM   CryptoProviderType = "belt-gcm"      // СТБ 34.101.31 (РБ)
	CryptoGOST      CryptoProviderType = "gost-28147-89" // ГОСТ (РФ) — stub
	CryptoSM4       CryptoProviderType = "sm4"           // 国密 (КНР) — stub
)

// CryptoPolicy определяет требования к шифрованию данных.
type CryptoPolicy struct {
	// Provider — используемый криптопровайдер.
	Provider CryptoProviderType `json:"provider"`
	// KeySize — размер ключа в битах (по умолчанию 256).
	KeySize int `json:"key_size"`
	// AADRequired — требуется ли Additional Authenticated Data.
	AADRequired bool `json:"aad_required"`
	// TLSMinVersion — минимальная версия TLS.
	TLSMinVersion string `json:"tls_min_version"`
}

// DefaultCryptoPolicy возвращает политику шифрования по умолчанию (AES-256-GCM).
func DefaultCryptoPolicy() CryptoPolicy {
	return CryptoPolicy{
		Provider:      CryptoAES256GCM,
		KeySize:       256,
		AADRequired:   false,
		TLSMinVersion: "1.2",
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HashPolicy — политика хеширования
// ────────────────────────────────────────────────────────────────────────────

// HashProviderType — тип хеш-провайдера.
type HashProviderType string

const (
	HashSHA256  HashProviderType = "sha256"      // Международный
	HashBash256 HashProviderType = "bash-256"    // СТБ 34.101.77 (РБ)
	HashStribog HashProviderType = "stribog-256" // ГОСТ Р 34.11-2012 (РФ) — stub
	HashSM3     HashProviderType = "sm3"         // 国密 SM3 (КНР) — stub
)

// HashPolicy определяет требования к хешированию.
type HashPolicy struct {
	// Provider — используемый хеш-провайдер.
	Provider HashProviderType `json:"provider"`
	// SaltRequired — требуется ли соль для хеширования.
	SaltRequired bool `json:"salt_required"`
	// OutputSizeBits — размер выхода в битах.
	OutputSizeBits int `json:"output_size_bits"`
}

// DefaultHashPolicy возвращает политику хеширования по умолчанию (SHA-256).
func DefaultHashPolicy() HashPolicy {
	return HashPolicy{
		Provider:       HashSHA256,
		SaltRequired:   false,
		OutputSizeBits: 256,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// SignaturePolicy — политика цифровых подписей
// ────────────────────────────────────────────────────────────────────────────

// SignatureProviderType — тип провайдера подписей.
type SignatureProviderType string

const (
	SignatureES256        SignatureProviderType = "es256"           // ECDSA P-256 (международный)
	SignatureBignCurve256 SignatureProviderType = "bign-curve256v1" // СТБ 34.101.45 (РБ)
	SignatureGOST3410     SignatureProviderType = "gost-3410-2012"  // ГОСТ Р 34.10-2012 (РФ) — stub
	SignatureSM2          SignatureProviderType = "sm2"             // 国密 SM2 (КНР) — stub
)

// SignaturePolicy определяет требования к цифровым подписям.
type SignaturePolicy struct {
	// Provider — используемый провайдер подписей.
	Provider SignatureProviderType `json:"provider"`
	// Curve — используемая эллиптическая кривая.
	Curve string `json:"curve"`
	// HashForSign — хеш-алгоритм для подписи.
	HashForSign HashProviderType `json:"hash_for_sign"`
}

// DefaultSignaturePolicy возвращает политику подписей по умолчанию (ES256).
func DefaultSignaturePolicy() SignaturePolicy {
	return SignaturePolicy{
		Provider:    SignatureES256,
		Curve:       "P-256",
		HashForSign: HashSHA256,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// PasswordPolicy — политика паролей
// ────────────────────────────────────────────────────────────────────────────

// PasswordHashProvider — тип хеширования паролей.
type PasswordHashProvider string

const (
	PasswordBCrypt   PasswordHashProvider = "bcrypt"    // Международный fallback
	PasswordArgon2ID PasswordHashProvider = "argon2id"  // Международный (рекомендуемый)
	PasswordBeltHash PasswordHashProvider = "belt-hash" // СТБ (РБ) — stub
)

// MFAType — тип MFA.
type MFAType string

const (
	MFANone  MFAType = "none"
	MFATOTP  MFAType = "totp"  // Time-based One-Time Password
	MFAFIDO2 MFAType = "fido2" // WebAuthn/FIDO2
	MFASMS   MFAType = "sms"   // SMS OTP
)

// PasswordPolicy определяет требования к паролям и аутентификации.
type PasswordPolicy struct {
	// HashProvider — алгоритм хеширования паролей.
	HashProvider PasswordHashProvider `json:"hash_provider"`
	// MinLength — минимальная длина пароля.
	MinLength int `json:"min_length"`
	// RequireMFA — обязательность MFA.
	RequireMFA bool `json:"require_mfa"`
	// MFATypes — доступные типы MFA.
	MFATypes []MFAType `json:"mfa_types"`
	// MaxAgeDays — максимальный возраст пароля (0 = без ограничения).
	MaxAgeDays int `json:"max_age_days"`
	// HistoryCount — количество предыдущих паролей для проверки.
	HistoryCount int `json:"history_count"`
	// RequireComplexity — требовать сложность (верхний/нижний/цифры/спецсимволы).
	RequireComplexity bool `json:"require_complexity"`
	// LockoutThreshold — количество неудачных попыток до блокировки.
	LockoutThreshold int `json:"lockout_threshold"`
	// LockoutDurationMinutes — длительность блокировки в минутах.
	LockoutDurationMinutes int `json:"lockout_duration_minutes"`
}

// DefaultPasswordPolicy возвращает политику паролей по умолчанию.
func DefaultPasswordPolicy() PasswordPolicy {
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

// ────────────────────────────────────────────────────────────────────────────
// DataResidencyPolicy — политика местонахождения данных
// ────────────────────────────────────────────────────────────────────────────

// StorageTier — уровень хранения.
type StorageTier string

const (
	StorageHot     StorageTier = "hot"
	StorageCold    StorageTier = "cold"
	StorageArchive StorageTier = "archive"
)

// DataResidencyPolicy определяет требования к местонахождению данных.
type DataResidencyPolicy struct {
	// AllowedRegions — список регионов, где могут храниться данные.
	AllowedRegions []string `json:"allowed_regions"`
	// CrossBorderTransferAllowed — разрешена ли трансграничная передача.
	CrossBorderTransferAllowed bool `json:"cross_border_transfer_allowed"`
	// ColdStorageRegion — регион для cold storage (пусто = то же, что primary).
	ColdStorageRegion string `json:"cold_storage_region"`
	// StorageTiers — доступные уровни хранения.
	StorageTiers []StorageTier `json:"storage_tiers"`
	// RequireEncryptionAtRest — обязательное шифрование at rest.
	RequireEncryptionAtRest bool `json:"require_encryption_at_rest"`
}

// DefaultDataResidencyPolicy возвращает политику местонахождения по умолчанию.
func DefaultDataResidencyPolicy() DataResidencyPolicy {
	return DataResidencyPolicy{
		AllowedRegions:             []string{"INTL"},
		CrossBorderTransferAllowed: true,
		ColdStorageRegion:          "",
		StorageTiers:               []StorageTier{StorageHot, StorageCold},
		RequireEncryptionAtRest:    true,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// RetentionPolicy — политика хранения данных
// ────────────────────────────────────────────────────────────────────────────

// RetentionPolicy определяет требования к срокам хранения данных.
type RetentionPolicy struct {
	// AuditLogDays — срок хранения логов аудита (дней).
	AuditLogDays int `json:"audit_log_days"`
	// EventDataDays — срок хранения событий (дней).
	EventDataDays int `json:"event_data_days"`
	// VideoDataDays — срок хранения видеоархива (дней).
	VideoDataDays int `json:"video_data_days"`
	// LegalHoldSupported — поддержка legal hold.
	LegalHoldSupported bool `json:"legal_hold_supported"`
	// AutoDeleteEnabled — автоматическое удаление по истечении срока.
	AutoDeleteEnabled bool `json:"auto_delete_enabled"`
}

// DefaultRetentionPolicy возвращает политику хранения по умолчанию.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		AuditLogDays:       365,
		EventDataDays:      90,
		VideoDataDays:      30,
		LegalHoldSupported: false,
		AutoDeleteEnabled:  true,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// AuditPolicy — политика аудита
// ────────────────────────────────────────────────────────────────────────────

// AuditPolicy определяет требования к аудиту и логированию.
type AuditPolicy struct {
	// HMACRequired — требуется ли HMAC-подпись для логов.
	HMACRequired bool `json:"hmac_required"`
	// ChainHashPrev — требуется ли связывание с предыдущим хешем (tamper detection).
	ChainHashPrev bool `json:"chain_hash_prev"`
	// RetentionYears — срок хранения аудита в годах.
	RetentionYears int `json:"retention_years"`
	// LogAllMutations — логировать все мутации данных.
	LogAllMutations bool `json:"log_all_mutations"`
	// IncludeTraceID — включать traceID в логи.
	IncludeTraceID bool `json:"include_trace_id"`
}

// DefaultAuditPolicy возвращает политику аудита по умолчанию.
func DefaultAuditPolicy() AuditPolicy {
	return AuditPolicy{
		HMACRequired:    true,
		ChainHashPrev:   false,
		RetentionYears:  1,
		LogAllMutations: true,
		IncludeTraceID:  true,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// SessionPolicy — политика сессий
// ────────────────────────────────────────────────────────────────────────────

// SessionPolicy определяет требования к управлению сессиями.
type SessionPolicy struct {
	// IdleTimeoutMinutes — таймаут бездействия (минут).
	IdleTimeoutMinutes int `json:"idle_timeout_minutes"`
	// MaxSessionHours — максимальная длительность сессии (часов).
	MaxSessionHours int `json:"max_session_hours"`
	// MaxConcurrentSessions — максимальное количество одновременных сессий.
	MaxConcurrentSessions int `json:"max_concurrent_sessions"`
	// FailedLoginLockout — количество неудачных попыток до блокировки.
	FailedLoginLockout int `json:"failed_login_lockout"`
	// RequireRefreshToken — требовать refresh token.
	RequireRefreshToken bool `json:"require_refresh_token"`
	// WarnBeforeTimeoutMinutes — предупреждение за N минут до таймаута.
	WarnBeforeTimeoutMinutes int `json:"warn_before_timeout_minutes"`
}

// DefaultSessionPolicy возвращает политику сессий по умолчанию.
func DefaultSessionPolicy() SessionPolicy {
	return SessionPolicy{
		IdleTimeoutMinutes:       60,
		MaxSessionHours:          24,
		MaxConcurrentSessions:    5,
		FailedLoginLockout:       5,
		RequireRefreshToken:      true,
		WarnBeforeTimeoutMinutes: 5,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ComplianceProfile — основной интерфейс
// ────────────────────────────────────────────────────────────────────────────

// ComplianceProfile определяет полный набор политик безопасности и соответствия
// для конкретного региона/рынка.
//
// Содержит 8 policy методов, покрывающих:
//   - Криптографию (шифрование, хеширование, подписи)
//   - Парольную политику
//   - Data residency
//   - Retention данных
//   - Аудит и логирование
//   - Управление сессиями
type ComplianceProfile interface {
	// Region возвращает код региона (BY, EU, INTL, и т.д.).
	Region() string

	// Name возвращает человекочитаемое название профиля.
	Name() string

	// Description возвращает описание профиля.
	Description() string

	// Crypto возвращает политику шифрования.
	Crypto() CryptoPolicy

	// Hash возвращает политику хеширования.
	Hash() HashPolicy

	// Signature возвращает политику цифровых подписей.
	Signature() SignaturePolicy

	// Password возвращает политику паролей.
	Password() PasswordPolicy

	// DataResidency возвращает политику местонахождения данных.
	DataResidency() DataResidencyPolicy

	// Retention возвращает политику хранения данных.
	Retention() RetentionPolicy

	// Audit возвращает политику аудита.
	Audit() AuditPolicy

	// Session возвращает политику сессий.
	Session() SessionPolicy
}

// ────────────────────────────────────────────────────────────────────────────
// BaseProfile — базовая структура для встраивания профилей
// ────────────────────────────────────────────────────────────────────────────

// BaseProfile содержит общие поля для всех ComplianceProfile реализаций.
// Используется для композиции в конкретных профилях.
type BaseProfile struct {
	region      string
	name        string
	description string

	cryptoPolicies    []func() CryptoPolicy
	hashPolicies      []func() HashPolicy
	signaturePolicies []func() SignaturePolicy
	passwordPolicies  []func() PasswordPolicy
	residencyPolicies []func() DataResidencyPolicy
	retentionPolicies []func() RetentionPolicy
	auditPolicies     []func() AuditPolicy
	sessionPolicies   []func() SessionPolicy
}

// NewBaseProfile создаёт новый базовый профиль.
func NewBaseProfile(region, name, description string) *BaseProfile {
	return &BaseProfile{
		region:      region,
		name:        name,
		description: description,
	}
}

func (b *BaseProfile) Region() string      { return b.region }
func (b *BaseProfile) Name() string        { return b.name }
func (b *BaseProfile) Description() string { return b.description }

// ────────────────────────────────────────────────────────────────────────────
// Validation
// ────────────────────────────────────────────────────────────────────────────

// ValidateProfile проверяет ComplianceProfile на корректность.
func ValidateProfile(p ComplianceProfile) error {
	if p == nil {
		return errors.New("compliance profile: cannot be nil")
	}

	region := p.Region()
	if region == "" {
		return errors.New("compliance profile: region cannot be empty")
	}

	crypto := p.Crypto()
	if crypto.Provider == "" {
		return fmt.Errorf("compliance profile %s: crypto provider cannot be empty", region)
	}

	hash := p.Hash()
	if hash.Provider == "" {
		return fmt.Errorf("compliance profile %s: hash provider cannot be empty", region)
	}

	sig := p.Signature()
	if sig.Provider == "" {
		return fmt.Errorf("compliance profile %s: signature provider cannot be empty", region)
	}

	pwd := p.Password()
	if pwd.HashProvider == "" {
		return fmt.Errorf("compliance profile %s: password hash provider cannot be empty", region)
	}
	if pwd.MinLength < 8 {
		return fmt.Errorf("compliance profile %s: password min length must be >= 8, got %d", region, pwd.MinLength)
	}

	residency := p.DataResidency()
	if len(residency.AllowedRegions) == 0 {
		return fmt.Errorf("compliance profile %s: at least one allowed region required", region)
	}

	retention := p.Retention()
	if retention.AuditLogDays <= 0 {
		return fmt.Errorf("compliance profile %s: audit log retention must be > 0", region)
	}

	session := p.Session()
	if session.IdleTimeoutMinutes <= 0 {
		return fmt.Errorf("compliance profile %s: idle timeout must be > 0", region)
	}

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	// ErrProfileNotFound возвращается, если профиль не найден в реестре.
	ErrProfileNotFound = errors.New("compliance profile not found")

	// ErrProfileAlreadyRegistered возвращается при повторной регистрации.
	ErrProfileAlreadyRegistered = errors.New("compliance profile already registered")

	// ErrRequiredProfileMissing возвращается при отсутствии обязательного профиля.
	ErrRequiredProfileMissing = errors.New("required compliance profile missing")

	// ErrRegionMismatch возвращается при несовпадении региона.
	ErrRegionMismatch = errors.New("compliance region mismatch")
)
