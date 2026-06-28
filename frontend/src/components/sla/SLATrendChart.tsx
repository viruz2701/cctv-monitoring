import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ResponsiveLine } from '@nivo/line';
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
    if (!filteredData.length) return [];
    return [
      {
        id: 'compliance',
        data: filteredData.map((d) => ({
          x: new Date(d.date).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
          }),
          y: Math.round(d.compliance * 10) / 10,
        })),
      },
    ];
  }, [filteredData]);

  // Compute min for Y-axis domain
  const minCompliance = useMemo(() => {
    if (!filteredData.length) return 0;
    const min = Math.min(...filteredData.map((d) => d.compliance));
    return Math.max(0, Math.floor(min / 10) * 10 - 5);
  }, [filteredData]);

  // Average compliance for display
  const avgCompliance = useMemo(() => {
    if (!filteredData.length) return 0;
    const sum = filteredData.reduce((s, d) => s + d.compliance, 0);
    return (sum / filteredData.length).toFixed(1);
  }, [filteredData]);

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
          {filteredData.length > 0 && (
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

      {chartData.length === 0 || chartData[0].data.length === 0 ? (
        <div className="flex items-center justify-center h-64 text-sm text-slate-500 dark:text-slate-400">
          {t('no_data') || 'No data available'}
        </div>
      ) : (
        <div className="h-72" style={{ position: 'relative' }}>
          <ResponsiveLine
            data={chartData}
            margin={{ top: 20, right: 80, bottom: 30, left: 50 }}
            xScale={{ type: 'point' }}
            yScale={{
              type: 'linear',
              min: minCompliance,
              max: 100,
            }}
            curve="monotoneX"
            lineWidth={2.5}
            colors={['#2563eb']}
            enablePoints={true}
            pointSize={6}
            pointColor="#2563eb"
            pointBorderWidth={0}
            enablePointLabel={false}
            enableGridX={false}
            enableGridY={true}
            gridYValues={[95]}
            axisBottom={{
              tickSize: 5,
              tickPadding: 5,
              tickRotation: 0,
              legend: undefined,
            }}
            axisLeft={{
              tickSize: 5,
              tickPadding: 5,
              tickRotation: 0,
              format: (v: number) => `${v}%`,
            }}
            theme={{
              axis: {
                ticks: {
                  text: { fontSize: 11, fill: '#94a3b8' },
                },
                domain: {
                  line: { stroke: '#e2e8f0', strokeWidth: 1 },
                },
              },
              grid: {
                line: {
                  stroke: '#e2e8f0',
                  strokeDasharray: '3 3',
                  strokeWidth: 1,
                },
              },
              crosshair: {
                line: { stroke: '#2563eb', strokeWidth: 1, strokeOpacity: 0.35 },
              },
            }}
            enableSlices="x"
            sliceTooltip={({ slice }) => {
              if (!slice.points.length) return null;
              const point = slice.points[0];
              return (
                <div
                  style={{
                    background: 'rgba(255,255,255,0.95)',
                    border: '1px solid #e2e8f0',
                    borderRadius: '8px',
                    padding: '6px 10px',
                    fontSize: '13px',
                    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
                  }}
                >
                  <div style={{ fontWeight: 600, color: '#1e293b', marginBottom: 2 }}>
                    {String(point.data.x)}
                  </div>
                  <div style={{ color: '#2563eb' }}>
                    {t('compliance') || 'Compliance'}: {Number(point.data.y).toFixed(1)}%
                  </div>
                </div>
              );
            }}
            markers={[
              {
                axis: 'y',
                value: 95,
                lineStyle: {
                  stroke: '#16a34a',
                  strokeDasharray: '6 4',
                  strokeWidth: 2,
                },
                legend: `${t('target') || 'Target'} 95%`,
                legendOrientation: 'horizontal',
                legendPosition: 'right',
                textStyle: {
                  fontSize: 11,
                  fill: '#16a34a',
                },
              },
            ]}
          />
        </div>
      )}
    </Card>
  );
}
