// ═══════════════════════════════════════════════════════════════════════
// TechnicianCalendar — Resource timeline calendar for technician scheduling
// P2-2.3: Resource Planning Calendar
//
// Uses FullCalendar resourceTimelineWeek view:
//   - Technicians as resources (rows)
//   - Work orders as draggable events
//   - Availability indicators (green/yellow/red) per day
//   - Conflict detection warnings
//   - Print-friendly @media print CSS
//
// Compliance:
//   - IEC 62443 SR 5.1 (Workflow — planning integrity)
//   - OWASP ASVS V1.8 (Stateless architecture)
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useCallback, useState } from 'react';
import FullCalendar from '@fullcalendar/react';
import resourceTimelinePlugin from '@fullcalendar/resource-timeline';
import interactionPlugin from '@fullcalendar/interaction';
import type { EventDropArg, EventContentArg, EventMountArg } from '@fullcalendar/core';
import type { User } from '../../services/api';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, Clock, User as UserIcon } from 'lucide-react';
import type {
  ScheduleSlot,
  ScheduleConflict,
  DayLoad,
  AvailabilityLevel,
} from '../../hooks/useTechnicianSchedule';

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const AVAILABILITY_STYLES: Record<AvailabilityLevel, { dot: string; bg: string; label: string }> = {
  green:  { dot: 'bg-emerald-500', bg: 'bg-emerald-50 dark:bg-emerald-900/20', label: 'free' },
  yellow: { dot: 'bg-amber-400',   bg: 'bg-amber-50 dark:bg-amber-900/20',   label: '>75% loaded' },
  red:    { dot: 'bg-red-500',     bg: 'bg-red-50 dark:bg-red-900/20',       label: 'overloaded' },
};

const PRIORITY_COLORS: Record<string, { bg: string; border: string; text: string }> = {
  critical:  { bg: '#FEE2E2', border: '#DC2626', text: '#991B1B' },
  high:      { bg: '#FED7AA', border: '#EA580C', text: '#9A3412' },
  medium:    { bg: '#DBEAFE', border: '#2563EB', text: '#1E40AF' },
  low:       { bg: '#F3F4F6', border: '#9CA3AF', text: '#374151' },
};

const STATUS_DOT: Record<string, string> = {
  open:        '#9CA3AF',
  in_progress: '#3B82F6',
  completed:   '#22C55E',
  cancelled:   '#EF4444',
};

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function hashHue(id: string): number {
  let hash = 0;
  for (let i = 0; i < id.length; i++) {
    hash = ((hash << 5) - hash) + id.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash) % 360;
}

