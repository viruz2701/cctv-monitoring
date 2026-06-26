// Package compliance — Provider Registry for ComplianceProfile.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.1: Provider Registry
//
// Реализует thread-safe реестр ComplianceProfile провайдеров для
// runtime-загрузки по региону. Используется DI контейнером для
// инжекции профиля на основе tenant/instance config.
//
// Compliance:
//   - IEC 62443-3-3 SR 5.1 (Zone-based access control)
//   - ISO 27001 A.8.1 (Asset management — региональные профили)
//   - OWASP ASVS V1 (Architecture — централизованная конфигурация)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"fmt"
	"log/slog"
	"sync"
)

// ────────────────────────────────────────────────────────────────────────────
// ProfileRegistry
// ────────────────────────────────────────────────────────────────────────────

// ProfileRegistry предоставляет thread-safe реестр ComplianceProfile
// провайдеров для runtime-загрузки по региону.
//
// Особенности:
//   - Thread-safe (sync.RWMutex)
//   - Registration с защитой от дубликатов
//   - Startup validation (проверка наличия required профилей)
//   - Graceful fallback на INTL профиль при отсутствии специфичного
type ProfileRegistry struct {
	mu         sync.RWMutex
	profiles   map[string]ComplianceProfile
	required   []string // регионы, обязательные для startup
	defaultKey string   // профиль по умолчанию (обычно INTL)
	logger     *slog.Logger
}

// RegistryOption — функциональная опция для ProfileRegistry.
type RegistryOption func(*ProfileRegistry)

// WithRequiredRegions устанавливает обязательные регионы для startup.
// Если профиль для required региона не зарегистрирован, startup фейлится.
func WithRequiredRegions(regions ...string) RegistryOption {
	return func(r *ProfileRegistry) {
		r.required = append(r.required, regions...)
	}
}

// WithDefaultProfile устанавливает профиль по умолчанию.
func WithDefaultProfile(region string) RegistryOption {
	return func(r *ProfileRegistry) {
		r.defaultKey = region
	}
}

// WithLogger устанавливает логгер.
func WithLogger(logger *slog.Logger) RegistryOption {
	return func(r *ProfileRegistry) {
		r.logger = logger
	}
}

// NewProfileRegistry создаёт новый ProfileRegistry.
//
// required: список регионов, обязательных для startup.
// Если не указать WithRequiredRegions, проверяются все зарегистрированные.
func NewProfileRegistry(opts ...RegistryOption) *ProfileRegistry {
	r := &ProfileRegistry{
		profiles:   make(map[string]ComplianceProfile),
		required:   []string{RegionINTL}, // INTL обязателен всегда
		defaultKey: RegionINTL,
		logger:     slog.Default().With("component", "compliance.registry"),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Register регистрирует ComplianceProfile в реестре.
//
// Ошибки:
//   - ErrProfileAlreadyRegistered — если профиль для региона уже зарегистрирован
//   - ErrRegionMismatch — если регион в профиле не совпадает с ключом
func (r *ProfileRegistry) Register(profile ComplianceProfile) error {
	if profile == nil {
		return fmt.Errorf("compliance registry: cannot register nil profile")
	}

	if err := ValidateProfile(profile); err != nil {
		return fmt.Errorf("compliance registry: invalid profile: %w", err)
	}

	region := profile.Region()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.profiles[region]; exists {
		return fmt.Errorf("%w: %s", ErrProfileAlreadyRegistered, region)
	}

	r.profiles[region] = profile
	r.logger.Info("compliance profile registered",
		"region", region,
		"name", profile.Name(),
	)

	return nil
}

// MustRegister регистрирует профиль и паникует при ошибке.
// Используется для startup-регистрации built-in профилей.
func (r *ProfileRegistry) MustRegister(profile ComplianceProfile) {
	if err := r.Register(profile); err != nil {
		panic(fmt.Sprintf("compliance registry: %v", err))
	}
}

// Get возвращает ComplianceProfile для указанного региона.
//
// Если профиль для региона не найден, возвращает:
//   - INTL профиль (если зарегистрирован) — graceful fallback
//   - ErrProfileNotFound — если INTL тоже не зарегистрирован
func (r *ProfileRegistry) Get(region string) (ComplianceProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if profile, ok := r.profiles[region]; ok {
		return profile, nil
	}

	// Graceful fallback на INTL
	if region != RegionINTL {
		if profile, ok := r.profiles[RegionINTL]; ok {
			r.logger.Warn("compliance profile fallback",
				"requested_region", region,
				"fallback_region", RegionINTL,
			)
			return profile, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, region)
}

// MustGet возвращает ComplianceProfile для региона.
// Паникует если профиль не найден (использовать только при уверенности).
func (r *ProfileRegistry) MustGet(region string) ComplianceProfile {
	profile, err := r.Get(region)
	if err != nil {
		panic(fmt.Sprintf("compliance registry: %v", err))
	}
	return profile
}

// IsRegistered проверяет, зарегистрирован ли профиль для региона.
func (r *ProfileRegistry) IsRegistered(region string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.profiles[region]
	return ok
}

// List возвращает список всех зарегистрированных регионов.
func (r *ProfileRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	regions := make([]string, 0, len(r.profiles))
	for region := range r.profiles {
		regions = append(regions, region)
	}
	return regions
}

// Count возвращает количество зарегистрированных профилей.
func (r *ProfileRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.profiles)
}

// Validate проверяет, что все required профили зарегистрированы.
// Вызывается при startup для предотвращения работы без обязательных профилей.
//
// Возвращает ErrRequiredProfileMissing если хотя бы один required
// профиль не зарегистрирован.
func (r *ProfileRegistry) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var missing []string
	for _, region := range r.required {
		if _, ok := r.profiles[region]; !ok {
			missing = append(missing, region)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %v", ErrRequiredProfileMissing, missing)
	}

	r.logger.Info("compliance registry validated",
		"profiles_count", len(r.profiles),
		"required_ok", len(r.required),
	)

	return nil
}

// WithProfile возвращает RegistryOption для регистрации профиля.
// Удобно для цепочки опций при создании реестра.
func WithProfile(profile ComplianceProfile) RegistryOption {
	return func(r *ProfileRegistry) {
		r.MustRegister(profile)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Context helpers
// ────────────────────────────────────────────────────────────────────────────

// contextKey — тип для context key (избегаем коллизий).
type contextKey string

const (
	// ContextKeyRegion — ключ для хранения региона в context.
	ContextKeyRegion contextKey = "compliance_region"
	// ContextKeyProfile — ключ для хранения профиля в context.
	ContextKeyProfile contextKey = "compliance_profile"
)

// RegionFromRegistry возвращает регион из реестра по имени профиля.
func RegionFromRegistry(registry *ProfileRegistry, profileName string) (string, error) {
	for _, region := range registry.List() {
		p, err := registry.Get(region)
		if err != nil {
			continue
		}
		if p.Name() == profileName {
			return region, nil
		}
	}
	return "", fmt.Errorf("%w: profile name %s", ErrProfileNotFound, profileName)
}
