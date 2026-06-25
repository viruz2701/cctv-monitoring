import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, DataGrid, Badge, StatsCard, EmptyState, SkeletonStatsCard, SkeletonChart, SkeletonTable } from '../components/ui';
import {
  Shield,
  AlertTriangle,
  DollarSign,
  TrendingUp,
  RefreshCw,
  PieChart,
  Activity,
} from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface ComplianceRisk {
  device_id: string;
  device_name?: string;
  device_type: string;
  site_id?: string;
  site_name?: string;
  total_downtime_min: number;
  downtime_hours: number;
  hourly_fine: number;
  total_exposure: number;
  risk_level: 'low' | 'medium' | 'high' | 'critical';
  updated_at: string;
}

interface ComplianceSummary {
  total_exposure: number;
  at_risk_devices: number;
  compliant_devices: number;
  total_devices: number;
  top_risks: ComplianceRisk[];
  risk_breakdown: Record<string, number>;
}

interface FineTable {
  [deviceType: string]: number;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const RISK_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  low: { bg: 'bg-green-100 dark:bg-green-900/20', text: 'text-green-700 dark:text-green-400', border: 'border-green-300 dark:border-green-700' },
  medium: { bg: 'bg-yellow-100 dark:bg-yellow-900/20', text: 'text-yellow-700 dark:text-yellow-400', border: 'border-yellow-300 dark:border-yellow-700' },
  high: { bg: 'bg-orange-100 dark:bg-orange-900/20', text: 'text-orange-700 dark:text-orange-400', border: 'border-orange-300 dark:border-orange-700' },
  critical: { bg: 'bg-red-100 dark:bg-red-900/20', text: 'text-red-700 dark:text-red-400', border: 'border-red-300 dark:border-red-700' },
};


// ═══════════════════════════════════════════════════════════════════════
// RiskBadge Component
// ═══════════════════════════════════════════════════════════════════════

function RiskBadge({ level, t }: { level: string; t: (key: string, options?: Record<string, unknown>) => string }) {
  const variantMap: Record<string, 'success' | 'warning' | 'danger' | 'neutral'> = {
    low: 'success',
    medium: 'warning',
    high: 'danger',
    critical: 'danger',
  };
  return <Badge variant={variantMap[level] || 'neutral'}>{t(`risk_level_${level}`, { defaultValue: level.toUpperCase() })}</Badge>;
}

// ═══════════════════════════════════════════════════════════════════════
// Simple PieChart SVG Component
// ═══════════════════════════════════════════════════════════════════════