function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00');
  return d.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' });
}

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface TechnicianCalendarProps {
  technicians: User[];
  slots: ScheduleSlot[];
  conflicts: ScheduleConflict[];
  dayLoads: Map<string, DayLoad[]>;
  isLoading: boolean;
  onEventDrop: (info: EventDropArg) => Promise<void>;
  onEventClick?: (workOrderId: string) => void;
  className?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export const TechnicianCalendar = React.memo(function TechnicianCalendar({
  technicians,
  slots,
  conflicts,
  dayLoads,
  isLoading,
  onEventDrop,
  onEventClick,
  className = '',
}: TechnicianCalendarProps) {
  const { t } = useTranslation();
  const [showConflicts, setShowConflicts] = useState(true);
  const [techFilter, setTechFilter] = useState<string>('all');

  // ── Conflicts by work order ID for quick lookup ──────────────────
  const conflictMap = useMemo(() => {
    const map = new Map<string, ScheduleConflict>();
    for (const c of conflicts) {
      for (const woId of c.workOrderIds) {
        map.set(woId, c);
      }
    }
    return map;
  }, [conflicts]);

  // ── FullCalendar resources (technician rows) ─────────────────────
  const resources = useMemo(() => {
    const filtered = techFilter === 'all'
      ? technicians
      : technicians.filter(t => t.id === techFilter);

    return filtered.map(tech => {
      const loads = dayLoads.get(tech.id) ?? [];
      const todayLoad = loads.find(l => l.date === new Date().toISOString().slice(0, 10));
      const av = todayLoad?.availability ?? 'green';
      const style = AVAILABILITY_STYLES[av];

      return {
        id: tech.id,
        title: tech.name || tech.username,
        // Extended props for custom rendering
        eventColor: `hsl(${hashHue(tech.id)}, 58%, 50%)`,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any;
    });
  }, [technicians, dayLoads, techFilter]);

  // ── FullCalendar events (work order slots) ───────────────────────
  const calendarEvents = useMemo(() => {
    return slots
      .filter(s => techFilter === 'all' || s.technicianId === techFilter)
      .map(slot => {
        const priColor = PRIORITY_COLORS[slot.priority] ?? PRIORITY_COLORS.medium;
        const hasConflict = conflictMap.has(slot.workOrderId);
        const statusColor = STATUS_DOT[slot.status] ?? '#9CA3AF';

        return {
          id: slot.id,
          resourceId: slot.technicianId,
          title: slot.title,
          start: slot.start,
          end: slot.end,
          backgroundColor: hasConflict ? '#FEF2F2' : priColor.bg,
          borderColor: hasConflict ? '#DC2626' : priColor.border,
          textColor: priColor.text,
          extendedProps: {
            workOrderId: slot.workOrderId,
            priority: slot.priority,
            status: slot.status,
            statusColor,
            hasConflict,
          },
          classNames: [
            'technician-event',
            `prio-${slot.priority}`,
            slot.status === 'in_progress' ? 'event-in-progress' : '',
            hasConflict ? 'event-conflict' : '',
          ].filter(Boolean),
        };
      });
  }, [slots, techFilter, conflictMap]);

  // ── Handlers ─────────────────────────────────────────────────────

  const handleEventClick = useCallback((info: { event: { extendedProps: Record<string, unknown> } }) => {
    const woId = info.event.extendedProps?.workOrderId as string | undefined;
    if (woId && onEventClick) onEventClick(woId);
  }, [onEventClick]);

  const handleEventDrop = useCallback(async (info: EventDropArg) => {
    await onEventDrop(info);
  }, [onEventDrop]);

  // ── Custom resource label rendering ──────────────────────────────
  const renderResourceLabel = useCallback((resource: {
    id: string;
    title: string;
    eventColor?: string;
  }) => {
    const tech = technicians.find(t => t.id === resource.id);
    const loads = dayLoads.get(resource.id) ?? [];
    const todayLoad = loads.find(l => l.date === new Date().toISOString().slice(0, 10));
    const av = todayLoad?.availability ?? 'green';
    const style = AVAILABILITY_STYLES[av];

    return (
      <div className="flex items-center gap-3 px-2 py-1.5 min-w-[200px]">
        <div
          className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold shrink-0"
          style={{ backgroundColor: resource.eventColor ?? `hsl(${hashHue(resource.id)}, 58%, 50%)` }}
        >
          {tech?.name?.charAt(0)?.toUpperCase() ?? tech?.username?.charAt(0)?.toUpperCase() ?? '?'}
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
            {resource.title}
          </p>
          <p className="text-xs text-slate-400 truncate">
            {tech?.role ?? 'Technician'}
          </p>
        </div>
        {/* Availability indicator */}
        <div className="flex items-center gap-1.5 shrink-0" title={`${av}: ${todayLoad?.totalHours.toFixed(1) ?? 0}h / ${todayLoad?.maxHours ?? 8}h`}>
          <span className={`inline-block w-2.5 h-2.5 rounded-full ${style.dot}`} />
          <span className="text-[10px] text-slate-400 hidden md:inline">
            {todayLoad ? `${todayLoad.totalHours.toFixed(1)}h` : '0h'}
          </span>
        </div>
        {/* Conflict badge */}
        {conflicts.some(c => c.technicianId === resource.id) && (
          <span className="flex items-center gap-1 text-[10px] text-red-500" title={t('has_conflicts') || 'Has conflicts'}>
            <AlertTriangle className="w-3 h-3" />
          </span>
        )}
      </div>
    );
  }, [technicians, dayLoads, conflicts, t]);

  // ── Custom event rendering ───────────────────────────────────────
  const renderEventContent = useCallback((info: EventContentArg) => {
    const props = info.event.extendedProps;
    const title = info.event.title;
    const hasConflict = props.hasConflict as boolean;
    const priority = props.priority as string;
    const status = props.status as string;
    const statusColor = props.statusColor as string;

    return (
      <div className={`
        flex items-center gap-1 px-1.5 py-0.5 text-xs leading-tight overflow-hidden rounded h-full
        ${hasConflict ? 'border-2 border-dashed border-red-400' : ''}
      `}>
        <span
          className="inline-block w-1.5 h-1.5 shrink-0 rounded-full"
          style={{ backgroundColor: statusColor ?? '#9CA3AF' }}
        />
        <span className="truncate font-medium">{title}</span>
        {priority === 'critical' && <span className="shrink-0">⚠</span>}
        {status === 'in_progress' && <Clock className="w-3 h-3 shrink-0 text-blue-500" />}
        {hasConflict && <AlertTriangle className="w-3 h-3 shrink-0 text-red-500" />}
      </div>
    );
  }, []);

  // ── Tooltip on hover ─────────────────────────────────────────────
  const handleEventMount = useCallback((info: EventMountArg) => {
    const props = info.event.extendedProps;
    const title = info.event.title;
    const conflict = props.hasConflict
      ? conflictMap.get(props.workOrderId as string)
      : null;

    const tip = document.createElement('div');
    tip.className = 'fc-resource-tooltip';
    tip.innerHTML = `
      <div class="tip-title">${title}</div>
      <div class="tip-body">
        <div><span>Status</span><strong>${(props.status as string)?.replace(/_/g, ' ') ?? '—'}</strong></div>
        <div><span>Priority</span><strong>${props.priority as string ?? '—'}</strong></div>
        <div><span>Start</span><strong>${new Date(info.event.start!).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</strong></div>
        <div><span>End</span><strong>${new Date(info.event.end!).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</strong></div>
        ${conflict ? `<div class="tip-conflict"><span>⚠</span><strong>${conflict.message}</strong></div>` : ''}
      </div>
    `;

    let hideTimeout: ReturnType<typeof setTimeout> | null = null;
    const show = (e: MouseEvent) => {
      if (hideTimeout) clearTimeout(hideTimeout);
      tip.style.display = 'block';
      const rect = info.el.getBoundingClientRect();
      tip.style.left = `${Math.min(e.clientX + 12, window.innerWidth - 280)}px`;
      tip.style.top = `${e.clientY + 12}px`;
    };
    const hide = () => {
      hideTimeout = setTimeout(() => { tip.style.display = 'none'; }, 80);
    };
    info.el.addEventListener('mouseenter', (e) => show(e as MouseEvent));
    info.el.addEventListener('mousemove', (e) => show(e as MouseEvent));
    info.el.addEventListener('mouseleave', hide);
    info.el.appendChild(tip);
  }, [conflictMap]);

  // ── Conflict summary panel ───────────────────────────────────────
  const conflictSummary = useMemo(() => {
    if (!showConflicts || conflicts.length === 0) return null;

    return (
      <div className="mb-3 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-semibold text-red-700 dark:text-red-400 flex items-center gap-1.5">
            <AlertTriangle className="w-4 h-4" />
            {t('schedule_conflicts') || 'Schedule Conflicts'} ({conflicts.length})
          </h3>
          <button
            onClick={() => setShowConflicts(false)}
            className="text-xs text-red-500 hover:text-red-700"
          >
            {t('dismiss') || 'Dismiss'}
          </button>
        </div>
        <ul className="space-y-1">
          {conflicts.map((c, i) => (
            <li key={i} className="text-xs text-red-600 dark:text-red-400 flex items-start gap-2">
              <span className="mt-0.5 shrink-0">•</span>
              <span>
                <strong>{formatDateLabel(c.date)}</strong> — {c.message}
              </span>
            </li>
          ))}
        </ul>
      </div>
    );
  }, [conflicts, showConflicts, t]);

  // ── Render ───────────────────────────────────────────────────────
  return (
    <div className={`technician-calendar ${className}`}>
      {/* Toolbar */}
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        {/* Technician filter */}
        <div className="flex items-center gap-2">
          <UserIcon className="w-4 h-4 text-slate-400" />
          <select
            value={techFilter}
            onChange={e => setTechFilter(e.target.value)}
            className="border rounded px-2.5 py-1.5 text-sm dark:bg-slate-800 dark:border-slate-600"
            aria-label={t('filter_technician') || 'Filter by technician'}
          >
            <option value="all">{t('all_technicians') || 'All Technicians'}</option>
            {technicians.map(t => (
              <option key={t.id} value={t.id}>{t.name || t.username}</option>
            ))}
          </select>
        </div>

        {/* Availability legend */}
        <div className="flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400 ml-auto">
          {Object.entries(AVAILABILITY_STYLES).map(([key, val]) => (
            <span key={key} className="flex items-center gap-1">
              <span className={`inline-block w-2.5 h-2.5 rounded-full ${val.dot}`} />
              {key === 'green' ? (t('free') || 'Free') :
               key === 'yellow' ? (t('loaded') || '>75%') :
               (t('overloaded') || 'Overloaded')}
            </span>
          ))}
          <span className="flex items-center gap-1 ml-2">
            <AlertTriangle className="w-3 h-3 text-red-500" />
            {t('conflict') || 'Conflict'}
          </span>
        </div>
      </div>

      {/* Conflict summary */}
      {conflictSummary}

      {/* Loading overlay */}
      {isLoading && (
        <div className="flex items-center justify-center py-12 text-slate-400">
          <div className="animate-spin w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full mr-2" />
          <span className="text-sm">{t('loading_schedule') || 'Loading schedule...'}</span>
        </div>
      )}

      {/* Calendar */}
      {!isLoading && (
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden print:border-none print:shadow-none">
          <FullCalendar
            plugins={[resourceTimelinePlugin, interactionPlugin]}
            initialView="resourceTimelineWeek"
            resources={resources}
            events={calendarEvents}
            resourceLabelContent={renderResourceLabel}
            eventContent={renderEventContent}
            eventDidMount={handleEventMount}
            eventClick={handleEventClick}
            editable
            eventDrop={handleEventDrop}
            eventResizableFromStart
            eventDurationEditable
            height="auto"
            contentHeight="auto"
            stickyHeaderDates
            nowIndicator
            firstDay={1}
            slotMinTime="07:00:00"
            slotMaxTime="19:00:00"
            slotDuration="01:00:00"
            headerToolbar={{
              left: 'prev,next today',
              center: 'title',
              right: 'resourceTimelineWeek,resourceTimelineDay',
            }}
            buttonText={{
              today: t('today') || 'Today',
              week: t('week') || 'Week',
              day: t('day') || 'Day',
            }}
            views={{
              resourceTimelineWeek: {
                type: 'resourceTimeline',
                duration: { weeks: 1 },
                buttonText: t('week') || 'Week',
              },
              resourceTimelineDay: {
                type: 'resourceTimeline',
                duration: { days: 1 },
                buttonText: t('day') || 'Day',
              },
            }}
            locale="en"
          />
        </div>
      )}
    </div>
  );
});
