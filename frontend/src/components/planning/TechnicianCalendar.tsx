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
import {
  AlertTriangle,
  BarChart3,
  Clock,
  Users,
  AlertCircle,
  User as UserIcon,
} from 'lucide-react';
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

export interface AnalyticsData {
  totalHours: number;
  techCount: number;
  conflictCount: number;
  utilizationRate: number;
}

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
// Analytics helpers
// ═══════════════════════════════════════════════════════════════════════

function slotDurationHours(start: string, end: string): number {
  return (new Date(end).getTime() - new Date(start).getTime()) / (1000 * 60 * 60);
}

function computeAnalytics(
  slots: ScheduleSlot[],
  technicians: User[],
  conflicts: ScheduleConflict[],
  dayLoads: Map<string, DayLoad[]>,
): AnalyticsData {
  const totalHours = slots.reduce(
    (sum, s) => sum + slotDurationHours(s.start, s.end),
    0,
  );

  let totalLoad = 0;
  let totalMax = 0;
  for (const loads of dayLoads.values()) {
    for (const dl of loads) {
      totalLoad += dl.totalHours;
      totalMax += dl.maxHours;
    }
  }
  const utilizationRate = totalMax > 0
    ? Math.round((totalLoad / totalMax) * 100)
    : 0;

  return {
    totalHours: Math.round(totalHours * 10) / 10,
    techCount: technicians.length,
    conflictCount: conflicts.length,
    utilizationRate,
  };
}

interface AnalyticsCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  accent?: 'blue' | 'amber' | 'emerald' | 'red';
}

const AnalyticsCard = React.memo(function AnalyticsCard({
  icon,
  label,
  value,
  accent = 'blue',
}: AnalyticsCardProps) {
  const accentStyles: Record<string, string> = {
    blue: 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300',
    amber: 'bg-amber-50 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800 text-amber-700 dark:text-amber-300',
    emerald: 'bg-emerald-50 dark:bg-emerald-900/20 border-emerald-200 dark:border-emerald-800 text-emerald-700 dark:text-emerald-300',
    red: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-700 dark:text-red-300',
  };

  return (
    <div className={`flex items-center gap-3 px-4 py-3 rounded-lg border ${accentStyles[accent]}`}>
      <div className="shrink-0">{icon}</div>
      <div>
        <p className="text-xs font-medium opacity-75">{label}</p>
        <p className="text-lg font-bold">{value}</p>
      </div>
    </div>
  );
});

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

  // ── Reactive analytics computation ──────────────────────────────
  const analytics = useMemo<AnalyticsData>(
    () => computeAnalytics(slots, technicians, conflicts, dayLoads),
    [slots, technicians, conflicts, dayLoads],
  );

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

  /**
   * Handle event drop — including cross-resource (inter-technician) moves.
   * FullCalendar resource-timeline attaches newResource when an event is
   * dragged to a different resource row.
   */
  const handleEventDrop = useCallback(async (info: EventDropArg) => {
    const drop = info as EventDropArg & {
      oldResource?: { id: string };
      newResource?: { id: string };
    };
    const resourceChanged =
      drop.newResource && drop.oldResource &&
      drop.newResource.id !== drop.oldResource.id;

    if (resourceChanged) {
      // Cross-resource move: patch the event's resourceId before the hook processes it.
      // Using internal API — resource-timeline plugin sets event.resourceIds after drop.
      const event = info.event as unknown as { resourceIds: string[]; setProp: (name: string, val: unknown) => void };
      if (event.setProp) {
        event.setProp('resourceIds', [drop.newResource!.id]);
      }
    }

    await onEventDrop(info);
  }, [onEventDrop]);

  /**
   * Handle event receive — when an external element is dropped onto
   * a resource row (e.g. from an external source or cross-resource move).
   * Delegates to the same onEventDrop pipeline.
   */
  const handleEventReceive = useCallback(async (info: {
    event: { id: string; title: string; start: Date | null; end: Date | null; setProp: (name: string, val: unknown) => void };
    resource?: { id: string };
    revert: () => void;
  }) => {
    const targetResourceId = info.resource?.id;
    if (!targetResourceId || !info.event.start) {
      info.revert();
      return;
    }

    // Assign to the target resource
    info.event.setProp('resourceIds', [targetResourceId]);

    // Build a minimal EventDropArg-like object to pass through the pipeline
    await onEventDrop({
      event: info.event as unknown as EventDropArg['event'],
      oldEvent: info.event as unknown as EventDropArg['oldEvent'],
      revert: info.revert,
      view: {} as EventDropArg['view'],
      el: {} as HTMLElement,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delta: { days: 0, milliseconds: 0 } as any,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      jsEvent: {} as any,
      relatedEvents: [],
    });
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

  // ── Print styles (component-level overrides) ────────────────────
  const printStyles = useMemo(() => (
    <style>{`
@media print {
  .technician-calendar .analytics-cards {
    display: grid !important;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
    margin-bottom: 12px;
  }
  .technician-calendar .fc-header-toolbar .fc-button {
    display: none !important;
  }
  .technician-calendar .fc-header-toolbar .fc-toolbar-title {
    font-size: 14pt !important;
    font-weight: 700 !important;
  }
  .technician-calendar .fc .fc-timeline-now-indicator {
    display: none !important;
  }
  .technician-calendar::before {
    content: "CCTV Health Monitor \\2014 Resource Planning";
    display: block;
    font-size: 18pt;
    font-weight: 700;
    text-align: center;
    margin-bottom: 12px;
    color: #1e293b;
  }
}
    `}</style>
  ), []);

  // ── Render ───────────────────────────────────────────────────────
  return (
    <div className={`technician-calendar ${className}`}>
      {printStyles}

      {/* Analytics cards */}
      <div className="analytics-cards grid grid-cols-2 md:grid-cols-4 gap-3 mb-4 print:gap-2 print:mb-3">
        <AnalyticsCard
          icon={<BarChart3 className="w-5 h-5" />}
          label={t('total_hours') || 'Total Hours'}
          value={`${analytics.totalHours}h`}
          accent="blue"
        />
        <AnalyticsCard
          icon={<Users className="w-5 h-5" />}
          label={t('available_technicians') || 'Technicians'}
          value={analytics.techCount}
          accent="emerald"
        />
        <AnalyticsCard
          icon={<Clock className="w-5 h-5" />}
          label={t('utilization_rate') || 'Utilization'}
          value={`${analytics.utilizationRate}%`}
          accent={
            analytics.utilizationRate > 85
              ? 'red'
              : analytics.utilizationRate > 65
              ? 'amber'
              : 'emerald'
          }
        />
        <AnalyticsCard
          icon={<AlertCircle className="w-5 h-5" />}
          label={t('conflicts') || 'Conflicts'}
          value={analytics.conflictCount}
          accent={analytics.conflictCount > 0 ? 'red' : 'emerald'}
        />
      </div>

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
            droppable
            eventDrop={handleEventDrop}
            eventReceive={handleEventReceive}
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
