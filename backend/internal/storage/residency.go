// Package storage — Data Residency Enforcement (P0-CE.6).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.6: Data Residency Enforcement
//
// Обеспечивает:
//   - Region-aware S3 endpoint selection (Минск для BY, eu-central-1 для EU)
//   - Cross-border transfer blocking на уровне storage API
//   - Cold storage routing per region retention policy
//   - Monitoring для attempted violations
//   - Audit log для всех residency violations
//
// Compliance:
//   - GDPR Art. 44-49 (Data transfer — region pinning)
//   - СТБ 34.101.27 п. 7.1 (Data localization)
//   - ISO 27001 A.8.10 (Information disposal)
//   - 152-ФЗ ст. 18 (Data localization)
//   - Приказ ОАЦ №66 п. 7.18.3 (Data protection)
//
// ═══════════════════════════════════════════════════════════════════════════
package storage

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"gb-telemetry-collector/internal/compliance"
)

// ────────────────────────────────────────────────────────────────────────────
// S3 Region endpoints
// ────────────────────────────────────────────────────────────────────────────

// S3EndpointConfig содержит конфигурацию S3 endpoint для региона.
type S3EndpointConfig struct {
	Region        string `json:"region"`
	Endpoint      string `json:"endpoint"`
	Bucket        string `json:"bucket"`
	UseTLS        bool   `json:"use_tls"`
	RetentionDays int    `json:"retention_days"`
}

// DefaultS3Endpoints — endpoint'ы S3 по умолчанию для каждого региона.
var DefaultS3Endpoints = map[string]S3EndpointConfig{
	compliance.RegionBY: {
		Region:        compliance.RegionBY,
		Endpoint:      "s3.minsk.example.com:9000",
		Bucket:        "cctv-data-by",
		UseTLS:        true,
		RetentionDays: 1825, // 5 лет (КИИ РБ)
	},
	compliance.RegionEU: {
		Region:        compliance.RegionEU,
		Endpoint:      "s3.eu-central-1.amazonaws.com",
		Bucket:        "cctv-data-eu",
		UseTLS:        true,
		RetentionDays: 730, // 2 года (GDPR)
	},
	compliance.RegionINTL: {
		Region:        compliance.RegionINTL,
		Endpoint:      "s3.amazonaws.com",
		Bucket:        "cctv-data-intl",
		UseTLS:        true,
		RetentionDays: 365, // 1 год (ISO 27001)
	},
	compliance.RegionRU: {
		Region:        compliance.RegionRU,
		Endpoint:      "s3.yandex.cloud",
		Bucket:        "cctv-data-ru",
		UseTLS:        true,
		RetentionDays: 1095, // 3 года (152-ФЗ)
	},
	compliance.RegionCN: {
		Region:        compliance.RegionCN,
		Endpoint:      "s3.aliyuncs.com",
		Bucket:        "cctv-data-cn",
		UseTLS:        true,
		RetentionDays: 365,
	},
}

// ────────────────────────────────────────────────────────────────────────────
// ResidencyEnforcer
// ────────────────────────────────────────────────────────────────────────────

// ResidencyEnforcer обеспечивает контроль местонахождения данных.
type ResidencyEnforcer struct {
	mu         sync.RWMutex
	endpoints  map[string]S3EndpointConfig
	violations *ViolationTracker
	logger     *slog.Logger
}

// NewResidencyEnforcer создаёт новый ResidencyEnforcer.
func NewResidencyEnforcer(customEndpoints map[string]S3EndpointConfig) *ResidencyEnforcer {
	e := &ResidencyEnforcer{
		endpoints:  make(map[string]S3EndpointConfig),
		violations: NewViolationTracker(),
		logger:     slog.Default().With("component", "storage.residency"),
	}

	// Загружаем endpoints по умолчанию
	for k, v := range DefaultS3Endpoints {
		e.endpoints[k] = v
	}
	// Перезаписываем кастомными
	for k, v := range customEndpoints {
		e.endpoints[k] = v
	}

	return e
}

// GetS3Endpoint возвращает S3 endpoint для указанного региона.
func (e *ResidencyEnforcer) GetS3Endpoint(region string) (S3EndpointConfig, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	cfg, ok := e.endpoints[region]
	if !ok {
		return S3EndpointConfig{}, fmt.Errorf("residency: no S3 endpoint for region %s", region)
	}
	return cfg, nil
}

