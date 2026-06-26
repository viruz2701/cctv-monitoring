// Package tenant — Tenant Compliance Profile (P0-CE.5).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.5: Tenant Compliance Profile (SaaS)
//
// Обеспечивает per-tenant compliance region:
//   - compliance_region в tenant_regions (VARCHAR(10), NOT NULL, DEFAULT 'INTL')
//   - compliance_locked (BOOLEAN, DEFAULT false) — immutable после first data
//   - Injected в request context через TenantMiddleware
//   - RLS policies включают compliance_region
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - ISO 27001 A.8.1 (Asset management — tenant classification)
//   - GDPR Art. 44-49 (Data transfer — region pinning)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
//
// ═══════════════════════════════════════════════════════════════════════════
package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gb-telemetry-collector/internal/compliance"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

// contextKey — тип для context key.
type contextKey string

const (
	// ContextKeyComplianceRegion — ключ для compliance_region в context.
	ContextKeyComplianceRegion contextKey = "compliance_region"
	// ContextKeyComplianceProfile — ключ для ComplianceProfile в context.
	ContextKeyComplianceProfile contextKey = "compliance_profile"
)

// ────────────────────────────────────────────────────────────────────────────
// TenantCompliance
// ────────────────────────────────────────────────────────────────────────────

// TenantComplianceStore управляет compliance регионом tenant'ов.
type TenantComplianceStore struct {
	pool     *pgxpool.Pool
	registry *compliance.ProfileRegistry
	cache    map[string]*cachedEntry
	mu       sync.RWMutex
	logger   *slog.Logger
}

type cachedEntry struct {
	region    string
	locked    bool
	expiresAt time.Time
}

const cacheTTL = 5 * time.Minute

// NewTenantComplianceStore создаёт новый TenantComplianceStore.
func NewTenantComplianceStore(pool *pgxpool.Pool, registry *compliance.ProfileRegistry) *TenantComplianceStore {
	return &TenantComplianceStore{
		pool:     pool,
		registry: registry,
		cache:    make(map[string]*cachedEntry),
		logger:   slog.Default().With("component", "tenant.compliance"),
	}
}

// GetComplianceRegion возвращает compliance_region для tenant'а.
func (s *TenantComplianceStore) GetComplianceRegion(ctx context.Context, tenantID string) (string, bool, error) {
	// Check cache
	if entry := s.getFromCache(tenantID); entry != nil {
		return entry.region, entry.locked, nil
	}

	// Query DB
	var region string
	var locked bool
	err := s.pool.QueryRow(ctx, `
		SELECT compliance_region, compliance_locked
		FROM tenant_regions
		WHERE tenant_id = $1
	`, tenantID).Scan(&region, &locked)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Tenant not found — use default
			return compliance.RegionINTL, false, nil
		}
		return "", false, fmt.Errorf("get tenant compliance region: %w", err)
	}

	// Cache
	s.addToCache(tenantID, region, locked)

	return region, locked, nil
}

// SetComplianceRegion устанавливает compliance_region для tenant'а.
// Возвращает ошибку если регион уже заблокирован.
func (s *TenantComplianceStore) SetComplianceRegion(ctx context.Context, tenantID, region string) error {
	// Validate region
	if !s.registry.IsRegistered(region) {
		return fmt.Errorf("%w: %s", compliance.ErrProfileNotFound, region)
	}

	// Check if locked
	_, locked, err := s.GetComplianceRegion(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("check region locked: %w", err)
	}
	if locked {
		return fmt.Errorf("tenant %s: compliance region is locked (immutable)", tenantID)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO tenant_regions (tenant_id, compliance_region, compliance_locked, primary_region, status, pinned_at)
		VALUES ($1, $2, false, $2, 'active', NOW())
		ON CONFLICT (tenant_id) DO UPDATE SET
			compliance_region = EXCLUDED.compliance_region,
			compliance_locked = false,
			updated_at = NOW()
	`, tenantID, region)
	if err != nil {
		return fmt.Errorf("set tenant compliance region: %w", err)
	}

	s.invalidateCache(tenantID)
	s.logger.Info("tenant compliance region set",
		"tenant_id", tenantID,
		"region", region,
	)
	return nil
}

// LockComplianceRegion блокирует compliance регион (после first data).
func (s *TenantComplianceStore) LockComplianceRegion(ctx context.Context, tenantID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenant_regions
		SET compliance_locked = true, updated_at = NOW()
		WHERE tenant_id = $1 AND compliance_locked = false
	`, tenantID)
	if err != nil {
		return fmt.Errorf("lock compliance region: %w", err)
	}

	s.invalidateCache(tenantID)
	return nil
}

