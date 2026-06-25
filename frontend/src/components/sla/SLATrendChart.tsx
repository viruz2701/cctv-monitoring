import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import { Card } from '../ui/Card';
import { Skeleton } from '../ui/Skeleton';

// P0-4.4: Line chart SLA trend за 30/90/180 дней с target line 95%

interface TrendPoint {
  date: string; // ISO date
  compliance: number;
}

interface SLATrendChartProps {
  data: TrendPoint[];
  loading?: boolean;
}

type Period = 30 | 90 | 180;

const PERIOD_OPTIONS: { value: Period; label: string }[] = [
  { value: 30, label: '30d' },
  { value: 90, label: '90d' },
  { value: 180, label: '180d' },
];

export function SLATrendChart({ data, loading = false }: SLATrendChartProps) {
  const { t } = useTranslation();
  const [period, setPeriod] = useState<Period>(30);

  const filteredData = useMemo(() => {
    if (!data.length) return [];
    const cutoff = new Date();
    cutoff.setDate(cutoff.getDate() - period);
    return data
      .filter((d) => new Date(d.date) >= cutoff)
      .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());
  }, [data, period]);

  const chartData = useMemo(() => {
    return filteredData.map((d) => ({
      date: new Date(d.date).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      }),
      compliance: Math.round(d.compliance * 10) / 10,
    }));
  }, [filteredData]);

  // Compute min for Y-axis domain
  const minCompliance = useMemo(() => {
    if (!chartData.length) return 0;
    const min = Math.min(...chartData.map((d) => d.compliance));
    return Math.max(0, Math.floor(min / 10) * 10 - 5);
  }, [chartData]);

  // Average compliance for display
  const avgCompliance = useMemo(() => {
    if (!chartData.length) return 0;
    const sum = chartData.reduce((s, d) => s + d.compliance, 0);
    return (sum / chartData.length).toFixed(1);
  }, [chartData]);

  if (loading) {
    return (
      <Card className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-8 w-32" />
        </div>
        <Skeleton className="h-64 w-full" />
      </Card>
    );
  }

  return (
    <Card className="mb-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
            {t('sla_trend') || 'SLA Trend'}
          </h3>
          {chartData.length > 0 && (
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              {t('avg_compliance') || 'Avg compliance'}: {avgCompliance}%
            </p>
          )}
        </div>
        <div className="flex gap-1 bg-slate-100 dark:bg-slate-700 rounded-lg p-0.5">
          {PERIOD_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setPeriod(opt.value)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                period === opt.value
                  ? 'bg-white dark:bg-slate-600 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {chartData.length === 0 ? (
        <div className="flex items-center justify-center h-64 text-sm text-slate-500 dark:text-slate-400">
          {t('no_data') || 'No data available'}
        </div>
      ) : (
        <div className="h-72">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart
              data={chartData}
              margin={{ top: 8, right: 8, left: -8, bottom: 8 }}
            >
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="#e2e8f0"
                className="dark:opacity-30"
              />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 11, fill: '#94a3b8' }}
                tickLine={false}
                axisLine={{ stroke: '#e2e8f0' }}
                interval="preserveStartEnd"
              />
              <YAxis
                domain={[minCompliance, 100]}
                tick={{ fontSize: 11, fill: '#94a3b8' }}
                tickLine={false}
                axisLine={false}
                tickFormatter={(v: number) => `${v}%`}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'rgba(255,255,255,0.95)',
                  border: '1px solid #e2e8f0',
                  borderRadius: '8px',
                  fontSize: '13px',
                }}
                formatter={(value) => {
                  const v = typeof value === 'number' ? value : 0;
                  return [`${v}%`, t('compliance') || 'Compliance'];
                }}
                labelStyle={{ fontWeight: 600, color: '#1e293b' }}
              />
              {/* Target line at 95% */}
              <ReferenceLine
                y={95}
                stroke="#16a34a"
                strokeDasharray="6 4"
                strokeWidth={2}
                label={{
                  value: `${t('target') || 'Target'} 95%`,
                  position: 'right',
                  fontSize: 11,
                  fill: '#16a34a',
                }}
              />
              <Line
                type="monotone"
                dataKey="compliance"
                stroke="#2563eb"
                strokeWidth={2.5}
                dot={{ r: 3, fill: '#2563eb', strokeWidth: 0 }}
                activeDot={{ r: 5, fill: '#2563eb', strokeWidth: 2, stroke: '#fff' }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </Card>
  );
}
