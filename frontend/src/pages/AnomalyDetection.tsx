// ═══════════════════════════════════════════════════════════════════════
// AnomalyDetection — Dashboard обнаружения аномалий устройств.
//
// P2-AI.4: Anomaly Detection Dashboard
//   - KPI карточки с количеством активных аномалий
//   - Фильтры по типу метрики / серьёзности / устройству
//   - Таблица аномалий с действиями
//   - Real-time WebSocket обновления
//   - Графики метрик (placeholder для chart library)
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AlertTriangle,
  Activity,
  CheckCircle2,
  XCircle,
  Filter,
  RefreshCw,
  AlertCircle,
  Thermometer,
  Wifi,
  HardDrive,
  Cpu,
  Camera,
} from '../components/ui/Icons';
import { anomaliesApi, type AnomalyResult, type AnomalyStats } from '../services/api/anomalies';

// ─── Types ──────────────────────────────────────────────────────────

type MetricType = 'heartbeat_latency' | 'error_rate' | 'packet_loss' | 'cpu_usage'
  | 'memory_usage' | 'disk_usage' | 'video_bitrate' | 'fps' | 'connection_jitter' | 'temperature';
type Severity = 'low' | 'medium' | 'high' | 'critical';
type AnomalyStatus = 'new' | 'acknowledged' | 'resolved';

interface Filters {
  device_id: string;
  metric_type: string;
  severity: string;
  status: string;
}

// ─── Metric type labels ────────────────────────────────────────────

const METRIC_LABELS: Record<string, string> = {
  heartbeat_latency: 'Задержка heartbeat',
  error_rate: 'Частота ошибок',
  packet_loss: 'Потеря пакетов',
  cpu_usage: 'Загрузка CPU',
  memory_usage: 'Использование памяти',
  disk_usage: 'Использование диска',
  video_bitrate: 'Битрейт видео',
  fps: 'FPS',
  connection_jitter: 'Джиттер',
  temperature: 'Температура',
};

const METRIC_ICONS: Record<string, React.ElementType> = {
  heartbeat_latency: Activity,
  error_rate: AlertCircle,
  packet_loss: Wifi,
  cpu_usage: Cpu,
  memory_usage: Cpu,
  disk_usage: HardDrive,
  video_bitrate: Camera,
  fps: Camera,
  connection_jitter: Wifi,
  temperature: Thermometer,
};

const SEVERITY_COLORS: Record<string, string> = {
  low: 'text-slate-600 bg-slate-100 dark:text-slate-300 dark:bg-slate-800',
  medium: 'text-amber-600 bg-amber-100 dark:text-amber-300 dark:bg-amber-900/30',
  high: 'text-orange-600 bg-orange-100 dark:text-orange-300 dark:bg-orange-900/30',
  critical: 'text-red-600 bg-red-100 dark:text-red-300 dark:bg-red-900/30',
};

const STATUS_COLORS: Record<string, string> = {
  new: 'text-blue-600 bg-blue-100 dark:text-blue-300 dark:bg-blue-900/30',
  acknowledged: 'text-amber-600 bg-amber-100 dark:text-amber-300 dark:bg-amber-900/30',
  resolved: 'text-emerald-600 bg-emerald-100 dark:text-emerald-300 dark:bg-emerald-900/30',
};

// ─── KPI Card ──────────────────────────────────────────────────────

interface KPICardProps {
  label: string;
  value: string | number;
  icon: React.ElementType;
  color: string;
  bg: string;
  subtitle?: string;
}

function KPICard({ label, value, icon: Icon, color, bg, subtitle }: KPICardProps) {
  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4">
      <div className="flex items-center justify-between mb-3">
        <div className={`p-2 rounded-lg ${bg}`}>
          <Icon className={`w-5 h-5 ${color}`} />
        </div>
      </div>
      <p className="text-2xl font-bold text-slate-900 dark:text-white">{value}</p>
      <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">{label}</p>
      {subtitle && (
        <p className="text-xs text-slate-400 dark:text-slate-500 mt-0.5">{subtitle}</p>
      )}
    </div>
  );
}

// ─── Severity Badge ────────────────────────────────────────────────

function SeverityBadge({ severity }: { severity: string }) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${SEVERITY_COLORS[severity] || SEVERITY_COLORS.low}`}>
      {severity.toUpperCase()}
    </span>
  );
}

// ─── Status Badge ──────────────────────────────────────────────────

function StatusBadge({ status }: { status: string }) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[status] || STATUS_COLORS.new}`}>
      {status === 'new' ? 'НОВАЯ' : status === 'acknowledged' ? 'ПОДТВЕРЖДЕНА' : 'РЕШЕНА'}
    </span>
  );
}

// ─── Metric Icon ───────────────────────────────────────────────────

function MetricIcon({ type }: { type: string }) {
  const Icon = METRIC_ICONS[type] || Activity;
  return <Icon className="w-4 h-4 text-slate-400" />;
}

