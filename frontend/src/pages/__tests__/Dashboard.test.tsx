import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { Dashboard } from '../Dashboard';
import { SettingsProvider } from '../../context/SettingsContext';
import { AuthProvider } from '../../hooks/useAuth';

vi.mock('../../hooks/useAuth', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useAuth: () => ({ user: { role: 'admin', name: 'Test' }, token: 'test-token', logout: vi.fn() }),
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <SettingsProvider>{ui}</SettingsProvider>
        </AuthProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

vi.mock('../../services/api', () => ({
  api: {
    getDashboardStats: () => Promise.resolve(null),
  },
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
  ResponsiveGridLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="responsive-grid">{children}</div>,
  useContainerWidth: () => 800,
  WidthProvider: (c: any) => c,
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
    // The customize button is available via text 'customize_layout' or aria-label
    const btn = await screen.findByRole('button', { name: /customize/i }, { timeout: 3000 });
    expect(btn).toBeTruthy();
  });

  it('shows clear filters button', async () => {
    renderWithProviders(<Dashboard />);
    const btn = await screen.findByText('clear_filters', {}, { timeout: 3000 });
    expect(btn).toBeTruthy();
  });
});
