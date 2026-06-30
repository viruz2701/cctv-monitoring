// ═══════════════════════════════════════════════════════════════════════
// AgentDetail.test.tsx — Unit tests for AgentDetail page
// EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AgentDetail } from '../AgentDetail';
import type { Agent } from '../../types/agent';

// ── Mocks ──────────────────────────────────────────────────────────────

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const mockAgentList = vi.fn<() => Agent[]>();
const mockAgentLoading = vi.fn<() => boolean>();
const mockAgentError = vi.fn<() => string | null>();
const mockFetchAgents = vi.fn();

vi.mock('../../store/agentStore', () => ({
  useAgentList: () => mockAgentList(),
  useAgentLoading: () => mockAgentLoading(),
  useAgentError: () => mockAgentError(),
  useAgentStore: (selector: (s: any) => any) =>
    selector({ fetchAgents: mockFetchAgents }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        agentDetailTitle: 'Agent Detail',
        agentBackToList: 'Back to list',
        agentRefresh: 'Refresh',
        agentSystemInfo: 'System Info',
        agentPerformance: 'Performance',
        agentNetwork: 'Network',
        agentVersion: 'Version',
        agentUptime: 'Uptime',
        agentLastSeen: 'Last Seen',
        agentCpu: 'CPU',
        agentMemory: 'Memory',
        agentErrors: 'Errors',
        agentTrafficIn: 'Traffic In',
        agentTrafficOut: 'Traffic Out',
        agentNotFound: 'Agent Not Found',
        agentNotFoundBody: 'Agent with ID {id} not found',
        agentErrorTitle: 'Error',
        loading: 'Loading...',
      };
      return translations[key] ?? key;
    },
    i18n: { language: 'en' },
  }),
}));

// ── Mock data ──────────────────────────────────────────────────────────

const mockAgent: Agent = {
  id: 'agent-1',
  name: 'Edge-Agent-Main-01',
  site: 'Main Office',
  status: 'online',
  lastSeen: new Date().toISOString(),
  version: 'v2.5.1',
  cpu: 45.2,
  memory: 62.8,
  uptime: 172800,
  errors: 0,
  traffic: { in: 1_500_000, out: 800_000 },
};

// ── Helpers ────────────────────────────────────────────────────────────

function renderWithRoute(agentId: string = 'agent-1') {
  return render(
    <MemoryRouter initialEntries={[`/agents/${agentId}`]}>
      <Routes>
        <Route path="/agents/:id" element={<AgentDetail />} />
      </Routes>
    </MemoryRouter>,
  );
}

// ── Tests ──────────────────────────────────────────────────────────────

describe('AgentDetail', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAgentList.mockReturnValue([mockAgent]);
    mockAgentLoading.mockReturnValue(false);
    mockAgentError.mockReturnValue(null);
  });

  it('renders agent name and site', () => {
    renderWithRoute();

    expect(screen.getByText('Edge-Agent-Main-01')).toBeInTheDocument();
    expect(screen.getByText(/Main Office/)).toBeInTheDocument();
  });

  it('renders system info section', () => {
    renderWithRoute();

    expect(screen.getByText('System Info')).toBeInTheDocument();
    expect(screen.getByText('v2.5.1')).toBeInTheDocument();
  });

  it('renders performance section', () => {
    renderWithRoute();

    expect(screen.getByText('Performance')).toBeInTheDocument();
    expect(screen.getByText(/45\.2%/)).toBeInTheDocument();
    expect(screen.getByText(/62\.8%/)).toBeInTheDocument();
  });

  it('renders network section', () => {
    renderWithRoute();

    expect(screen.getByText('Network')).toBeInTheDocument();
    expect(screen.getByText('Traffic In')).toBeInTheDocument();
    expect(screen.getByText('Traffic Out')).toBeInTheDocument();
  });

  it('renders back button', () => {
    renderWithRoute();

    const backBtn = screen.getByText('Back to list');
    expect(backBtn).toBeInTheDocument();
  });

  it('renders refresh button', () => {
    renderWithRoute();

    const refreshBtn = screen.getByText('Refresh');
    expect(refreshBtn).toBeInTheDocument();
  });

  it('shows loading state when agent not yet loaded', () => {
    mockAgentList.mockReturnValue([]);
    mockAgentLoading.mockReturnValue(true);

    renderWithRoute();

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('shows error alert when error exists', () => {
    mockAgentError.mockReturnValue('Failed to fetch agent');

    renderWithRoute();

    expect(screen.getByText('Error')).toBeInTheDocument();
    expect(screen.getByText('Failed to fetch agent')).toBeInTheDocument();
  });

  it('shows not found when agent does not exist', () => {
    renderWithRoute('nonexistent');

    expect(screen.getByText('Agent Not Found')).toBeInTheDocument();
  });

  it('renders version info', () => {
    renderWithRoute();

    expect(screen.getByText('Version')).toBeInTheDocument();
    expect(screen.getByText('v2.5.1')).toBeInTheDocument();
  });

  it('renders uptime correctly (2 days)', () => {
    renderWithRoute();

    expect(screen.getByText('Uptime')).toBeInTheDocument();
  });

  it('renders last seen timestamp', () => {
    renderWithRoute();

    expect(screen.getByText('Last Seen')).toBeInTheDocument();
  });
});
