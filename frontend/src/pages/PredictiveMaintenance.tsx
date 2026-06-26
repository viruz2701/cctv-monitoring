// ═══════════════════════════════════════════════════════════════════════
// PredictiveMaintenance — Dashboard предиктивного обслуживания.
//
// P2-1.3: Predictive Maintenance Dashboard
//   - KPI cards с at-risk count
//   - Risk distribution chart (placeholder)
//   - Failure by type breakdown (placeholder)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, TrendingUp, Activity, Calendar } from 'lucide-react';

const kpiCards = [
  {
    id: 'at-risk',
    label: 'at_risk_devices',
    value: '12',
    icon: AlertTriangle,
    color: 'text-amber-600',
    bg: 'bg-amber-50 dark:bg-amber-900/20',
    change: '+3',
    changeDirection: 'up' as const,
  },
  {
    id: 'predicted-failures',
    label: 'predicted_failures_30d',
    value: '8',
    icon: TrendingUp,
    color: 'text-red-600',
    bg: 'bg-red-50 dark:bg-red-900/20',
    change: '-2',
    changeDirection: 'down' as const,
  },
  {
    id: 'avg-reliability',
    label: 'avg_reliability_score',
    value: '94.2%',
    icon: Activity,
    color: 'text-emerald-600',
    bg: 'bg-emerald-50 dark:bg-emerald-900/20',
    change: '+0.5%',
    changeDirection: 'down' as const,
  },
  {
    id: 'next-maintenance',
    label: 'next_maintenance_due',
    value: '5',
    icon: Calendar,
    color: 'text-blue-600',
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    change: '2 overdue',
    changeDirection: 'up' as const,
  },
];

export function PredictiveMaintenance() {
  const { t } = useTranslation();

  return (
    <div className="p-4 md:p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          {t('predictive_maintenance')}
        </h1>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {t('predictive_maintenance_desc') || 'AI-powered failure prediction'}
        </p>
      </div>

      {/* KPI Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {kpiCards.map((kpi) => {
          const Icon = kpi.icon;
          return (
            <div
              key={kpi.id}
              className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4"
            >
              <div className="flex items-center justify-between mb-3">
                <div className={`p-2 rounded-lg ${kpi.bg}`}>
                  <Icon className={`w-5 h-5 ${kpi.color}`} />
                </div>
                <span
                  className={`text-xs font-medium ${
                    kpi.changeDirection === 'up'
                      ? 'text-red-500'
                      : 'text-emerald-500'
                  }`}
                >
                  {kpi.change}
                </span>
              </div>
              <p className="text-2xl font-bold text-slate-900 dark:text-white">
                {kpi.value}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                {t(kpi.label)}
              </p>
            </div>
          );
        })}
      </div>

      {/* Placeholder Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Risk Distribution Chart Placeholder */}
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-6">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            {t('risk_distribution') || 'Risk Distribution'}
          </h3>
          <div className="h-64 flex items-center justify-center bg-slate-50 dark:bg-slate-700/50 rounded-lg border-2 border-dashed border-slate-200 dark:border-slate-600">
            <p className="text-sm text-slate-400">
              {t('chart_coming_soon') || 'Chart — Coming Soon'}
            </p>
          </div>
        </div>

        {/* Failure by Type Chart Placeholder */}
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-6">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            {t('failure_by_type') || 'Failure by Type'}
          </h3>
          <div className="h-64 flex items-center justify-center bg-slate-50 dark:bg-slate-700/50 rounded-lg border-2 border-dashed border-slate-200 dark:border-slate-600">
            <p className="text-sm text-slate-400">
              {t('chart_coming_soon') || 'Chart — Coming Soon'}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
