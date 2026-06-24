import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Activity, AlertTriangle, CheckCircle, Clock, XCircle, FileText, RefreshCw, Save, X, Edit2 } from 'lucide-react';
import { request } from '../services/api';
import { Card, DataGrid, Badge, Gauge, StatsCard, Button, useToast } from '../components/ui';

interface SLAConfig {
  id: string;
  priority: string;
  response_time_minutes: number;
  resolution_time_minutes: number;
}

interface SLAComplianceReport {
  priority: string;
  total_work_orders: number;
  within_sla: number;
  breached_sla: number;
  compliance_percent: number;
  avg_response_minutes: number;
  avg_resolution_minutes: number;
}

const GAUGE_THRESHOLDS = [
  { value: 90, color: '#16a34a', label: '≥90%' },
  { value: 70, color: '#d97706', label: '≥70%' },
  { value: 0, color: '#dc2626', label: '<70%' },
];

export const SLADashboard: React.FC = () => {
  const { t } = useTranslation();
  const toast = useToast();
  const [configs, setConfigs] = useState<SLAConfig[]>([]);
  const [reports, setReports] = useState<SLAComplianceReport[]>([]);
  const [loading, setLoading] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date>(new Date());
  const [autoRefresh, setAutoRefresh] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // ═══ Inline Editing States ═══
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<{ response_time_minutes: number; resolution_time_minutes: number }>({
    response_time_minutes: 0,
    resolution_time_minutes: 0,
  });
  const [savingId, setSavingId] = useState<string | null>(null);

  // SLA-6.3.2: Real-time auto-refresh every 30s
  const fetchData = useCallback(async () => {
    try {
      const [c, r] = await Promise.all([
        request<SLAConfig[]>('/sla/config'),
        request<SLAComplianceReport[]>('/reports/sla-compliance'),
      ]);
      setConfigs(c || []);
      setReports(r || []);
      setLastUpdated(new Date());
    } catch (err) {
      console.error('SLA refresh failed', err);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    fetchData().finally(() => setLoading(false));

    // Auto-refresh interval
    if (intervalRef.current) clearInterval(intervalRef.current);
    intervalRef.current = setInterval(fetchData, 30000); // 30s

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [fetchData]);

  // ═══ Inline Edit Handlers ═══
  const startEditing = (config: SLAConfig) => {
    setEditingId(config.id);
    setEditForm({
      response_time_minutes: config.response_time_minutes,
      resolution_time_minutes: config.resolution_time_minutes,
    });
  };

  const cancelEditing = () => {
    setEditingId(null);
  };

  const saveEditing = async (config: SLAConfig) => {
    if (editForm.response_time_minutes <= 0 || editForm.resolution_time_minutes <= 0) {
      toast.error(t('times_must_be_positive') || 'Times must be positive');
      return;
    }
    setSavingId(config.id);
    try {
      await request(`/sla/config/${config.priority}`, {
        method: 'PUT',
        body: JSON.stringify(editForm),
      });
      toast.success(t('sla_config_updated') || 'SLA configuration updated');
      setEditingId(null);
      // Refresh data
      await fetchData();
    } catch (err) {
      const message = err instanceof Error ? err.message : (t('update_failed') || 'Failed to update');
      toast.error(message);
    } finally {
      setSavingId(null);
    }
  };

  const overallCompliance = useMemo(() => {
    if (reports.length === 0) return 0;
    const total = reports.reduce((s, r) => s + r.total_work_orders, 0);
    const within = reports.reduce((s, r) => s + r.within_sla, 0);
    return total > 0 ? (within / total) * 100 : 0;
  }, [reports]);

  const getComplianceColor = (percent: number) => {
    if (percent >= 90) return 'success';
    if (percent >= 70) return 'warning';
    return 'danger';
  };

  const configColumns = [
    { key: 'priority', header: t('priority'), sortable: true, render: (item: SLAConfig) => <Badge variant="info">{t(item.priority)}</Badge> },
    {
      key: 'response_time_minutes',
      header: t('response_time'),
      sortable: true,
      render: (item: SLAConfig) => {
        if (editingId === item.id) {
          return (
            <input
              type="number"
              min={1}
              className="w-24 px-2 py-1 text-sm border border-blue-300 dark:border-blue-600 rounded bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={editForm.response_time_minutes}
              onChange={(e) => setEditForm(prev => ({ ...prev, response_time_minutes: parseInt(e.target.value) || 0 }))}
            />
          );
        }
        return `${item.response_time_minutes} min`;
      },
    },
    {
      key: 'resolution_time_minutes',
      header: t('resolution_time'),
      sortable: true,
      render: (item: SLAConfig) => {
        if (editingId === item.id) {
          return (
            <input
              type="number"
              min={1}
              className="w-24 px-2 py-1 text-sm border border-blue-300 dark:border-blue-600 rounded bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
              value={editForm.resolution_time_minutes}
              onChange={(e) => setEditForm(prev => ({ ...prev, resolution_time_minutes: parseInt(e.target.value) || 0 }))}
            />
          );
        }
        return `${item.resolution_time_minutes} min`;
      },
    },
    {
      key: 'actions',
      header: '',
      align: 'right' as const,
      render: (item: SLAConfig) => {
        if (editingId === item.id) {
          return (
            <div className="flex justify-end gap-1">
              <button
                onClick={() => saveEditing(item)}
                disabled={savingId === item.id}
                className="p-1.5 hover:bg-emerald-50 dark:hover:bg-emerald-900/20 rounded-lg transition-colors"
                title={t('save')}
              >
                {savingId === item.id
                  ? <RefreshCw className="w-4 h-4 text-emerald-500 animate-spin" />
                  : <Save className="w-4 h-4 text-emerald-500" />
                }
              </button>
              <button
                onClick={cancelEditing}
                className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                title={t('cancel')}
              >
                <X className="w-4 h-4 text-red-500" />
              </button>
            </div>
          );
        }
        return (
          <div className="flex justify-end">
            <button
              onClick={() => startEditing(item)}
              className="p-1.5 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
              title={t('edit')}
            >
              <Edit2 className="w-4 h-4 text-slate-400 hover:text-blue-500" />
            </button>
          </div>
        );
      },
    },
  ];

  const reportColumns = [
    { key: 'priority', header: t('priority'), sortable: true, render: (item: SLAComplianceReport) => <Badge variant="info">{t(item.priority)}</Badge> },
    { key: 'total_work_orders', header: t('total'), sortable: true },
    { key: 'within_sla', header: t('within_sla'), sortable: true, render: (item: SLAComplianceReport) => <span className="text-green-600">{item.within_sla}</span> },
    { key: 'breached_sla', header: t('breached'), sortable: true, render: (item: SLAComplianceReport) => <span className="text-red-600">{item.breached_sla}</span> },
    {
      key: 'compliance_percent',
      header: t('compliance'),
      sortable: true,
      render: (item: SLAComplianceReport) => (
        <Badge variant={getComplianceColor(item.compliance_percent) as 'success' | 'warning' | 'danger'}>
          {item.compliance_percent.toFixed(1)}%
        </Badge>
      ),
    },
    { key: 'avg_response_minutes', header: t('avg_response'), sortable: true, render: (item: SLAComplianceReport) => `${item.avg_response_minutes.toFixed(1)} min` },
    { key: 'avg_resolution_minutes', header: t('avg_resolution'), sortable: true, render: (item: SLAComplianceReport) => `${item.avg_resolution_minutes.toFixed(1)} min` },
  ];

  // ── KPI Cards (SLA-6.3.1) ──────────────────────────────────────────

  const kpiData = useMemo(() => {
    if (reports.length === 0) return null;
    const total = reports.reduce((s, r) => s + r.total_work_orders, 0);
    const within = reports.reduce((s, r) => s + r.within_sla, 0);
    const breached = reports.reduce((s, r) => s + r.breached_sla, 0);
    const atRisk = reports.filter(r => r.compliance_percent > 0 && r.compliance_percent < 90)
      .reduce((s, r) => s + r.total_work_orders, 0);
    return { total, within, breached, atRisk, compliance: overallCompliance };
  }, [reports, overallCompliance]);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">{t('sla_dashboard')}</h1>
        <div className="flex items-center gap-3 text-xs text-slate-500">
          <span>{t('last_updated') || 'Last updated'}: {lastUpdated.toLocaleTimeString()}</span>
          <button
            onClick={() => { setAutoRefresh(!autoRefresh); if (!autoRefresh) fetchData(); }}
            className={`inline-flex items-center gap-1 px-2 py-1 rounded transition-colors ${
              autoRefresh ? 'bg-blue-50 text-blue-700' : 'bg-slate-100 text-slate-500'
            }`}
            title={autoRefresh ? (t('disable_auto_refresh') || 'Disable auto-refresh') : (t('enable_auto_refresh') || 'Enable auto-refresh')}
          >
            <RefreshCw className={`w-3 h-3 ${autoRefresh ? 'animate-spin' : ''}`} />
            {autoRefresh ? '30s' : (t('manual') || 'Manual')}
          </button>
        </div>
      </div>

      {/* KPI Cards Row */}
      {kpiData && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <StatsCard
            title={t('total_work_orders')}
            value={kpiData.total}
            icon={FileText}
            iconBgColor="bg-blue-50"
            iconColor="text-blue-600"
          />
          <StatsCard
            title={t('within_sla')}
            value={kpiData.within}
            subtitle={`${kpiData.compliance.toFixed(1)}% ${t('compliance')}`}
            icon={CheckCircle}
            iconBgColor="bg-emerald-50"
            iconColor="text-emerald-600"
          />
          <StatsCard
            title={t('breached')}
            value={kpiData.breached}
            icon={XCircle}
            iconBgColor="bg-red-50"
            iconColor="text-red-600"
          />
          <StatsCard
            title={t('at_risk')}
            value={kpiData.atRisk}
            icon={Clock}
            iconBgColor="bg-amber-50"
            iconColor="text-amber-600"
          />
        </div>
      )}

      {reports.length > 0 && (
        <Card className="mb-6">
          <div className="flex items-center gap-2 mb-6">
            <Activity className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <h3 className="text-lg font-semibold">{t('overall_sla_compliance')}</h3>
          </div>
          <div className="flex flex-wrap items-center justify-center gap-8">
            <Gauge
              value={overallCompliance}
              max={100}
              label={t('overall_compliance')}
              size="lg"
              thresholds={GAUGE_THRESHOLDS}
              unit="%"
            />
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              {reports.map((r) => (
                <Gauge
                  key={r.priority}
                  value={r.compliance_percent}
                  max={100}
                  label={t(r.priority)}
                  size="md"
                  thresholds={GAUGE_THRESHOLDS}
                  unit="%"
                />
              ))}
            </div>
          </div>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <h3 className="text-lg font-semibold mb-4">{t('sla_configuration')}</h3>
          <DataGrid data={configs} columns={configColumns} keyExtractor={(item) => item.id} loading={loading} variant="striped" defaultDensity="compact" pageSize={10} exportFilename="sla-config.csv" />
        </Card>

        <Card>
          <h3 className="text-lg font-semibold mb-4">{t('sla_compliance_30d')}</h3>
          <DataGrid data={reports} columns={reportColumns} keyExtractor={(item) => item.priority} loading={loading} variant="striped" defaultDensity="standard" pageSize={10} exportFilename="sla-compliance.csv" />
        </Card>
      </div>
    </div>
  );
};
