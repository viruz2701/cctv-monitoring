// ═══════════════════════════════════════════════════════════════════════
// AgentDashboard.test.tsx — Unit tests for AgentDashboard page
// EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { AgentDashboard } from '../AgentDashboard';
import type { Agent, AgentStats } from '../../types/agent';

// ── Mocks ──────────────────────────────────────────────────────────────

const mockAgentList = vi.fn<() => Agent[]>();
const mockAgentStats = vi.fn<() => AgentStats>();
const mockAgentLoading = vi.fn<() => boolean>();
const mockAgentError = vi.fn<() => string | null>();
const mockFetchAgents = vi.fn();
const mockSendCommand = vi.fn<() => Promise<boolean>>();
const mockDeleteAgent = vi.fn<() => Promise<boolean>>();
const mockClearError = vi.fn();

vi.mock('../../store/agentStore', () => ({
  useAgentList: () => mockAgentList(),
  useAgentStats: () => mockAgentStats(),
  useAgentLoading: () => mockAgentLoading(),
  useAgentError: () => mockAgentError(),
  useAgentStore: (selector: (s: any) => any) =>
    selector({
      fetchAgents: mockFetchAgents,
      sendCommand: mockSendCommand,
      deleteAgent: mockDeleteAgent,
      clearError: mockClearError,
    }),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        agentDashboardTitle: 'Agent Dashboard',
        agentDashboardSubtitle: 'Monitor and manage edge agents',
        agentRefresh: 'Refresh',
        agentOfflineAlertTitle: 'Offline Agents',
        agentOfflineAlertBody: 'There are {count} offline agents',
        agentErrorTitle: 'Error',
        agentListTitle: 'All Agents',
        agentKeyboardHint: 'Press / to search',
        agentStatsTotal: 'Total Agents',
        agentStatsOnline: 'Online',
        agentStatsOffline: 'Offline',
        agentStatsErrors: 'Errors',
        agentColName: 'Name',
        agentColSite: 'Site',
        agentColStatus: 'Status',
        agentColLastSeen: 'Last Seen',
        agentColVersion: 'Version',
        agentColTraffic: 'Traffic',
        agentColActions: 'Actions',
        agentViewDetail: 'View details',
        agentSendCommand: 'Send command',
        agentDelete: 'Delete',
        agentEmptyMessage: 'No agents found',
        agentCommandPrompt: 'Enter command:',
        agentDeleteConfirm: 'Delete this agent?',
        agentStatusOnline: 'Online',
        agentStatsRegionLabel: 'Agent Statistics',
      };
      return translations[key] ?? key;
    },
    i18n: { language: 'en' },
  }),
}));

// Mock child components
vi.mock('../../components/agents/AgentStatsCard', () => ({
  AgentStatsCard: ({ stats }: { stats: AgentStats }) => (
    <div data-testid="agent-stats-card">
      <span>Total: {stats.total}</span>
      <span>Online: {stats.online}</span>
      <span>Offline: {stats.offline}</span>
      <span>Errors: {stats.errors}</span>
    </div>
  ),
}));

vi.mock('../../components/agents/AgentTable', () => ({
  AgentTable: ({
    agents,
    loading,
    onSendCommand,
    onDeleteAgent,
  }: {
    agents: Agent[];
    loading: boolean;
    onSendCommand: (id: string) => void;
    onDeleteAgent: (id: string) => void;
  }) => (
    <div data-testid="agent-table">
      <span>Agents: {agents.length}</span>
      <span>Loading: {String(loading)}</span>
      <button data-testid="mock-send" onClick={() => onSendCommand('agent-1')}>
        Send
      </button>
      <button data-testid="mock-delete" onClick={() => onDeleteAgent('agent-2')}>
        Delete
      </button>
    </div>
  ),
}));

vi.mock('../../components/ui/Alert', () => ({
  Alert: ({
    variant,
    title,
    children,
    onClose,
  }: {
    variant: string;
    title: string;
    children: React.ReactNode;
    onClose?: () => void;
  }) => (
    <div data-testid={`alert-${variant}`} role="alert">
      <span>{title}</span>
      <span>{children}</span>
      {onClose && (
        <button data-testid="alert-close" onClick={onClose}>
          Close
        </button>
      )}
    </div>
  ),
}));

// ── Mock data ──────────────────────────────────────────────────────────

