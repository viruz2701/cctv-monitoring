// ═══════════════════════════════════════════════════════════════════════
// DashboardHub.test.tsx — Unit tests for unified DashboardHub
// P1-UX.1: Dashboard Unification
//   - Проверяет что DashboardHub рендерится
//   - Проверяет role-based tab visibility
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { DashboardHub } from '../DashboardHub';
import type { AuthUser } from '../../hooks/useAuth';

// ── Mocks ──────────────────────────────────────────────────────────────

const mockUseAuth = vi.fn<() => { user: AuthUser | null; token: string | null; logout: () => void }>();

vi.mock('../../hooks/useAuth', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useAuth: () => mockUseAuth(),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: 'ru' },
  }),
}));

vi.mock('../../components/dashboard/DragDropDashboard', () => ({
  DragDropDashboard: ({ widgets }: { widgets: unknown[] }) => (
    <div data-testid="drag-drop-dashboard">{widgets.length} widgets</div>
  ),
}));

vi.mock('../../components/dashboard/WidgetRegistry', () => ({
  getWidgetsForTab: (tabId: string, _role: string) => {
    const mockWidgets: Record<string, { id: string; titleKey: string; minW: number; minH: number }[]> = {
      overview: [
        { id: 'statsOverview', titleKey: 'device_statistics', minW: 2, minH: 1 },
        { id: 'ticketAnalytics', titleKey: 'ticket_analytics', minW: 2, minH: 1 },
      ],
      sla: [],
      performance: [],
      maintenance: [
        { id: 'maintenanceOverview', titleKey: 'maintenance_overview', minW: 2, minH: 1 },
      ],
    };
    return mockWidgets[tabId] ?? [];
  },
}));

// Lazy-loaded tab components
vi.mock('../../components/dashboard/tabs/OverviewTab', () => ({
  default: () => <div data-testid="overview-tab">Overview Tab Content</div>,
}));

vi.mock('../../components/dashboard/tabs/SLAComplianceTab', () => ({
  default: () => <div data-testid="sla-tab">SLA Tab Content</div>,
}));

vi.mock('../../components/dashboard/tabs/PerformanceTab', () => ({
  default: () => <div data-testid="performance-tab">Performance Tab Content</div>,
}));

vi.mock('../../components/dashboard/tabs/MaintenanceTab', () => ({
  default: () => <div data-testid="maintenance-tab">Maintenance Tab Content</div>,
}));

// ── Test helpers ────────────────────────────────────────────────────────

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter initialEntries={['/dashboard']}>
      <QueryClientProvider client={queryClient}>
        {ui}
      </QueryClientProvider>
    </MemoryRouter>
  );
}

function createUser(role: AuthUser['role']): AuthUser {
  return {
    id: `test-${role}-id`,
    username: `test_${role}`,
    role,
    name: `Test ${role.charAt(0).toUpperCase() + role.slice(1)}`,
    email: `${role}@test.com`,
  };
}

// ── Tests ───────────────────────────────────────────────────────────────

describe('DashboardHub', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without crashing for admin', async () => {
    mockUseAuth.mockReturnValue({
      user: createUser('admin'),
      token: 'test-token',
      logout: vi.fn(),
    });

    renderWithProviders(<DashboardHub />);
    const title = await screen.findByText('dashboard', {}, { timeout: 3000 });
    expect(title).toBeTruthy();
  });

  it('shows all tabs for admin role', async () => {
    mockUseAuth.mockReturnValue({
      user: createUser('admin'),
      token: 'test-token',
      logout: vi.fn(),
    });

    renderWithProviders(<DashboardHub />);

    // Admin should see all tabs: overview, sla, performance, maintenance
    expect(await screen.findByText('overview', {}, { timeout: 3000 })).toBeTruthy();
    expect(await screen.findByText('sla_compliance')).toBeTruthy();
    expect(await screen.findByText('performance')).toBeTruthy();
    expect(await screen.findByText('maintenance_schedule')).toBeTruthy();
  });

  it('shows all tabs for manager role', async () => {
    mockUseAuth.mockReturnValue({
      user: createUser('manager'),
      token: 'test-token',
      logout: vi.fn(),
    });

    renderWithProviders(<DashboardHub />);

    // Manager should see all tabs: overview, sla, performance, maintenance
    expect(await screen.findByText('overview', {}, { timeout: 3000 })).toBeTruthy();
    expect(await screen.findByText('sla_compliance')).toBeTruthy();
    expect(await screen.findByText('performance')).toBeTruthy();
    expect(await screen.findByText('maintenance_schedule')).toBeTruthy();
  });

  it('shows only Overview and Maintenance for technician role', async () => {
    mockUseAuth.mockReturnValue({
      user: createUser('technician'),
      token: 'test-token',
      logout: vi.fn(),
    });

    renderWithProviders(<DashboardHub />);

    // Technician should see: overview, maintenance
    expect(await screen.findByText('overview', {}, { timeout: 3000 })).toBeTruthy();
    expect(await screen.findByText('maintenance_schedule')).toBeTruthy();

    // Technician should NOT see: sla, performance
    expect(screen.queryByText('sla_compliance')).toBeNull();
    expect(screen.queryByText('performance')).toBeNull();
  });

  it('shows only Overview for viewer role', async () => {
    mockUseAuth.mockReturnValue({
      user: createUser('viewer'),
      token: 'test-token',
      logout: vi.fn(),
    });

    renderWithProviders(<DashboardHub />);

    // Viewer should see only: overview
    expect(await screen.findByText('overview', {}, { timeout: 3000 })).toBeTruthy();

    // Viewer should NOT see: sla, performance, maintenance
    expect(screen.queryByText('sla_compliance')).toBeNull();
    expect(screen.queryByText('performance')).toBeNull();
    expect(screen.queryByText('maintenance_schedule')).toBeNull();
  });

  it('renders DragDropDashboard for overview tab (useWidgets=true)', async () => {
    mockUseAuth.mockReturnValue({
      user: createUser('admin'),
      token: 'test-token',
      logout: vi.fn(),
    });

    renderWithProviders(<DashboardHub />);

    // Overview tab has useWidgets=true, so DragDropDashboard should render
    const ddDashboard = await screen.findByTestId('drag-drop-dashboard', {}, { timeout: 3000 });
    expect(ddDashboard).toBeTruthy();
    expect(ddDashboard.textContent).toContain('widgets');
  });
});