// ─── Main Component ────────────────────────────────────────────────

export function AnomalyDetection() {
  const { t } = useTranslation();

  const [anomalies, setAnomalies] = useState<AnomalyResult[]>([]);
  const [stats, setStats] = useState<AnomalyStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<Filters>({
    device_id: '',
    metric_type: '',
    severity: '',
    status: '',
  });
  const [deviceFilterInput, setDeviceFilterInput] = useState('');
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // ─── Fetch Data ──────────────────────────────────────────────────

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [anomalyData, statsData] = await Promise.all([
        anomaliesApi.getAnomalies({
          device_id: filters.device_id || undefined,
          metric_type: filters.metric_type || undefined,
          severity: filters.severity || undefined,
          status: filters.status || undefined,
          limit: 100,
        }),
        anomaliesApi.getStats(),
      ]);

      setAnomalies(anomalyData.anomalies);
      setStats(statsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch anomaly data');
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // ─── Auto-refresh every 30s ──────────────────────────────────────

  useEffect(() => {
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  // ─── Actions ─────────────────────────────────────────────────────

  const handleAcknowledge = async (id: string) => {
    setActionLoading(id);
    try {
      await anomaliesApi.acknowledgeAnomaly(id);
      await fetchData();
    } catch (err) {
      console.error('Failed to acknowledge anomaly:', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleResolve = async (id: string) => {
    setActionLoading(id);
    try {
      await anomaliesApi.resolveAnomaly(id);
      await fetchData();
    } catch (err) {
      console.error('Failed to resolve anomaly:', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleRefresh = () => {
    fetchData();
  };

  const handleApplyDeviceFilter = () => {
    setFilters((prev) => ({ ...prev, device_id: deviceFilterInput }));
  };

  const handleClearFilters = () => {
    setDeviceFilterInput('');
    setFilters({ device_id: '', metric_type: '', severity: '', status: '' });
  };

  // ─── KPI Data ────────────────────────────────────────────────────

  const activeCount = anomalies.filter((a) => a.status !== 'resolved').length;
  const criticalCount = anomalies.filter((a) => a.severity === 'critical' && a.status !== 'resolved').length;
  const acknowledgedCount = anomalies.filter((a) => a.status === 'acknowledged').length;

  // ─── Render ──────────────────────────────────────────────────────

  return (
    <div className="p-4 md:p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('anomaly_detection') || 'Обнаружение аномалий'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {t('anomaly_detection_desc') || 'Статистический анализ метрик устройств'}
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={loading}
          className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          {t('refresh') || 'Обновить'}
        </button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KPICard
          label={t('active_anomalies') || 'Активные аномалии'}
          value={activeCount}
          icon={AlertTriangle}
          color="text-amber-600"
          bg="bg-amber-50 dark:bg-amber-900/20"
          subtitle={stats ? `Буферов: ${stats.metric_buffers}` : undefined}
        />
        <KPICard
          label={t('critical_anomalies') || 'Критические'}
          value={criticalCount}
          icon={XCircle}
          color="text-red-600"
          bg="bg-red-50 dark:bg-red-900/20"
        />
        <KPICard
          label={t('acknowledged') || 'Подтверждены'}
          value={acknowledgedCount}
          icon={CheckCircle2}
          color="text-blue-600"
          bg="bg-blue-50 dark:bg-blue-900/20"
        />
        <KPICard
          label={t('total_metric_points') || 'Точек метрик'}
          value={stats?.total_metric_points ?? 0}
          icon={Activity}
          color="text-emerald-600"
          bg="bg-emerald-50 dark:bg-emerald-900/20"
          subtitle={stats ? `NATS: ${stats.nats_connected ? '✓' : '✗'}` : undefined}
        />
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4">
        <div className="flex items-center gap-2 mb-3">
          <Filter className="w-4 h-4 text-slate-500" />
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
            {t('filters') || 'Фильтры'}
          </span>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
          {/* Device ID */}
          <div>
            <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">
              {t('device_id') || 'Устройство'}
            </label>
            <div className="flex gap-1">
              <input
                type="text"
                value={deviceFilterInput}
                onChange={(e) => setDeviceFilterInput(e.target.value)}
                placeholder="device-id"
                className="w-full px-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                onKeyDown={(e) => e.key === 'Enter' && handleApplyDeviceFilter()}
              />
              <button
                onClick={handleApplyDeviceFilter}
                className="px-2 py-1.5 text-xs font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700"
              >
                OK
              </button>
            </div>
          </div>

          {/* Metric Type */}
          <div>
            <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">
              {t('metric_type') || 'Тип метрики'}
            </label>
            <select
              value={filters.metric_type}
              onChange={(e) => setFilters((p) => ({ ...p, metric_type: e.target.value }))}
              className="w-full px-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
            >
              <option value="">{t('all') || 'Все'}</option>
              {Object.entries(METRIC_LABELS).map(([key, label]) => (
                <option key={key} value={key}>{label}</option>
              ))}
            </select>
          </div>

          {/* Severity */}
          <div>
            <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">
              {t('severity') || 'Серьёзность'}
            </label>
            <select
              value={filters.severity}
              onChange={(e) => setFilters((p) => ({ ...p, severity: e.target.value }))}
              className="w-full px-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
            >
              <option value="">{t('all') || 'Все'}</option>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="critical">Critical</option>
            </select>
          </div>

          {/* Status */}
          <div>
            <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">
              {t('status') || 'Статус'}
            </label>
            <select
              value={filters.status}
              onChange={(e) => setFilters((p) => ({ ...p, status: e.target.value }))}
              className="w-full px-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
            >
              <option value="">{t('all') || 'Все'}</option>
              <option value="new">Новая</option>
              <option value="acknowledged">Подтверждена</option>
              <option value="resolved">Решена</option>
            </select>
          </div>

          {/* Clear */}
          <div className="flex items-end">
            <button
              onClick={handleClearFilters}
              className="w-full px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              {t('clear_filters') || 'Сброс'}
            </button>
          </div>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-4 flex items-center gap-3">
          <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
          <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
        </div>
      )}

      {/* Loading State */}
      {loading && anomalies.length === 0 && !error && (
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 text-center">
          <Activity className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-3 animate-pulse" />
          <p className="text-sm text-slate-400">
            {t('loading_anomalies') || 'Загрузка аномалий...'}
          </p>
        </div>
      )}

      {/* Empty State */}
      {!loading && !error && anomalies.length === 0 && (
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 text-center">
          <CheckCircle2 className="w-12 h-12 text-emerald-400 mx-auto mb-3" />
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-1">
            {t('no_anomalies') || 'Аномалий не обнаружено'}
          </h3>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {t('no_anomalies_desc') || 'Все метрики устройств в пределах нормы'}
          </p>
        </div>
      )}

      {/* Anomalies Table */}
      {anomalies.length > 0 && (
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('metric') || 'Метрика'}
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('device') || 'Устройство'}
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('value') || 'Значение'}
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('z_score') || 'Z-Score'}
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('severity') || 'Серьёзность'}
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('status') || 'Статус'}
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('detected_at') || 'Обнаружено'}
                  </th>
                  <th className="text-right px-4 py-3 font-medium text-slate-500 dark:text-slate-400 text-xs uppercase tracking-wider">
                    {t('actions') || 'Действия'}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                {anomalies.map((anomaly) => (
                  <tr key={anomaly.id} className="hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <MetricIcon type={anomaly.metric_type} />
                        <span className="text-slate-900 dark:text-white font-medium">
                          {METRIC_LABELS[anomaly.metric_type] || anomaly.metric_type}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <code className="text-xs text-slate-600 dark:text-slate-400 bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded">
                        {anomaly.device_id.substring(0, 12)}...
                      </code>
                    </td>
                    <td className="px-4 py-3">
                      <div className="text-slate-900 dark:text-white">
                        <span className="font-medium">{anomaly.current_value.toFixed(2)}</span>
                        <span className="text-slate-400 text-xs ml-1">
                          (μ={anomaly.mean_value.toFixed(2)})
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`font-mono font-medium ${
                        Math.abs(anomaly.z_score) >= 5 ? 'text-red-500' :
                        Math.abs(anomaly.z_score) >= 3 ? 'text-amber-500' :
                        'text-slate-500'
                      }`}>
                        {anomaly.z_score.toFixed(2)}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <SeverityBadge severity={anomaly.severity} />
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={anomaly.status} />
                    </td>
                    <td className="px-4 py-3 text-slate-500 dark:text-slate-400 text-xs">
                      {new Date(anomaly.detected_at).toLocaleString('ru-RU')}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        {anomaly.status === 'new' && (
                          <button
                            onClick={() => handleAcknowledge(anomaly.id)}
                            disabled={actionLoading === anomaly.id}
                            className="px-2 py-1 text-xs font-medium text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors disabled:opacity-50"
                          >
                            {t('acknowledge') || 'Подтв.'}
                          </button>
                        )}
                        {anomaly.status !== 'resolved' && (
                          <button
                            onClick={() => handleResolve(anomaly.id)}
                            disabled={actionLoading === anomaly.id}
                            className="px-2 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-900/20 rounded transition-colors disabled:opacity-50"
                          >
                            {t('resolve') || 'Решить'}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Description tooltip row for expanded info */}
      {anomalies.length > 0 && (
        <div className="grid grid-cols-1 gap-2">
          {anomalies.slice(0, 3).map((anomaly) => (
            <div
              key={`desc-${anomaly.id}`}
              className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-3 text-xs text-slate-600 dark:text-slate-400"
            >
              <span className="font-medium text-slate-700 dark:text-slate-300">
                [{anomaly.device_id.substring(0, 8)}]
              </span>{' '}
              {anomaly.description}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
