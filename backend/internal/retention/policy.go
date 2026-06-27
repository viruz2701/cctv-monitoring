// Package retention — Regional retention policies with lifecycle management.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.1: Regional Retention Policies
//   - 5+ retention profiles (BY/RU/EU/US/CN)
//   - Automated lifecycle transitions (hot → cold → archive → delete)
//   - Compliance-aware deletion (legal hold support)
//   - Audit log for all retention actions
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — logging of retention actions)
//   - IEC 62443 SR 2.3 (Data integrity — retention enforcement)
//   - CTБ 34.101.27 п. 7.2 (Целостность журналов аудита)
//   - GDPR Art. 5(1)(e) (Storage limitation — data minimization)
//   - GDPR Art. 17-19 (Right to erasure / restriction)
//   - Приказ ОАЦ №66 п. 7.18.3 (Хранение логов)
//
// ═══════════════════════════════════════════════════════════════════════════
package retention

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// DataType — тип данных, для которых применяется политика хранения.
type DataType string

const (
	DataAudit      DataType = "audit"
	DataTelemetry  DataType = "telemetry"
	DataAlerts     DataType = "alerts"
	DataImages     DataType = "images"
	DataReports    DataType = "reports"
	DataWorkOrders DataType = "work_orders"
	DataVideo      DataType = "video"
)

// Region — регион применения политики хранения.
type Region string

const (
	RegionBY Region = "BY" // Беларусь — КИИ, 5 лет audit
	RegionRU Region = "RU" // Россия — ФЗ-152, 3 года audit
	RegionEU Region = "EU" // EU — GDPR, min necessary
	RegionUS Region = "US" // USA — SOX/HIPAA, 7 лет audit
	RegionCN Region = "CN" // Китай — Cybersecurity Law, 30d telemetry
)

// LifecycleStage — стадия жизненного цикла данных.
type LifecycleStage string

const (
	StageHot     LifecycleStage = "hot"
	StageCold    LifecycleStage = "cold"
	StageArchive LifecycleStage = "archive"
	StageDelete  LifecycleStage = "delete"
)

// ────────────────────────────────────────────────────────────────────────────
// RetentionPolicy
// ────────────────────────────────────────────────────────────────────────────

// RetentionPolicy определяет политику хранения для региона + типа данных.
// Использует TTL (time-to-live) для каждой стадии lifecycle.
type RetentionPolicy struct {
	Region     Region        `json:"region"`
	DataType   DataType      `json:"data_type"`
	HotTTL     time.Duration `json:"hot_ttl"`     // Быстрый доступ (hot storage)
	ColdTTL    time.Duration `json:"cold_ttl"`    // Холодное хранение (cold storage)
	ArchiveTTL time.Duration `json:"archive_ttl"` // Архивное хранение
	DeleteTTL  time.Duration `json:"delete_ttl"`  // Полное удаление
	TotalTTL   time.Duration `json:"total_ttl"`   // Максимальный срок жизни (сумма)
}

// Validate проверяет корректность политики.
func (p *RetentionPolicy) Validate() error {
	if p.Region == "" {
		return fmt.Errorf("region is required")
	}
	if p.DataType == "" {
		return fmt.Errorf("data_type is required")
	}
	if p.TotalTTL <= 0 {
		return fmt.Errorf("total_ttl must be positive")
	}
	if p.DeleteTTL > 0 && p.DeleteTTL < p.TotalTTL {
		return fmt.Errorf("delete_ttl must be >= total_ttl (got %s < %s)", p.DeleteTTL, p.TotalTTL)
	}
	return nil
}

