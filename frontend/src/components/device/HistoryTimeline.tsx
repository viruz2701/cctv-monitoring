// ═══════════════════════════════════════════════════════════════════════
// HistoryTimeline — вертикальная шкала времени событий устройства
// UX-2.5: Device History Timeline
//
// Особенности:
//   - Vertical timeline: alarms, WO, config changes, firmware updates
//   - Filter by event type
//   - Virtualized list (react-window)
//   - Export to CSV
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { FixedSizeList as List } from 'react-window';
import {
  AlertTriangle,
  Wrench,
  Download,
  Settings,
  RotateCcw,
  Info,
  Filter,
  XCircle,
  CheckCircle,
} from '../ui/Icons';
import { Button } from '../ui/Button';
import { Card, CardHeader, CardBody } from '../ui/Card';

// ── Types ────────────────────────────────────────────────────────────

export type HistoryEventType =
  | 'alarm'
  | 'work_order'
  | 'config_change'
  | 'firmware_update'
  | 'restart';

export type HistoryEventSeverity = 'info' | 'warning' | 'error' | 'success';

export interface HistoryEvent {
  id: string;
  deviceId: string;
  type: HistoryEventType;
  severity: HistoryEventSeverity;
  title: string;
  description?: string;
  timestamp: string;
  /** Optional link to related entity */
  relatedId?: string;
  /** Optional user who triggered the event */
  actor?: string;
}

interface HistoryTimelineProps {
  /** Array of history events */
  events: HistoryEvent[];
  /** Whether data is still loading */
  isLoading?: boolean;
  /** Device ID for context */
  deviceId?: string;
}

// ── Constants ────────────────────────────────────────────────────────

const EVENT_TYPE_CONFIG: Record<HistoryEventType, { icon: React.ElementType; label: string; color: string }> = {
  alarm: { icon: AlertTriangle, label: 'alarm', color: 'text-red-500 border-red-500' },
  work_order: { icon: Wrench, label: 'work_order', color: 'text-blue-500 border-blue-500' },
  config_change: { icon: Settings, label: 'config_change', color: 'text-purple-500 border-purple-500' },
  firmware_update: { icon: Download, label: 'firmware_update', color: 'text-amber-500 border-amber-500' },
  restart: { icon: RotateCcw, label: 'restart', color: 'text-orange-500 border-orange-500' },
};

const SEVERITY_BORDER: Record<HistoryEventSeverity, string> = {
  success: 'border-emerald-500',
  info: 'border-blue-500',
  warning: 'border-amber-500',
  error: 'border-red-500',
};

const ITEM_HEIGHT = 80; // px per row in virtualized list
const LIST_HEIGHT = 600; // max visible height

// ── Helpers ──────────────────────────────────────────────────────────

function formatDateTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

/** Parse severity from event type for backward compatibility */
function getSeverity(ev: HistoryEvent): HistoryEventSeverity {
  return ev.severity || (ev.type === 'alarm' ? 'error' : 'info');
}

