package cmms

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// Tests: AdapterRegistry
// ═══════════════════════════════════════════════════════════════════════

func TestAdapterRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)

	// Register
	internalAdapter := &mockAdapter{}
	atlasAdapter := &mockAdapter{}

	if err := reg.Register("internal", internalAdapter); err != nil {
		t.Fatalf("failed to register internal adapter: %v", err)
	}
	if err := reg.Register("atlas", atlasAdapter); err != nil {
		t.Fatalf("failed to register atlas adapter: %v", err)
	}

	// Get existing
	adapter, err := reg.Get("internal")
	if err != nil {
		t.Fatalf("failed to get internal adapter: %v", err)
	}
	if adapter != internalAdapter {
		t.Error("expected internal adapter to match")
	}

	// Get non-existing
	_, err = reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent adapter")
	}

	// Validate existing
	if err := reg.Validate("atlas"); err != nil {
		t.Errorf("expected atlas to be valid: %v", err)
	}

	// Validate non-existing
	if err := reg.Validate("nonexistent"); err == nil {
		t.Error("expected validation error for nonexistent adapter")
	}

	// List
	names := reg.List()
	if len(names) != 2 {
		t.Errorf("expected 2 adapters, got %d", len(names))
	}

	// Remove
	reg.Remove("atlas")
	names = reg.List()
	if len(names) != 1 {
		t.Errorf("expected 1 adapter after remove, got %d", len(names))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Tests: TenantRouter
// ═══════════════════════════════════════════════════════════════════════

func TestTenantRouterDefault(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)

	internalAdapter := &mockAdapter{}
	_ = reg.Register("internal", internalAdapter)

	router := NewTenantRouter(reg, nil, TenantRouterConfig{
		DefaultAdapter: "internal",
	}, logger)

	// Get adapter for tenant without override
	adapter, err := router.GetAdapter(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("failed to get adapter: %v", err)
	}
	if adapter != internalAdapter {
		t.Error("expected default internal adapter")
	}
}

func TestTenantRouterOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)

	internalAdapter := &mockAdapter{}
	atlasAdapter := &mockAdapter{}
	_ = reg.Register("internal", internalAdapter)
	_ = reg.Register("atlas", atlasAdapter)

	router := NewTenantRouter(reg, nil, TenantRouterConfig{
		DefaultAdapter: "internal",
		PerTenantOverrides: map[string]string{
			"tenant-premium": "atlas",
		},
	}, logger)

	// Get adapter for premium tenant
	adapter, err := router.GetAdapter(context.Background(), "tenant-premium")
	if err != nil {
		t.Fatalf("failed to get adapter: %v", err)
	}
	if adapter != atlasAdapter {
		t.Error("expected atlas adapter for premium tenant")
	}

	// Get adapter for regular tenant (no override)
	adapter, err = router.GetAdapter(context.Background(), "tenant-regular")
	if err != nil {
		t.Fatalf("failed to get adapter: %v", err)
	}
	if adapter != internalAdapter {
		t.Error("expected internal adapter for regular tenant")
	}
}

func TestTenantRouterResolver(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)

	internalAdapter := &mockAdapter{}
	servicenowAdapter := &mockAdapter{}
	_ = reg.Register("internal", internalAdapter)
	_ = reg.Register("servicenow", servicenowAdapter)

	resolver := TenantAdapterResolverFunc(func(_ context.Context, tenantID string) (string, error) {
		if tenantID == "enterprise-1" {
			return "servicenow", nil
		}
		return "internal", nil
	})

	router := NewTenantRouter(reg, resolver, TenantRouterConfig{
		DefaultAdapter: "internal",
	}, logger)

	// Enterprise tenant → ServiceNow
	adapter, err := router.GetAdapter(context.Background(), "enterprise-1")
	if err != nil {
		t.Fatalf("failed to get adapter: %v", err)
	}
	if adapter != servicenowAdapter {
		t.Error("expected servicenow adapter for enterprise tenant")
	}

	// Small tenant → internal
	adapter, err = router.GetAdapter(context.Background(), "small-tenant")
	if err != nil {
		t.Fatalf("failed to get adapter: %v", err)
	}
	if adapter != internalAdapter {
		t.Error("expected internal adapter for small tenant")
	}
}

func TestTenantRouterSetOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)

	internalAdapter := &mockAdapter{}
	atlasAdapter := &mockAdapter{}
	_ = reg.Register("internal", internalAdapter)
	_ = reg.Register("atlas", atlasAdapter)

	router := NewTenantRouter(reg, nil, TenantRouterConfig{
		DefaultAdapter: "internal",
	}, logger)

	// Set override dynamically
	if err := router.SetOverride("tenant-3", "atlas"); err != nil {
		t.Fatalf("failed to set override: %v", err)
	}

	// Verify override
	adapter, err := router.GetAdapter(context.Background(), "tenant-3")
	if err != nil {
		t.Fatalf("failed to get adapter: %v", err)
	}
	if adapter != atlasAdapter {
		t.Error("expected atlas adapter after override")
	}

	// Verify adapter name
	name := router.AdapterName(context.Background(), "tenant-3")
	if name != "atlas" {
		t.Errorf("expected adapter name 'atlas', got %q", name)
	}

	// Remove override
	router.RemoveOverride("tenant-3")
	adapter, err = router.GetAdapter(context.Background(), "tenant-3")
	if err != nil {
		t.Fatalf("failed to get adapter after remove: %v", err)
	}
	if adapter != internalAdapter {
		t.Error("expected internal adapter after override removed")
	}

	// List overrides
	overrides := router.ListOverrides()
	if len(overrides) != 0 {
		t.Errorf("expected 0 overrides, got %d", len(overrides))
	}
}

func TestTenantRouterSetInvalidOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)
	_ = reg.Register("internal", &mockAdapter{})

	router := NewTenantRouter(reg, nil, TenantRouterConfig{
		DefaultAdapter: "internal",
	}, logger)

	// Set override to nonexistent adapter
	err := router.SetOverride("tenant-4", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent adapter override")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Tests: Context helpers
// ═══════════════════════════════════════════════════════════════════════

func TestTenantContext(t *testing.T) {
	ctx := context.Background()

	// Empty context
	if id := TenantIDFromContext(ctx); id != "" {
		t.Errorf("expected empty tenant ID, got %q", id)
	}

	// Context with tenant ID
	ctx = ContextWithTenantID(ctx, "tenant-42")
	if id := TenantIDFromContext(ctx); id != "tenant-42" {
		t.Errorf("expected 'tenant-42', got %q", id)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Tests: TenantRouterWrapper
// ═══════════════════════════════════════════════════════════════════════

func TestTenantRouterWrapper(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := NewAdapterRegistry(logger)

	internalAdapter := &mockAdapter{}
	_ = reg.Register("internal", internalAdapter)

	router := NewTenantRouter(reg, nil, TenantRouterConfig{
		DefaultAdapter: "internal",
	}, logger)

	wrapper := NewTenantRouterWrapper(router, logger)

	// Test with tenant context
	ctx := ContextWithTenantID(context.Background(), "tenant-1")
	err := wrapper.CreateWorkOrder(ctx, nil)
	if err != nil {
		t.Fatalf("CreateWorkOrder failed: %v", err)
	}

	// Test without tenant context (should use default)
	ctx2 := context.Background()
	err = wrapper.CreateWorkOrder(ctx2, nil)
	if err != nil {
		t.Fatalf("CreateWorkOrder without tenant failed: %v", err)
	}

	// Test with override
	_ = router.SetOverride("tenant-premium", "internal")
	ctx3 := ContextWithTenantID(context.Background(), "tenant-premium")
	err = wrapper.CreateWorkOrder(ctx3, nil)
	if err != nil {
		t.Fatalf("CreateWorkOrder with override failed: %v", err)
	}
}
