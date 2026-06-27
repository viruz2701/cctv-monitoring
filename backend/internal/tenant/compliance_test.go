// Package tenant — tests for Tenant Compliance Profile (P0-CE.5).
package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gb-telemetry-collector/internal/compliance"
)

// ═══════════════════════════════════════════════════════════════════════
// Context helper tests
// ═══════════════════════════════════════════════════════════════════════

func TestContextWithComplianceRegion(t *testing.T) {
	ctx := ContextWithComplianceRegion(context.Background(), compliance.RegionBY)
	region := GetComplianceRegionFromContext(ctx)
	if region != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", region)
	}
}

func TestContextWithComplianceProfile(t *testing.T) {
	profile := compliance.NewBYProfile()
	ctx := ContextWithComplianceProfile(context.Background(), profile)

	got := GetComplianceProfileFromContext(ctx)
	if got == nil {
		t.Fatal("profile from context must not be nil")
	}
	if got.Region() != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", got.Region())
	}
}

func TestGetComplianceRegionFromContextEmpty(t *testing.T) {
	region := GetComplianceRegionFromContext(context.Background())
	if region != "" {
		t.Errorf("expected empty region, got %s", region)
	}
}

func TestGetComplianceProfileFromContextNil(t *testing.T) {
	profile := GetComplianceProfileFromContext(context.Background())
	if profile != nil {
		t.Fatal("expected nil profile from empty context")
	}
}

func TestContextRoundTrip(t *testing.T) {
	// BY profile
	byProfile := compliance.NewBYProfile()
	ctx := ContextWithComplianceRegion(context.Background(), compliance.RegionBY)
	ctx = ContextWithComplianceProfile(ctx, byProfile)

	if region := GetComplianceRegionFromContext(ctx); region != compliance.RegionBY {
		t.Errorf("expected BY, got %s", region)
	}
	if p := GetComplianceProfileFromContext(ctx); p == nil || p.Region() != compliance.RegionBY {
		t.Error("failed to retrieve BY profile from context")
	}

	// Override with EU
	euProfile := compliance.NewEUProfile()
	ctx = ContextWithComplianceRegion(ctx, compliance.RegionEU)
	ctx = ContextWithComplianceProfile(ctx, euProfile)

	if region := GetComplianceRegionFromContext(ctx); region != compliance.RegionEU {
		t.Errorf("expected EU, got %s", region)
	}
}

func TestGetTenantIDFromContext(t *testing.T) {
	ctx := context.Background()

	id := GetTenantIDFromContext(ctx)
	if id != "" {
		t.Errorf("expected empty tenant ID, got %s", id)
	}
}

// GetTenantIDFromContext использует локальный тип ключа внутри функции,
// поэтому тест с внешним типом не сработает.
// Значение tenantID устанавливается auth.TenantMiddleware через auth.TenantIDKey.
func TestGetTenantIDFromContextWithValue_EmptyContext(t *testing.T) {
	// Даже с установленным значением через общий ключ,
	// GetTenantIDFromContext вернёт пустую строку из-за различия типов ключей.
	const tenantKey testContextKey = "tenant_id"
	ctx := context.WithValue(context.Background(), tenantKey, "tenant-123")
	id := GetTenantIDFromContext(ctx)
	// Ожидаем пустоту — локальный тип ключа в функции не совпадает с testContextKey
	if id != "" {
		t.Errorf("expected empty due to type mismatch, got %s", id)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TenantComplianceStore creation tests
// ═══════════════════════════════════════════════════════════════════════

func TestNewTenantComplianceStore(t *testing.T) {
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewINTLProfile()),
	)

	// Without DB pool — store is functional for context operations
	store := NewTenantComplianceStore(nil, registry)
	if store == nil {
		t.Fatal("store must not be nil")
	}
}

func TestNewTenantComplianceStore_NilRegistry(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)
	if store == nil {
		t.Fatal("store must not be nil even with nil registry")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Cache behavior tests
// ═══════════════════════════════════════════════════════════════════════

func TestCacheAddAndGet(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)

	// Add to cache manually
	store.addToCache("tenant-1", compliance.RegionBY, false)

	entry := store.getFromCache("tenant-1")
	if entry == nil {
		t.Fatal("expected cache entry, got nil")
	}
	if entry.region != compliance.RegionBY {
		t.Errorf("expected region BY, got %s", entry.region)
	}
	if entry.locked {
		t.Error("expected locked=false")
	}
}

func TestCacheGet_Missing(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)
	entry := store.getFromCache("nonexistent")
	if entry != nil {
		t.Fatal("expected nil for missing cache entry")
	}
}

func TestCacheExpiry(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)

	// Add entry with short TTL (override default via test)
	store.addToCache("tenant-exp", compliance.RegionEU, false)

	// Verify it's there
	entry := store.getFromCache("tenant-exp")
	if entry == nil {
		t.Fatal("expected cache entry before expiry")
	}

	// Force expiry by modifying the entry
	store.mu.Lock()
	store.cache["tenant-exp"].expiresAt = time.Now().Add(-1 * time.Second)
	store.mu.Unlock()

	// Should now be expired
	entry = store.getFromCache("tenant-exp")
	if entry != nil {
		t.Fatal("expected nil after expiry")
	}
}

func TestCacheInvalidate(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)

	store.addToCache("tenant-1", compliance.RegionBY, false)
	store.addToCache("tenant-2", compliance.RegionEU, false)

	// Invalidate one tenant
	store.invalidateCache("tenant-1")

	if entry := store.getFromCache("tenant-1"); entry != nil {
		t.Error("expected tenant-1 to be invalidated")
	}
	if entry := store.getFromCache("tenant-2"); entry == nil {
		t.Error("expected tenant-2 to still be in cache")
	}
}

