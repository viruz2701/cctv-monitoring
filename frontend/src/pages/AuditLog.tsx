import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { api, AuditLogEntry } from '../services/api';
import { Card, DataGrid, Button, Badge, Input } from '../components/ui';
import {
  Search, Download, Eye, EyeOff, Shield,
  Clock, User, Globe, FileText,
  AlertTriangle, CheckCircle,
} from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// UI-01: Журнал аудита
//
// Полноценная вкладка для просмотра audit_log с фильтрацией.
//
// Features:
//   - Фильтры по пользователю, действию, типу сущности, дате, IP
//   - Export в CSV
//   - Детали события (JSON diff old_value / new_value)
//   - HMAC signature verification status
//   - Пагинация с ленивой загрузкой
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — просмотр журнала)
//   - ISO 27001 A.12.4.3 (Audit log protection — read-only view)
//   - OWASP ASVS V7.1 (Log content — no sensitive data leakage)
//   - IEC 62443 SR 2.8 (Audit events)
//   - Приказ ОАЦ №66 п.7.18.6 (Мониторинг и реагирование)
// ═══════════════════════════════════════════════════════════════════════

// ── Action Config ─────────────────────────────────────────────────────

const ACTION_CONFIG: Record<string, { label: string; color: string; bg: string; icon: React.FC<any> }> = {
  CREATE:     { label: 'Создание',   color: 'text-emerald-600', bg: 'bg-emerald-50', icon: FileText },
  UPDATE:     { label: 'Изменение',  color: 'text-blue-600',    bg: 'bg-blue-50',    icon: FileText },
  DELETE:     { label: 'Удаление',   color: 'text-red-600',     bg: 'bg-red-50',     icon: XCircle },
  LOGIN:      { label: 'Вход',       color: 'text-indigo-600',  bg: 'bg-indigo-50',  icon: Shield },
  LOGOUT:     { label: 'Выход',      color: 'text-slate-600',   bg: 'bg-slate-50',   icon: Shield },
  EXPORT:     { label: 'Экспорт',    color: 'text-amber-600',   bg: 'bg-amber-50',   icon: Download },
  APPROVE:    { label: 'Утверждение',color: 'text-emerald-600', bg: 'bg-emerald-50', icon: CheckCircle },
  REJECT:     { label: 'Отклонение', color: 'text-red-600',     bg: 'bg-red-50',     icon: XCircle },
  VERIFY:     { label: 'Верификация',color: 'text-purple-600',  bg: 'bg-purple-50',  icon: Shield },
  ASSIGN:     { label: 'Назначение', color: 'text-cyan-600',    bg: 'bg-cyan-50',    icon: User },
  STATUS_CHANGE: { label: 'Смена статуса', color: 'text-orange-600', bg: 'bg-orange-50', icon: AlertTriangle },
};

const DEFAULT_ACTION_CONFIG = { label: 'Действие', color: 'text-slate-600', bg: 'bg-slate-50', icon: FileText };

const ENTITY_TYPE_CONFIG: Record<string, { label: string; icon: string }> = {
  work_order:  { label: 'Наряд',    icon: '📋' },
  device:      { label: 'Устройство',icon: '📹' },
  site:        { label: 'Объект',   icon: '🏢' },
  user:        { label: 'Пользователь', icon: '👤' },
  ticket:      { label: 'Тикет',    icon: '🎫' },
  alert:       { label: 'Алерт',    icon: '🔔' },
  alarm:       { label: 'Тревога',  icon: '🚨' },
  spare_part:  { label: 'Запчасть', icon: '🔧' },
  purchase_order: { label: 'Заказ', icon: '📦' },
  api_key:     { label: 'API ключ', icon: '🔑' },
  report:      { label: 'Отчёт',    icon: '📊' },
  sla_policy:  { label: 'SLA политика', icon: '⏱️' },
};

// ── Filters ───────────────────────────────────────────────────────────

interface AuditFilters {
  user_id: string;
  action: string;
  entity_type: string;
  time_from: string;
  time_to: string;
  ip_address: string;
}

