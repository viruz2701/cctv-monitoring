import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, DataGrid, Badge, StatsCard } from '../components/ui';
import {
  DollarSign, Briefcase, Wrench, Truck, TrendingUp,
  PieChart, BarChart3,
} from 'lucide-react';

// ── Types ────────────────────────────────────────────────────────────

interface WorkOrderCostSummary {
  total_work_orders: number;
  total_labor_cost: number;
  total_parts_cost: number;
  total_additional_cost: number;
  total_cost: number;
  avg_cost_per_order: number;
  currency: string;
}

interface WorkOrderCostBreakdown {
  category: string;
  amount: number;
  count: number;
  percent: number;
}

interface CostResponse {
  summary: WorkOrderCostSummary;
  breakdown: WorkOrderCostBreakdown[];
}

// ── Page Component ───────────────────────────────────────────────────

export const TotalCostDashboard: React.FC = () => {
  const { t } = useTranslation();
  const [data, setData] = useState<CostResponse | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetch = async () => {
      setLoading(true);
      try {
        const result = await request<CostResponse>('/analytics/wo-costs');
        setData(result);
      } catch (err) {
        console.error('Failed to fetch cost data', err);
      } finally {
        setLoading(false);
      }
    };
    fetch();
  }, []);

  const summary = data?.summary;
  const breakdown = data?.breakdown || [];

  const categoryConfig: Record<string, { label: string; icon: React.FC<{ size?: number; className?: string }>; color: string; bg: string }> = {
    labor: {
      label: t('labor_cost'),
      icon: Briefcase,
      color: 'text-blue-600',
      bg: 'bg-blue-50 dark:bg-blue-900/30',
    },
    parts: {
      label: t('parts_cost'),
      icon: Wrench,
      color: 'text-emerald-600',
      bg: 'bg-emerald-50 dark:bg-emerald-900/30',
    },
    additional: {
      label: t('additional_cost'),
      icon: Truck,
      color: 'text-amber-600',
      bg: 'bg-amber-50 dark:bg-amber-900/30',
    },
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
        <DollarSign className="w-6 h-6" />
        {t('total_cost_dashboard') || 'Total Cost Dashboard'}
      </h1>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('total_cost') || 'Total Cost'}
          value={summary ? `${summary.total_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={DollarSign}
          iconBgColor="bg-slate-50 dark:bg-slate-900/30"
          iconColor="text-slate-600"
        />
        <StatsCard
          title={t('labor_cost') || 'Labor Cost'}
          value={summary ? `${summary.total_labor_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={Briefcase}
          iconBgColor="bg-blue-50 dark:bg-blue-900/30"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('parts_cost') || 'Parts Cost'}
          value={summary ? `${summary.total_parts_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={Wrench}
          iconBgColor="bg-emerald-50 dark:bg-emerald-900/30"
          iconColor="text-emerald-600"
        />
        <StatsCard
          title={t('additional_cost') || 'Additional Cost'}
          value={summary ? `${summary.total_additional_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={Truck}
          iconBgColor="bg-amber-50 dark:bg-amber-900/30"
          iconColor="text-amber-600"
        />
      </div>

      {/* Secondary Metrics */}
      {summary && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <div className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-purple-50 dark:bg-purple-900/30 rounded-xl">
                  <BarChart3 className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                </div>
                <div>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    {t('total_work_orders') || 'Total Work Orders'}
                  </p>
                  <p className="text-xl font-bold text-slate-900 dark:text-white">
                    {summary.total_work_orders}
                  </p>
                </div>
              </div>
            </div>
          </Card>
          <Card>
            <div className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-indigo-50 dark:bg-indigo-900/30 rounded-xl">
                  <TrendingUp className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                </div>
                <div>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    {t('avg_cost_per_order') || 'Avg Cost/Order'}
                  </p>
                  <p className="text-xl font-bold text-slate-900 dark:text-white">
                    ${summary.avg_cost_per_order.toLocaleString('en-US', { minimumFractionDigits: 2 })}
                  </p>
                </div>
              </div>
            </div>
          </Card>
          <Card>
            <div className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-rose-50 dark:bg-rose-900/30 rounded-xl">
                  <PieChart className="w-5 h-5 text-rose-600 dark:text-rose-400" />
                </div>
                <div>
                  <p className="text-xs text-slate-500 dark:text-slate-400">{t('currency') || 'Currency'}</p>
                  <p className="text-xl font-bold text-slate-900 dark:text-white">{summary.currency}</p>
                </div>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Cost Breakdown Table */}
      <Card>
        <div className="p-5">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
            <PieChart className="w-4 h-4" />
            {t('cost_breakdown') || 'Cost Breakdown by Category'}
          </h3>
          <DataGrid
            data={breakdown}
            columns={[
              {
                key: 'category',
                header: t('category') || 'Category',
                sortable: true,
                render: (item: WorkOrderCostBreakdown) => {
                  const cfg = categoryConfig[item.category] || categoryConfig.additional;
                  const Icon = cfg.icon;
                  return (
                    <div className="flex items-center gap-2">
                      <div className={`p-1.5 rounded-lg ${cfg.bg}`}>
                        <Icon className={`w-4 h-4 ${cfg.color}`} />
                      </div>
                      <span className="font-medium">{cfg.label}</span>
                    </div>
                  );
                },
              },
              {
                key: 'amount',
                header: t('amount') || 'Amount',
                sortable: true,
                render: (item: WorkOrderCostBreakdown) => (
                  <span className="font-mono font-medium">
                    ${item.amount.toLocaleString('en-US', { minimumFractionDigits: 2 })}
                  </span>
                ),
              },
              {
                key: 'count',
                header: t('entries') || 'Entries',
                sortable: true,
                render: (item: WorkOrderCostBreakdown) => (
                  <Badge variant="info">{item.count}</Badge>
                ),
              },
              {
                key: 'percent',
                header: '%',
                sortable: true,
                render: (item: WorkOrderCostBreakdown) => (
                  <div className="flex items-center gap-2">
                    <div className="w-24 bg-slate-200 dark:bg-slate-700 rounded-full h-2">
                      <div
                        className={`h-2 rounded-full ${
                          item.category === 'labor' ? 'bg-blue-500' :
                          item.category === 'parts' ? 'bg-emerald-500' :
                          'bg-amber-500'
                        }`}
                        style={{ width: `${Math.min(item.percent, 100)}%` }}
                      />
                    </div>
                    <span className="text-xs font-mono">{item.percent.toFixed(1)}%</span>
                  </div>
                ),
              },
            ]}
            keyExtractor={(item) => item.category}
            emptyMessage={t('no_cost_data') || 'No cost data available'}
            variant="striped"
            defaultDensity="compact"
            exportFilename="cost-breakdown.csv"
          />
        </div>
      </Card>
    </div>
  );
};
