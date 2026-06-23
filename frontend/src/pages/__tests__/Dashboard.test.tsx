import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Dashboard } from '../Dashboard';

// Mock contexts
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
  it('renders without crashing', () => {
    render(<Dashboard />);
    expect(screen.getByText('total_devices')).toBeTruthy();
  });

  it('shows filter controls', () => {
    render(<Dashboard />);
    expect(screen.getByText('all_sites')).toBeTruthy();
    expect(screen.getByText('all_types')).toBeTruthy();
    expect(screen.getByText('all_statuses')).toBeTruthy();
  });

  it('shows customize layout button', () => {
    render(<Dashboard />);
    expect(screen.getByText('customize_layout')).toBeTruthy();
  });

  it('shows clear filters button', () => {
    render(<Dashboard />);
    expect(screen.getByText('clear_filters')).toBeTruthy();
  });
});