// LifecycleDecision — результат оценки lifecycle для порции данных.
type LifecycleDecision struct {
	TenantID     string           `json:"tenant_id"`
	DataType     DataType         `json:"data_type"`
	Region       Region           `json:"region"`
	DataAge      time.Duration    `json:"data_age"`
	CurrentStage LifecycleStage   `json:"current_stage"`
	NextStage    LifecycleStage   `json:"next_stage,omitempty"`
	Action       string           `json:"action"` // "none", "transition", "archive", "delete"
	Policy       *RetentionPolicy `json:"policy,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Lifecycle Evaluation
// ────────────────────────────────────────────────────────────────────────────

// EvaluateLifecycle определяет текущую стадию и следующее действие
// для данных указанного возраста.
func EvaluateLifecycle(age time.Duration, policy *RetentionPolicy) LifecycleStage {
	hotEnd := policy.HotTTL
	coldEnd := hotEnd + policy.ColdTTL
	archiveEnd := coldEnd + policy.ArchiveTTL
	deleteEnd := archiveEnd + policy.DeleteTTL

	switch {
	case age < hotEnd:
		return StageHot
	case age < coldEnd:
		return StageCold
	case age < archiveEnd:
		return StageArchive
	case deleteEnd > 0 && age < deleteEnd:
		return StageDelete
	case age < policy.TotalTTL:
		return StageDelete
	default:
		return StageDelete
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ProfileManager
// ────────────────────────────────────────────────────────────────────────────

// ProfileManager — thread-safe менеджер профилей хранения.
type ProfileManager struct {
	mu       sync.RWMutex
	profiles map[Region]map[DataType]*RetentionPolicy
	logger   *slog.Logger
}

// NewProfileManager создаёт ProfileManager с профилями по умолчанию.
func NewProfileManager(logger *slog.Logger) *ProfileManager {
	if logger == nil {
		logger = slog.Default()
	}
	pm := &ProfileManager{
		profiles: make(map[Region]map[DataType]*RetentionPolicy),
		logger:   logger.With("component", "retention-profile-manager"),
	}
	pm.registerDefaults()
	return pm
}

// registerDefaults регистрирует профили по умолчанию для всех регионов.
//
// ═════════════════════════════════════════════════════════════════════════
// Retention Profiles by Region
//
// BY (Беларусь — КИИ, ISO 27001, CTБ 34.101.27):
//   - audit:     5 лет (1800d) — требования КИИ РБ
//   - telemetry: 90d hot + 180d cold = 270d
//   - images:    30d hot + 60d cold + 275d archive = 365d (1 год)
//   - video:     7d hot + 23d cold = 30d
//
// RU (Россия — ФЗ-152, ФЗ-59):
//   - audit:     3 года (1095d) — ФЗ-152
//   - telemetry: 180d
//   - images:    365d
//
// EU (EU — GDPR Art. 5(1)(e), data minimization):
//   - audit:     365d — minimum necessary
//   - telemetry: 30d
//   - images:    90d
//
// US (USA — SOX, HIPAA, FINRA):
//   - audit:     7 лет (2555d) — SOX
//   - telemetry: 90d
//   - images:    365d
//
// CN (Китай — Cybersecurity Law, Personal Information Protection Law):
//   - audit:     365d
//   - telemetry: 30d
//   - images:    90d
//
// ═════════════════════════════════════════════════════════════════════════
func (pm *ProfileManager) registerDefaults() {
	d := 24 * time.Hour

	// ── BY ─────────────────────────────────────────────────────────────
	pm.profiles[RegionBY] = map[DataType]*RetentionPolicy{
		DataAudit: {
			Region: RegionBY, DataType: DataAudit,
			HotTTL: 30 * d, ColdTTL: 270 * d, ArchiveTTL: 1500 * d,
			TotalTTL: 1800 * d,
		},
		DataTelemetry: {
			Region: RegionBY, DataType: DataTelemetry,
			HotTTL: 7 * d, ColdTTL: 83 * d,
			TotalTTL: 90 * d,
		},
		DataAlerts: {
			Region: RegionBY, DataType: DataAlerts,
			HotTTL: 30 * d, ColdTTL: 335 * d,
			TotalTTL: 365 * d,
		},
		DataImages: {
			Region: RegionBY, DataType: DataImages,
			HotTTL: 30 * d, ColdTTL: 60 * d, ArchiveTTL: 275 * d,
			TotalTTL: 365 * d,
		},
		DataVideo: {
			Region: RegionBY, DataType: DataVideo,
			HotTTL: 7 * d, ColdTTL: 23 * d,
			TotalTTL: 30 * d,
		},
		DataReports: {
			Region: RegionBY, DataType: DataReports,
			HotTTL: 30 * d, ColdTTL: 335 * d,
			TotalTTL: 365 * d,
		},
		DataWorkOrders: {
			Region: RegionBY, DataType: DataWorkOrders,
			HotTTL: 90 * d, ColdTTL: 275 * d,
			TotalTTL: 365 * d,
		},
	}

	// ── RU ─────────────────────────────────────────────────────────────
	pm.profiles[RegionRU] = map[DataType]*RetentionPolicy{
		DataAudit: {
			Region: RegionRU, DataType: DataAudit,
			HotTTL: 30 * d, ColdTTL: 165 * d, ArchiveTTL: 900 * d,
			TotalTTL: 1095 * d,
		},
		DataTelemetry: {
			Region: RegionRU, DataType: DataTelemetry,
			HotTTL: 30 * d, ColdTTL: 150 * d,
			TotalTTL: 180 * d,
		},
		DataAlerts: {
			Region: RegionRU, DataType: DataAlerts,
			HotTTL: 30 * d, ColdTTL: 335 * d,
			TotalTTL: 365 * d,
		},
		DataImages: {
			Region: RegionRU, DataType: DataImages,
			HotTTL: 30 * d, ColdTTL: 60 * d, ArchiveTTL: 275 * d,
			TotalTTL: 365 * d,
		},
		DataVideo: {
			Region: RegionRU, DataType: DataVideo,
			HotTTL: 7 * d, ColdTTL: 23 * d,
			TotalTTL: 30 * d,
		},
	}

	// ── EU ─────────────────────────────────────────────────────────────
	pm.profiles[RegionEU] = map[DataType]*RetentionPolicy{
		DataAudit: {
			Region: RegionEU, DataType: DataAudit,
			HotTTL: 30 * d, ColdTTL: 335 * d,
			TotalTTL: 365 * d,
		},
		DataTelemetry: {
			Region: RegionEU, DataType: DataTelemetry,
			HotTTL:   30 * d,
			TotalTTL: 30 * d,
		},
		DataAlerts: {
			Region: RegionEU, DataType: DataAlerts,
			HotTTL: 30 * d, ColdTTL: 60 * d,
			TotalTTL: 90 * d,
		},
		DataImages: {
			Region: RegionEU, DataType: DataImages,
			HotTTL: 30 * d, ColdTTL: 60 * d,
			TotalTTL: 90 * d,
		},
		DataVideo: {
			Region: RegionEU, DataType: DataVideo,
			HotTTL: 7 * d, ColdTTL: 23 * d,
			TotalTTL: 30 * d,
		},
	}

	// ── US ─────────────────────────────────────────────────────────────
	pm.profiles[RegionUS] = map[DataType]*RetentionPolicy{
		DataAudit: {
			Region: RegionUS, DataType: DataAudit,
			HotTTL: 90 * d, ColdTTL: 365 * d, ArchiveTTL: 2100 * d,
			TotalTTL: 2555 * d,
		},
		DataTelemetry: {
			Region: RegionUS, DataType: DataTelemetry,
			HotTTL: 30 * d, ColdTTL: 60 * d,
			TotalTTL: 90 * d,
		},
		DataAlerts: {
			Region: RegionUS, DataType: DataAlerts,
			HotTTL: 30 * d, ColdTTL: 335 * d,
			TotalTTL: 365 * d,
		},
		DataImages: {
			Region: RegionUS, DataType: DataImages,
			HotTTL: 30 * d, ColdTTL: 60 * d, ArchiveTTL: 275 * d,
			TotalTTL: 365 * d,
		},
		DataVideo: {
			Region: RegionUS, DataType: DataVideo,
			HotTTL: 7 * d, ColdTTL: 23 * d,
			TotalTTL: 30 * d,
		},
	}

	// ── CN ─────────────────────────────────────────────────────────────
	pm.profiles[RegionCN] = map[DataType]*RetentionPolicy{
		DataAudit: {
			Region: RegionCN, DataType: DataAudit,
			HotTTL: 30 * d, ColdTTL: 335 * d,
			TotalTTL: 365 * d,
		},
		DataTelemetry: {
			Region: RegionCN, DataType: DataTelemetry,
			HotTTL:   30 * d,
			TotalTTL: 30 * d,
		},
		DataAlerts: {
			Region: RegionCN, DataType: DataAlerts,
			HotTTL: 30 * d, ColdTTL: 60 * d,
			TotalTTL: 90 * d,
		},
		DataImages: {
			Region: RegionCN, DataType: DataImages,
			HotTTL: 30 * d, ColdTTL: 60 * d,
			TotalTTL: 90 * d,
		},
		DataVideo: {
			Region: RegionCN, DataType: DataVideo,
			HotTTL: 3 * d, ColdTTL: 27 * d,
			TotalTTL: 30 * d,
		},
	}
}

// GetProfile возвращает политику хранения для региона и типа данных.
// Возвращает nil, если политика не найдена.
func (pm *ProfileManager) GetProfile(region Region, dataType DataType) (*RetentionPolicy, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	regionProfiles, ok := pm.profiles[region]
	if !ok {
		return nil, fmt.Errorf("retention: no profile for region %q", region)
	}
	policy, ok := regionProfiles[dataType]
	if !ok {
		return nil, fmt.Errorf("retention: no profile for region %q, data_type %q", region, dataType)
	}
	return policy, nil
}

// SetProfile регистрирует или обновляет политику хранения.
func (pm *ProfileManager) SetProfile(policy *RetentionPolicy) error {
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("retention: invalid policy: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	region := policy.Region
	if _, ok := pm.profiles[region]; !ok {
		pm.profiles[region] = make(map[DataType]*RetentionPolicy)
	}
	pm.profiles[region][policy.DataType] = policy

	return nil
}

// ListProfiles возвращает все зарегистрированные профили.
func (pm *ProfileManager) ListProfiles() []*RetentionPolicy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []*RetentionPolicy
	for _, regionProfiles := range pm.profiles {
		for _, policy := range regionProfiles {
			result = append(result, policy)
		}
	}
	return result
}

// ListRegions возвращает список регионов, для которых есть профили.
func (pm *ProfileManager) ListRegions() []Region {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	regions := make([]Region, 0, len(pm.profiles))
	for region := range pm.profiles {
		regions = append(regions, region)
	}
	return regions
}

// ────────────────────────────────────────────────────────────────────────────
// LegalHoldManager
// ────────────────────────────────────────────────────────────────────────────

// LegalHoldStatus — статус legal hold.
type LegalHoldStatus string

const (
	LegalHoldActive   LegalHoldStatus = "active"
	LegalHoldReleased LegalHoldStatus = "released"
	LegalHoldExpired  LegalHoldStatus = "expired"
)

// LegalHold — запись о legal hold на данные тенанта.
type LegalHold struct {
	TenantID    string          `json:"tenant_id"`
	DataType    DataType        `json:"data_type,omitempty"` // empty = all data
	Reason      string          `json:"reason"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	ReleasedAt  *time.Time      `json:"released_at,omitempty"`
	ReleasedBy  string          `json:"released_by,omitempty"`
	Status      LegalHoldStatus `json:"status"`
	ReferenceID string          `json:"reference_id,omitempty"` // Case/Litigation ID
}