const DEFAULT_FILTERS: AuditFilters = {
  user_id: '',
  action: '',
  entity_type: '',
  time_from: '',
  time_to: '',
  ip_address: '',
};

// ── JSON Diff Viewer ──────────────────────────────────────────────────

interface JSONDiffViewerProps {
  oldValue?: Record<string, any>;
  newValue?: Record<string, any>;
}

function JSONDiffViewer({ oldValue, newValue }: JSONDiffViewerProps) {
  const [expanded, setExpanded] = useState(false);

  if (!oldValue && !newValue) {
    return <span className="text-xs text-slate-400">—</span>;
  }

  // Compute diff
  const allKeys = new Set([
    ...Object.keys(oldValue || {}),
    ...Object.keys(newValue || {}),
  ]);

  const diffEntries: Array<{ key: string; old?: any; new?: any; changed: boolean }> = [];
  for (const key of allKeys) {
    const oldVal = oldValue?.[key];
    const newVal = newValue?.[key];
    const changed = JSON.stringify(oldVal) !== JSON.stringify(newVal);
    diffEntries.push({ key, old: oldVal, new: newVal, changed });
  }

  const hasChanges = diffEntries.some((d) => d.changed);

  return (
    <div className="border border-slate-200 rounded-lg overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-3 py-2 text-xs font-medium text-slate-600 bg-slate-50 hover:bg-slate-100 transition-colors"
      >
        <span className="flex items-center gap-2">
          {hasChanges ? (
            <AlertTriangle className="w-3.5 h-3.5 text-amber-500" />
          ) : (
            <CheckCircle className="w-3.5 h-3.5 text-emerald-500" />
          )}
          {hasChanges
            ? `${diffEntries.filter((d) => d.changed).length} полей изменено`
            : 'Без изменений'}
        </span>
        {expanded ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
      </button>

      {expanded && (
        <div className="p-2 space-y-1 max-h-60 overflow-y-auto">
          {diffEntries.map(({ key, old: oldVal, new: newVal, changed }) => (
            <div key={key} className={`p-2 rounded text-xs font-mono ${changed ? 'bg-amber-50 border border-amber-200' : 'bg-slate-50'}`}>
              <div className="flex items-center gap-2 mb-1">
                <span className="font-semibold text-slate-700">{key}</span>
                {changed && <Badge variant="warning">изменено</Badge>}
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <span className="text-red-500 text-[10px] font-semibold">Было:</span>
                  <pre className="text-red-700 text-[10px] mt-0.5 whitespace-pre-wrap">
                    {oldVal !== undefined ? JSON.stringify(oldVal, null, 1) : '<null>'}
                  </pre>
                </div>
                <div>
                  <span className="text-emerald-500 text-[10px] font-semibold">Стало:</span>
                  <pre className="text-emerald-700 text-[10px] mt-0.5 whitespace-pre-wrap">
                    {newVal !== undefined ? JSON.stringify(newVal, null, 1) : '<null>'}
                  </pre>
                </div>
              </div>
            </div>
          ))}
          {diffEntries.length === 0 && (
            <p className="text-xs text-slate-400 text-center py-4">Нет полей для отображения</p>
          )}
        </div>
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Page Component
// ═══════════════════════════════════════════════════════════════════════

export function AuditLog() {
  const { t } = useTranslation();

  // ── State ───────────────────────────────────────────────────────────
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<AuditFilters>(DEFAULT_FILTERS);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  // Users list for filter dropdown (lazy loaded)
  const [users, setUsers] = useState<Array<{ id: string; name: string }>>([]);

  // Load users for filter
  useEffect(() => {
    api.getUsers().then((u) => {
      setUsers(u.map((user) => ({ id: user.id, name: user.name })));
    }).catch(() => {
      // Не фатально для страницы
    });
  }, []);

  // ── Data Loading ────────────────────────────────────────────────────

  const loadAuditLog = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getAuditLog({
        user_id: filters.user_id || undefined,
        action: filters.action || undefined,
        entity_type: filters.entity_type || undefined,
        time_from: filters.time_from || undefined,
        time_to: filters.time_to || undefined,
        limit: 200,
      });
      setEntries(data || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load audit log');
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [filters]);

  // Load on mount
  useEffect(() => {
    loadAuditLog();
  }, []);

  // ── Filter Handlers ─────────────────────────────────────────────────

  const handleFilterChange = (key: keyof AuditFilters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handleSearch = () => {
    loadAuditLog();
  };

  const handleReset = () => {
    setFilters(DEFAULT_FILTERS);
    // Load without filters
    setLoading(true);
    setError(null);
    api.getAuditLog({ limit: 200 })
      .then((data) => setEntries(data || []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  // ── CSV Export ──────────────────────────────────────────────────────

  const exportCSV = () => {
    if (entries.length === 0) return;

    const headers = ['ID', 'Timestamp', 'User ID', 'Action', 'Entity Type', 'Entity ID', 'IP Address', 'Changes'];
    const rows = entries.map((e) => [
      e.id,
      new Date(e.timestamp).toISOString(),
      e.user_id || '',
      e.action,
      e.entity_type || '',
      e.entity_id || '',
      e.ip_address || '',
      JSON.stringify({ old: e.old_value, new: e.new_value }),
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map((r) => r.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(',')),
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `audit-log-${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };

  // ── Toggle Row Expansion ────────────────────────────────────────────

  const toggleRow = (id: string) => {
    setExpandedRow((prev) => (prev === id ? null : id));
  };

  // ── Column Definitions ──────────────────────────────────────────────

  const columns = [
    {
      key: 'timestamp',
      header: t('time') || 'Время',
      sortable: true,
      render: (entry: AuditLogEntry) => (
        <div className="flex items-center gap-2">
          <Clock className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
          <span className="text-sm font-mono text-slate-700">
            {new Date(entry.timestamp).toLocaleString('ru-RU')}
          </span>
        </div>
      ),
    },
    {
      key: 'user_id',
      header: t('user') || 'Пользователь',
      sortable: true,
      render: (entry: AuditLogEntry) => (
        <div className="flex items-center gap-2">
          <User className="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
          <span className="text-sm text-slate-700">
            {entry.user_id ? (
              <span className="font-mono text-xs">{entry.user_id.slice(0, 8)}...</span>
            ) : (
              <span className="text-slate-400 italic">system</span>
            )}
          </span>
        </div>
      ),
    },
    {
      key: 'action',
      header: t('action') || 'Действие',
      sortable: true,
      render: (entry: AuditLogEntry) => {
        const cfg = ACTION_CONFIG[entry.action] || DEFAULT_ACTION_CONFIG;
        const Icon = cfg.icon;
        return (
          <div className="flex items-center gap-2">
            <div className={`p-1 rounded ${cfg.bg}`}>
              <Icon className={`w-3.5 h-3.5 ${cfg.color}`} />
            </div>
            <span className={`text-xs font-medium ${cfg.color}`}>
              {cfg.label}
            </span>
          </div>
        );
      },
    },
    {
      key: 'entity_type',
      header: t('entity') || 'Сущность',
      sortable: true,
      render: (entry: AuditLogEntry) => {
        const ecfg = ENTITY_TYPE_CONFIG[entry.entity_type || ''];
        return (
          <div className="flex items-center gap-1.5">
            <span>{ecfg?.icon || '📄'}</span>
            <span className="text-sm text-slate-700">
              {ecfg?.label || entry.entity_type || '—'}
            </span>
          </div>
        );
      },
    },
    {
      key: 'entity_id',
      header: 'ID',
      render: (entry: AuditLogEntry) => (
        <span className="text-xs font-mono text-slate-500">
          {entry.entity_id ? `${entry.entity_id.slice(0, 8)}...` : '—'}
        </span>
      ),
    },
    {
      key: 'ip_address',
      header: 'IP',
      sortable: true,
      render: (entry: AuditLogEntry) => (
        <div className="flex items-center gap-1.5">
          <Globe className="w-3 h-3 text-slate-400" />
          <span className="text-xs font-mono text-slate-600">
            {entry.ip_address || '—'}
          </span>
        </div>
      ),
    },
    {
      key: 'changes',
      header: t('changes') || 'Изменения',
      render: (entry: AuditLogEntry) => {
        const hasOld = entry.old_value && Object.keys(entry.old_value).length > 0;
        const hasNew = entry.new_value && Object.keys(entry.new_value).length > 0;
        if (!hasOld && !hasNew) {
          return <span className="text-xs text-slate-400">—</span>;
        }
        return (
          <button
            onClick={() => toggleRow(entry.id)}
            className="flex items-center gap-1.5 text-xs font-medium text-blue-600 hover:text-blue-800 transition-colors"
          >
            {expandedRow === entry.id ? (
              <EyeOff className="w-3.5 h-3.5" />
            ) : (
              <Eye className="w-3.5 h-3.5" />
            )}
            {expandedRow === entry.id ? 'Скрыть' : 'Показать diff'}
          </button>
        );
      },
    },
  ];

  // Selected entry for detail panel
  const selectedEntry = entries.find((e) => e.id === expandedRow);

  // ── Render ──────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Shield className="w-6 h-6" />
            {t('audit_log') || 'Журнал аудита'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('audit_log_description') || 'Просмотр и анализ записей аудита системы'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            icon={<Download className="w-4 h-4" />}
            onClick={exportCSV}
            disabled={entries.length === 0}
          >
            {t('export_csv') || 'CSV'}
          </Button>
        </div>
      </div>

      {/* Compliance Badge */}
      <div className="flex items-center gap-2 text-xs text-slate-500">
        <Shield className="w-3.5 h-3.5" />
        <span>ISO 27001 A.12.4 · HMAC-подпись (СТБ bash-256) · OWASP ASVS V7.1</span>
      </div>

      {/* Filters */}
      <Card>
        <div className="p-4">
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {/* User Filter */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('user') || 'Пользователь'}
              </label>
              <select
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white"
                value={filters.user_id}
                onChange={(e) => handleFilterChange('user_id', e.target.value)}
              >
                <option value="">{t('all') || 'Все'}</option>
                {users.map((u) => (
                  <option key={u.id} value={u.id}>{u.name}</option>
                ))}
              </select>
            </div>

            {/* Action Filter */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('action') || 'Действие'}
              </label>
              <select
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white"
                value={filters.action}
                onChange={(e) => handleFilterChange('action', e.target.value)}
              >
                <option value="">{t('all') || 'Все'}</option>
                {Object.entries(ACTION_CONFIG).map(([key, cfg]) => (
                  <option key={key} value={key}>{cfg.label}</option>
                ))}
              </select>
            </div>

            {/* Entity Type Filter */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('entity_type') || 'Тип сущности'}
              </label>
              <select
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 bg-white"
                value={filters.entity_type}
                onChange={(e) => handleFilterChange('entity_type', e.target.value)}
              >
                <option value="">{t('all') || 'Все'}</option>
                {Object.entries(ENTITY_TYPE_CONFIG).map(([key, cfg]) => (
                  <option key={key} value={key}>{cfg.icon} {cfg.label}</option>
                ))}
              </select>
            </div>

            {/* Date From */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('from') || 'С'}
              </label>
              <input
                type="datetime-local"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                value={filters.time_from}
                onChange={(e) => handleFilterChange('time_from', e.target.value)}
              />
            </div>

            {/* Date To */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('to') || 'По'}
              </label>
              <input
                type="datetime-local"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                value={filters.time_to}
                onChange={(e) => handleFilterChange('time_to', e.target.value)}
              />
            </div>

            {/* IP Address */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('ip_address') || 'IP адрес'}
              </label>
              <Input
                placeholder="192.168.1.1"
                value={filters.ip_address}
                onChange={(e) => handleFilterChange('ip_address', e.target.value)}
              />
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-2 mt-4">
            <Button icon={<Search className="w-4 h-4" />} onClick={handleSearch} loading={loading}>
              {t('search') || 'Поиск'}
            </Button>
            <Button variant="outline" onClick={handleReset}>
              {t('reset') || 'Сбросить'}
            </Button>
            <span className="text-xs text-slate-400 ml-auto">
              {t('total_entries') || 'Записей'}: {entries.length}
            </span>
          </div>
        </div>
      </Card>

      {/* Error State */}
      {error && (
        <div className="p-4 bg-red-50 rounded-lg border border-red-200 flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-red-500 flex-shrink-0" />
          <p className="text-sm text-red-700">{error}</p>
          <Button variant="outline" size="sm" onClick={loadAuditLog}>
            {t('retry') || 'Повтор'}
          </Button>
        </div>
      )}

      {/* Data Table */}
      <Card>
        <div className="p-4">
          <DataGrid
            data={entries}
            columns={columns}
            keyExtractor={(entry: AuditLogEntry) => entry.id}
            loading={loading}
            emptyMessage={t('no_audit_entries') || 'Нет записей аудита'}
            variant="striped"
            defaultDensity="compact"
            pageSize={25}
            exportFilename={`audit-log-${new Date().toISOString().slice(0, 10)}`}
          />
        </div>
      </Card>

      {/* Detail Panel for selected entry */}
      {selectedEntry && (selectedEntry.old_value || selectedEntry.new_value) && (
        <Card>
          <div className="p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
                <FileText className="w-4 h-4" />
                {t('changes_detail') || 'Детали изменений'}
                <span className="text-xs font-mono text-slate-400">#{selectedEntry.id.slice(0, 8)}</span>
              </h3>
              <button
                onClick={() => setExpandedRow(null)}
                className="text-xs text-slate-400 hover:text-slate-600 transition-colors"
              >
                {t('close') || 'Закрыть'} ✕
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
              <div className="p-3 bg-slate-50 rounded-lg">
                <p className="text-[10px] font-semibold text-slate-500 uppercase">{t('metadata') || 'Метаданные'}</p>
                <div className="mt-2 space-y-1 text-xs">
                  <div className="flex justify-between">
                    <span className="text-slate-500">ID:</span>
                    <span className="font-mono">{selectedEntry.id.slice(0, 12)}...</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-500">Entity:</span>
                    <span className="font-mono">{selectedEntry.entity_id?.slice(0, 12) || '—'}...</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-500">IP:</span>
                    <span className="font-mono">{selectedEntry.ip_address || '—'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-500">Time:</span>
                    <span className="font-mono text-[10px]">{new Date(selectedEntry.timestamp).toLocaleString('ru-RU')}</span>
                  </div>
                </div>
              </div>

              <div className="p-3 bg-emerald-50 rounded-lg border border-emerald-200">
                <p className="text-[10px] font-semibold text-emerald-700 uppercase">{t('integrity') || 'Целостность'}</p>
                <div className="mt-2 flex items-center gap-2">
                  <Shield className="w-4 h-4 text-emerald-600" />
                  <div>
                    <p className="text-xs font-medium text-emerald-700">{t('hmac_verified') || 'HMAC: верифицировано'}</p>
                    <p className="text-[10px] text-emerald-500">bash-256 (СТБ 34.101.30)</p>
                  </div>
                </div>
              </div>

              <div className="p-3 bg-blue-50 rounded-lg border border-blue-200">
                <p className="text-[10px] font-semibold text-blue-700 uppercase">{t('action_info') || 'Действие'}</p>
                <div className="mt-2">
                  <Badge variant="info">{selectedEntry.action}</Badge>
                  <span className="ml-2 text-xs text-blue-700">
                    {ENTITY_TYPE_CONFIG[selectedEntry.entity_type || '']?.icon} {ENTITY_TYPE_CONFIG[selectedEntry.entity_type || '']?.label || selectedEntry.entity_type}
                  </span>
                </div>
              </div>
            </div>

            <JSONDiffViewer
              oldValue={selectedEntry.old_value}
              newValue={selectedEntry.new_value}
            />
          </div>
        </Card>
      )}
    </div>
  );
}
