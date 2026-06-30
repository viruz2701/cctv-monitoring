// ═══════════════════════════════════════════════════════════════════════
// AgentTable — таблица edge-агентов с сортировкой и действиями
// EDGE-11: Agent Monitoring Dashboard
//
// Использует существующий Table компонент из components/ui/Table
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Table } from '../ui/Table';
import { Button } from '../ui/Button';
import { AgentStatusBadge } from './AgentStatusBadge';
import { Trash2, Send, ExternalLink } from '../ui/Icons';
import type { Agent } from '../../types/agent';

interface AgentTableProps {
  agents: Agent[];
  loading: boolean;
  onSendCommand: (agentId: string) => void;
  onDeleteAgent: (agentId: string) => void;
  /** Порог offline агентов для алерта */
  offlineThreshold?: number;
}

/**
 * AgentTable — сортируемая таблица агентов с действиями.
 * WCAG 2.1 AA: сортируемые заголовки, aria-sort, keyboard navigation.
 */
export function AgentTable({
  agents,
  loading,
  onSendCommand,
  onDeleteAgent,
}: AgentTableProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [sortColumn, setSortColumn] = useState<string>('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const sortedAgents = useMemo(() => {
    const sorted = [...agents];
    sorted.sort((a, b) => {
      let aVal: unknown = a[sortColumn as keyof Agent];
      let bVal: unknown = b[sortColumn as keyof Agent];

      // Вложенные поля: traffic.in, traffic.out
      if (sortColumn === 'trafficIn') {
        aVal = a.traffic?.in ?? 0;
        bVal = b.traffic?.in ?? 0;
      } else if (sortColumn === 'trafficOut') {
        aVal = a.traffic?.out ?? 0;
        bVal = b.traffic?.out ?? 0;
      }

      // Сравнение строк
      if (typeof aVal === 'string' && typeof bVal === 'string') {
        return sortDirection === 'asc'
          ? aVal.localeCompare(bVal)
          : bVal.localeCompare(aVal);
      }

      // Сравнение чисел
      const aNum = Number(aVal) || 0;
      const bNum = Number(bVal) || 0;
      return sortDirection === 'asc' ? aNum - bNum : bNum - aNum;
    });
    return sorted;
  }, [agents, sortColumn, sortDirection]);

  /** Форматирование трафика */
  const formatTraffic = (bytesPerSec: number): string => {
    if (bytesPerSec === 0) return '0 B/s';
    const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    const i = Math.floor(Math.log(bytesPerSec) / Math.log(1024));
    const value = bytesPerSec / Math.pow(1024, i);
    return `${value.toFixed(1)} ${units[i]}`;
  };

  const columns = [
    {
      key: 'name',
      header: t('agentColName'),
      sortable: true,
      render: (agent: Agent) => (
        <button
          onClick={() => navigate(`/agents/${agent.id}`)}
          className="font-medium text-blue-600 dark:text-blue-400 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
          aria-label={`${t('agentViewDetail')}: ${agent.name}`}
        >
          {agent.name}
        </button>
      ),
    },
    {
      key: 'site',
      header: t('agentColSite'),
      sortable: true,
    },
    {
      key: 'status',
      header: t('agentColStatus'),
      sortable: true,
      render: (agent: Agent) => <AgentStatusBadge status={agent.status} />,
    },
    {
      key: 'lastSeen',
      header: t('agentColLastSeen'),
      sortable: true,
      render: (agent: Agent) => {
        const date = new Date(agent.lastSeen);
        return (
          <span className="tabular-nums">
            {date.toLocaleString()}
          </span>
        );
      },
    },
    {
      key: 'version',
      header: t('agentColVersion'),
      sortable: true,
    },
    {
      key: 'trafficIn',
      header: t('agentColTraffic'),
      sortable: true,
      render: (agent: Agent) => (
        <span className="tabular-nums text-xs" title={`↓ ${formatTraffic(agent.traffic.in)} / ↑ ${formatTraffic(agent.traffic.out)}`}>
          ↓ {formatTraffic(agent.traffic.in)}
        </span>
      ),
    },
    {
      key: 'actions',
      header: t('agentColActions'),
      align: 'right' as const,
      render: (agent: Agent) => (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/agents/${agent.id}`);
            }}
            aria-label={`${t('agentViewDetail')}: ${agent.name}`}
          >
            <ExternalLink className="w-4 h-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              onSendCommand(agent.id);
            }}
            aria-label={`${t('agentSendCommand')}: ${agent.name}`}
          >
            <Send className="w-4 h-4" />
          </Button>
          <Button
            variant="danger"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              onDeleteAgent(agent.id);
            }}
            aria-label={`${t('agentDelete')}: ${agent.name}`}
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      ),
    },
  ];

  return (
    <Table<Agent>
      data={sortedAgents}
      columns={columns}
      keyExtractor={(agent) => agent.id}
      sortColumn={sortColumn}
      sortDirection={sortDirection}
      onSort={handleSort}
      loading={loading}
      emptyMessage={t('agentEmptyMessage')}
    />
  );
}