// IsActive возвращает true, если hold активен и не истёк.
func (h *LegalHold) IsActive() bool {
	if h.Status != LegalHoldActive {
		return false
	}
	if h.ExpiresAt != nil && time.Now().After(*h.ExpiresAt) {
		return false
	}
	return true
}

// LegalHoldManager управляет legal hold для предотвращения удаления данных.
type LegalHoldManager struct {
	mu     sync.RWMutex
	holds  map[string][]*LegalHold // tenantID -> holds
	logger *slog.Logger
}

// NewLegalHoldManager создаёт новый LegalHoldManager.
func NewLegalHoldManager(logger *slog.Logger) *LegalHoldManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &LegalHoldManager{
		holds:  make(map[string][]*LegalHold),
		logger: logger.With("component", "legal-hold-manager"),
	}
}

// AddHold добавляет legal hold для тенанта.
func (m *LegalHoldManager) AddHold(hold *LegalHold) error {
	if hold.TenantID == "" {
		return fmt.Errorf("legal hold: tenant_id is required")
	}
	if hold.Reason == "" {
		return fmt.Errorf("legal hold: reason is required")
	}
	if hold.CreatedBy == "" {
		return fmt.Errorf("legal hold: created_by is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	hold.CreatedAt = time.Now().UTC()
	hold.Status = LegalHoldActive
	m.holds[hold.TenantID] = append(m.holds[hold.TenantID], hold)

	m.logger.Info("legal hold added",
		"tenant", hold.TenantID,
		"data_type", hold.DataType,
		"reason", hold.Reason,
		"reference", hold.ReferenceID,
	)
	return nil
}

// ReleaseHold снимает legal hold.
func (m *LegalHoldManager) ReleaseHold(tenantID string, releasedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	holds, ok := m.holds[tenantID]
	if !ok {
		return fmt.Errorf("legal hold: no holds found for tenant %q", tenantID)
	}

	released := false
	now := time.Now().UTC()
	for _, hold := range holds {
		if hold.Status == LegalHoldActive {
			hold.Status = LegalHoldReleased
			hold.ReleasedAt = &now
			hold.ReleasedBy = releasedBy
			released = true
		}
	}

	if !released {
		return fmt.Errorf("legal hold: no active holds for tenant %q", tenantID)
	}

	m.logger.Info("legal hold released",
		"tenant", tenantID,
		"released_by", releasedBy,
	)
	return nil
}

// IsHeld проверяет, есть ли активный legal hold для тенанта и типа данных.
func (m *LegalHoldManager) IsHeld(tenantID string, dataType DataType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	holds, ok := m.holds[tenantID]
	if !ok {
		return false
	}

	for _, hold := range holds {
		if !hold.IsActive() {
			continue
		}
		// Если dataType не указан (пустая строка) — hold на все данные
		if hold.DataType == "" || hold.DataType == dataType {
			return true
		}
	}
	return false
}

// GetHolds возвращает все holds для тенанта.
func (m *LegalHoldManager) GetHolds(tenantID string) []*LegalHold {
	m.mu.RLock()
	defer m.mu.RUnlock()

	holds, ok := m.holds[tenantID]
	if !ok {
		return nil
	}

	result := make([]*LegalHold, len(holds))
	copy(result, holds)
	return result
}

// ListActive возвращает все активные holds.
func (m *LegalHoldManager) ListActive() []*LegalHold {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*LegalHold
	for _, holds := range m.holds {
		for _, hold := range holds {
			if hold.IsActive() {
				result = append(result, hold)
			}
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────────────────────
// RetentionAction — запись о действии retention для audit log
// ────────────────────────────────────────────────────────────────────────────

// RetentionAction представляет действие retention lifecycle.
type RetentionAction struct {
	TenantID     string         `json:"tenant_id"`
	DataType     DataType       `json:"data_type"`
	Region       Region         `json:"region"`
	FromStage    LifecycleStage `json:"from_stage"`
	ToStage      LifecycleStage `json:"to_stage"`
	RecordsCount int64          `json:"records_count"`
	StorageBytes int64          `json:"storage_bytes,omitempty"`
	TraceID      string         `json:"trace_id"`
	Timestamp    time.Time      `json:"timestamp"`
	LegalHold    bool           `json:"legal_hold_skipped,omitempty"`
}