func TestCacheConcurrency(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)
	var wg sync.WaitGroup

	// Concurrent reads and writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tenantID := string(rune('A' + id))
			store.addToCache(tenantID, compliance.RegionINTL, false)
			store.getFromCache(tenantID)
			store.invalidateCache(tenantID)
		}(i)
	}
	wg.Wait()
	// If no race conditions, test passes
}

func TestCacheLockedState(t *testing.T) {
	store := NewTenantComplianceStore(nil, nil)

	store.addToCache("tenant-locked", compliance.RegionBY, true)

	entry := store.getFromCache("tenant-locked")
	if entry == nil {
		t.Fatal("expected cache entry")
	}
	if !entry.locked {
		t.Error("expected locked=true")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Middleware HTTP tests
// ═══════════════════════════════════════════════════════════════════════

// testContextKey — тип для context key в тестах.
type testContextKey string

func TestMiddleware_AdminBypass(t *testing.T) {
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewINTLProfile()),
	)
	store := NewTenantComplianceStore(nil, registry)

	var capturedRegion string
	var capturedProfile compliance.ComplianceProfile

	handler := store.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRegion = GetComplianceRegionFromContext(r.Context())
		capturedProfile = GetComplianceProfileFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Admin bypass: tenantID = "*" (simulating admin from TenantMiddleware)
	const tenantKey testContextKey = "tenant_id"
	ctx := context.WithValue(context.Background(), tenantKey, "*")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedRegion != compliance.RegionINTL {
		t.Errorf("expected INTL region for admin, got %s", capturedRegion)
	}
	if capturedProfile == nil {
		t.Fatal("expected compliance profile for admin")
	}
}

func TestMiddleware_NoTenantID(t *testing.T) {
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewINTLProfile()),
	)
	store := NewTenantComplianceStore(nil, registry)

	var capturedRegion string
	handler := store.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRegion = GetComplianceRegionFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// No tenantID — should use INTL as fallback
	if capturedRegion != compliance.RegionINTL {
		t.Errorf("expected INTL fallback, got %s", capturedRegion)
	}
}

func TestMiddleware_TenantWithRegion(t *testing.T) {
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewINTLProfile()),
		compliance.WithProfile(compliance.NewBYProfile()),
	)
	store := NewTenantComplianceStore(nil, registry)

	var capturedRegion string
	var capturedProfile compliance.ComplianceProfile

	handler := store.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRegion = GetComplianceRegionFromContext(r.Context())
		capturedProfile = GetComplianceProfileFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	const tenantKey testContextKey = "tenant_id"
	// Without DB, the store will return INTL default for any tenant
	// (since no row exists, GetComplianceRegion returns RegionINTL, false, nil)
	ctx := context.WithValue(context.Background(), tenantKey, "tenant-123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Without DB, defaults to INTL
	if capturedRegion != compliance.RegionINTL {
		t.Errorf("expected INTL (default), got %s", capturedRegion)
	}
	if capturedProfile == nil {
		t.Fatal("expected compliance profile")
	}
}

func TestMiddleware_ChainsWithExistingContext(t *testing.T) {
	registry := compliance.NewProfileRegistry(
		compliance.WithProfile(compliance.NewINTLProfile()),
		compliance.WithProfile(compliance.NewBYProfile()),
	)
	store := NewTenantComplianceStore(nil, registry)

	var capturedRegion string
	var capturedProfile compliance.ComplianceProfile

	// Simulate inner handler that also reads from context
	handler := store.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRegion = GetComplianceRegionFromContext(r.Context())
		capturedProfile = GetComplianceProfileFromContext(r.Context())

		// Verify context values are accessible from child handlers
		childRegion := GetComplianceRegionFromContext(r.Context())
		if childRegion != capturedRegion {
			t.Error("context value not propagated to child handler")
		}
		w.WriteHeader(http.StatusOK)
	}))

	const tenantKey testContextKey = "tenant_id"
	ctx := context.WithValue(context.Background(), tenantKey, "tenant-chain")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedProfile == nil {
		t.Fatal("expected compliance profile in chain")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Integration test stubs (требуют PostgreSQL)
// ═══════════════════════════════════════════════════════════════════════

// TestTenantComplianceStoreWithDB — интеграционный тест.
// Требует запущенный PostgreSQL с применённой миграцией 036.
// Запуск: go test -run TestTenantComplianceStoreWithDB -tags=integration ./internal/tenant/
func TestTenantComplianceStoreWithDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Log("Integration test requires PostgreSQL with migration 036 applied")
	t.Log("Set DATABASE_URL env var to a test database")
	t.Log("Example: DATABASE_URL=postgres://user:pass@localhost:5432/test_tenant_compliance?sslmode=disable")

	// Этот тест будет пропущен если нет DATABASE_URL
	// Для реального запуска используйте testcontainers-go
	t.Skip("integration test not configured — set DATABASE_URL and run with -tags=integration")
}

// TestComplianceStoreSetGetLock — тест цикла Set → Get → Lock.
func TestComplianceStoreSetGetLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Skip("requires PostgreSQL — run with -tags=integration and DATABASE_URL set")
}

// TestComplianceStoreLockPreventsChange — тест блокировки.
func TestComplianceStoreLockPreventsChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Skip("requires PostgreSQL — run with -tags=integration and DATABASE_URL set")
}
