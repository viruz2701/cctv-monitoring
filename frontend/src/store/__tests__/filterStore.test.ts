// ═══════════════════════════════════════════════════════════════════════
// filterStore — Unit Tests
// P1-UX.9: Saved Filters — export/import, URL helpers, default views
// ═══════════════════════════════════════════════════════════════════════

import { describe, it, expect, beforeEach } from 'vitest';
import { useFilterStore, encodeFilterState, decodeFilterState, ROLE_DEFAULT_FILTERS } from '../filterStore';

// ═══════════════════════════════════════════════════════════════════════
// Reset store before each test
// ═══════════════════════════════════════════════════════════════════════

beforeEach(() => {
  useFilterStore.setState({ savedViews: [] });
});

// ═══════════════════════════════════════════════════════════════════════
// CRUD Tests
// ═══════════════════════════════════════════════════════════════════════

describe('filterStore — CRUD', () => {
  it('should save a view and retrieve it by page', () => {
    const store = useFilterStore.getState();

    store.saveView(
      'Critical Overdue',
      'work-orders',
      { status: 'overdue', priority: 'critical' },
      { column: 'due_date', direction: 'asc' },
    );

    const views = useFilterStore.getState().getViewsForPage('work-orders');
    expect(views).toHaveLength(1);
    expect(views[0].name).toBe('Critical Overdue');
    expect(views[0].filters).toEqual({ status: 'overdue', priority: 'critical' });
    expect(views[0].sort).toEqual({ column: 'due_date', direction: 'asc' });
  });

  it('should delete a view by id', () => {
    const store = useFilterStore.getState();

    store.saveView('Test View', 'devices', { status: 'active' }, { column: 'name', direction: 'asc' });
    const viewId = useFilterStore.getState().savedViews[0].id;

    store.deleteView(viewId);
    expect(useFilterStore.getState().savedViews).toHaveLength(0);
  });

  it('should rename a view', () => {
    const store = useFilterStore.getState();

    store.saveView('Old Name', 'alerts', {}, { column: 'id', direction: 'desc' });
    const viewId = useFilterStore.getState().savedViews[0].id;

    store.renameView(viewId, 'New Name');
    const renamed = useFilterStore.getState().loadView(viewId);
    expect(renamed?.name).toBe('New Name');
  });

  it('should load a view by id', () => {
    const store = useFilterStore.getState();

    store.saveView('My View', 'sites', { region: 'east' }, { column: 'name', direction: 'asc' });
    const viewId = useFilterStore.getState().savedViews[0].id;

    const loaded = store.loadView(viewId);
    expect(loaded).toBeDefined();
    expect(loaded?.name).toBe('My View');
  });

  it('should return undefined for non-existent id', () => {
    const loaded = useFilterStore.getState().loadView('non-existent');
    expect(loaded).toBeUndefined();
  });

  it('should filter views by page', () => {
    const store = useFilterStore.getState();

    store.saveView('View A', 'devices', {}, { column: 'id', direction: 'asc' });
    store.saveView('View B', 'work-orders', {}, { column: 'id', direction: 'asc' });
    store.saveView('View C', 'devices', {}, { column: 'id', direction: 'asc' });

    const deviceViews = useFilterStore.getState().getViewsForPage('devices');
    expect(deviceViews).toHaveLength(2);

    const woViews = useFilterStore.getState().getViewsForPage('work-orders');
    expect(woViews).toHaveLength(1);
  });
});

// ═══════════════════════════════════════════════════════════════════════
// Default Views Per Role
// ═══════════════════════════════════════════════════════════════════════

describe('filterStore — Default Views Per Role', () => {
  it('should set and get default view for a role', () => {
    const store = useFilterStore.getState();

    store.saveView('Technician Default', 'devices', { status: 'active', assigned_to: 'me' }, { column: 'id', direction: 'asc' });
    const viewId = useFilterStore.getState().savedViews[0].id;

    store.setDefaultForRole(viewId, 'devices', 'technician');
    const defaultView = store.getDefaultViewForRole('devices', 'technician');

    expect(defaultView).toBeDefined();
    expect(defaultView?.defaultForRole).toBe('technician');
    expect(defaultView?.name).toBe('Technician Default');
  });

  it('should clear default for a role', () => {
    const store = useFilterStore.getState();

    store.saveView('Manager Default', 'work-orders', { status: 'open' }, { column: 'id', direction: 'asc' });
    const viewId = useFilterStore.getState().savedViews[0].id;

    store.setDefaultForRole(viewId, 'work-orders', 'manager');
    store.clearDefaultForRole('work-orders', 'manager');

    expect(store.getDefaultViewForRole('work-orders', 'manager')).toBeUndefined();
  });

  it('should only allow one default per role per page', () => {
    const store = useFilterStore.getState();

    store.saveView('First Default', 'devices', { status: 'active' }, { column: 'id', direction: 'asc' });
    const firstId = useFilterStore.getState().savedViews[0].id;
    store.setDefaultForRole(firstId, 'devices', 'technician');

    store.saveView('Second Default', 'devices', { status: 'inactive' }, { column: 'id', direction: 'asc' });
    const secondId = useFilterStore.getState().savedViews[1].id;
    store.setDefaultForRole(secondId, 'devices', 'technician');

    const defaultView = store.getDefaultViewForRole('devices', 'technician');
    expect(defaultView?.id).toBe(secondId);

    const firstView = store.loadView(firstId);
    expect(firstView?.defaultForRole).toBeUndefined();
  });
});

