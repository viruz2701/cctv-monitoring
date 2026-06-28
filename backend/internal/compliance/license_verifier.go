// Package compliance — License Verification System (P1-REG.7).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-REG.7: License Verification System
//
// Обеспечивает:
//   - Auto-check license expiration
//   - Alert 30 дней до expiration
//   - Block WO assignment to unlicensed vendors
//   - Integration с government registries (где есть API)
//
// Compliance:
//   - Приказ МЧС №55 (лицензия обязательна с 01.02.2026)
//   - СН 3.02.19-2025 (ОАЦ лицензия для КИИ РБ)
//   - РД 25.964-90 (МЧС лицензия для РФ)
//   - ISO 27001 A.9.2.1 (Vendor management)
//   - IEC 62443 SR 2.1 (Account management — vendor access)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// LicenseStatus — статус лицензии.
type LicenseStatus string

const (
	LicenseStatusActive       LicenseStatus = "active"
	LicenseStatusExpiringSoon LicenseStatus = "expiring_soon" // < 30 дней
	LicenseStatusExpired      LicenseStatus = "expired"
	LicenseStatusRevoked      LicenseStatus = "revoked"
	LicenseStatusSuspended    LicenseStatus = "suspended"
)

// LicenseType — тип лицензии.
type LicenseType string

const (
	LicenseTypeMCHS    LicenseType = "mchs"    // МЧС (РФ, РБ)
	LicenseTypeOAC     LicenseType = "oac"     // ОАЦ (РБ)
	LicenseTypeFSTEC   LicenseType = "fstek"   // ФСТЭК (РФ)
	LicenseTypeKVKK    LicenseType = "kvkk"    // KVKK (TR)
	LicenseTypePOPIA   LicenseType = "popia"   // POPIA (ZA)
	LicenseTypeGeneral LicenseType = "general" // Общая
)

