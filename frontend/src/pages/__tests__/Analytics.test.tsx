import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { Analytics } from '../Analytics';

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

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({ token: 'test-token' }),
}));

vi.mock('../../services/api', () => ({
  api: {
    getPredictions: () => Promise.resolve([
      { device_id: 'CAM-001', prediction_date: '2026-07-01', failure_probability: 85, explanation: 'High temperature readings' },
      { device_id: 'CAM-002', prediction_date: '2026-07-01', failure_probability: 25, explanation: 'Normal operation' },
    ]),
  },
}));

vi.mock('../../hooks/useApiQuery', () => ({
  usePredictions: () => ({
    data: [
      { device_id: 'CAM-001', prediction_date: '2026-07-01', failure_probability: 85, explanation: 'High temperature readings' },
      { device_id: 'CAM-002', prediction_date: '2026-07-01', failure_probability: 25, explanation: 'Normal operation' },
    ],
    isLoading: false,
  }),
}));

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AreaChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  BarChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  LineChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  PieChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Pie: () => null,
  Bar: () => null,
  Area: () => null,
  Line: () => null,
  Cell: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  CartesianGrid: () => null,
  Legend: () => null,
}));

describe('Analytics', () => {
  it('renders predictions table', async () => {
    renderWithProviders(<Analytics />);
    const deviceId = await screen.findByText('CAM-001');
    expect(deviceId).toBeTruthy();
  });

  it('shows prediction risk badges', async () => {
    renderWithProviders(<Analytics />);
    const highRisk = await screen.findByText('85%');
    expect(highRisk).toBeTruthy();
  });

  it('shows analytics title', async () => {
    renderWithProviders(<Analytics />);
    const title = await screen.findByText('analytics_predictions', {}, { timeout: 3000 });
    expect(title).toBeTruthy();
  });
});
