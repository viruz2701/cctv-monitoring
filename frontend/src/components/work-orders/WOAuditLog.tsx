import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { api, AuditLogEntry } from '../../services/api';
import { Timeline, TimelineEvent, DiffEntry } from '../ui/Timeline';
import { Card, CardHeader, CardBody, Badge, Button } from '../ui';
import {
  Clock, Download, Filter, X, Loader2, AlertTriangle,
} from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// Типы событий для фильтрации
// ═══════════════════════════════════════════════════════════════════════

const EVENT_TYPE_CHIPS = [
  { value: '', label: 'Все' },
  { value: 'STATUS_CHANGE', label: 'Смена статуса', color: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400' },
  { value: 'UPDATE',        label: 'Изменение',     color: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  { value: 'ASSIGN',        label: 'Назначение',    color: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { value: 'NOTE',          label: 'Заметка',       color: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400' },
  { value: 'PART_USED',     label: 'Запчасть',      color: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400' },
  { value: 'PHOTO',         label: 'Фото',          color: 'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-400' },
  { value: 'SYSTEM',        label: 'Система',       color: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { value: 'CREATE',        label: 'Создание',      color: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' },
];

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

const ACTION_TO_TIMELINE_TYPE: Record<string, TimelineEvent['type']> = {
  CREATE:       'system',
  UPDATE:       'status_change',
  DELETE:       'status_change',
  STATUS_CHANGE: 'status_change',
  ASSIGN:       'assignment',
  NOTE:         'note',
  PART_USED:    'part_used',
  PHOTO:        'photo',
  SYSTEM:       'system',
};

function mapActionToTimelineType(action: string): TimelineEvent['type'] {
  return ACTION_TO_TIMELINE_TYPE[action] || 'system';
}

function buildDiffEntries(entry: AuditLogEntry): DiffEntry[] | undefined {
  if (!entry.old_value && !entry.new_value) return undefined;
  const allKeys = new Set([
    ...Object.keys(entry.old_value || {}),
    ...Object.keys(entry.new_value || {}),
  ]);
  const diffs: DiffEntry[] = [];
  for (const key of allKeys) {
    const oldVal = entry.old_value?.[key];
    const newVal = entry.new_value?.[key];
    if (JSON.stringify(oldVal) !== JSON.stringify(newVal)) {
      diffs.push({
        field: key,
        oldValue: oldVal != null ? String(oldVal) : null,
        newValue: newVal != null ? String(newVal) : null,
      });
    }
  }
  return diffs.length > 0 ? diffs : undefined;
}

function actionToLabel(action: string): string {
  const map: Record<string, string> = {
    CREATE: 'Создание',
    UPDATE: 'Изменение',
    DELETE: 'Удаление',
    STATUS_CHANGE: 'Смена статуса',
    ASSIGN: 'Назначение',
    NOTE: 'Добавлена заметка',
    PART_USED: 'Использована запчасть',
    PHOTO: 'Добавлено фото',
    SYSTEM: 'Системное событие',
  };
  return map[action] || action;
}

// ═══════════════════════════════════════════════════════════════════════
// Props
// ═══════════════════════════════════════════════════════════════════════

interface WOAuditLogProps {
  workOrderId: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export const WOAuditLog: React.FC<WOAuditLogProps> = ({ workOrderId }) => {
  const { t } = useTranslation();

  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState('');

  // ── Load audit log ──────────────────────────────────────────────────

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getAuditLog({
        entity_type: 'work_order',
        entity_id: workOrderId,
        limit: 200,
      });
      setEntries(data || []);
    } catch (err: any) {
      setError(err.message || 'Не удалось загрузить историю');
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [workOrderId]);

  useEffect(() => {
    load();
  }, [load]);

  // ── Filter ──────────────────────────────────────────────────────────

  const filtered = filter
    ? entries.filter((e) => e.action === filter)
    : entries;

  // ── Map to Timeline events ──────────────────────────────────────────

  const timelineEvents: TimelineEvent[] = filtered.map((entry, i) => ({
    id: entry.id || `audit-${i}`,
    timestamp: entry.timestamp,
    type: mapActionToTimelineType(entry.action),
    title: actionToLabel(entry.action),
    description: entry.entity_id
      ? `${entry.entity_type || 'объект'}: ${entry.entity_id}`
      : undefined,
    user: entry.user_id,
    diff: buildDiffEntries(entry),
    details: entry.ip_address ? (
      <div className="flex items-center gap-2 text-xs text-slate-500">
        <span>IP: {entry.ip_address}</span>
      </div>
    ) : undefined,
  }));

  // ── CSV Export ──────────────────────────────────────────────────────

  const exportCSV = () => {
    if (filtered.length === 0) return;

    const headers = ['ID', 'Timestamp', 'Action', 'User ID', 'Entity Type', 'Entity ID', 'IP Address'];
    const rows = filtered.map((e) => [
      e.id,
      new Date(e.timestamp).toISOString(),
      e.action,
      e.user_id || '',
      e.entity_type || '',
      e.entity_id || '',
      e.ip_address || '',
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map((r) => r.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(',')),
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `wo-audit-${workOrderId}-${new Date().toISOString().slice(0, 10)}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  };

  // ── Render ──────────────────────────────────────────────────────────

  if (loading) {
    return (
      <Card>
        <CardBody>
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-6 h-6 animate-spin text-blue-600" />
          </div>
        </CardBody>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardBody>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <AlertTriangle className="w-8 h-8 text-red-500 mb-2" />
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            <Button variant="outline" size="sm" className="mt-3" onClick={load}>
              Повторить
            </Button>
          </div>
        </CardBody>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div className="flex items-center gap-2">
          <Clock className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span className="font-medium">История изменений</span>
          <Badge variant="info" size="sm">{filtered.length}</Badge>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            icon={<Download className="w-4 h-4" />}
            onClick={exportCSV}
            disabled={filtered.length === 0}
          >
            CSV
          </Button>
        </div>
      </CardHeader>

      {/* Filter chips */}
      <div className="px-6 pb-4 flex flex-wrap gap-1.5">
        {EVENT_TYPE_CHIPS.map((chip) => (
          <button
            key={chip.value}
            onClick={() => setFilter(chip.value === filter ? '' : chip.value)}
            className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium transition-all ${
              filter === chip.value
                ? 'ring-2 ring-blue-500 ring-offset-1 dark:ring-offset-slate-900 ' + (chip.color || 'bg-blue-100 text-blue-700')
                : chip.color || 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'
            }`}
          >
            {chip.label}
            {filter === chip.value && (
              <X className="w-3 h-3" />
            )}
          </button>
        ))}
        {filter && (
          <button
            onClick={() => setFilter('')}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
          >
            <Filter className="w-3 h-3" />
            Сбросить
          </button>
        )}
      </div>

      <CardBody>
        {filtered.length === 0 ? (
          <div className="text-center py-8">
            <Clock className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {filter ? 'Нет событий по выбранному фильтру' : 'История изменений пуста'}
            </p>
          </div>
        ) : (
          <Timeline events={timelineEvents} />
        )}
      </CardBody>
    </Card>
  );
};
