import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../../services/api';
import { Card, Badge, StatsCard, EmptyState, SkeletonStatsCard, SkeletonTable } from '../../components/ui';
import {
  Shield,
  Globe,
  Download,
  FileText,
  FileCode,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  ExternalLink,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface RegionCompliance {
  region: string;
  status: 'compliant' | 'partial' | 'non_compliant' | 'not_assessed';
  score: number;
  devices: number;
  exposure: number;
  gaps: number;
  updated_at: string;
}

interface GapItem {
  id: string;
  category: string;
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: string;
  detected_at: string;
}

interface RiskSummary {
  total_exposure: number;
  at_risk_devices: number;
  compliant_devices: number;
  total_devices: number;
  severity_breakdown: Record<string, number>;
}

interface ComplianceDashboard {
  tenant_id: string;
  overall_status: 'compliant' | 'partial' | 'non_compliant' | 'not_assessed';
  overall_score: number;
  regions: RegionCompliance[];
  recent_gaps?: GapItem[];
  risk_summary: RiskSummary;
  generated_at: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const REGION_LABELS: Record<string, string> = {
  BY: 'Belarus (СТБ)',
  RU: 'Russia (ГОСТ/ФСТЭК)',
  EU: 'European Union (GDPR)',
  US: 'United States (NIST)',
  CN: 'China (MLPS 2.0)',
  INTL: 'International (ISO 27001)',
};

const REGION_FLAGS: Record<string, string> = {
  BY: '🇧🇾',
  RU: '🇷🇺',
  EU: '🇪🇺',
  US: '🇺🇸',
  CN: '🇨🇳',
  INTL: '🌐',
};

const STATUS_CONFIG: Record<string, { bg: string; text: string; border: string; icon: React.ElementType; label: string }> = {
  compliant: {
    bg: 'bg-emerald-50 dark:bg-emerald-900/20',
    text: 'text-emerald-700 dark:text-emerald-400',
    border: 'border-emerald-300 dark:border-emerald-700',
    icon: CheckCircle,
    label: 'Compliant',
  },
  partial: {
    bg: 'bg-amber-50 dark:bg-amber-900/20',
    text: 'text-amber-700 dark:text-amber-400',
    border: 'border-amber-300 dark:border-amber-700',
    icon: AlertTriangle,
    label: 'Partial',
  },
  non_compliant: {
    bg: 'bg-red-50 dark:bg-red-900/20',
    text: 'text-red-700 dark:text-red-400',
    border: 'border-red-300 dark:border-red-700',
    icon: XCircle,
    label: 'Non-Compliant',
  },
  not_assessed: {
    bg: 'bg-slate-50 dark:bg-slate-800/20',
    text: 'text-slate-500 dark:text-slate-400',
    border: 'border-slate-300 dark:border-slate-600',
    icon: Clock,
    label: 'Not Assessed',
  },
};

const SEVERITY_COLORS: Record<string, string> = {
  low: 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-300',
  medium: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  high: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  critical: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
};

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

function StatusBadge({ status }: { status: string }) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.not_assessed;
  const Icon = config.icon;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${config.bg} ${config.text} ${config.border}`}>
      <Icon className="w-3.5 h-3.5" />
      {config.label}
    </span>
  );
}

function SeverityBadge({ severity }: { severity: string }) {
  const colorClass = SEVERITY_COLORS[severity] || SEVERITY_COLORS.low;
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colorClass}`}>
      {severity.toUpperCase()}
    </span>
  );
}