function RiskPieChart({ breakdown, t }: { breakdown: Record<string, number> | null | undefined; t: (key: string, options?: Record<string, unknown>) => string }) {
  if (!breakdown) return <div className="text-slate-400 text-sm text-center py-4">{t('no_data_short')}</div>;
  const total = Object.values(breakdown).reduce((a, b) => a + b, 0);
  if (total === 0) return <div className="text-slate-400 text-sm text-center py-4">{t('no_data_short')}</div>;

  const COLORS: Record<string, string> = {
    low: '#16a34a',
    medium: '#d97706',
    high: '#ea580c',
    critical: '#dc2626',
  };

  const segments = Object.entries(breakdown).map(([level, count]) => ({
    level,
    count,
    percent: (count / total) * 100,
    color: COLORS[level] || '#94a3b8',
  }));

  // Generate SVG pie chart
  let cumulativePercent = 0;
  const arcs = segments.map((seg) => {
    const startPercent = cumulativePercent;
    const endPercent = cumulativePercent + seg.percent;
    cumulativePercent = endPercent;

    const startAngle = (startPercent / 100) * 360 - 90;
    const endAngle = (endPercent / 100) * 360 - 90;

    const startRad = (startAngle * Math.PI) / 180;
    const endRad = (endAngle * Math.PI) / 180;

    const r = 80;
    const cx = 100;
    const cy = 100;

    const x1 = cx + r * Math.cos(startRad);
    const y1 = cy + r * Math.sin(startRad);
    const x2 = cx + r * Math.cos(endRad);
    const y2 = cy + r * Math.sin(endRad);

    const largeArc = seg.percent > 50 ? 1 : 0;

    const d = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`;

    return { d, color: seg.color, label: seg.level, percent: seg.percent };
  });

  return (
    <div className="flex flex-col items-center gap-4">
      <svg viewBox="0 0 200 200" className="w-40 h-40">
        {arcs.map((arc, i) => (
          <path key={i} d={arc.d} fill={arc.color} stroke="white" strokeWidth="2" />
        ))}
        <circle cx="100" cy="100" r="40" fill="white" className="dark:fill-slate-800" />
        <text x="100" y="100" textAnchor="middle" dominantBaseline="middle" className="text-lg font-bold fill-slate-900 dark:fill-white" fontSize="24">
          {total}
        </text>
        <text x="100" y="118" textAnchor="middle" dominantBaseline="middle" className="text-xs fill-slate-500" fontSize="10">
          {t('pie_devices')}
        </text>
      </svg>
      <div className="flex flex-wrap gap-3 justify-center">
        {segments.map((seg) => (
          <div key={seg.level} className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full" style={{ backgroundColor: seg.color }} />
            <span className="text-xs text-slate-600 dark:text-slate-400 capitalize">{seg.level}</span>
            <span className="text-xs font-medium text-slate-900 dark:text-white">{seg.count}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Exposure by DeviceType Chart
// ═══════════════════════════════════════════════════════════════════════

function ExposureByTypeChart({ risks, t }: { risks: ComplianceRisk[]; t: (key: string, options?: Record<string, unknown>) => string }) {
  if (risks.length === 0) return <div className="text-slate-400 text-sm text-center py-4">{t('no_data_short')}</div>;

  // Aggregate by device type
  const byType: Record<string, number> = {};
  for (const r of risks) {
    const label = t(`device_type_${r.device_type}`, { defaultValue: r.device_type });
    byType[label] = (byType[label] || 0) + r.total_exposure;
  }

  const entries = Object.entries(byType)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 8);

  const maxExposure = Math.max(...entries.map(([, v]) => v), 1);

  return (
    <div className="space-y-2">
      {entries.map(([type, exposure]) => (
        <div key={type} className="flex items-center gap-3">
          <span className="text-xs text-slate-600 dark:text-slate-400 w-24 truncate shrink-0">{type}</span>
          <div className="flex-1 h-5 bg-slate-100 dark:bg-slate-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all duration-500"
              style={{ width: `${(exposure / maxExposure) * 100}%` }}
            />
          </div>
          <span className="text-xs font-medium text-slate-700 dark:text-slate-300 w-20 text-right shrink-0">
            ${exposure.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
          </span>
        </div>
      ))}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Fine Rates Table
// ═══════════════════════════════════════════════════════════════════════

function FineRatesTable({ fines, t }: { fines: FineTable; t: (key: string, options?: Record<string, unknown>) => string }) {
  const entries = Object.entries(fines).sort(([, a], [, b]) => b - a);

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 dark:border-slate-700">
            <th className="text-left py-2 px-3 text-slate-500 dark:text-slate-400 font-medium">{t('fine_device_type')}</th>
            <th className="text-right py-2 px-3 text-slate-500 dark:text-slate-400 font-medium">{t('fine_hourly_rate')}</th>
          </tr>
        </thead>
        <tbody>
          {entries.map(([type, fine]) => (
            <tr key={type} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
              <td className="py-2 px-3 text-slate-700 dark:text-slate-300 capitalize">
                {t(`device_type_${type}`, { defaultValue: type.replace(/_/g, ' ') })}
              </td>
              <td className="py-2 px-3 text-right font-medium text-slate-900 dark:text-white">
                ${fine.toFixed(2)}/h
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export const ComplianceShield: React.FC = () => {
  const { t } = useTranslation();
  const [summary, setSummary] = useState<ComplianceSummary | null>(null);
  const [risks, setRisks] = useState<ComplianceRisk[]>([]);
  const [fines, setFines] = useState<FineTable>({});
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [summaryData, risksData, finesData] = await Promise.all([
        request<ComplianceSummary>('/compliance/summary'),
        request<ComplianceRisk[]>('/compliance/risks'),
        request<FineTable>('/compliance/fines'),
      ]);
      setSummary(summaryData);
      setRisks(risksData || []);
      setFines(finesData || {});
    } catch (err) {
      console.error('Failed to fetch compliance data', err);
      setError(t('compliance_load_error'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await request('/compliance/refresh', { method: 'POST' });
      await fetchData();
    } catch (err) {
      console.error('Failed to refresh compliance data', err);
    } finally {
      setRefreshing(false);
    }
  };

  const riskScore = summary
    ? Math.round(
        ((summary.total_devices - summary.at_risk_devices) / Math.max(summary.total_devices, 1)) * 100
      )
    : 0;

  const trendUp = { value: 100, label: t('target_label'), direction: 'up' as const };
  const trendDown = { value: 0, label: t('at_risk_label'), direction: 'down' as const };

  const columns: { key: string; header: string; render: (item: ComplianceRisk) => React.ReactNode }[] = [
    {
      key: 'device_id',
      header: t('col_device'),
      render: (row: ComplianceRisk) => (
        <div>
          <div className="font-medium text-slate-900 dark:text-white">{row.device_name || row.device_id}</div>
          <div className="text-xs text-slate-500">{row.device_type}</div>
        </div>
      ),
    },
    {
      key: 'device_type',
      header: t('col_type'),
      render: (row: ComplianceRisk) => (
        <Badge variant="primary">{t('device_type_' + row.device_type, { defaultValue: row.device_type })}</Badge>
      ),
    },
    {
      key: 'total_downtime_min',
      header: t('col_downtime'),
      render: (row: ComplianceRisk) => (
        <span className="text-sm text-slate-700 dark:text-slate-300">
          {row.downtime_hours >= 1
            ? `${row.downtime_hours.toFixed(1)}h`
            : `${row.total_downtime_min}m`}
        </span>
      ),
    },
    {
      key: 'hourly_fine',
      header: t('col_fine_hourly'),
      render: (row: ComplianceRisk) => (
        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
          ${row.hourly_fine.toFixed(2)}
        </span>
      ),
    },
    {
      key: 'total_exposure',
      header: t('col_exposure'),
      render: (row: ComplianceRisk) => (
        <span className={`text-sm font-semibold ${
          row.total_exposure >= 5000
            ? 'text-red-600 dark:text-red-400'
            : row.total_exposure >= 1000
            ? 'text-orange-600 dark:text-orange-400'
            : 'text-slate-700 dark:text-slate-300'
        }`}>
          ${row.total_exposure.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
        </span>
      ),
    },
    {
      key: 'risk_level',
      header: t('col_risk'),
      render: (row: ComplianceRisk) => <RiskBadge level={row.risk_level} t={t} />,
    },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Shield className="w-7 h-7 text-blue-500" />
            {t('compliance_shield') || 'Compliance & Fines Shield'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('compliance_shield_desc') || 'Downtime-to-monetary-risk conversion for CCTV cameras'}
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={refreshing}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg transition-colors text-sm font-medium"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          {refreshing ? t('compliance_refreshing') : t('compliance_refresh')}
        </button>
      </div>

      {/* Error State */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-red-500 shrink-0" />
          <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Loading State */}
      {loading && !summary && (
        <div className="space-y-6">
          {/* Skeleton KPI Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <SkeletonStatsCard />
            <SkeletonStatsCard />
            <SkeletonStatsCard />
            <SkeletonStatsCard />
          </div>

          {/* Skeleton Charts */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <SkeletonChart />
            <SkeletonChart />
            <SkeletonChart />
          </div>

          {/* Skeleton Table */}
          <SkeletonTable rows={5} columns={5} />
        </div>
      )}

      {/* KPI Cards */}
      {summary && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatsCard
            title={t('total_exposure_kpi')}
            value={`$${summary.total_exposure.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`}
            icon={DollarSign}
            trend={summary.at_risk_devices > 0 ? { value: summary.at_risk_devices, label: t('at_risk_label'), direction: 'down' } : trendUp}
          />
          <StatsCard
            title={t('at_risk_devices')}
            value={summary.at_risk_devices.toString()}
            subtitle={t('out_of_total', { total: summary.total_devices })}
            icon={AlertTriangle}
            iconColor="text-red-600"
            iconBgColor="bg-red-50"
            trend={summary.at_risk_devices > 0 ? { value: Math.round((summary.at_risk_devices / Math.max(summary.total_devices, 1)) * 100), label: t('of_total_label'), direction: 'down' } : trendUp}
          />
          <StatsCard
            title={t('compliant_devices_kpi')}
            value={summary.compliant_devices.toString()}
            subtitle={t('percent_compliant', { percent: summary.total_devices > 0 ? Math.round((summary.compliant_devices / summary.total_devices) * 100) : 0 })}
            icon={Shield}
            iconColor="text-emerald-600"
            iconBgColor="bg-emerald-50"
            trend={{ value: Math.round((summary.compliant_devices / Math.max(summary.total_devices, 1)) * 100), label: t('compliant_label'), direction: 'up' }}
          />
          <StatsCard
            title={t('risk_score_kpi')}
            value={`${riskScore}%`}
            subtitle={riskScore >= 80 ? t('good_label') : riskScore >= 50 ? t('fair_label') : t('poor_label')}
            icon={TrendingUp}
            trend={riskScore >= 80 ? { value: riskScore, label: t('score_label'), direction: 'up' } : { value: riskScore, label: t('score_label'), direction: 'down' }}
          />
        </div>
      )}

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Risk Breakdown Pie */}
        <Card className="lg:col-span-1">
          <div className="p-4">
            <div className="flex items-center gap-2 mb-4">
              <PieChart className="w-5 h-5 text-slate-500" />
              <h3 className="font-semibold text-slate-900 dark:text-white">{t('risk_breakdown')}</h3>
            </div>
            {summary?.risk_breakdown ? (
              <RiskPieChart breakdown={summary.risk_breakdown} t={t} />
            ) : (
              <EmptyState icon={<PieChart className="w-8 h-8" />} title={t('no_data_short')} description={t('no_risk_data')} size="sm" />
            )}
          </div>
        </Card>

        {/* Exposure by Type */}
        <Card className="lg:col-span-1">
          <div className="p-4">
            <div className="flex items-center gap-2 mb-4">
              <BarChart3Icon className="w-5 h-5 text-slate-500" />
              <h3 className="font-semibold text-slate-900 dark:text-white">{t('exposure_by_type')}</h3>
            </div>
            <ExposureByTypeChart risks={risks} t={t} />
          </div>
        </Card>

        {/* Fine Rates */}
        <Card className="lg:col-span-1">
          <div className="p-4">
            <div className="flex items-center gap-2 mb-4">
              <DollarSign className="w-5 h-5 text-slate-500" />
              <h3 className="font-semibold text-slate-900 dark:text-white">{t('fine_rates')}</h3>
            </div>
            {Object.keys(fines).length > 0 ? (
              <FineRatesTable fines={fines} t={t} />
            ) : (
              <EmptyState icon={<DollarSign className="w-8 h-8" />} title={t('no_fines_data')} description={t('fines_not_configured')} size="sm" />
            )}
          </div>
        </Card>
      </div>

      {/* Risks Table */}
      <Card>
        <div className="p-4 border-b border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-slate-500" />
            <h3 className="font-semibold text-slate-900 dark:text-white">{t('device_risk_details')}</h3>
          </div>
        </div>
        {risks.length > 0 ? (
          <DataGrid
            data={risks}
            columns={columns}
            keyExtractor={(item: ComplianceRisk) => item.device_id}
            pageSize={20}
          />
        ) : (
          <div className="p-8">
            <EmptyState
              icon={<Shield className="w-12 h-12" />}
              title={t('no_compliance_risks')}
              description={t('all_compliant_desc')}
            />
          </div>
        )}
      </Card>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// BarChart3 Icon (lucide-react doesn't export it directly as used above)
// ═══════════════════════════════════════════════════════════════════════

function BarChart3Icon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <path d="M3 3v18h18" />
      <path d="M7 16v-3" />
      <path d="M12 16v-7" />
      <path d="M17 16v-5" />
    </svg>
  );
}

export default ComplianceShield;
