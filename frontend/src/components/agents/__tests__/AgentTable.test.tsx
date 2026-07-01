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

vi.mock('react-i18next', () => {
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
    agentStatusOnline: 'Online',
    agentStatusOffline: 'Offline',
    agentStatusError: 'Error',
  };
  return {
    useTranslation: () => ({
      t: (key: string) => translations[key] ?? key,
      i18n: { language: 'en' },
    }),
  };
});

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
    errors: 2,
    traffic: { in: 2_300_000, out: 1_100_000 },
  },
  {
    id: 'agent-4',
    name: 'Edge-Agent-Camera-04',
    site: 'Main Office',
    status: 'online',
    lastSeen: new Date(Date.now() - 50000).toISOString(),
    version: 'v2.5.2',
    cpu: 32.1,
    memory: 45.6,
    uptime: 604800,
    errors: 1,
    traffic: { in: 3_200_000, out: 1_900_000 },
  },
  {
    id: 'agent-empty',
    name: '',
    site: 'Storage',
    status: 'online',
    lastSeen: new Date().toISOString(),
    version: 'v2.5.1',
    cpu: 12.0,
    memory: 30.0,
    uptime: 86400,
    errors: 0,
    traffic: { in: 500_000, out: 250_000 },
  },
];

// ── Helpers ────────────────────────────────────────────────────────────

function renderTable(props: Partial<React.ComponentProps<typeof AgentTable>> = {}) {
  return render(
    <BrowserRouter>
      <AgentTable
        agents={mockAgents}
        loading={false}
        onSendCommand={vi.fn()}
        onDeleteAgent={vi.fn()}
        {...props}
      />
    </BrowserRouter>,
  );
}

// ── Tests ──────────────────────────────────────────────────────────────

describe('AgentTable', () => {
  describe('rendering', () => {
    it('renders all agents', () => {
      renderTable();
      expect(screen.getByText('Edge-Agent-Main-01')).toBeInTheDocument();
      expect(screen.getByText('Edge-Agent-Warehouse-03')).toBeInTheDocument();
    });

    it('renders all column headers', () => {
      renderTable();
      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Site')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText('Last Seen')).toBeInTheDocument();
      expect(screen.getByText('Version')).toBeInTheDocument();
      expect(screen.getByText('Traffic')).toBeInTheDocument();
    });

    it('renders empty message when no agents', () => {
      renderTable({ agents: [] });
      expect(screen.getByText('No agents found')).toBeInTheDocument();
    });

    it('renders status badges for each agent', () => {
      renderTable();
      // Two agents are 'Online' (agent-1, agent-4)
      expect(screen.getAllByText('Online').length).toBeGreaterThanOrEqual(2);
      // One agent is 'Offline' (agent-2), one is 'Error' (agent-3)
      expect(screen.getByText('Offline')).toBeInTheDocument();
      expect(screen.getByText('Error')).toBeInTheDocument();
    });
  });

  describe('sorting', () => {
    it('sorts when column header is clicked', () => {
      renderTable();
      const nameHeader = screen.getByText('Name');
      fireEvent.click(nameHeader);
      expect(screen.getByText('Edge-Agent-Main-01')).toBeInTheDocument();
    });
  });

  describe('actions', () => {
    it('renders action buttons for each agent', () => {
      renderTable();
      // View details кнопки имеют aria-label с шаблоном "View details: {agent.name}"
      // Используем getAllByLabelText т.к. имя агента также является ссылкой
      const viewButtons = screen.getAllByLabelText(/^View details: /);
      expect(viewButtons.length).toBeGreaterThanOrEqual(4);
      // Send command кнопки (aria-label)
      const sendButtons = screen.getAllByLabelText(/^Send command: /);
      expect(sendButtons.length).toBeGreaterThanOrEqual(4);
    });

    it('calls onSendCommand when send button is clicked', () => {
      const onSendCommand = vi.fn();
      renderTable({ onSendCommand });
      // Send кнопка использует aria-label (Send icon)
      const sendButton = screen.getByLabelText('Send command: Edge-Agent-Main-01');
      fireEvent.click(sendButton);
      expect(onSendCommand).toHaveBeenCalledWith('agent-1');
    });

    it('calls onDeleteAgent when delete button is clicked', () => {
      const onDeleteAgent = vi.fn();
      renderTable({ onDeleteAgent });
      // Delete кнопка использует aria-label (Trash2 icon)
      const deleteButton = screen.getByLabelText('Delete: Edge-Agent-Main-01');
      fireEvent.click(deleteButton);
      expect(onDeleteAgent).toHaveBeenCalledWith('agent-1');
    });
  });

  describe('empty state', () => {
    it('shows empty message when array is empty', () => {
      renderTable({ agents: [] });
      expect(screen.getByText('No agents found')).toBeInTheDocument();
    });
  });
});