function RegionCard({ region, onExport }: { region: RegionCompliance; onExport: (region: string, format: 'pdf' | 'xml') => void }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const config = STATUS_CONFIG[region.status] || STATUS_CONFIG.not_assessed;
  const StatusIcon = config.icon;
  const flag = REGION_FLAGS[region.region] || '🌍';
  const label = REGION_LABELS[region.region] || region.region;

  const scoreColor = region.score >= 90
    ? 'text-emerald-600 dark:text-emerald-400'
    : region.score >= 60
    ? 'text-amber-600 dark:text-amber-400'
    : 'text-red-600 dark:text-red-400';

  return (
    <Card className="overflow-hidden">
      <div className="p-4">
        {/* Header */}
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center gap-2">
            <span className="text-xl">{flag}</span>
            <div>
              <h3 className="font-semibold text-slate-900 dark:text-white">{label}</h3>
              <span className="text-xs text-slate-500 dark:text-slate-400">{t('region_code')}: {region.region}</span>
            </div>
          </div>
          <StatusIcon className={`w-5 h-5 ${config.text}`} />
        </div>

        {/* Score ring */}
        <div className="flex items-center gap-4 mb-3">
          <div className="relative w-16 h-16">
            <svg className="w-16 h-16 -rotate-90" viewBox="0 0 36 36">
              <circle cx="18" cy="18" r="15.5" fill="none" stroke="currentColor" strokeWidth="3"
                className="text-slate-200 dark:text-slate-700" />
              <circle cx="18" cy="18" r="15.5" fill="none" stroke="currentColor" strokeWidth="3"
                strokeDasharray={`${region.score} ${100 - region.score}`}
                strokeLinecap="round"
                className={scoreColor}
              />
            </svg>
            <span className={`absolute inset-0 flex items-center justify-center text-sm font-bold ${scoreColor}`}>
              {Math.round(region.score)}%
            </span>
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2 text-sm">
              <span className="text-slate-500 dark:text-slate-400">{t('status')}:</span>
              <StatusBadge status={region.status} />
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400">
              {region.devices} {t('devices')} · ${region.exposure.toLocaleString()} {t('exposure')} · {region.gaps} {t('gaps')}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => onExport(region.region, 'pdf')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-md transition-colors"
          >
            <FileText className="w-3.5 h-3.5" />
            PDF
          </button>
          <button
            onClick={() => onExport(region.region, 'xml')}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-md transition-colors"
          >
            <FileCode className="w-3.5 h-3.5" />
            XML
          </button>
          <button
            onClick={() => setExpanded(!expanded)}
            className="ml-auto flex items-center gap-1 text-xs text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 transition-colors"
          >
            {t('details')}
            {expanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
          </button>
        </div>

        {/* Expanded details */}
        {expanded && (
          <div className="mt-3 pt-3 border-t border-slate-200 dark:border-slate-700 space-y-2">
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div className="text-slate-500 dark:text-slate-400">{t('devices_monitored')}:</div>
              <div className="text-slate-900 dark:text-white font-medium text-right">{region.devices}</div>
              <div className="text-slate-500 dark:text-slate-400">{t('total_exposure')}:</div>
              <div className="text-slate-900 dark:text-white font-medium text-right">${region.exposure.toLocaleString()}</div>
              <div className="text-slate-500 dark:text-slate-400">{t('open_gaps')}:</div>
              <div className="text-slate-900 dark:text-white font-medium text-right">{region.gaps}</div>
              <div className="text-slate-500 dark:text-slate-400">{t('last_updated')}:</div>
              <div className="text-slate-900 dark:text-white font-medium text-right">
                {new Date(region.updated_at).toLocaleDateString()}
              </div>
            </div>
          </div>
        )}
      </div>
    </Card>
  );
}

function GapTable({ gaps }: { gaps: GapItem[] }) {
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);
  const displayed = showAll ? gaps : gaps.slice(0, 5);

  if (gaps.length === 0) {
    return (
      <div className="p-6">
        <EmptyState
          icon={<Shield className="w-10 h-10" />}
          title={t('no_gaps_found')}
          description={t('all_compliant_desc')}
          size="sm"
        />
      </div>
    );
  }

  return (
    <div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 dark:border-slate-700">
              <th className="text-left py-2.5 px-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('gap_category')}</th>
              <th className="text-left py-2.5 px-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('gap_title')}</th>
              <th className="text-center py-2.5 px-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('gap_severity')}</th>
              <th className="text-center py-2.5 px-3 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('gap_status')}</th>
            </tr>
          </thead>
          <tbody>
            {displayed.map((gap) => (
              <tr key={gap.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                <td className="py-2.5 px-3 text-slate-600 dark:text-slate-400 capitalize">{gap.category}</td>
                <td className="py-2.5 px-3">
                  <div className="text-slate-900 dark:text-white font-medium">{gap.title}</div>
                  <div className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{gap.description}</div>
                </td>
                <td className="py-2.5 px-3 text-center">
                  <SeverityBadge severity={gap.severity} />
                </td>
                <td className="py-2.5 px-3 text-center">
                  <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${
                    gap.status === 'open'
                      ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                      : gap.status === 'in_progress'
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                      : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                  }`}>
                    {gap.status.replace('_', ' ')}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {gaps.length > 5 && (
        <div className="flex justify-center py-3 border-t border-slate-200 dark:border-slate-700">
          <button
            onClick={() => setShowAll(!showAll)}
            className="flex items-center gap-1 text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
          >
            {showAll ? t('show_less') : t('show_all_gaps', { count: gaps.length })}
            {showAll ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </button>
        </div>
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export const ComplianceDashboard: React.FC = () => {
  const { t } = useTranslation();
  const [dashboard, setDashboard] = useState<ComplianceDashboard | null>(null);
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchDashboard = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await request<ComplianceDashboard>('/compliance/dashboard');
      setDashboard(data);
    } catch (err) {
      console.error('Failed to fetch compliance dashboard', err);
      setError(t('dashboard_load_error'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  const handleExport = async (region: string, format: 'pdf' | 'xml') => {
    const key = `${region}_${format}`;
    setExporting(key);
    try {
      const blob = await request<Blob>(`/compliance/report/${region}?format=${format}`, {
        headers: { Accept: format === 'pdf' ? 'application/pdf' : 'application/xml' },
      });
      const url = URL.createObjectURL(blob as unknown as Blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `compliance-report-${region}-${new Date().toISOString().slice(0, 10)}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error(`Failed to export ${format} report for ${region}`, err);
    } finally {
      setExporting(null);
    }
  };

  const handleRefresh = () => {
    fetchDashboard();
  };

  // ─── Loading State ────────────────────────────────────────────────
  if (loading && !dashboard) {
    return (
      <div className="p-6 max-w-7xl mx-auto space-y-6">
        {/* Skeleton Header */}
        <div className="flex items-center justify-between">
          <div className="space-y-2">
            <div className="h-8 w-64 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
            <div className="h-4 w-48 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
          </div>
          <div className="h-10 w-28 bg-slate-200 dark:bg-slate-700 rounded-lg animate-pulse" />
        </div>

        {/* Skeleton KPI Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <SkeletonStatsCard />
          <SkeletonStatsCard />
          <SkeletonStatsCard />
          <SkeletonStatsCard />
        </div>

        {/* Skeleton Region Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-48 bg-slate-200 dark:bg-slate-700 rounded-xl animate-pulse" />
          ))}
        </div>

        {/* Skeleton Gap Table */}
        <SkeletonTable rows={3} columns={4} />
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* ═══ Header ═══ */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Globe className="w-7 h-7 text-blue-500" />
            {t('compliance_dashboard') || 'Regional Compliance Dashboard'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('compliance_dashboard_desc') || 'Real-time compliance monitoring across all supported regions'}
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={loading}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg transition-colors text-sm font-medium"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          {loading ? t('refreshing') : t('refresh')}
        </button>
      </div>

      {/* ═══ Error State ═══ */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-red-500 shrink-0" />
          <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* ═══ KPI Cards ═══ */}
      {dashboard && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatsCard
              title={t('overall_compliance') || 'Overall Compliance'}
              value={`${Math.round(dashboard.overall_score)}%`}
              icon={Shield}
              trend={{
                value: Math.round(dashboard.overall_score),
                label: t('compliance_score_label'),
                direction: dashboard.overall_score >= 80 ? 'up' : 'down',
              }}
            />
            <StatsCard
              title={t('regions_monitored') || 'Regions Monitored'}
              value={dashboard.regions.length.toString()}
              icon={Globe}
            />
            <StatsCard
              title={t('total_exposure_kpi') || 'Total Exposure'}
              value={`$${dashboard.risk_summary.total_exposure.toLocaleString('en-US', { minimumFractionDigits: 2 })}`}
              icon={AlertTriangle}
              iconColor="text-red-600"
              iconBgColor="bg-red-50"
              trend={{
                value: dashboard.risk_summary.at_risk_devices,
                label: t('at_risk_label'),
                direction: dashboard.risk_summary.at_risk_devices > 0 ? 'down' : 'up',
              }}
            />
            <StatsCard
              title={t('overall_status') || 'Overall Status'}
              value={dashboard.overall_status.replace('_', ' ')}
              icon={CheckCircle}
              iconColor={dashboard.overall_status === 'compliant' ? 'text-emerald-600' : 'text-amber-600'}
              iconBgColor={dashboard.overall_status === 'compliant' ? 'bg-emerald-50' : 'bg-amber-50'}
            />
          </div>

          {/* ═══ Region Cards ═══ */}
          <div>
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <Globe className="w-5 h-5 text-slate-500" />
              {t('regional_compliance') || 'Regional Compliance'}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {dashboard.regions.map((region) => (
                <RegionCard key={region.region} region={region} onExport={handleExport} />
              ))}
            </div>
          </div>

          {/* ═══ Gap Analysis ═══ */}
          <Card>
            <div className="p-4 border-b border-slate-200 dark:border-slate-700">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="w-5 h-5 text-slate-500" />
                  <h3 className="font-semibold text-slate-900 dark:text-white">
                    {t('gap_analysis') || 'Gap Analysis'}
                  </h3>
                  <span className="text-xs text-slate-500 dark:text-slate-400">
                    ({(dashboard.recent_gaps || []).length} {t('gaps_found')})
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  {Object.entries(dashboard.risk_summary.severity_breakdown || {}).map(([level, count]) =>
                    count > 0 ? (
                      <div key={level} className="flex items-center gap-1">
                        <SeverityBadge severity={level} />
                        <span className="text-xs font-medium text-slate-600 dark:text-slate-400">×{count}</span>
                      </div>
                    ) : null
                  )}
                </div>
              </div>
            </div>
            <GapTable gaps={dashboard.recent_gaps || []} />
          </Card>

          {/* ═══ Footer ═══ */}
          <div className="flex items-center justify-between text-xs text-slate-400 dark:text-slate-500">
            <span>
              {t('last_updated')}: {new Date(dashboard.generated_at).toLocaleString()}
            </span>
            <span className="flex items-center gap-1">
              <ExternalLink className="w-3 h-3" />
              {t('auto_generated')}
            </span>
          </div>
        </>
      )}
    </div>
  );
};

export default ComplianceDashboard;