const mockAgents: Agent[] = [
  {
    id: 'agent-1', name: 'Agent-1', site: 'Site A', status: 'online',
    lastSeen: new Date().toISOString(), version: 'v1', traffic: { in: 0, out: 0 },
    errors: 0, uptime: 100, cpu: 20, memory: 30,
  },
  {
    id: 'agent-2', name: 'Agent-2', site: 'Site B', status: 'offline',
    lastSeen: new Date().toISOString(), version: 'v1', traffic: { in: 0, out: 0 },
    errors: 2, uptime: 0, cpu: 0, memory: 0,
  },
];

const defaultStats: AgentStats = { total: 2, online: 1, offline: 1, errors: 0 };

// ── Tests ──────────────────────────────────────────────────────────────

function renderDashboard() {
  return render(
    <MemoryRouter>
      <AgentDashboard />
    </MemoryRouter>,
  );
}

describe('AgentDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAgentList.mockReturnValue(mockAgents);
    mockAgentStats.mockReturnValue(defaultStats);
    mockAgentLoading.mockReturnValue(false);
    mockAgentError.mockReturnValue(null);
    mockSendCommand.mockResolvedValue(true);
    mockDeleteAgent.mockResolvedValue(true);
  });

  it('renders dashboard title', () => {
    renderDashboard();
    expect(screen.getByText('Agent Dashboard')).toBeInTheDocument();
  });

  it('renders subtitle', () => {
    renderDashboard();
    expect(screen.getByText('Monitor and manage edge agents')).toBeInTheDocument();
  });

  it('renders stats card', () => {
    renderDashboard();
    const statsCard = screen.getByTestId('agent-stats-card');
    expect(statsCard).toBeInTheDocument();
    expect(screen.getByText('Total: 2')).toBeInTheDocument();
    expect(screen.getByText('Online: 1')).toBeInTheDocument();
    expect(screen.getByText('Offline: 1')).toBeInTheDocument();
  });

  it('renders agent table', () => {
    renderDashboard();
    const table = screen.getByTestId('agent-table');
    expect(table).toBeInTheDocument();
    expect(screen.getByText('Agents: 2')).toBeInTheDocument();
  });

  it('renders refresh button', () => {
    renderDashboard();
    const refreshBtn = screen.getByText('Refresh');
    expect(refreshBtn).toBeInTheDocument();
  });

  it('calls fetchAgents on mount', () => {
    renderDashboard();
    expect(mockFetchAgents).toHaveBeenCalledTimes(1);
  });

  it('shows offline alert when there are offline agents', () => {
    renderDashboard();
    const alert = screen.getByTestId('alert-warning');
    expect(alert).toBeInTheDocument();
    expect(screen.getByText('Offline Agents')).toBeInTheDocument();
  });

  it('shows error alert when error exists', () => {
    mockAgentError.mockReturnValue('Connection failed');
    renderDashboard();

    const alert = screen.getByTestId('alert-error');
    expect(alert).toBeInTheDocument();
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('clears error when close button is clicked', () => {
    mockAgentError.mockReturnValue('Connection failed');
    renderDashboard();

    const closeBtn = screen.getByTestId('alert-close');
    fireEvent.click(closeBtn);
    expect(mockClearError).toHaveBeenCalled();
  });

  it('does not show offline alert when all agents are online', () => {
    mockAgentStats.mockReturnValue({ total: 2, online: 2, offline: 0, errors: 0 });
    renderDashboard();

    const warningAlert = screen.queryByTestId('alert-warning');
    expect(warningAlert).not.toBeInTheDocument();
  });

  it('renders keyboard hint', () => {
    renderDashboard();
    expect(screen.getByText('Press / to search')).toBeInTheDocument();
  });

  it('has main region with aria-label', () => {
    renderDashboard();
    const main = screen.getByRole('main');
    expect(main).toBeInTheDocument();
  });

  it('refresh button is disabled while loading', () => {
    mockAgentLoading.mockReturnValue(true);
    renderDashboard();

    const refreshBtn = screen.getByText('Refresh').closest('button');
    expect(refreshBtn).toBeDisabled();
  });

  it('calls fetchAgents on refresh click', () => {
    renderDashboard();
    mockFetchAgents.mockClear();

    const refreshBtn = screen.getByText('Refresh');
    fireEvent.click(refreshBtn);
    expect(mockFetchAgents).toHaveBeenCalledTimes(1);
  });
});
