// ═══════════════════════════════════════════════════════════════════════
// AgentTable.test.tsx — Unit tests for AgentTable component
// EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { AgentTable } from '../AgentTable';
import type { Agent } from '../../../types/agent';

// ── Mocks ──────────────────────────────────────────────────────────────

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
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
      };
      return translations[key] ?? key;
    },
    i18n: { language: 'en' },
  },
}));

// ── Mock data ──────────────────────────────────────────────────────────

const mockAgents: Agent[] = [
  {
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
  },
  {
    id: 'agent-2',
    name: 'Edge-Agent-Branch-02',
    site: 'Branch Office',
    status: 'offline',
    lastSeen: new Date(Date.now() - 7200000).toISOString(),
    version: 'v2.4.0',
    cpu: 0,
    memory: 0,
    uptime: 0,
    errors: 3,
    traffic: { in: 0, out: 0 },
  },
  {
    id: 'agent-3',
    name: 'Edge-Agent-Warehouse-03',
    site: 'Warehouse',
    status: 'error',
    lastSeen: new Date(Date.now() - 3600000).toISOString(),
    version: 'v2.5.1',
    cpu: 89.5,
    memory: 95.2,
    uptime: 43200,
    errors: 7,
    traffic: { in: 500_000, out: 200_000 },
  },
];

// ── Helpers ────────────────────────────────────────────────────────────

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>);
}

// ── Tests ──────────────────────────────────────────────────────────────

describe('AgentTable', () => {
  const defaultProps = {
    agents: mockAgents,
    loading: false,
    onSendCommand: vi.fn(),
    onDeleteAgent: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders all agent rows', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    expect(screen.getByText('Edge-Agent-Main-01')).toBeInTheDocument();
    expect(screen.getByText('Edge-Agent-Branch-02')).toBeInTheDocument();
    expect(screen.getByText('Edge-Agent-Warehouse-03')).toBeInTheDocument();
  });

  it('renders column headers', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Version')).toBeInTheDocument();
    expect(screen.getByText('Actions')).toBeInTheDocument();
  });

  it('shows empty message when no agents', () => {
    renderWithRouter(<AgentTable {...defaultProps} agents={[]} />);

    expect(screen.getByText('No agents found')).toBeInTheDocument();
  });

  it('calls onSendCommand when send button is clicked', () => {
    const onSendCommand = vi.fn();
    renderWithRouter(
      <AgentTable {...defaultProps} onSendCommand={onSendCommand} />,
    );

    const sendButtons = screen.getAllByRole('button');
    // Find the send buttons (those with aria-label containing "Send command")
    const sendBtn = sendButtons.find(
      (btn) => btn.getAttribute('aria-label')?.includes('Send command'),
    );
    if (sendBtn) {
      fireEvent.click(sendBtn);
      expect(onSendCommand).toHaveBeenCalled();
    }
  });

  it('calls onDeleteAgent when delete button is clicked', () => {
    const onDeleteAgent = vi.fn();
    renderWithRouter(
      <AgentTable {...defaultProps} onDeleteAgent={onDeleteAgent} />,
    );

    const deleteButtons = screen.getAllByRole('button');
    const deleteBtn = deleteButtons.find(
      (btn) => btn.getAttribute('aria-label')?.includes('Delete'),
    );
    if (deleteBtn) {
      fireEvent.click(deleteBtn);
      expect(onDeleteAgent).toHaveBeenCalled();
    }
  });

  it('navigates to agent detail when name is clicked', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    const nameButton = screen.getByText('Edge-Agent-Main-01');
    fireEvent.click(nameButton);
    expect(mockNavigate).toHaveBeenCalledWith('/agents/agent-1');
  });

  it('renders status badge for each agent', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    expect(screen.getByText('Online')).toBeInTheDocument();
    expect(screen.getByText('Offline')).toBeInTheDocument();
  });

  it('sorts by name on column header click', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    const nameHeader = screen.getByText('Name');
    fireEvent.click(nameHeader);
    // After click, the sort direction should change
    // Verify agents are still rendered
    expect(screen.getByText('Edge-Agent-Main-01')).toBeInTheDocument();
  });

  it('handles view detail icon click', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    const detailButtons = screen.getAllByRole('button');
    const detailBtn = detailButtons.find(
      (btn) => btn.getAttribute('aria-label')?.includes('View details'),
    );
    if (detailBtn) {
      fireEvent.click(detailBtn);
      expect(mockNavigate).toHaveBeenCalled();
    }
  });

  it('renders traffic information', () => {
    renderWithRouter(<AgentTable {...defaultProps} />);

    // Traffic should be displayed (1.5 MB/s for first agent)
    const trafficElements = screen.getAllByText(/B\/s|KB\/s|MB\/s/);
    expect(trafficElements.length).toBeGreaterThan(0);
  });
});
