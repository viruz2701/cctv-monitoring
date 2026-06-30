// ═══════════════════════════════════════════════════════════════════════
// AgentStatsCard — карточка статистики для Agent Dashboard
// EDGE-11: Agent Monitoring Dashboard
//
// Использует существующий StatsCard компонент из components/ui/StatsCard
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { StatsCard } from '../ui/StatsCard';
import {
  Server,
  Wifi,
  WifiOff,
  AlertTriangle,
  type LucideIcon,
} from '../ui/Icons';
import type { AgentStats } from '../../types/agent';

interface AgentStatsCardProps {
  stats: AgentStats;
}

/** Маппинг метрик на конфигурацию карточек */
const cardConfig: Record<
  keyof AgentStats,
  { titleKey: string; icon: LucideIcon; iconColor: string; iconBgColor: string }
> = {
  total: {
    titleKey: 'agentStatsTotal',
    icon: Server,
    iconColor: 'text-blue-600',
    iconBgColor: 'bg-blue-50 dark:bg-blue-900/30',
  },
  online: {
    titleKey: 'agentStatsOnline',
    icon: Wifi,
    iconColor: 'text-emerald-600',
    iconBgColor: 'bg-emerald-50 dark:bg-emerald-900/30',
  },
  offline: {
    titleKey: 'agentStatsOffline',
    icon: WifiOff,
    iconColor: 'text-red-600',
    iconBgColor: 'bg-red-50 dark:bg-red-900/30',
  },
  errors: {
    titleKey: 'agentStatsErrors',
    icon: AlertTriangle,
    iconColor: 'text-amber-600',
    iconBgColor: 'bg-amber-50 dark:bg-amber-900/30',
  },
};

/**
 * AgentStatsCard — отображает одну карточку статистики агентов.
 * Обёртка над StatsCard с предустановленными иконками и цветами.
 */
export function AgentStatsCard({ stats }: AgentStatsCardProps) {
  const { t } = useTranslation();

  return (
    <div
      className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4"
      role="region"
      aria-label={t('agentStatsRegionLabel')}
    >
      {(Object.keys(cardConfig) as Array<keyof AgentStats>).map((key) => {
        const config = cardConfig[key];
        return (
          <StatsCard
            key={key}
            title={t(config.titleKey)}
            value={stats[key]}
            icon={config.icon}
            iconColor={config.iconColor}
            iconBgColor={config.iconBgColor}
          />
        );
      })}
    </div>
  );
}
