// ═══════════════════════════════════════════════════════════════════════
// AgentDashboard — мониторинг edge-агентов
// EDGE-11: P1-EDGE Agent Monitoring Dashboard
//
// Compliance:
// - WCAG 2.1 AA (aria, contrast, keyboard navigation)
// - Dark mode + Light mode (useTheme)
// - i18n (useTranslation)
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useAgentStore, useAgentList, useAgentStats, useAgentLoading, useAgentError } from '../store/agentStore';
import { AgentStatsCard } from '../components/agents/AgentStatsCard';
import { AgentTable } from '../components/agents/AgentTable';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { RefreshCw } from '../components/ui/Icons';

/**
 * AgentDashboard — главная страница мониторинга edge-агентов.
 *
 * Особенности:
 * - Автообновление каждые 30 секунд
 * - Inline alerts при проблемах
 * - Статистика в карточках
 * - Сортируемая таблица
 */
export function AgentDashboard() {
  const { t } = useTranslation();

  const agents = useAgentList();
  const stats = useAgentStats();
  const loading = useAgentLoading();
  const error = useAgentError();
  const fetchAgents = useAgentStore((s) => s.fetchAgents);
  const sendCommand = useAgentStore((s) => s.sendCommand);
  const deleteAgent = useAgentStore((s) => s.deleteAgent);
  const clearError = useAgentStore((s) => s.clearError);

  // ── Initial fetch + auto-refresh ──────────────────────────────────
  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  useEffect(() => {
    const interval = setInterval(() => {
      fetchAgents();
    }, 30_000);
    return () => clearInterval(interval);
  }, [fetchAgents]);

  // ── Handlers ──────────────────────────────────────────────────────
  const handleRefresh = useCallback(() => {
    fetchAgents();
  }, [fetchAgents]);

  const handleSendCommand = useCallback((agentId: string) => {
    const command = prompt(t('agentCommandPrompt'));
    if (command) {
      sendCommand(agentId, command).then((success) => {
        if (success) {
          fetchAgents();
        }
      });
    }
  }, [sendCommand, fetchAgents, t]);

  const handleDeleteAgent = useCallback((agentId: string) => {
    if (window.confirm(t('agentDeleteConfirm'))) {
      deleteAgent(agentId);
    }
  }, [deleteAgent, t]);

  // ── Offline threshold alert ───────────────────────────────────────
  const showOfflineAlert = stats.offline > 0;

  return (
    <div className="space-y-6" role="main" aria-label={t('agentDashboardTitle')}>
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('agentDashboardTitle')}
          </h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {t('agentDashboardSubtitle')}
          </p>
        </div>
        <Button
          variant="primary"
          onClick={handleRefresh}
          disabled={loading}
          aria-label={t('agentRefresh')}
        >
          <RefreshCw className={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          {t('agentRefresh')}
        </Button>
      </div>

      {/* Offline alert */}
      {showOfflineAlert && (
        <Alert
          variant="warning"
          title={t('agentOfflineAlertTitle')}
          assertive
        >
          {t('agentOfflineAlertBody', { count: stats.offline })}
        </Alert>
      )}

      {/* Error alert */}
      {error && (
        <Alert
          variant="error"
          title={t('agentErrorTitle')}
          onClose={clearError}
          assertive
        >
          {error}
        </Alert>
      )}

      {/* Stats cards */}
      <AgentStatsCard stats={stats} />

      {/* Agent table */}
      <div>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
          {t('agentListTitle')}
        </h2>
        <AgentTable
          agents={agents}
          loading={loading}
          onSendCommand={handleSendCommand}
          onDeleteAgent={handleDeleteAgent}
        />
      </div>

      {/* Keyboard hint */}
      <p className="text-xs text-slate-400 dark:text-slate-500 text-center">
        {t('agentKeyboardHint')}
      </p>
    </div>
  );
}
