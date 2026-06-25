import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { Dashboard } from '../Dashboard';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    </MemoryRouter>
  );
}

// Mock all contexts and hooks used by Dashboard
vi.mock('../../context/DataContext', () => ({
  useTickets: () => ({ tickets: [] }),
  useAlerts: () => ({ alerts: [] }),
  useDevicesSites: () => ({
    devices: [],
    sites: [],
  }),
  useSettings: () => ({
    dashboardConfig: {
      showStatsRow: true,
      showTicketStats: true,
      showRecentAlerts: true,
      showLatestTickets: true,
      showQuickActions: true,
    },
    updateDashboardConfig: vi.fn(),
  }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: 'ru' },
  }),
}));

vi.mock('../../hooks/useApiQuery', () => ({
  useDevices: () => ({ data: [], isLoading: false }),
  useSites: () => ({ data: [], isLoading: false }),
  useAlerts: () => ({ data: [], isLoading: false }),
  useTickets: () => ({ data: [], isLoading: false }),
  useAlarms: () => ({ data: [], isLoading: false }),
}));

vi.mock('../../components/dashboard/AlertBanner', () => ({
  AlertBanner: () => <div data-testid="alert-banner" />,
}));

vi.mock('react-grid-layout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="grid-layout">{children}</div>,
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  BarChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  LineChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Bar: () => null,
  Area: () => null,
  Line: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  CartesianGrid: () => null,
}));

describe('Dashboard', () => {
  it('renders without crashing', async () => {
    renderWithProviders(<Dashboard />);
    const title = await screen.findByText('total_devices', {}, { timeout: 3000 });
    expect(title).toBeTruthy();
  });

  it('shows filter controls', async () => {
    renderWithProviders(<Dashboard />);
    const sites = await screen.findByText('all_sites', {}, { timeout: 3000 });
    expect(sites).toBeTruthy();
    expect(screen.getByText('all_types')).toBeTruthy();
    expect(screen.getByText('all_statuses')).toBeTruthy();
  });

  it('shows customize layout button', async () => {
    renderWithProviders(<Dashboard />);
    const btn = await screen.findByText('customize_layout', {}, { timeout: 3000 });
    expect(btn).toBeTruthy();
  });

  it('shows clear filters button', async () => {
    renderWithProviders(<Dashboard />);
    const btn = await screen.findByText('clear_filters', {}, { timeout: 3000 });
    expect(btn).toBeTruthy();
  });
});
