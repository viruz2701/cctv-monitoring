// P1-REG.6: Regulatory Dashboard
//
// Widget в Compliance Shield для мониторинга региональных требований ТО:
//   - KPI: upcoming TO, overdue, completed
//   - Retention status (archive / hot / cold)
//   - License expiration alerts
//   - Regional compliance score
//
// Compliance:
//   - РД 25.964-90 (плановое ТО)
//   - СН 3.02.19-2025 (CCTV)
//   - Приказ МЧС №55 (пожарная автоматика)
//   - ISO 27001 A.12.4 (Audit trail)
//   - IEC 62443 SR 3.1 (Data integrity)

import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../../services/api';
import {
  Card,
  CardHeader,
  Badge,
  StatsCard,
  EmptyState,
  SkeletonStatsCard,
  SkeletonChart,
  SkeletonTable,
} from '../../components/ui';
import {
  Shield,
  AlertTriangle,
  Calendar,
  FileText,
  Clock,
  CheckCircle2,
  Archive,
  HardDrive,
  Thermometer,
  RefreshCw,
} from '../../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface MaintenanceKPI {
  upcoming: number;
  overdue: number;
  completed: number;
  total: number;
  compliance_rate: number;
}

interface LicenseInfo {
  id: string;
  vendor_name: string;
  license_type: string;
  region: string;
  issued_at: string;
  expires_at: string;
  status: 'active' | 'expiring_soon' | 'expired';
  days_until_expiry: number;
}

interface RegulationStatus {
  regulation_id: string;
  regulation_name: string;
  region: string;
  device_count: number;
  compliant: number;
  non_compliant: number;
  last_check: string;
}

interface RetentionStatus {
  storage_tier: 'hot' | 'cold' | 'archive';
  total_size_gb: number;
  used_gb: number;
  retention_days: number;
  region: string;
}

interface RegionalComplianceScore {
  region: string;
  score: number;
  trend: 'up' | 'down' | 'stable';
  total_checks: number;
  passed: number;
  failed: number;
}