// ═══════════════════════════════════════════════════════════════════════
// Export / Import
// ═══════════════════════════════════════════════════════════════════════

describe('filterStore — Export/Import', () => {
  it('should export all views as JSON string', () => {
    const store = useFilterStore.getState();

    store.saveView('View 1', 'devices', { status: 'active' }, { column: 'id', direction: 'asc' });
    store.saveView('View 2', 'work-orders', { status: 'open' }, { column: 'id', direction: 'desc' });

    const exported = store.exportViews();
    const parsed = JSON.parse(exported);

    expect(parsed.version).toBe(1);
    expect(parsed.exportedAt).toBeDefined();
    expect(parsed.views).toHaveLength(2);
  });

  it('should export views filtered by page', () => {
    const store = useFilterStore.getState();

    store.saveView('Devices View', 'devices', {}, { column: 'id', direction: 'asc' });
    store.saveView('Work Orders View', 'work-orders', {}, { column: 'id', direction: 'asc' });

    const exported = store.exportViews('devices');
    const parsed = JSON.parse(exported);

    expect(parsed.views).toHaveLength(1);
    expect(parsed.views[0].page).toBe('devices');
  });

  it('should import valid views', () => {
    const store = useFilterStore.getState();

    const importJson = JSON.stringify({
      version: 1,
      exportedAt: new Date().toISOString(),
      views: [
        {
          id: 'imported-1',
          name: 'Imported View',
          page: 'devices',
          filters: { status: 'active' },
          sort: { column: 'name', direction: 'asc' },
        },
      ],
    });

    const result = store.importViews(importJson);
    expect(result.success).toBe(true);
    expect(result.count).toBe(1);

    const views = useFilterStore.getState().savedViews;
    expect(views).toHaveLength(1);
    expect(views[0].name).toBe('Imported View');
  });

  it('should reject invalid import format', () => {
    const result = useFilterStore.getState().importViews('{"invalid": true}');
    expect(result.success).toBe(false);
    expect(result.count).toBe(0);
    expect(result.errors).toHaveLength(1);
  });

  it('should reject malformed JSON', () => {
    const result = useFilterStore.getState().importViews('not-json-at-all');
    expect(result.success).toBe(false);
    expect(result.count).toBe(0);
  });
});

// ═══════════════════════════════════════════════════════════════════════
// URL Encode / Decode
// ═══════════════════════════════════════════════════════════════════════

describe('filterStore — URL Encoders', () => {
  it('should encode and decode filter state', () => {
    const filters = { status: 'active', priority: 'high' };
    const sort = { column: 'name', direction: 'asc' as const };

    const encoded = encodeFilterState(filters, sort);
    expect(encoded).toBeTruthy();
    expect(typeof encoded).toBe('string');

    const decoded = decodeFilterState(encoded);
    expect(decoded).not.toBeNull();
    expect(decoded?.filters).toEqual(filters);
    expect(decoded?.sort).toEqual(sort);
  });

  it('should decode with URL-safe base64', () => {
    const filters = { status: 'active' };
    const sort = { column: 'id', direction: 'desc' as const };

    const encoded = encodeFilterState(filters, sort);
    // Should not contain URL-unsafe characters
    expect(encoded).not.toContain('+');
    expect(encoded).not.toContain('/');
    expect(encoded).not.toMatch(/=+$/);

    const decoded = decodeFilterState(encoded);
    expect(decoded?.filters.status).toBe('active');
  });

  it('should return null for invalid encoded string', () => {
    const decoded = decodeFilterState('this-is-not-valid-base64url');
    expect(decoded).toBeNull();
  });
});

// ═══════════════════════════════════════════════════════════════════════
// ROLE_DEFAULT_FILTERS
// ═══════════════════════════════════════════════════════════════════════

describe('ROLE_DEFAULT_FILTERS', () => {
  it('should have role-specific filters for devices', () => {
    expect(ROLE_DEFAULT_FILTERS.devices).toBeDefined();
    expect(ROLE_DEFAULT_FILTERS.devices.technician).toEqual({ status: 'active', assigned_to: 'me' });
    expect(ROLE_DEFAULT_FILTERS.devices.viewer).toEqual({ status: 'active' });
  });

  it('should have role-specific filters for work-orders', () => {
    expect(ROLE_DEFAULT_FILTERS['work-orders']).toBeDefined();
    expect(ROLE_DEFAULT_FILTERS['work-orders'].manager).toEqual({ status: 'open,in_progress', priority: 'high,critical' });
  });

  it('should have empty filters for admin roles', () => {
    expect(ROLE_DEFAULT_FILTERS.devices.admin).toEqual({});
    expect(ROLE_DEFAULT_FILTERS['work-orders'].admin).toEqual({});
    expect(ROLE_DEFAULT_FILTERS.alerts.admin).toEqual({});
  });
});
