import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import {
  Card,
  CardHeader,
  CardBody,
  Badge,
  Button,
  Modal,
  Input,
  StatsCard,
  SkeletonTable,
  EmptyState,
  useToast,
} from '../components/ui';
import {
  HardDrive,
  AlertTriangle,
  Clock,
  Download,
  Trash2,
  Eye,
  Plus,
  Search,
  Activity,
  FileText,
  Camera,
  Archive,
  X,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface IncidentReportListItem {
  id: string;
  device_id: string;
  device_name?: string;
  site_id?: string;
  triggered_by: string;
  trigger_ref?: string;
  timestamp: string;
  recording_status: string;
  status: string;
  alert_count: number;
  log_count: number;
}

interface ListReportsResponse {
  reports: IncidentReportListItem[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface AlarmSnapshot {
  timestamp: string;
  priority: number;
  description?: string;
  method?: number;
}

interface LogSnapshot {
  time: string;
  level: string;
  message?: string;
  source?: string;
}

interface DowntimeSnapshot {
  started_at: string;
  ended_at?: string;
  duration_minutes: number;
  reason: string;
  description?: string;
}

interface IncidentReport {
  id: string;
  device_id: string;
  device_name?: string;
  site_id?: string;
  triggered_by: string;
  trigger_ref?: string;
  timestamp: string;
  device_snapshot: Record<string, unknown>;
  recent_alerts: AlarmSnapshot[];
  recent_logs: LogSnapshot[];
  recording_status: string;
  downtime_history: DowntimeSnapshot[];
  sla_data: Record<string, unknown>;
  photos: string[];
  notes: string;
  status: string;
  created_at: string;
  updated_at: string;
}

interface TriggerIncidentResponse {
  report_id: string;
  status: string;
  timestamp: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

type TLabelFn = (k: string) => string;

function getTriggerLabel(t: TLabelFn, key?: string): { label: string; color: string } {
  const labels: Record<string, { label: string; color: string }> = {
    alarm: { label: t('trigger_alarm'), color: 'bg-red-100 text-red-700 dark:bg-red-900/20 dark:text-red-400' },
    manual: { label: t('trigger_manual'), color: 'bg-blue-100 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400' },
    sla_breach: { label: t('trigger_sla_breach'), color: 'bg-orange-100 text-orange-700 dark:bg-orange-900/20 dark:text-orange-400' },
    downtime: { label: t('trigger_downtime'), color: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400' },
  };
  return labels[key ?? ''] ?? { label: key ?? t('unknown_label'), color: 'bg-slate-100 text-slate-600' };
}

function getStatusLabel(t: TLabelFn, key?: string): { label: string; color: string } {
  const labels: Record<string, { label: string; color: string }> = {
    draft: { label: t('status_draft'), color: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300' },
    finalized: { label: t('status_finalized'), color: 'bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400' },
    archived: { label: t('status_archived'), color: 'bg-purple-100 text-purple-700 dark:bg-purple-900/20 dark:text-purple-400' },
  };
  return labels[key ?? ''] ?? { label: key ?? t('unknown_label'), color: 'bg-slate-100 text-slate-600' };
}

function getPriorityLabel(t: TLabelFn, priority: number): { label: string; color: string } {
  switch (priority) {
    case 1: return { label: t('priority_low'), color: 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400' };
    case 2: return { label: t('priority_medium'), color: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400' };
    case 3: return { label: t('priority_high'), color: 'bg-red-100 text-red-700 dark:bg-red-900/20 dark:text-red-400' };
    default: return { label: `${t('unknown_label')} (${priority})`, color: 'bg-slate-100 text-slate-600' };
  }
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return d.toLocaleString('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return dateStr;
  }
}

function formatDuration(minutes: number): string {
  if (minutes < 60) return `${minutes} min`;
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

// ═══════════════════════════════════════════════════════════════════════
// Trigger Modal Component
// ═══════════════════════════════════════════════════════════════════════

interface TriggerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onTriggered: () => void;
}

function TriggerModal({ isOpen, onClose, onTriggered }: TriggerModalProps) {
  const { t } = useTranslation();
  const toast = useToast();
  const [deviceId, setDeviceId] = useState('');
  const [notes, setNotes] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!deviceId.trim()) {
      toast.error(t('device_id_required'));
      return;
    }

    setLoading(true);
    try {
      const resp = await request<TriggerIncidentResponse>('/blackbox/trigger', {
        method: 'POST',
        body: JSON.stringify({
          device_id: deviceId.trim(),
          notes: notes.trim(),
          trigger_type: 'manual',
        }),
      });
      toast.success(t('incident_triggered', { id: resp.report_id }));
      setDeviceId('');
      setNotes('');
      onClose();
      onTriggered();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('trigger_failed'));
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-white dark:bg-slate-800 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 max-w-lg w-full mx-4 p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white">{t('trigger_manual_incident')}</h2>
          <button onClick={onClose} className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label={t('device_id_label')}
            value={deviceId}
            onChange={(e) => setDeviceId(e.target.value)}
            placeholder={t('enter_device_uuid')}
            required
            disabled={loading}
          />
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              {t('notes_optional')}
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t('notes_placeholder')}
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg
                bg-white dark:bg-slate-800 text-slate-900 dark:text-white
                placeholder:text-slate-400 dark:placeholder:text-slate-500
                focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
                disabled:opacity-50 disabled:cursor-not-allowed
                transition-colors resize-none"
              rows={3}
              disabled={loading}
            />
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-700 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
            >
              {t('cancel')}
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {loading ? t('triggering') : t('trigger_incident')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Detail Modal Component
// ═══════════════════════════════════════════════════════════════════════

interface DetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  reportId: string | null;
  onDeleted: () => void;
}

function DetailModal({ isOpen, onClose, reportId, onDeleted }: DetailModalProps) {
  const { t } = useTranslation();
  const toast = useToast();
  const [report, setReport] = useState<IncidentReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'alerts' | 'logs' | 'downtime' | 'sla'>('overview');

  useEffect(() => {
    if (isOpen && reportId) {
      setLoading(true);
      setActiveTab('overview');
      request<IncidentReport>(`/blackbox/reports/${reportId}`)
        .then(setReport)
        .catch((err) => toast.error(err instanceof Error ? err.message : t('load_report_failed')))
        .finally(() => setLoading(false));
    }
  }, [isOpen, reportId, toast, t]);

  const handleExport = async (format: 'json' | 'pdf') => {
    if (!reportId) return;
    try {
      const url = `/api/v1/blackbox/reports/${reportId}/export?format=${format}`;
      window.open(url, '_blank');
    } catch {
      toast.error(t('export_failed'));
    }
  };

  const handleDelete = async () => {
    if (!reportId || !confirm(t('delete_confirm'))) return;
    try {
      await request(`/blackbox/reports/${reportId}`, { method: 'DELETE' });
      toast.success(t('report_deleted'));
      onClose();
      onDeleted();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('delete_failed'));
    }
  };

  const triggerInfo = getTriggerLabel(t, report?.triggered_by);
  const statusInfo = getStatusLabel(t, report?.status);

  const TABS = [
    { key: 'overview' as const, label: t('tab_overview') },
    { key: 'alerts' as const, label: t('tab_alerts', { count: report?.recent_alerts?.length ?? 0 }) },
    { key: 'logs' as const, label: t('tab_logs', { count: report?.recent_logs?.length ?? 0 }) },
    { key: 'downtime' as const, label: t('tab_downtime', { count: report?.downtime_history?.length ?? 0 }) },
    { key: 'sla' as const, label: t('tab_sla') },
  ];

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-white dark:bg-slate-800 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 max-w-4xl w-full mx-4 max-h-[85vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 px-6 py-4 flex items-start justify-between z-10">
          <div className="space-y-1">
            <div className="flex items-center gap-3">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                {report?.device_name || report?.device_id || t('report_detail')}
              </h3>
              {report && (
                <>
                  <Badge className={triggerInfo.color}>{triggerInfo.label}</Badge>
                  <Badge className={statusInfo.color}>{statusInfo.label}</Badge>
                </>
              )}
            </div>
            {report && (
              <p className="text-sm text-slate-500 dark:text-slate-400">
                ID: {report.id} · {formatDate(report.timestamp)}
                {report.trigger_ref && ` · Ref: ${report.trigger_ref}`}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => handleExport('json')}
              className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-700 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
            >
              <Download className="w-4 h-4" /> {t('export_json')}
            </button>
            <button
              onClick={() => handleExport('pdf')}
              className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-700 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
            >
              <Download className="w-4 h-4" /> {t('bb_export_pdf')}
            </button>
            <button
              onClick={handleDelete}
              className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors"
            >
              <Trash2 className="w-4 h-4" /> {t('delete_action')}
            </button>
            <button onClick={onClose} className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded">
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        <div className="p-6">
          {loading ? (
            <SkeletonTable rows={5} />
          ) : report ? (
            <div className="space-y-6">
              {/* Tabs */}
              <div className="flex gap-1 border-b border-slate-200 dark:border-slate-700">
                {TABS.map((tab) => (
                  <button
                    key={tab.key}
                    onClick={() => setActiveTab(tab.key)}
                    className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                      activeTab === tab.key
                        ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                        : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>

              {/* Tab Content */}
              <div className="min-h-[200px]">
                {activeTab === 'overview' && (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Card>
                      <CardHeader>{t('device_snapshot')}</CardHeader>
                      <CardBody>
                        <pre className="text-xs text-slate-600 dark:text-slate-400 overflow-auto max-h-60 font-mono bg-slate-50 dark:bg-slate-800/50 p-3 rounded-lg">
                          {JSON.stringify(report.device_snapshot, null, 2)}
                        </pre>
                      </CardBody>
                    </Card>
                    <Card>
                      <CardHeader>{t('bb_recording_status')}</CardHeader>
                      <CardBody>
                        <div className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
                          {report.recording_status || t('unknown_label')}
                        </div>
                        {report.notes && (
                          <div className="mt-4">
                            <h4 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">{t('notes_section')}</h4>
                            <p className="text-sm text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-800/50 p-3 rounded-lg">
                              {report.notes}
                            </p>
                          </div>
                        )}
                      </CardBody>
                    </Card>
                  </div>
                )}

                {activeTab === 'alerts' && (
                  <>
                    {report.recent_alerts.length === 0 ? (
                      <div className="py-12 text-center">
                        <AlertTriangle className="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600 mb-3" />
                        <p className="text-sm font-medium text-slate-500 dark:text-slate-400">{t('bb_no_alerts')}</p>
                      </div>
                    ) : (
                      <div className="space-y-2">
                        {report.recent_alerts.map((alert, idx) => {
                          const p = getPriorityLabel(t, alert.priority);
                          return (
                            <div key={idx} className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                              <div className="flex items-center gap-3">
                                <Badge className={p.color}>{p.label}</Badge>
                                <div>
                                  <p className="text-sm text-slate-700 dark:text-slate-300">{alert.description || 'No description'}</p>
                                  <p className="text-xs text-slate-400">{formatDate(alert.timestamp)}</p>
                                </div>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </>
                )}

                {activeTab === 'logs' && (
                  <>
                    {report.recent_logs.length === 0 ? (
                      <div className="py-12 text-center">
                        <FileText className="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600 mb-3" />
                        <p className="text-sm font-medium text-slate-500 dark:text-slate-400">{t('no_logs')}</p>
                      </div>
                    ) : (
                      <div className="space-y-1 max-h-80 overflow-y-auto">
                        {report.recent_logs.map((log, idx) => (
                          <div key={idx} className="flex items-start gap-3 p-2 text-xs font-mono hover:bg-slate-50 dark:hover:bg-slate-800/30 rounded">
                            <span className="text-slate-400 w-16 flex-shrink-0">{log.level}</span>
                            <span className="text-slate-500 w-24 flex-shrink-0">{formatDate(log.time)}</span>
                            <span className="text-slate-600 dark:text-slate-400">{log.message}</span>
                            {log.source && <span className="text-slate-400 ml-auto">[{log.source}]</span>}
                          </div>
                        ))}
                      </div>
                    )}
                  </>
                )}

                {activeTab === 'downtime' && (
                  <>
                    {report.downtime_history.length === 0 ? (
                      <div className="py-12 text-center">
                        <Clock className="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600 mb-3" />
                        <p className="text-sm font-medium text-slate-500 dark:text-slate-400">{t('no_downtime')}</p>
                      </div>
                    ) : (
                      <div className="space-y-2">
                        {report.downtime_history.map((dt, idx) => (
                          <div key={idx} className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                            <div>
                              <p className="text-sm font-medium text-slate-700 dark:text-slate-300 capitalize">{dt.reason}</p>
                              <p className="text-xs text-slate-400">{dt.description}</p>
                              <p className="text-xs text-slate-400">
                                {formatDate(dt.started_at)} → {dt.ended_at ? formatDate(dt.ended_at) : t('ongoing')}
                              </p>
                            </div>
                            <Badge className="bg-red-100 text-red-700 dark:bg-red-900/20 dark:text-red-400">
                              {formatDuration(dt.duration_minutes)}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    )}
                  </>
                )}

                {activeTab === 'sla' && (
                  <Card>
                    <CardBody>
                      <pre className="text-xs text-slate-600 dark:text-slate-400 overflow-auto max-h-60 font-mono bg-slate-50 dark:bg-slate-800/50 p-3 rounded-lg">
                        {JSON.stringify(report.sla_data, null, 2)}
                      </pre>
                    </CardBody>
                  </Card>
                )}
              </div>
            </div>
          ) : (
            <div className="py-12 text-center">
              <HardDrive className="w-12 h-12 mx-auto text-slate-300 dark:text-slate-600 mb-3" />
              <p className="text-sm font-medium text-slate-500 dark:text-slate-400">{t('report_not_found')}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main BlackBox Page Component
// ═══════════════════════════════════════════════════════════════════════

export function BlackBox() {
  const { t } = useTranslation();
  const toast = useToast();

  const [reports, setReports] = useState<IncidentReportListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [loading, setLoading] = useState(true);

  // Filters
  const [filterDeviceId, setFilterDeviceId] = useState('');
  const [filterTrigger, setFilterTrigger] = useState('');

  // Modals
  const [triggerModalOpen, setTriggerModalOpen] = useState(false);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null);

  const fetchReports = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.set('limit', String(pageSize));
      params.set('offset', String((page - 1) * pageSize));
      if (filterDeviceId.trim()) params.set('device_id', filterDeviceId.trim());
      if (filterTrigger) params.set('trigger', filterTrigger);

      const resp = await request<ListReportsResponse>(`/blackbox/reports?${params.toString()}`);
      setReports(resp.reports);
      setTotal(resp.total);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load reports');
      setReports([]);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filterDeviceId, filterTrigger, toast]);

  useEffect(() => {
    fetchReports();
  }, [fetchReports]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleViewDetail = (id: string) => {
    setSelectedReportId(id);
    setDetailModalOpen(true);
  };

  const handleRefresh = () => {
    fetchReports();
  };

  // Stats
  const stats = useMemo(() => {
    let totalAlerts = 0;
    let totalLogs = 0;
    let draftCount = 0;
    for (const r of reports) {
      totalAlerts += r.alert_count;
      totalLogs += r.log_count;
      if (r.status === 'draft') draftCount++;
    }
    return { totalAlerts, totalLogs, draftCount };
  }, [reports]);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Archive className="w-6 h-6 text-blue-500" />
            {t('blackbox_title')}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('blackbox_desc')}
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleRefresh}
            disabled={loading}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-slate-600 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 transition-colors"
          >
            <Activity className="w-4 h-4" /> {t('refresh_action')}
          </button>
          <button
            onClick={() => setTriggerModalOpen(true)}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Plus className="w-4 h-4" /> {t('trigger_manual_action')}
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard title={t('total_reports')} value={total} icon={Archive} iconColor="text-blue-600" iconBgColor="bg-blue-50" />
        <StatsCard title={t('draft_reports')} value={stats.draftCount} icon={FileText} iconColor="text-amber-600" iconBgColor="bg-amber-50" />
        <StatsCard title={t('blackbox_total_alerts')} value={stats.totalAlerts} icon={AlertTriangle} iconColor="text-red-600" iconBgColor="bg-red-50" />
        <StatsCard title={t('total_logs')} value={stats.totalLogs} icon={FileText} iconColor="text-purple-600" iconBgColor="bg-purple-50" />
      </div>

      {/* Filters */}
      <Card>
        <CardBody>
          <div className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-[200px]">
              <Input
                label={t('device_id_label')}
                value={filterDeviceId}
                onChange={(e) => { setFilterDeviceId(e.target.value); setPage(1); }}
                placeholder={t('filter_by_device_id')}
              />
            </div>
            <div className="w-40">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {t('trigger_label')}
              </label>
              <select
                value={filterTrigger}
                onChange={(e) => { setFilterTrigger(e.target.value); setPage(1); }}
                className="w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">{t('filter_all')}</option>
                <option value="alarm">{t('filter_alarm')}</option>
                <option value="manual">{t('filter_manual')}</option>
                <option value="sla_breach">{t('filter_sla_breach')}</option>
                <option value="downtime">{t('filter_downtime')}</option>
              </select>
            </div>
            <button
              onClick={() => { setFilterDeviceId(''); setFilterTrigger(''); setPage(1); }}
              className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              {t('bb_clear_filters')}
            </button>
          </div>
        </CardBody>
      </Card>

      {/* Reports Table */}
      <Card>
        <CardBody className="p-0">
          {loading ? (
            <SkeletonTable rows={8} />
          ) : reports.length === 0 ? (
            <EmptyState
              icon={<Archive className="w-12 h-12" />}
              title={t('no_incidents')}
              description={t('no_incidents_desc')}
              action={{ label: t('trigger_manual_action'), onClick: () => setTriggerModalOpen(true) }}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('device_col')}</th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('timestamp_col')}</th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('trigger_col')}</th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('status_col')}</th>
                    <th className="text-center px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('alerts_col')}</th>
                    <th className="text-center px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('logs_col')}</th>
                    <th className="text-right px-4 py-3 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('actions_col')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                  {reports.map((report) => {
                    const triggerInfo = getTriggerLabel(t, report.triggered_by);
                    const statusInfo = getStatusLabel(t, report.status);
                    return (
                      <tr key={report.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/30 transition-colors">
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Camera className="w-4 h-4 text-slate-400" />
                            <div>
                              <p className="text-sm font-medium text-slate-900 dark:text-white">
                                {report.device_name || report.device_id}
                              </p>
                              {report.device_name && report.device_id !== report.device_name && (
                                <p className="text-xs text-slate-400">{report.device_id}</p>
                              )}
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-400">
                          {formatDate(report.timestamp)}
                        </td>
                        <td className="px-4 py-3">
                          <Badge className={triggerInfo.color}>{triggerInfo.label}</Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge className={statusInfo.color}>{statusInfo.label}</Badge>
                        </td>
                        <td className="px-4 py-3 text-center text-sm text-slate-600 dark:text-slate-400">
                          {report.alert_count}
                        </td>
                        <td className="px-4 py-3 text-center text-sm text-slate-600 dark:text-slate-400">
                          {report.log_count}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => handleViewDetail(report.id)}
                            className="p-1.5 text-slate-400 hover:text-blue-600 dark:hover:text-blue-400 rounded hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                          >
                            <Eye className="w-4 h-4" />
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between px-4 py-3 bg-white dark:bg-slate-800 border-t border-slate-200 dark:border-slate-700 rounded-lg">
          <div className="text-sm text-slate-500 dark:text-slate-300">
            {t('page_info', { page, totalPages, total })}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
              className="px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t('previous_page')}
            </button>
            <span className="text-sm text-slate-500 dark:text-slate-400 px-2">{page}</span>
            <button
              onClick={() => setPage(Math.min(totalPages, page + 1))}
              disabled={page === totalPages}
              className="px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t('next_page')}
            </button>
          </div>
        </div>
      )}

      {/* Modals */}
      <TriggerModal
        isOpen={triggerModalOpen}
        onClose={() => setTriggerModalOpen(false)}
        onTriggered={handleRefresh}
      />
      <DetailModal
        isOpen={detailModalOpen}
        onClose={() => { setDetailModalOpen(false); setSelectedReportId(null); }}
        reportId={selectedReportId}
        onDeleted={handleRefresh}
      />
    </div>
  );
}