interface RegulatoryDashboardData {
  maintenance_kpi: MaintenanceKPI;
  licenses: LicenseInfo[];
  regulation_statuses: RegulationStatus[];
  retention: RetentionStatus[];
  regional_scores: RegionalComplianceScore[];
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const SCORE_COLORS: Record<string, string> = {
  excellent: '#059669',
  good: '#3B82F6',
  fair: '#F59E0B',
  poor: '#DC2626',
};

function getScoreColor(score: number): string {
  if (score >= 90) return SCORE_COLORS.excellent;
  if (score >= 75) return SCORE_COLORS.good;
  if (score >= 60) return SCORE_COLORS.fair;
  return SCORE_COLORS.poor;
}

function getScoreLabel(score: number): string {
  if (score >= 90) return 'excellent';
  if (score >= 75) return 'good';
  if (score >= 60) return 'fair';
  return 'poor';
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function RegulatoryCompliance() {
  const { t } = useTranslation();
  const [data, setData] = useState<RegulatoryDashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedRegion, setSelectedRegion] = useState<string>('all');

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const query = selectedRegion !== 'all' ? `?region=${encodeURIComponent(selectedRegion)}` : '';
      const response = await request<RegulatoryDashboardData>(
        `/api/v1/compliance/regulatory-dashboard${query}`,
      );

      setData(response);
    } catch (err) {
      console.error('[RegulatoryCompliance] Failed to load:', err);
      setError(t('common:error_loading'));
    } finally {
      setLoading(false);
    }
  }, [selectedRegion, t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // ── Maintenance KPI Widget ─────────────────────────────────────────
  const renderMaintenanceKPI = () => {
    if (!data) return <SkeletonStatsCard />;

    const kpi = data.maintenance_kpi;

    return (
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('compliance:upcoming_to')}
          value={kpi.upcoming}
          icon={<Calendar className="w-5 h-5 text-blue-600" />}
          className="border-l-4 border-blue-500"
        />
        <StatsCard
          title={t('compliance:overdue_to')}
          value={kpi.overdue}
          icon={<AlertTriangle className="w-5 h-5 text-red-600" />}
          className="border-l-4 border-red-500"
          trend={kpi.overdue > 0 ? 'down' : 'stable'}
        />
        <StatsCard
          title={t('compliance:completed_to')}
          value={kpi.completed}
          icon={<CheckCircle2 className="w-5 h-5 text-green-600" />}
          className="border-l-4 border-green-500"
        />
        <StatsCard
          title={t('compliance:compliance_rate')}
          value={`${kpi.compliance_rate.toFixed(1)}%`}
          icon={<Shield className="w-5 h-5 text-purple-600" />}
          className="border-l-4 border-purple-500"
          trend={kpi.compliance_rate >= 90 ? 'up' : kpi.compliance_rate >= 75 ? 'stable' : 'down'}
        />
      </div>
    );
  };

  // ── License Alerts ─────────────────────────────────────────────────
  const renderLicenseAlerts = () => {
    if (!data) return <SkeletonTable rows={3} />;

    const expiringSoon = data.licenses.filter((l) => l.status === 'expiring_soon');
    const expired = data.licenses.filter((l) => l.status === 'expired');

    if (data.licenses.length === 0) {
      return (
        <EmptyState
          icon={<FileText className="w-12 h-12 text-gray-400" />}
          title={t('compliance:no_licenses')}
          description={t('compliance:no_licenses_description')}
        />
      );
    }

    return (
      <div className="space-y-3">
        {/* Alerts */}
        {expiringSoon.length > 0 && (
          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
            <div className="flex items-center gap-2 mb-2">
              <Clock className="w-4 h-4 text-yellow-600" />
              <span className="text-sm font-medium text-yellow-800">
                {t('compliance:licenses_expiring_soon', { count: expiringSoon.length })}
              </span>
            </div>
            {expiringSoon.slice(0, 3).map((license) => (
              <div key={license.id} className="text-sm text-yellow-700 ml-6">
                {license.vendor_name} — {t('compliance:expires_in_days', { days: license.days_until_expiry })}
              </div>
            ))}
          </div>
        )}

        {expired.length > 0 && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-3">
            <div className="flex items-center gap-2 mb-2">
              <AlertTriangle className="w-4 h-4 text-red-600" />
              <span className="text-sm font-medium text-red-800">
                {t('compliance:licenses_expired', { count: expired.length })}
              </span>
            </div>
            {expired.map((license) => (
              <div key={license.id} className="text-sm text-red-700 ml-6">
                {license.vendor_name} — {t('compliance:expired_days_ago', { days: Math.abs(license.days_until_expiry) })}
              </div>
            ))}
          </div>
        )}

        {/* License table */}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="text-left py-2 px-3 font-medium text-gray-600">
                  {t('compliance:vendor')}
                </th>
                <th className="text-left py-2 px-3 font-medium text-gray-600">
                  {t('compliance:region')}
                </th>
                <th className="text-left py-2 px-3 font-medium text-gray-600">
                  {t('compliance:type')}
                </th>
                <th className="text-left py-2 px-3 font-medium text-gray-600">
                  {t('compliance:expires')}
                </th>
                <th className="text-left py-2 px-3 font-medium text-gray-600">
                  {t('compliance:status')}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.licenses.map((license) => (
                <tr key={license.id} className="border-b border-gray-100 hover:bg-gray-50">
                  <td className="py-2 px-3">{license.vendor_name}</td>
                  <td className="py-2 px-3">
                    <Badge variant="outline">{license.region}</Badge>
                  </td>
                  <td className="py-2 px-3">{license.license_type}</td>
                  <td className="py-2 px-3">
                    {new Date(license.expires_at).toLocaleDateString()}
                  </td>
                  <td className="py-2 px-3">
                    <Badge
                      variant={
                        license.status === 'active'
                          ? 'success'
                          : license.status === 'expiring_soon'
                          ? 'warning'
                          : 'danger'
                      }
                    >
                      {t(`compliance:license_status_${license.status}`)}
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  };

  // ── Retention Status ───────────────────────────────────────────────
  const renderRetentionStatus = () => {
    if (!data) return <SkeletonChart />;

    if (data.retention.length === 0) {
      return (
        <EmptyState
          icon={<Archive className="w-12 h-12 text-gray-400" />}
          title={t('compliance:no_retention_data')}
        />
      );
    }

    const tierIcons: Record<string, React.ReactNode> = {
      hot: <Activity className="w-4 h-4 text-red-500" />,
      cold: <Thermometer className="w-4 h-4 text-blue-500" />,
      archive: <Archive className="w-4 h-4 text-gray-500" />,
    };

    return (
      <div className="space-y-3">
        {data.retention.map((item, idx) => (
          <div key={idx} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
            <div className="flex items-center gap-3">
              {tierIcons[item.storage_tier] || <HardDrive className="w-4 h-4" />}
              <div>
                <div className="text-sm font-medium text-gray-900">
                  {t(`compliance:tier_${item.storage_tier}`)}
                </div>
                <div className="text-xs text-gray-500">
                  {t('compliance:region')}: {item.region}
                </div>
              </div>
            </div>
            <div className="text-right">
              <div className="text-sm font-medium text-gray-900">
                {(item.used_gb / 1024).toFixed(1)} GB / {(item.total_size_gb / 1024).toFixed(1)} GB
              </div>
              <div className="text-xs text-gray-500">
                {item.retention_days} {t('compliance:days_retention')}
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  };

  // ── Regional Compliance Scores ─────────────────────────────────────
  const renderRegionalScores = () => {
    if (!data) return <SkeletonTable rows={3} />;

    if (data.regional_scores.length === 0) {
      return (
        <EmptyState
          icon={<Shield className="w-12 h-12 text-gray-400" />}
          title={t('compliance:no_regional_scores')}
        />
      );
    }

    return (
      <div className="space-y-4">
        {data.regional_scores.map((score) => (
          <div key={score.region} className="bg-white border border-gray-200 rounded-lg p-4">
            <div className="flex items-center justify-between mb-3">
              <div>
                <span className="font-medium text-gray-900">{score.region}</span>
                <span className="ml-2 text-xs text-gray-500">
                  {score.total_checks} {t('compliance:checks')}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <div
                  className="text-2xl font-bold"
                  style={{ color: getScoreColor(score.score) }}
                >
                  {score.score}%
                </div>
                {score.trend === 'up' && (
                  <TrendingUp className="w-4 h-4 text-green-500" />
                )}
                {score.trend === 'down' && (
                  <TrendingDown className="w-4 h-4 text-red-500" />
                )}
              </div>
            </div>

            {/* Score bar */}
            <div className="w-full bg-gray-200 rounded-full h-2.5">
              <div
                className="h-2.5 rounded-full transition-all duration-500"
                style={{
                  width: `${score.score}%`,
                  backgroundColor: getScoreColor(score.score),
                }}
              />
            </div>

            <div className="flex gap-4 mt-2 text-xs text-gray-500">
              <span className="text-green-600">
                {score.passed} {t('compliance:passed')}
              </span>
              <span className="text-red-600">
                {score.failed} {t('compliance:failed')}
              </span>
            </div>
          </div>
        ))}
      </div>
    );
  };

  // ── Regulation Status ──────────────────────────────────────────────
  const renderRegulationStatus = () => {
    if (!data) return <SkeletonTable rows={4} />;

    if (data.regulation_statuses.length === 0) {
      return (
        <EmptyState
          icon={<FileText className="w-12 h-12 text-gray-400" />}
          title={t('compliance:no_regulations')}
        />
      );
    }

    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="text-left py-2 px-3 font-medium text-gray-600">
                {t('compliance:regulation')}
              </th>
              <th className="text-left py-2 px-3 font-medium text-gray-600">
                {t('compliance:region')}
              </th>
              <th className="text-center py-2 px-3 font-medium text-gray-600">
                {t('compliance:devices')}
              </th>
              <th className="text-center py-2 px-3 font-medium text-gray-600">
                {t('compliance:compliant')}
              </th>
              <th className="text-center py-2 px-3 font-medium text-gray-600">
                {t('compliance:non_compliant')}
              </th>
              <th className="text-center py-2 px-3 font-medium text-gray-600">
                {t('compliance:compliance_rate_short')}
              </th>
            </tr>
          </thead>
          <tbody>
            {data.regulation_statuses.map((reg) => {
              const rate = reg.device_count > 0
                ? ((reg.compliant / reg.device_count) * 100).toFixed(1)
                : '100.0';
              return (
                <tr key={reg.regulation_id} className="border-b border-gray-100 hover:bg-gray-50">
                  <td className="py-2 px-3 font-medium text-gray-900">
                    {reg.regulation_name}
                  </td>
                  <td className="py-2 px-3">
                    <Badge variant="outline">{reg.region}</Badge>
                  </td>
                  <td className="py-2 px-3 text-center">{reg.device_count}</td>
                  <td className="py-2 px-3 text-center text-green-600">
                    {reg.compliant}
                  </td>
                  <td className="py-2 px-3 text-center">
                    {reg.non_compliant > 0 ? (
                      <span className="text-red-600">{reg.non_compliant}</span>
                    ) : (
                      <span className="text-gray-400">0</span>
                    )}
                  </td>
                  <td className="py-2 px-3 text-center">
                    <Badge
                      variant={
                        parseFloat(rate) >= 90
                          ? 'success'
                          : parseFloat(rate) >= 75
                          ? 'warning'
                          : 'danger'
                      }
                    >
                      {rate}%
                    </Badge>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  };

  // ── Main Render ────────────────────────────────────────────────────
  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-red-600" />
            <span className="text-red-800">{error}</span>
          </div>
          <button
            onClick={fetchData}
            className="mt-2 text-sm text-red-600 hover:text-red-800 flex items-center gap-1"
          >
            <RefreshCw className="w-4 h-4" />
            {t('common:retry')}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            {t('compliance:regulatory_dashboard')}
          </h1>
          <p className="text-sm text-gray-500 mt-1">
            {t('compliance:regulatory_dashboard_description')}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={selectedRegion}
            onChange={(e) => setSelectedRegion(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            aria-label={t('compliance:select_region')}
          >
            <option value="all">{t('compliance:all_regions')}</option>
            <option value="BY">🇧🇾 BY</option>
            <option value="RU">🇷🇺 RU</option>
            <option value="KZ">🇰🇿 KZ</option>
            <option value="EU">🇪🇺 EU</option>
            <option value="INTL">🌍 INTL</option>
          </select>
          <button
            onClick={fetchData}
            className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            aria-label={t('common:refresh')}
            disabled={loading}
          >
            <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">{t('compliance:maintenance_kpi')}</h3>
        </CardHeader>
        {renderMaintenanceKPI()}
      </Card>

      {/* Regulation Status & Regional Scores */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <h3 className="text-lg font-semibold">{t('compliance:regulation_status')}</h3>
            </CardHeader>
            {renderRegulationStatus()}
          </Card>
        </div>
        <div>
          <Card>
            <CardHeader>
              <h3 className="text-lg font-semibold">{t('compliance:regional_scores')}</h3>
            </CardHeader>
            {renderRegionalScores()}
          </Card>
        </div>
      </div>

      {/* Licenses & Retention */}
      <Card>
        <CardHeader>
          <h3 className="text-lg font-semibold">{t('compliance:maintenance_kpi')}</h3>
        </CardHeader>
        {renderMaintenanceKPI()}
      </Card>

      {/* Regulation Status & Regional Scores */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <h3 className="text-lg font-semibold">{t('compliance:regulation_status')}</h3>
            </CardHeader>
            {renderRegulationStatus()}
          </Card>
        </div>
        <div>
          <Card>
            <CardHeader>
              <h3 className="text-lg font-semibold">{t('compliance:regional_scores')}</h3>
            </CardHeader>
            {renderRegionalScores()}
          </Card>
        </div>
      </div>

      {/* Licenses & Retention */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="border-t-4 border-yellow-400">
          <CardHeader>
            <h3 className="text-lg font-semibold">{t('compliance:license_alerts')}</h3>
          </CardHeader>
          {renderLicenseAlerts()}
        </Card>
        <Card className="border-t-4 border-blue-400">
          <CardHeader>
            <h3 className="text-lg font-semibold">{t('compliance:retention_status')}</h3>
          </CardHeader>
          {renderRetentionStatus()}
        </Card>
      </div>
    </div>
  );
}

// ── Helper components ──────────────────────────────────────────────────

function Activity(props: { className?: string }) {
  return <ActivityIcon className={props.className} />;
}

function ActivityIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M13 10V3L4 14h7v7l9-11h-7z"
      />
    </svg>
  );
}

function TrendingUp({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
      />
    </svg>
  );
}

function TrendingDown({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={2}
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M13 17h8m0 0v-8m0 8l-8-8-4 4-6-6"
      />
    </svg>
  );
}

export default RegulatoryCompliance;
