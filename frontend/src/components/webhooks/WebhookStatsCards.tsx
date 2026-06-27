// ═══════════════════════════════════════════════════════════════════════
// WebhookStatsCards — дашборд статистики вебхука (P2-3.1)
//
// Features:
//   - Total deliveries (24h/7d/30d)
//   - Success rate percentage
//   - Average latency
//   - Active/inactive status
//   - Auto-refresh every 30s via TanStack Query
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — async data fetching)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  Activity,
  CheckCircle,
  Clock,
  Power,
  PowerOff,
  RefreshCw,
} from 'lucide-react';
import { MiniStatsCard } from '../ui';
import { useWebhookStats } from '../../hooks/useWebhooks';

// ═══════════════════════════════════════════════════════════════════════
// Props
// ═══════════════════════════════════════════════════════════════════════

interface WebhookStatsCardsProps {
  webhookId: string | undefined;
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function WebhookStatsCards({ webhookId }: WebhookStatsCardsProps) {
  const { t } = useTranslation();
  const { data: stats, isLoading, isError } = useWebhookStats(webhookId);

  if (!webhookId) return null;

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="h-24 bg-slate-50 dark:bg-slate-800/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (isError || !stats) {
    return (
      <div className="flex items-center gap-2 px-4 py-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-xl">
        <RefreshCw className="w-4 h-4 text-amber-500" />
        <p className="text-xs text-amber-700 dark:text-amber-300">
          {t('stats_load_error') || 'Unable to load webhook statistics'}
        </p>
      </div>
    );
  }

  const successRateColor: 'green' | 'amber' | 'red' =
    stats.success_rate >= 0.95 ? 'green' : stats.success_rate >= 0.8 ? 'amber' : 'red';

  const latencyColor: 'green' | 'amber' | 'red' =
    stats.avg_latency_ms < 500 ? 'green' : stats.avg_latency_ms < 2000 ? 'amber' : 'red';

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
      <MiniStatsCard
        title={t('deliveries_24h') || 'Deliveries (24h)'}
        value={stats.total_deliveries_24h.toLocaleString()}
        icon={Activity}
        color="blue"
      />

      <MiniStatsCard
        title={t('success_rate') || 'Success Rate'}
        value={`${(stats.success_rate * 100).toFixed(1)}%`}
        icon={CheckCircle}
        color={successRateColor}
      />

      <MiniStatsCard
        title={t('avg_latency') || 'Avg Latency'}
        value={`${stats.avg_latency_ms}ms`}
        icon={Clock}
        color={latencyColor}
      />

      <MiniStatsCard
        title={t('status') || 'Status'}
        value={stats.active ? (t('active') || 'Active') : (t('inactive') || 'Inactive')}
        icon={stats.active ? Power : PowerOff}
        color={stats.active ? 'green' : 'red'}
      />
    </div>
  );
}
