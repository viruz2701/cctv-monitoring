// ═══════════════════════════════════════════════════════════════════════
// AgentDetail — детальная информация об edge-агенте
// EDGE-11: P1-EDGE Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, useNavigate } from 'react-router-dom';
import { useAgentStore, useAgentList, useAgentLoading, useAgentError } from '../store/agentStore';
import { AgentStatusBadge } from '../components/agents/AgentStatusBadge';
import { Button } from '../components/ui/Button';
import { Alert } from '../components/ui/Alert';
import { ArrowLeft, RefreshCw, Server, Cpu, HardDrive, Wifi, Activity } from '../components/ui/Icons';
import type { Agent } from '../types/agent';

/** Форматирование байт/с */
function formatTraffic(bytesPerSec: number): string {
  if (bytesPerSec === 0) return '0 B/s';
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
  const i = Math.floor(Math.log(bytesPerSec) / Math.log(1024));
  const value = bytesPerSec / Math.pow(1024, i);
  return `${value.toFixed(1)} ${units[i]}`;
}

/** Форматирование секунд в дни/часы */
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  parts.push(`${mins}m`);
  return parts.join(' ');
}

/** StatRow — строка метрики */
function StatRow({
  label,
  value,
  icon: Icon,
  color = 'text-slate-600',
}: {
  label: string;
  value: string | number;
  icon: React.ComponentType<{ className?: string }>;
  color?: string;
}) {
  return (
    <div className="flex items-center gap-3 py-3 border-b border-slate-100 dark:border-slate-700 last:border-0">
      <Icon className={`w-5 h-5 ${color}`} aria-hidden="true" />
      <span className="text-sm text-slate-500 dark:text-slate-400 flex-1">{label}</span>
      <span className="text-sm font-medium text-slate-900 dark:text-white tabular-nums">
        {value}
      </span>
    </div>
  );
}

/**
 * AgentDetail — страница с детальной информацией об агенте.
 * Загружает данные из store, отображает метрики и статус.
 */
export function AgentDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const agents = useAgentList();
  const loading = useAgentLoading();
  const error = useAgentError();
  const fetchAgents = useAgentStore((s) => s.fetchAgents);

  const agent = agents.find((a) => a.id === id) ?? null;

  useEffect(() => {
    if (agents.length === 0 && !loading) {
      fetchAgents();
    }
  }, [agents.length, loading, fetchAgents]);

  if (loading && !agent) {
    return (
      <div className="space-y-6 animate-pulse" role="status" aria-label={t('loading')}>
        <div className="h-8 bg-slate-200 dark:bg-slate-700 rounded w-1/3" />
        <div className="h-48 bg-slate-200 dark:bg-slate-700 rounded-xl" />
      </div>
    );
  }

  if (!agent) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" onClick={() => navigate('/agents')}>
          <ArrowLeft className="w-4 h-4 mr-2" />
          {t('agentBackToList')}
        </Button>
        <Alert variant="error" title={t('agentNotFound')} assertive>
          {t('agentNotFoundBody', { id: id ?? '' })}
        </Alert>
      </div>
    );
  }

  return (
    <div className="space-y-6" role="main" aria-label={`${t('agentDetailTitle')}: ${agent.name}`}>
      {/* Back button */}
      <Button variant="ghost" onClick={() => navigate('/agents')}>
        <ArrowLeft className="w-4 h-4 mr-2" />
        {t('agentBackToList')}
      </Button>

      {/* Error alert */}
      {error && (
        <Alert variant="error" title={t('agentErrorTitle')} assertive>
          {error}
        </Alert>
      )}

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-3">
          <Server className="w-8 h-8 text-blue-600 dark:text-blue-400" aria-hidden="true" />
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {agent.name}
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {agent.site} · <AgentStatusBadge status={agent.status} />
            </p>
          </div>
        </div>
        <Button variant="primary" onClick={fetchAgents} disabled={loading}>
          <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          {t('agentRefresh')}
        </Button>
      </div>

      {/* Detail cards */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* System info */}
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-6">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white uppercase tracking-wider mb-4">
            {t('agentSystemInfo')}
          </h2>
          <StatRow label={t('agentVersion')} value={agent.version} icon={Server} />
          <StatRow label={t('agentUptime')} value={formatUptime(agent.uptime)} icon={Activity} />
          <StatRow
            label={t('agentLastSeen')}
            value={new Date(agent.lastSeen).toLocaleString()}
            icon={RefreshCw}
          />
        </div>

        {/* Performance */}
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-6">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white uppercase tracking-wider mb-4">
            {t('agentPerformance')}
          </h2>
          <StatRow
            label={t('agentCpu')}
            value={`${agent.cpu.toFixed(1)}%`}
            icon={Cpu}
            color={agent.cpu > 80 ? 'text-red-500' : agent.cpu > 50 ? 'text-amber-500' : 'text-emerald-500'}
          />
          <StatRow
            label={t('agentMemory')}
            value={`${agent.memory.toFixed(1)}%`}
            icon={HardDrive}
            color={agent.memory > 80 ? 'text-red-500' : agent.memory > 50 ? 'text-amber-500' : 'text-emerald-500'}
          />
          <StatRow
            label={t('agentErrors')}
            value={agent.errors}
            icon={Activity}
            color={agent.errors > 0 ? 'text-red-500' : 'text-emerald-500'}
          />
        </div>

        {/* Network */}
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-6 lg:col-span-2">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white uppercase tracking-wider mb-4">
            {t('agentNetwork')}
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="flex items-center gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-lg">
              <Wifi className="w-5 h-5 text-emerald-500" aria-hidden="true" />
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">{t('agentTrafficIn')}</p>
                <p className="text-lg font-bold text-slate-900 dark:text-white tabular-nums">
                  {formatTraffic(agent.traffic.in)}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-lg">
              <Wifi className="w-5 h-5 text-blue-500" aria-hidden="true" />
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">{t('agentTrafficOut')}</p>
                <p className="text-lg font-bold text-slate-900 dark:text-white tabular-nums">
                  {formatTraffic(agent.traffic.out)}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