// VendorLicense представляет лицензию подрядчика.
type VendorLicense struct {
	ID              string        `json:"id"`
	VendorID        string        `json:"vendor_id"`
	VendorName      string        `json:"vendor_name"`
	LicenseType     LicenseType   `json:"license_type"`
	LicenseNumber   string        `json:"license_number"`
	Region          string        `json:"region"`
	IssuedAt        time.Time     `json:"issued_at"`
	ExpiresAt       time.Time     `json:"expires_at"`
	Status          LicenseStatus `json:"status"`
	VerificationURL string        `json:"verification_url,omitempty"`
	Notes           string        `json:"notes,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// LicenseCheckResult — результат проверки лицензии.
type LicenseCheckResult struct {
	LicenseID       string        `json:"license_id"`
	VendorID        string        `json:"vendor_id"`
	VendorName      string        `json:"vendor_name"`
	PreviousStatus  LicenseStatus `json:"previous_status"`
	NewStatus       LicenseStatus `json:"new_status"`
	DaysUntilExpiry int           `json:"days_until_expiry"`
	Changed         bool          `json:"changed"`
	CheckedAt       time.Time     `json:"checked_at"`
}

// ────────────────────────────────────────────────────────────────────────────
// LicenseStore — интерфейс для хранения лицензий
// ────────────────────────────────────────────────────────────────────────────

// LicenseStore определяет методы для работы с лицензиями подрядчиков.
type LicenseStore interface {
	// GetByID возвращает лицензию по ID.
	GetByID(ctx context.Context, id string) (*VendorLicense, error)

	// GetByVendor возвращает все лицензии подрядчика.
	GetByVendor(ctx context.Context, vendorID string) ([]VendorLicense, error)

	// ListByStatus возвращает лицензии с указанным статусом.
	ListByStatus(ctx context.Context, status LicenseStatus) ([]VendorLicense, error)

	// ListExpiringSoon возвращает лицензии, истекающие в ближайшие N дней.
	ListExpiringSoon(ctx context.Context, days int) ([]VendorLicense, error)

	// ListAll возвращает все лицензии.
	ListAll(ctx context.Context) ([]VendorLicense, error)

	// UpdateStatus обновляет статус лицензии.
	UpdateStatus(ctx context.Context, id string, status LicenseStatus) error

	// LogCheck записывает результат проверки в audit_log.
	LogCheck(ctx context.Context, result LicenseCheckResult) error

	// GetVendorsByRegion возвращает подрядчиков для региона.
	GetVendorsByRegion(ctx context.Context, region string) ([]string, error)
}

// ────────────────────────────────────────────────────────────────────────────
// LicenseVerifier
// ────────────────────────────────────────────────────────────────────────────

// LicenseVerifier проверяет статус лицензий подрядчиков.
type LicenseVerifier struct {
	store  LicenseStore
	logger *slog.Logger

	// alertBeforeDays — за сколько дней до истечения начинать alert.
	alertBeforeDays int

	// checkInterval — интервал автоматической проверки.
	checkInterval time.Duration

	mu     sync.RWMutex
	checks map[string]time.Time // license_id → last check time
}

// LicenseVerifierOption — функциональная опция для LicenseVerifier.
type LicenseVerifierOption func(*LicenseVerifier)

// WithAlertBeforeDays устанавливает количество дней до истечения для alert.
func WithAlertBeforeDays(days int) LicenseVerifierOption {
	return func(v *LicenseVerifier) {
		if days > 0 {
			v.alertBeforeDays = days
		}
	}
}

// WithCheckInterval устанавливает интервал автоматической проверки.
func WithCheckInterval(interval time.Duration) LicenseVerifierOption {
	return func(v *LicenseVerifier) {
		if interval > 0 {
			v.checkInterval = interval
		}
	}
}

// WithVerifierLogger устанавливает логгер для LicenseVerifier.
func WithVerifierLogger(logger *slog.Logger) LicenseVerifierOption {
	return func(v *LicenseVerifier) {
		v.logger = logger
	}
}

// NewLicenseVerifier создаёт новый LicenseVerifier.
func NewLicenseVerifier(store LicenseStore, opts ...LicenseVerifierOption) *LicenseVerifier {
	v := &LicenseVerifier{
		store:           store,
		logger:          slog.Default().With("component", "compliance.license_verifier"),
		alertBeforeDays: 30,
		checkInterval:   24 * time.Hour,
		checks:          make(map[string]time.Time),
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// ────────────────────────────────────────────────────────────────────────────
// Core operations
// ────────────────────────────────────────────────────────────────────────────

// CheckLicense проверяет статус конкретной лицензии.
// Обновляет статус на основе даты истечения.
func (v *LicenseVerifier) CheckLicense(ctx context.Context, licenseID string) (*LicenseCheckResult, error) {
	license, err := v.store.GetByID(ctx, licenseID)
	if err != nil {
		return nil, fmt.Errorf("get license %s: %w", licenseID, err)
	}
	if license == nil {
		return nil, fmt.Errorf("license %s not found", licenseID)
	}

	result := v.evaluateLicense(license)

	// Если статус изменился — обновляем
	if result.Changed {
		if err := v.store.UpdateStatus(ctx, licenseID, result.NewStatus); err != nil {
			return nil, fmt.Errorf("update license %s status: %w", licenseID, err)
		}

		v.logger.Warn("license status changed",
			"vendor_id", license.VendorID,
			"vendor_name", license.VendorName,
			"previous", result.PreviousStatus,
			"new", result.NewStatus,
			"days_remaining", result.DaysUntilExpiry,
		)
	}

	// Логируем проверку в audit_log
	if err := v.store.LogCheck(ctx, *result); err != nil {
		v.logger.Error("failed to log license check",
			"license_id", licenseID,
			"error", err,
		)
	}

	// Кешируем время проверки
	v.mu.Lock()
	v.checks[licenseID] = time.Now()
	v.mu.Unlock()

	return result, nil
}

// CheckAllLicenses проверяет все лицензии в системе.
// Возвращает список изменений.
func (v *LicenseVerifier) CheckAllLicenses(ctx context.Context) ([]LicenseCheckResult, error) {
	licenses, err := v.store.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all licenses: %w", err)
	}

	var results []LicenseCheckResult
	for _, license := range licenses {
		result, err := v.CheckLicense(ctx, license.ID)
		if err != nil {
			v.logger.Error("license check failed",
				"license_id", license.ID,
				"vendor_id", license.VendorID,
				"error", err,
			)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// CheckVendorLicenses проверяет все лицензии конкретного подрядчика.
func (v *LicenseVerifier) CheckVendorLicenses(ctx context.Context, vendorID string) ([]LicenseCheckResult, error) {
	licenses, err := v.store.GetByVendor(ctx, vendorID)
	if err != nil {
		return nil, fmt.Errorf("get vendor %s licenses: %w", vendorID, err)
	}

	var results []LicenseCheckResult
	for _, license := range licenses {
		result, err := v.CheckLicense(ctx, license.ID)
		if err != nil {
			return nil, fmt.Errorf("check license %s: %w", license.ID, err)
		}
		results = append(results, *result)
	}

	return results, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Vendor verification (WO assignment)
// ────────────────────────────────────────────────────────────────────────────

// VerifyVendorForWO проверяет, может ли подрядчик выполнять работы.
//
// Возвращает ошибку если:
//   - У подрядчика нет активной лицензии для региона
//   - Лицензия истекает (предупреждение)
//   - Лицензия отозвана или приостановлена
func (v *LicenseVerifier) VerifyVendorForWO(ctx context.Context, vendorID, region string) error {
	licenses, err := v.store.GetByVendor(ctx, vendorID)
	if err != nil {
		return fmt.Errorf("verify vendor %s: %w", vendorID, err)
	}

	if len(licenses) == 0 {
		return fmt.Errorf("vendor %s has no licenses for region %s", vendorID, region)
	}

	// Проверяем каждую лицензию
	var activeLicense bool
	var expiringLicense bool

	for _, license := range licenses {
		// Пропускаем лицензии не для этого региона
		if license.Region != region {
			continue
		}

		switch license.Status {
		case LicenseStatusActive:
			activeLicense = true
		case LicenseStatusExpiringSoon:
			activeLicense = true
			expiringLicense = true
		case LicenseStatusExpired, LicenseStatusRevoked, LicenseStatusSuspended:
			// Неактивные лицензии
			continue
		}
	}

	if !activeLicense {
		return fmt.Errorf("vendor %s has no valid license for region %s: "+
			"WO assignment blocked (compliance: Приказ МЧС №55 / СН 3.02.19-2025)",
			vendorID, region)
	}

	// Если лицензия истекает — логируем предупреждение
	if expiringLicense {
		v.logger.Warn("vendor license expiring soon",
			"vendor_id", vendorID,
			"region", region,
		)
	}

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Alert generation
// ────────────────────────────────────────────────────────────────────────────

// ExpirationAlert содержит информацию об истекающей лицензии.
type ExpirationAlert struct {
	License        VendorLicense `json:"license"`
	DaysRemaining  int           `json:"days_remaining"`
	AlertThreshold int           `json:"alert_threshold"`
	Severity       string        `json:"severity"` // "info" | "warning" | "critical"
}

// GetExpirationAlerts возвращает список лицензий, требующих внимания.
//
// Уровни severity:
//   - critical: просрочена или < 7 дней
//   - warning: 7-14 дней
//   - info: 15-30 дней
func (v *LicenseVerifier) GetExpirationAlerts(ctx context.Context) ([]ExpirationAlert, error) {
	expiring, err := v.store.ListExpiringSoon(ctx, v.alertBeforeDays)
	if err != nil {
		return nil, fmt.Errorf("list expiring licenses: %w", err)
	}

	alerts := make([]ExpirationAlert, 0, len(expiring))
	for _, license := range expiring {
		daysRemaining := int(time.Until(license.ExpiresAt).Hours() / 24)

		severity := "info"
		if daysRemaining <= 0 {
			severity = "critical"
		} else if daysRemaining <= 7 {
			severity = "critical"
		} else if daysRemaining <= 14 {
			severity = "warning"
		}

		alerts = append(alerts, ExpirationAlert{
			License:        license,
			DaysRemaining:  daysRemaining,
			AlertThreshold: v.alertBeforeDays,
			Severity:       severity,
		})
	}

	return alerts, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ────────────────────────────────────────────────────────────────────────────

// evaluateLicense определяет новый статус лицензии на основе даты истечения.
func (v *LicenseVerifier) evaluateLicense(license *VendorLicense) *LicenseCheckResult {
	daysUntilExpiry := int(time.Until(license.ExpiresAt).Hours() / 24)
	newStatus := license.Status

	switch {
	case daysUntilExpiry <= 0:
		newStatus = LicenseStatusExpired
	case daysUntilExpiry <= v.alertBeforeDays:
		newStatus = LicenseStatusExpiringSoon
	default:
		newStatus = LicenseStatusActive
	}

	// Не перезаписываем revoked/suspended — они требуют ручного вмешательства
	if license.Status == LicenseStatusRevoked || license.Status == LicenseStatusSuspended {
		newStatus = license.Status
	}

	changed := newStatus != license.Status

	return &LicenseCheckResult{
		LicenseID:       license.ID,
		VendorID:        license.VendorID,
		VendorName:      license.VendorName,
		PreviousStatus:  license.Status,
		NewStatus:       newStatus,
		DaysUntilExpiry: daysUntilExpiry,
		Changed:         changed,
		CheckedAt:       time.Now().UTC(),
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Government registry integration (stub)
// ────────────────────────────────────────────────────────────────────────────

// GovernmentRegistryClient — интерфейс для проверки лицензий через
// государственные реестры.
//
// Поддерживаемые реестры:
//   - МЧС РФ: https://reestr.mchs.gov.ru
//   - ОАЦ РБ: https://oac.gov.by/reestr
//   - МЧС РК: https://emer.gov.kz/reestr
type GovernmentRegistryClient interface {
	// VerifyLicense проверяет лицензию через госреестр.
	VerifyLicense(ctx context.Context, licenseNumber, region string) (*GovernmentRegistryResponse, error)
}

// GovernmentRegistryResponse — ответ от госреестра.
type GovernmentRegistryResponse struct {
	Valid           bool   `json:"valid"`
	LicenseNumber   string `json:"license_number"`
	HolderName      string `json:"holder_name"`
	Status          string `json:"status"`
	ExpirationDate  string `json:"expiration_date"`
	RegisteredAt    string `json:"registered_at"`
	VerificationURL string `json:"verification_url"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// VerifyWithGovernmentRegistry проверяет лицензию через госреестр.
// Если регистр недоступен — возвращает warning, но не блокирует операцию.
func (v *LicenseVerifier) VerifyWithGovernmentRegistry(ctx context.Context, client GovernmentRegistryClient, licenseID string) (*GovernmentRegistryResponse, error) {
	license, err := v.store.GetByID(ctx, licenseID)
	if err != nil {
		return nil, fmt.Errorf("get license %s: %w", licenseID, err)
	}
	if license == nil {
		return nil, fmt.Errorf("license %s not found", licenseID)
	}

	resp, err := client.VerifyLicense(ctx, license.LicenseNumber, license.Region)
	if err != nil {
		v.logger.Warn("government registry check failed (non-blocking)",
			"license_id", licenseID,
			"license_number", license.LicenseNumber,
			"region", license.Region,
			"error", err,
		)
		return nil, fmt.Errorf("government registry unavailable for %s: %w", license.Region, err)
	}

	return resp, nil
}
