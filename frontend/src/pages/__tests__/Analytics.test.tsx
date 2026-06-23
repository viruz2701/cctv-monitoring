import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Analytics } from '../Analytics';

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
    render(<Analytics />);
    // Wait for data to load
    const deviceId = await screen.findByText('CAM-001');
    expect(deviceId).toBeTruthy();
  });

  it('shows prediction risk badges', async () => {
    render(<Analytics />);
    const highRisk = await screen.findByText('85%');
    expect(highRisk).toBeTruthy();
  });

  it('shows analytics title', () => {
    render(<Analytics />);
    expect(screen.getByText('analytics_predictions')).toBeTruthy();
  });
});
