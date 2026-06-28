// ═══════════════════════════════════════════════════════════════════════
// TechnicianCalendar — Resource calendar for technician scheduling
// P1-PERF-BUNDLE.1: Schedule-X (~80KB) replaces FullCalendar (~328KB)
// P2-2.3: Resource Planning Calendar
//
// P1-UX.5: Calendar Date Mode Toggle (day/week)
//   - Toggle between day/week views
//   - Preference сохраняется в localStorage
//   - Color coding: per-technician with availability indicators
//
// Использует Schedule-X с calendars для color-coded per-technician events:
//   - Technicians as calendars (color-coded)
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
import { useCalendarApp, ScheduleXCalendar } from '@schedule-x/react';
import { viewWeek, viewDay } from '@schedule-x/calendar';
import { createDragAndDropPlugin } from '@schedule-x/drag-and-drop';
import { createCurrentTimePlugin } from '@schedule-x/current-time';
import type { User } from '../../services/api';
import { useTranslation } from 'react-i18next';
import {
  AlertTriangle,
  BarChart3,
  Clock,
  Users,
  AlertCircle,
  User as UserIcon,
  CalendarDays,
  CalendarRange,
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

/** Per-technician calendar color palette */
const TECH_CALENDAR_COLORS = [
  { colorName: 'blue', lightColors: { main: '#3B82F6', container: '#DBEAFE', onContainer: '#1E40AF' }, darkColors: { main: '#60A5FA', container: '#1E3A5F', onContainer: '#BFDBFE' } },
  { colorName: 'green', lightColors: { main: '#22C55E', container: '#DCFCE7', onContainer: '#166534' }, darkColors: { main: '#4ADE80', container: '#14532D', onContainer: '#BBF7D0' } },
  { colorName: 'orange', lightColors: { main: '#F97316', container: '#FED7AA', onContainer: '#9A3412' }, darkColors: { main: '#FB923C', container: '#7C2D12', onContainer: '#FED7AA' } },
  { colorName: 'purple', lightColors: { main: '#A855F7', container: '#F3E8FF', onContainer: '#6B21A8' }, darkColors: { main: '#C084FC', container: '#4C1D95', onContainer: '#E9D5FF' } },
  { colorName: 'red', lightColors: { main: '#EF4444', container: '#FEE2E2', onContainer: '#991B1B' }, darkColors: { main: '#F87171', container: '#7F1D1D', onContainer: '#FECACA' } },
  { colorName: 'teal', lightColors: { main: '#14B8A6', container: '#CCFBF1', onContainer: '#115E59' }, darkColors: { main: '#2DD4BF', container: '#134E4A', onContainer: '#CCFBF1' } },
  { colorName: 'pink', lightColors: { main: '#EC4899', container: '#FCE7F3', onContainer: '#9D174D' }, darkColors: { main: '#F472B6', container: '#831843', onContainer: '#FBCFE8' } },
  { colorName: 'yellow', lightColors: { main: '#EAB308', container: '#FEF9C3', onContainer: '#854D0E' }, darkColors: { main: '#FACC15', container: '#713F12', onContainer: '#FEF08A' } },
];

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

// ── Date mode types ───────────────────────────────────────────────────

export type CalendarDateMode = 'deadline' | 'creation';

// ── View mode types ───────────────────────────────────────────────────

export type CalendarViewMode = 'day' | 'week';

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
  /** P1-UX.5: Date mode: show by deadline or creation date */
  dateMode?: CalendarDateMode;
  /** P1-UX.5: Date mode change callback */
  onDateModeChange?: (mode: CalendarDateMode) => void;
  /** P1-UX.5: Current view mode */
  viewMode?: CalendarViewMode;
  /** P1-UX.5: View mode change callback */
  onViewModeChange?: (mode: CalendarViewMode) => void;
  technicians: User[];
  slots: ScheduleSlot[];
  conflicts: ScheduleConflict[];
  dayLoads: Map<string, DayLoad[]>;
  isLoading: boolean;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  onEventDrop: (info: any) => Promise<void>;
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
  dateMode = 'deadline',
  onDateModeChange,
  viewMode: externalViewMode,
  onViewModeChange,
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

  // P1-UX.5: View mode state with localStorage persistence
  const [internalViewMode, setInternalViewMode] = useState<CalendarViewMode>(() => {
    const stored = localStorage.getItem('technicianCalendar_viewMode');
    if (stored === 'day' || stored === 'week') return stored;
    return 'week';
  });

  const viewMode = externalViewMode ?? internalViewMode;

  const setViewMode = useCallback((mode: CalendarViewMode) => {
    setInternalViewMode(mode);
    localStorage.setItem('technicianCalendar_viewMode', mode);
    onViewModeChange?.(mode);
  }, [onViewModeChange]);

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

  // ── Build calendars config (one per technician) ──────────────────
  const calendars = useMemo(() => {
    const map: Record<string, { colorName: string; lightColors: { main: string; container: string; onContainer: string }; darkColors: { main: string; container: string; onContainer: string } }> = {};
    technicians.forEach((tech, i) => {
      map[tech.id] = TECH_CALENDAR_COLORS[i % TECH_CALENDAR_COLORS.length];
    });
    return map;
  }, [technicians]);

  // ── Schedule-X events ────────────────────────────────────────────
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const calendarEvents = useMemo<any[]>(() => {
    return slots
      .filter(s => techFilter === 'all' || s.technicianId === techFilter)
      .map(slot => {
        const priColor = PRIORITY_COLORS[slot.priority] ?? PRIORITY_COLORS.medium;
        const hasConflict = conflictMap.has(slot.workOrderId);
        const statusColor = STATUS_DOT[slot.status] ?? '#9CA3AF';
        const techCalColor = calendars[slot.technicianId]?.lightColors;

        return {
          id: slot.id,
          title: slot.title,
          start: slot.start,
          end: slot.end,
          calendarId: slot.technicianId,
          workOrderId: slot.workOrderId,
          priority: slot.priority,
          status: slot.status,
          statusColor,
          hasConflict,
          backgroundColor: hasConflict ? '#FEF2F2' : (techCalColor?.container || priColor.bg),
          borderColor: hasConflict ? '#DC2626' : priColor.border,
          textColor: priColor.text,
          _customContent: {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            timeGrid: buildEventHTML(slot, hasConflict, statusColor),
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            dateGrid: buildEventHTML(slot, hasConflict, statusColor),
          },
        };
      });
  }, [slots, techFilter, conflictMap, calendars]);

  // ── Calendar instance ────────────────────────────────────────────
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark');

  const calendar = useCalendarApp({
    views: [viewWeek, viewDay],
    defaultView: viewMode === 'day' ? 'day' : 'week',
    events: calendarEvents,
    calendars,
    plugins: [
      createDragAndDropPlugin(),
      createCurrentTimePlugin(),
    ],
    isDark,
    callbacks: {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      onEventClick: (event: any) => {
        const woId = event.workOrderId as string | undefined;
        if (woId && onEventClick) onEventClick(woId);
      },
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      onEventUpdate: (event: any) => {
        // Build a minimal event drop info object
        const info = {
          event: {
            id: event.id,
            startStr: event.start?.toString?.() || event.start,
            endStr: event.end?.toString?.() || event.end,
            extendedProps: {
              workOrderId: event.workOrderId,
            },
          },
        };
        onEventDrop(info);
      },
    },
    dayBoundaries: { start: '07:00', end: '19:00' },
    firstDayOfWeek: 1,
  });

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

  // ── Print styles ────────────────────────────────────────────────
  const printStyles = useMemo(() => (
    <style>{`
@media print {
  .technician-calendar .analytics-cards {
    display: grid !important;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
    margin-bottom: 12px;
  }
  .technician-calendar .sx__calendar-header .sx__button {
    display: none !important;
  }
  .technician-calendar .sx__calendar-header .sx__title {
    font-size: 14pt !important;
    font-weight: 700 !important;
  }
  .technician-calendar .sx__current-time-line {
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

        {/* P1-UX.5: View mode toggle (day/week) */}
        <div className="flex items-center border border-slate-200 dark:border-slate-600 rounded-lg overflow-hidden">
          <button
            onClick={() => setViewMode('day')}
            className={`p-2 transition-colors ${
              viewMode === 'day'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
            }`}
            title={t('day_view') || 'Day View'}
            aria-label={t('day_view') || 'Day View'}
            aria-pressed={viewMode === 'day'}
          >
            <CalendarDays size={16} />
          </button>
          <button
            onClick={() => setViewMode('week')}
            className={`p-2 transition-colors ${
              viewMode === 'week'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
            }`}
            title={t('week_view') || 'Week View'}
            aria-label={t('week_view') || 'Week View'}
            aria-pressed={viewMode === 'week'}
          >
            <CalendarRange size={16} />
          </button>
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
          <ScheduleXCalendar calendarApp={calendar} />
        </div>
      )}
    </div>
  );
});

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

/** Build HTML string for custom event content */
function buildEventHTML(slot: ScheduleSlot, hasConflict: boolean, statusColor: string): string {
  const priColor = PRIORITY_COLORS[slot.priority] ?? PRIORITY_COLORS.medium;
  const borderStyle = hasConflict ? 'border-2 border-dashed border-red-400' : '';

  return `
    <div class="flex items-center gap-1 px-1.5 py-0.5 text-xs leading-tight overflow-hidden rounded h-full ${borderStyle}" style="border-left: 3px solid ${priColor.border};">
      <span class="inline-block w-1.5 h-1.5 shrink-0 rounded-full" style="background-color: ${statusColor};"></span>
      <span class="truncate font-medium">${slot.title}</span>
      ${slot.priority === 'critical' ? '<span class="shrink-0">⚠</span>' : ''}
      ${slot.status === 'in_progress' ? '<span class="shrink-0"><svg class="w-3 h-3 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg></span>' : ''}
      ${hasConflict ? '<span class="shrink-0"><svg class="w-3 h-3 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>' : ''}
    </div>
  `;
}