/** Export events array to CSV and trigger download */
function exportToCSV(events: HistoryEvent[], filename = 'device-history.csv'): void {
  const headers = ['Type', 'Severity', 'Title', 'Description', 'Timestamp', 'Actor', 'Related ID'];
  const rows = events.map((ev) => [
    ev.type,
    getSeverity(ev),
    `"${ev.title.replace(/"/g, '""')}"`,
    `"${(ev.description || '').replace(/"/g, '""')}"`,
    ev.timestamp,
    ev.actor || '',
    ev.relatedId || '',
  ]);

  const csv = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// ── Row renderer (virtualized) ───────────────────────────────────────

interface RowProps {
  index: number;
  style: React.CSSProperties;
  data: {
    events: HistoryEvent[];
    t: (key: string) => string;
    locale: string;
  };
}

const TimelineRow = React.memo<RowProps>(({ index, style, data }) => {
  const ev = data.events[index];
  const config = EVENT_TYPE_CONFIG[ev.type];
  const Icon = config?.icon || Info;
  const severity = getSeverity(ev);
  const borderClass = SEVERITY_BORDER[severity] || 'border-slate-300';

  return (
    <div style={style} className="flex items-start gap-4 pr-4">
      {/* Timeline dot + line */}
      <div className="flex flex-col items-center shrink-0">
        <div
          className={`flex items-center justify-center w-9 h-9 rounded-full border-2 bg-white dark:bg-slate-800 ${borderClass}`}
        >
          <Icon className={`w-4 h-4 ${config?.color.split(' ')[0] || 'text-slate-500'}`} />
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0 pt-1">
        <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
          {ev.title}
        </p>
        {ev.description && (
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-2">
            {ev.description}
          </p>
        )}
        <div className="flex items-center gap-3 mt-1">
          <span className="text-[11px] text-slate-400 dark:text-slate-500">
            {formatDateTime(ev.timestamp)}
          </span>
          {ev.actor && (
            <span className="text-[11px] text-slate-400 dark:text-slate-500">
              {ev.actor}
            </span>
          )}
          <span className="text-[10px] uppercase tracking-wider text-slate-400 dark:text-slate-500 font-medium">
            {config?.label || ev.type}
          </span>
        </div>
      </div>
    </div>
  );
});

TimelineRow.displayName = 'TimelineRow';

// ── Main Component ───────────────────────────────────────────────────

export function HistoryTimeline({ events, isLoading = false, deviceId }: HistoryTimelineProps) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<HistoryEventType | 'all'>('all');

  const filteredEvents = useMemo(() => {
    if (filter === 'all') return events;
    return events.filter((ev) => ev.type === filter);
  }, [events, filter]);

  const eventTypes = useMemo(() => {
    const types = new Set(events.map((ev) => ev.type));
    return Array.from(types);
  }, [events]);

  const handleExport = useCallback(() => {
    const filename = deviceId
      ? `device-history-${deviceId.slice(0, 8)}.csv`
      : 'device-history.csv';
    exportToCSV(filteredEvents, filename);
  }, [filteredEvents, deviceId]);

  // ── Loading state ──────────────────────────────────────────────
  if (isLoading) {
    return (
      <Card>
        <CardHeader>{t('history_timeline') || 'History Timeline'}</CardHeader>
        <CardBody>
          <div className="space-y-4 animate-pulse">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-start gap-4">
                <div className="w-9 h-9 rounded-full bg-slate-200 dark:bg-slate-700 shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-3/4" />
                  <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded w-1/2" />
                </div>
              </div>
            ))}
          </div>
        </CardBody>
      </Card>
    );
  }

  // ── Empty state ────────────────────────────────────────────────
  if (events.length === 0) {
    return (
      <Card>
        <CardHeader>{t('history_timeline') || 'History Timeline'}</CardHeader>
        <CardBody>
          <div className="text-center py-10">
            <Info className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
            <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
              {t('no_history_events') || 'No history events'}
            </p>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('no_history_events_description') || 'Events will appear here as they occur'}
            </p>
          </div>
        </CardBody>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader
        action={
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              icon={<Download className="w-4 h-4" />}
              onClick={handleExport}
              disabled={filteredEvents.length === 0}
            >
              {t('export_csv') || 'CSV'}
            </Button>
          </div>
        }
      >
        {t('history_timeline') || 'History Timeline'}
      </CardHeader>

      <CardBody>
        {/* ── Filter bar ────────────────────────────────────────── */}
        <div className="flex items-center gap-2 mb-4 flex-wrap">
          <Filter className="w-4 h-4 text-slate-400 shrink-0" />
          <button
            onClick={() => setFilter('all')}
            className={`px-3 py-1 text-xs rounded-full font-medium transition-colors ${
              filter === 'all'
                ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300'
                : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700'
            }`}
          >
            {t('all') || 'All'}
          </button>
          {eventTypes.map((type) => (
            <button
              key={type}
              onClick={() => setFilter(type)}
              className={`px-3 py-1 text-xs rounded-full font-medium transition-colors flex items-center gap-1 ${
                filter === type
                  ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300'
                  : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700'
              }`}
            >
              {React.createElement(EVENT_TYPE_CONFIG[type]?.icon || Info, {
                className: 'w-3 h-3',
              })}
              {t(type) || EVENT_TYPE_CONFIG[type]?.label || type}
            </button>
          ))}
        </div>

        {/* ── Event counter ─────────────────────────────────────── */}
        <p className="text-xs text-slate-400 dark:text-slate-500 mb-3">
          {filteredEvents.length} {t('events') || 'events'}
          {filter !== 'all' && ` (${t('filtered') || 'filtered'})`}
        </p>

        {/* ── Virtualized list ──────────────────────────────────── */}
        {filteredEvents.length === 0 ? (
          <div className="text-center py-8 text-slate-400">
            <p className="text-sm">{t('no_matching_events') || 'No matching events'}</p>
          </div>
        ) : (
          <div className="relative pl-0">
            {/* Vertical line */}
            <div className="absolute left-[17px] top-3 bottom-3 w-px bg-slate-200 dark:bg-slate-700" />

            <List
              height={Math.min(filteredEvents.length * ITEM_HEIGHT, LIST_HEIGHT)}
              itemCount={filteredEvents.length}
              itemSize={ITEM_HEIGHT}
              width="100%"
              itemData={{ events: filteredEvents, t }}
              overscanCount={5}
            >
              {TimelineRow}
            </List>
          </div>
        )}
      </CardBody>
    </Card>
  );
}

export default HistoryTimeline;
