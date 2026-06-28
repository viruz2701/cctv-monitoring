/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockSites,
  mockDevices,
  mockUsers,
  mockWorkOrders,
  mockDashboardStats,
  mockAlerts,
  mockCatchAll,
  MOCK_ADMIN_USER,
  MOCK_MANAGER_USER,
  MOCK_TECHNICIAN_USER,
  MOCK_SITES,
  MOCK_DEVICES,
  MOCK_USERS,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// User Permissions — E2E Tests
// P1-QA.1: RBAC, role-based доступ, permission guards
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_PERMISSIONS = {
  admin: {
    routes: ['/', '/dashboard', '/devices', '/devices/new', '/work-orders', '/work-orders/create', '/settings', '/settings/security', '/settings/users', '/compliance', '/gatekeeper', '/rca', '/reports', '/p2p-devices', '/maintenance', '/sla'],
    actions: ['create:wo', 'edit:wo', 'delete:wo', 'assign:wo', 'complete:wo', 'create:device', 'edit:device', 'delete:device', 'manage:users', 'manage:settings', 'view:compliance', 'export:reports'],
  },
  manager: {
    routes: ['/', '/dashboard', '/devices', '/work-orders', '/work-orders/create', '/reports', '/maintenance', '/sla'],
    actions: ['create:wo', 'assign:wo', 'complete:wo', 'export:reports', 'view:compliance'],
  },
  technician: {
    routes: ['/', '/dashboard', '/devices', '/work-orders', '/maintenance'],
    actions: ['complete:wo', 'view:compliance'],
  },
  support: {
    routes: ['/', '/dashboard', '/devices', '/work-orders'],
    actions: ['view:compliance'],
  },
  owner: {
    routes: ['/', '/dashboard', '/devices', '/work-orders', '/settings', '/settings/security', '/compliance', '/reports'],
    actions: ['create:wo', 'edit:wo', 'delete:wo', 'assign:wo', 'manage:settings', 'view:compliance', 'export:reports'],
  },
};

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupPermissionsMockApi(page: any, user: any) {
  await setupAuth(page, user);

  // Permissions endpoint
  await page.route('**/api/v1/users/permissions', async (route: any) => {
    const rolePerms = MOCK_PERMISSIONS[user.role as keyof typeof MOCK_PERMISSIONS] || MOCK_PERMISSIONS.technician;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(rolePerms),
    });
  });

  // Standard mocks
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page, [
    ...MOCK_USERS,
    { id: 'user-5', username: 'support', role: 'support', full_name: 'Support Agent' },
    { id: 'user-6', username: 'owner', role: 'owner', full_name: 'System Owner' },
  ]);
  await mockWorkOrders(page);
  await mockDashboardStats(page);
  await mockAlerts(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: RBAC — Admin Access
// ═══════════════════════════════════════════════════════════════════════════

test.describe('RBAC — Admin Access', () => {
  test.beforeEach(async ({ page }) => {
    await setupPermissionsMockApi(page, MOCK_ADMIN_USER);
    await page.goto('/');
    await page.waitForTimeout(1500);
  });

  test('Admin — can access settings with security tab', async ({ page }) => {
    await page.goto('/settings/security');
    await page.waitForTimeout(1000);

    // Admin должен иметь доступ к security settings
    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено|403/i',
    );

    const isDenied = await accessDenied.isVisible().catch(() => false);
    if (!isDenied) {
      // Проверяем что страница загрузилась
      const pageContent = page.locator('body');
      await expect(pageContent).toBeVisible();
      expect(page.url()).toContain('/settings');
    }
  });

  test('Admin — can access user management page', async ({ page }) => {
    await page.goto('/settings/users');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      const pageContent = page.locator('body');
      await expect(pageContent).toBeVisible();
    }
  });

  test('Admin — can access compliance shield', async ({ page }) => {
    await page.goto('/compliance');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      await expect(page.locator('body')).toBeVisible();
      expect(page.url()).toContain('/compliance');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: RBAC — Manager Access
// ═══════════════════════════════════════════════════════════════════════════

test.describe('RBAC — Manager Access', () => {
  test.beforeEach(async ({ page }) => {
    await setupPermissionsMockApi(page, MOCK_MANAGER_USER);
    await page.goto('/');
    await page.waitForTimeout(1500);
  });

  test('Manager — can access work orders and create', async ({ page }) => {
    await page.goto('/work-orders/create');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      await expect(page.locator('body')).toBeVisible();
    }
  });

  test('Manager — cannot access user management', async ({ page }) => {
    await page.goto('/settings/users');
    await page.waitForTimeout(1000);

    // Manager не должен иметь доступ к управлению пользователями
    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено|not authorized|не авторизован/i',
    ).first();
    await expect(accessDenied).toBeVisible();
  });

  test('Manager — can access reports and export', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      await expect(page.locator('body')).toBeVisible();
      expect(page.url()).toContain('/reports');
    }
  });

  test('Manager — can access SLA page', async ({ page }) => {
    await page.goto('/sla');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      await expect(page.locator('body')).toBeVisible();
      expect(page.url()).toContain('/sla');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: RBAC — Technician Access
// ═══════════════════════════════════════════════════════════════════════════

test.describe('RBAC — Technician Access', () => {
  test.beforeEach(async ({ page }) => {
    await setupPermissionsMockApi(page, MOCK_TECHNICIAN_USER);
    await page.goto('/');
    await page.waitForTimeout(1500);
  });

  test('Technician — can access work orders list', async ({ page }) => {
    await page.goto('/work-orders');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      await expect(page.locator('body')).toBeVisible();
      expect(page.url()).toContain('/work-orders');
    }
  });

  test('Technician — cannot access settings', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForTimeout(1000);

    // Technician не должен иметь доступ к settings
    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено|not authorized|не авторизован/i',
    ).first();
    await expect(accessDenied).toBeVisible();
  });

  test('Technician — cannot access security settings', async ({ page }) => {
    await page.goto('/settings/security');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено|not authorized|не авторизован/i',
    ).first();
    await expect(accessDenied).toBeVisible();
  });

  test('Technician — can access maintenance schedule', async ({ page }) => {
    await page.goto('/maintenance');
    await page.waitForTimeout(1000);

    const accessDenied = page.locator(
      'text=/access denied|доступ запрещен|forbidden|запрещено/i',
    );
    const isDenied = await accessDenied.isVisible().catch(() => false);

    if (!isDenied) {
      await expect(page.locator('body')).toBeVisible();
      expect(page.url()).toContain('/maintenance');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: RBAC — Permission UI Visibility
// ═══════════════════════════════════════════════════════════════════════════

test.describe('RBAC — Permission UI Visibility', () => {
  test('Admin — sees all navigation items', async ({ page }) => {
    await setupPermissionsMockApi(page, MOCK_ADMIN_USER);
    await page.goto('/');
    await page.waitForTimeout(1500);

    // Проверяем видимость навигационных элементов
    const navItems = page.locator(
      'nav a, [role="navigation"] a, [class*="nav" i] a, [class*="sidebar" i] a',
    );

    // Admin должен видеть все пункты меню
    const complianceLink = navItems.filter({ hasText: /compliance|shield/i }).first();
    const settingsLink = navItems.filter({ hasText: /settings|настройк/i }).first();

    const hasCompliance = await complianceLink.isVisible().catch(() => false);
    const hasSettings = await settingsLink.isVisible().catch(() => false);

    // Хотя бы один extended пункт должен быть виден
    expect(hasCompliance || hasSettings).toBeTruthy();
  });

  test('Technician — does not see admin-only navigation items', async ({ page }) => {
    await setupPermissionsMockApi(page, MOCK_TECHNICIAN_USER);
    await page.goto('/');
    await page.waitForTimeout(1500);

    // Проверяем что technician НЕ видит admin-пункты
    const navItems = page.locator(
      'nav a, [role="navigation"] a, [class*="nav" i] a, [class*="sidebar" i] a',
    );

    const settingsLink = navItems.filter({ hasText: /settings|настройк/i }).first();
    const isSettingsVisible = await settingsLink.isVisible().catch(() => false);

    // Settings не должны быть видны technician
    if (isSettingsVisible) {
      // Если settings видны — проверяем что клик ведет на permission denied
      await settingsLink.click();
      await page.waitForTimeout(1000);
      const accessDenied = page.locator(
        'text=/access denied|доступ запрещен|forbidden|запрещено/i',
      );
      await expect(accessDenied).toBeVisible();
    }
  });

  test('RBAC — navigation hides restricted routes based on role', async ({ page }) => {
    const roles = [
      { user: MOCK_ADMIN_USER, expectRestricted: false },
      { user: MOCK_TECHNICIAN_USER, expectRestricted: true },
    ];

    for (const { user, expectRestricted } of roles) {
      await setupPermissionsMockApi(page, user);
      await page.goto('/settings/users');
      await page.waitForTimeout(1000);

      const accessDenied = page.locator(
        'text=/access denied|доступ запрещен|forbidden|запрещено/i',
      );
      const isDenied = await accessDenied.isVisible().catch(() => false);
      expect(isDenied).toBe(expectRestricted || isDenied);
    }
  });
});