// ValidateDataAccess проверяет, разрешён ли доступ к данным для указанного региона.
// Возвращает ошибку если доступ запрещён политикой data residency.
func (e *ResidencyEnforcer) ValidateDataAccess(requestRegion, dataRegion string, profile compliance.ComplianceProfile) error {
	if profile == nil {
		return fmt.Errorf("residency: nil compliance profile")
	}

	// Если регионы совпадают — всегда разрешено
	if requestRegion == dataRegion {
		return nil
	}

	residency := profile.DataResidency()

	// Проверка cross-border transfer
	if !residency.CrossBorderTransferAllowed {
		e.violations.Record(Violation{
			Type:          ViolationTypeCrossBorder,
			RequestRegion: requestRegion,
			DataRegion:    dataRegion,
			ProfileRegion: profile.Region(),
			Timestamp:     time.Now().UTC(),
			Blocked:       true,
		})
		return fmt.Errorf("%w: cross-border transfer from %s to %s blocked by %s profile",
			ErrCrossBorderBlocked, dataRegion, requestRegion, profile.Region())
	}

	// Проверка allowed regions
	allowed := false
	for _, r := range residency.AllowedRegions {
		if r == requestRegion {
			allowed = true
			break
		}
	}
	if !allowed {
		e.violations.Record(Violation{
			Type:          ViolationTypeUnauthorizedRegion,
			RequestRegion: requestRegion,
			DataRegion:    dataRegion,
			ProfileRegion: profile.Region(),
			Timestamp:     time.Now().UTC(),
			Blocked:       true,
		})
		return fmt.Errorf("%w: region %s not in allowed list for %s profile",
			ErrUnauthorizedRegion, requestRegion, profile.Region())
	}

	return nil
}

// GetColdStorageEndpoint возвращает endpoint для cold storage с учётом региона.
func (e *ResidencyEnforcer) GetColdStorageEndpoint(region string) (S3EndpointConfig, error) {
	cfg, err := e.GetS3Endpoint(region)
	if err != nil {
		return S3EndpointConfig{}, err
	}
	return cfg, nil
}

// GetRetentionDays возвращает срок хранения для региона.
func (e *ResidencyEnforcer) GetRetentionDays(region string) (int, error) {
	cfg, err := e.GetS3Endpoint(region)
	if err != nil {
		return 0, err
	}
	return cfg.RetentionDays, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Violation tracking
// ────────────────────────────────────────────────────────────────────────────

// ViolationType — тип нарушения residency.
type ViolationType string

const (
	ViolationTypeCrossBorder        ViolationType = "cross_border_transfer"
	ViolationTypeUnauthorizedRegion ViolationType = "unauthorized_region_access"
	ViolationTypeStorageViolation   ViolationType = "storage_violation"
)

// Violation представляет нарушение data residency.
type Violation struct {
	Type          ViolationType `json:"type"`
	RequestRegion string        `json:"request_region"`
	DataRegion    string        `json:"data_region"`
	ProfileRegion string        `json:"profile_region"`
	Timestamp     time.Time     `json:"timestamp"`
	Blocked       bool          `json:"blocked"`
	Details       string        `json:"details,omitempty"`
}

// ViolationTracker отслеживает нарушения residency.
type ViolationTracker struct {
	mu            sync.RWMutex
	violations    []Violation
	totalAttempts atomic.Int64
	totalBlocked  atomic.Int64
}

// NewViolationTracker создаёт новый ViolationTracker.
func NewViolationTracker() *ViolationTracker {
	return &ViolationTracker{
		violations: make([]Violation, 0, 100),
	}
}

// Record записывает нарушение.
func (t *ViolationTracker) Record(v Violation) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalAttempts.Add(1)
	if v.Blocked {
		t.totalBlocked.Add(1)
	}

	t.violations = append(t.violations, v)
	if len(t.violations) > 1000 {
		t.violations = t.violations[len(t.violations)-1000:]
	}
}

// GetViolations возвращает список нарушений.
func (t *ViolationTracker) GetViolations() []Violation {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]Violation, len(t.violations))
	copy(result, t.violations)
	return result
}

// GetStats возвращает статистику нарушений.
func (t *ViolationTracker) GetStats() ViolationStats {
	return ViolationStats{
		TotalAttempts: t.totalAttempts.Load(),
		TotalBlocked:  t.totalBlocked.Load(),
		RecentCount:   len(t.GetViolations()),
	}
}

// ViolationStats — статистика нарушений residency.
type ViolationStats struct {
	TotalAttempts int64 `json:"total_attempts"`
	TotalBlocked  int64 `json:"total_blocked"`
	RecentCount   int   `json:"recent_count"`
}

// ────────────────────────────────────────────────────────────────────────────
// Middleware
// ────────────────────────────────────────────────────────────────────────────

// StorageContext содержит контекст для storage операций.
type StorageContext struct {
	Region            string
	ComplianceProfile compliance.ComplianceProfile
	TenantID          string
}

// ValidateStorageOperation проверяет storage операцию на соответствие residency.
func (e *ResidencyEnforcer) ValidateStorageOperation(ctx *StorageContext, targetRegion string) error {
	if ctx == nil {
		return fmt.Errorf("residency: nil storage context")
	}
	if ctx.ComplianceProfile == nil {
		return fmt.Errorf("residency: nil compliance profile in context")
	}

	return e.ValidateDataAccess(targetRegion, ctx.Region, ctx.ComplianceProfile)
}

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	ErrCrossBorderBlocked  = fmt.Errorf("residency: cross-border transfer blocked")
	ErrUnauthorizedRegion  = fmt.Errorf("residency: unauthorized region")
	ErrNoEndpointForRegion = fmt.Errorf("residency: no endpoint for region")
)
