// ═══════════════════════════════════════════════════════════════════════
// AgentStatusBadge.test.tsx — Unit tests for AgentStatusBadge component
// EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { AgentStatusBadge } from '../AgentStatusBadge';

// ── Mocks ──────────────────────────────────────────────────────────────

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        agentStatusOnline: 'Online',
        agentStatusOffline: 'Offline',
        agentStatusError: 'Error',
      };
      return translations[key] ?? key;
    },
    i18n: { language: 'en' },
  }),
}));

// ── Tests ──────────────────────────────────────────────────────────────

describe('AgentStatusBadge', () => {
  it('renders online status with correct text', () => {
    render(<AgentStatusBadge status="online" />);
    expect(screen.getByText('Online')).toBeInTheDocument();
  });

  it('renders offline status with correct text', () => {
    render(<AgentStatusBadge status="offline" />);
    expect(screen.getByText('Offline')).toBeInTheDocument();
  });

  it('renders error status with correct text', () => {
    render(<AgentStatusBadge status="error" />);
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('has aria-label for screen readers', () => {
    render(<AgentStatusBadge status="online" />);
    const badge = screen.getByText('Online');
    expect(badge).toBeInTheDocument();
  });

  it('applies success variant for online status', () => {
    const { container } = render(<AgentStatusBadge status="online" />);
    const badge = container.querySelector('span');
    expect(badge).toBeInTheDocument();
  });

  it('applies danger variant for offline status', () => {
    const { container } = render(<AgentStatusBadge status="offline" />);
    const badge = container.querySelector('span');
    expect(badge).toBeInTheDocument();
  });

  it('applies warning variant for error status', () => {
    const { container } = render(<AgentStatusBadge status="error" />);
    const badge = container.querySelector('span');
    expect(badge).toBeInTheDocument();
  });

  it('falls back to offline for unknown status', () => {
    render(<AgentStatusBadge status={'unknown' as any} />);
    expect(screen.getByText('Offline')).toBeInTheDocument();
  });

  it('renders with dot indicator', () => {
    const { container } = render(<AgentStatusBadge status="online" />);
    expect(container.firstChild).toBeInTheDocument();
  });
});
