// ═══════════════════════════════════════════════════════════════════════
// AgentStatsCard.test.tsx — Unit tests for AgentStatsCard component
// EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { AgentStatsCard } from '../AgentStatsCard';
import type { AgentStats } from '../../../types/agent';

// ── Mocks ──────────────────────────────────────────────────────────────

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        agentStatsTotal: 'Total Agents',
        agentStatsOnline: 'Online',
        agentStatsOffline: 'Offline',
        agentStatsErrors: 'Errors',
        agentStatsRegionLabel: 'Agent Statistics',
      };
      return translations[key] ?? key;
    },
    i18n: { language: 'en' },
  }),
}));

// ── Tests ──────────────────────────────────────────────────────────────

describe('AgentStatsCard', () => {
  const defaultStats: AgentStats = {
    total: 10,
    online: 7,
    offline: 2,
    errors: 1,
  };

  it('renders all four stat cards', () => {
    render(<AgentStatsCard stats={defaultStats} />);

    expect(screen.getByText('Total Agents')).toBeInTheDocument();
    expect(screen.getByText('Online')).toBeInTheDocument();
    expect(screen.getByText('Offline')).toBeInTheDocument();
    expect(screen.getByText('Errors')).toBeInTheDocument();
  });

  it('displays correct total value', () => {
    render(<AgentStatsCard stats={defaultStats} />);

    const totalValue = screen.getByText('10');
    expect(totalValue).toBeInTheDocument();
  });

  it('displays correct online value', () => {
    render(<AgentStatsCard stats={defaultStats} />);

    const onlineValue = screen.getByText('7');
    expect(onlineValue).toBeInTheDocument();
  });

  it('displays correct offline value', () => {
    render(<AgentStatsCard stats={defaultStats} />);

    const offlineValue = screen.getByText('2');
    expect(offlineValue).toBeInTheDocument();
  });

  it('displays correct errors value', () => {
    render(<AgentStatsCard stats={defaultStats} />);

    const errorsValue = screen.getByText('1');
    expect(errorsValue).toBeInTheDocument();
  });

  it('renders with zero values', () => {
    const zeroStats: AgentStats = { total: 0, online: 0, offline: 0, errors: 0 };
    render(<AgentStatsCard stats={zeroStats} />);

    expect(screen.getAllByText('0')).toHaveLength(4);
  });

  it('renders with all agents offline', () => {
    const allOffline: AgentStats = { total: 5, online: 0, offline: 5, errors: 0 };
    render(<AgentStatsCard stats={allOffline} />);

    // Five appears twice (total + offline), zero appears twice (online + errors)
    expect(screen.getAllByText('5')).toHaveLength(2);
    expect(screen.getAllByText('0')).toHaveLength(2);
  });

  it('has region role and label', () => {
    const { container } = render(<AgentStatsCard stats={defaultStats} />);

    const region = container.querySelector('[role="region"]');
    expect(region).toBeInTheDocument();
  });

  it('renders grid layout container', () => {
    const { container } = render(<AgentStatsCard stats={defaultStats} />);

    const gridContainer = container.querySelector('.grid');
    expect(gridContainer).toBeInTheDocument();
  });
});