// IsRegionLocked проверяет, заблокирован ли регион.
func (s *TenantComplianceStore) IsRegionLocked(ctx context.Context, tenantID string) (bool, error) {
	_, locked, err := s.GetComplianceRegion(ctx, tenantID)
	return locked, err
}

// ────────────────────────────────────────────────────────────────────────────
// Context helpers
// ────────────────────────────────────────────────────────────────────────────

// ContextWithComplianceRegion добавляет compliance_region в context.
func ContextWithComplianceRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, ContextKeyComplianceRegion, region)
}

// GetComplianceRegionFromContext извлекает compliance_region из context.
func GetComplianceRegionFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyComplianceRegion).(string); ok {
		return v
	}
	return ""
}

// ContextWithComplianceProfile добавляет ComplianceProfile в context.
func ContextWithComplianceProfile(ctx context.Context, profile compliance.ComplianceProfile) context.Context {
	return context.WithValue(ctx, ContextKeyComplianceProfile, profile)
}

// GetComplianceProfileFromContext извлекает ComplianceProfile из context.
func GetComplianceProfileFromContext(ctx context.Context) compliance.ComplianceProfile {
	if v, ok := ctx.Value(ContextKeyComplianceProfile).(compliance.ComplianceProfile); ok {
		return v
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Middleware
// ────────────────────────────────────────────────────────────────────────────

// Middleware создаёт HTTP middleware, который инжектит compliance профиль
// в контекст запроса на основе tenantID из JWT.
//
// Используется после TenantMiddleware (который устанавливает tenantID в контекст).
func (s *TenantComplianceStore) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем tenantID из контекста (установлен TenantMiddleware)
		tenantID := GetTenantIDFromContext(r.Context())
		if tenantID == "" || tenantID == "*" {
			// Admin bypass или нет tenant — используем INTL
			profile, err := s.registry.Get(compliance.RegionINTL)
			if err != nil {
				s.logger.Error("failed to get default compliance profile",
					"error", err,
				)
				next.ServeHTTP(w, r)
				return
			}
			ctx := ContextWithComplianceRegion(r.Context(), compliance.RegionINTL)
			ctx = ContextWithComplianceProfile(ctx, profile)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Получаем compliance_region для tenant'а
		region, _, err := s.GetComplianceRegion(r.Context(), tenantID)
		if err != nil {
			s.logger.Error("failed to get tenant compliance region",
				"tenant_id", tenantID,
				"error", err,
			)
			// Fallback на INTL
			region = compliance.RegionINTL
		}

		// Получаем ComplianceProfile для региона
		profile, err := s.registry.Get(region)
		if err != nil {
			s.logger.Error("failed to get compliance profile for region",
				"region", region,
				"error", err,
			)
			next.ServeHTTP(w, r)
			return
		}

		// Инжектим в контекст
		ctx := ContextWithComplianceRegion(r.Context(), region)
		ctx = ContextWithComplianceProfile(ctx, profile)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Cache helpers
// ────────────────────────────────────────────────────────────────────────────

func (s *TenantComplianceStore) getFromCache(tenantID string) *cachedEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cache[tenantID]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry
}

func (s *TenantComplianceStore) addToCache(tenantID, region string, locked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[tenantID] = &cachedEntry{
		region:    region,
		locked:    locked,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

func (s *TenantComplianceStore) invalidateCache(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, tenantID)
}

// ────────────────────────────────────────────────────────────────────────────
// TenantID from context helper
// ────────────────────────────────────────────────────────────────────────────

// GetTenantIDFromContext извлекает tenantID из контекста.
// Использует ключ из auth пакета.
func GetTenantIDFromContext(ctx context.Context) string {
	type tenantContextKey string
	const key tenantContextKey = "tenant_id"
	if v, ok := ctx.Value(key).(string); ok {
		return v
	}
	return ""
}
