import React, { useMemo, useState, useCallback } from 'react';
import FullCalendar from '@fullcalendar/react';
import dayGridPlugin from '@fullcalendar/daygrid';
import interactionPlugin from '@fullcalendar/interaction';
import type { EventClickArg, DateSelectArg, EventDropArg, EventContentArg, EventMountArg } from '@fullcalendar/core';
import type { WorkOrder } from '../../services/workOrdersApi';
import type { User as ApiUser } from '../../services/api';
import { useTranslation } from 'react-i18next';
import { CalendarDays, CalendarClock, Info } from 'lucide-react';
import { useLocalStorage } from '../../hooks/useLocalStorage';

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

type DateMode = 'deadline' | 'creation';

const PRIORITY_CONFIG: Record<string, { bg: string; border: string; text: string }> = {
  critical: { bg: '#FEE2E2', border: '#DC2626', text: '#991B1B' },
  high:      { bg: '#FED7AA', border: '#EA580C', text: '#9A3412' },
  medium:    { bg: '#DBEAFE', border: '#2563EB', text: '#1E40AF' },
  low:       { bg: '#DCFCE7', border: '#16A34A', text: '#166534' },
};

const STATUS_DOT: Record<string, string> = {
  open:        '#9CA3AF',
  in_progress: '#3B82F6',
  completed:   '#22C55E',
  cancelled:   '#EF4444',
};

/** Color coding for date modes (P1-UX.6) — deadline (red), creation (blue) */
const DATE_MODE_COLORS: Record<DateMode, { bg: string; border: string; text: string; label: string }> = {
  deadline: { bg: '#FEE2E2', border: '#EF4444', text: '#991B1B', label: 'By Deadline' },
  creation: { bg: '#DBEAFE', border: '#3B82F6', text: '#1E40AF', label: 'By Creation' },
};

// ── Deterministic colour per technician (HSL) ─────────────────────────
function hashHue(id: string): number {
  let hash = 0;
  for (let i = 0; i < id.length; i++) {
    hash = ((hash << 5) - hash) + id.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash) % 360;
}

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface WorkOrderCalendarProps {
  workOrders: WorkOrder[];
  technicians: ApiUser[];
  currentUserId?: string;
  onDateChange: (id: string, newDate: string) => Promise<void>;
  onEventClick: (workOrder: WorkOrder) => void;
  onDateClick: (date: Date) => void;
  className?: string;
  /** P1-UX.6: Controlled date mode from parent (optional) */
  dateMode?: DateMode;
  /** P1-UX.6: Callback when date mode changes */
  onDateModeChange?: (mode: DateMode) => void;
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderCalendar
// ═══════════════════════════════════════════════════════════════════════

export const WorkOrderCalendar = React.memo(function WorkOrderCalendar({
  workOrders,
  technicians,
  currentUserId,
  onDateChange,
  onEventClick,
  onDateClick,
  className = '',
  dateMode: controlledDateMode,
  onDateModeChange,
}: WorkOrderCalendarProps) {
  const { t } = useTranslation();
  const [techFilter, setTechFilter] = useState<string>('all');
  // P1-UX.6: Date mode toggle — controlled or uncontrolled via localStorage
  const [localDateMode, setLocalDateMode] = useLocalStorage<DateMode>(
    'woCalendar_dateMode',
    'deadline',
  );
  const dateMode = controlledDateMode ?? localDateMode;
  const setDateMode = useCallback(
    (mode: DateMode) => {
      if (onDateModeChange) {
        onDateModeChange(mode);
      } else {
        setLocalDateMode(mode);
      }
    },
    [onDateModeChange, setLocalDateMode],
  );

  // ── Technician colour map ───────────────────────────────────────────
  const techColorMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const t of technicians) {
      map[t.id] = `hsl(${hashHue(t.id)}, 58%, 50%)`;
    }
    return map;
  }, [technicians]);

  // ── Filtered work orders ────────────────────────────────────────────
  const filteredOrders = useMemo(() => {
    if (techFilter === 'all') return workOrders;
    if (techFilter === 'mine') {
      return workOrders.filter(wo => wo.assigned_to === currentUserId);
    }
    return workOrders.filter(wo => wo.assigned_to === techFilter);
  }, [workOrders, techFilter, currentUserId]);

  // ── Convert to FullCalendar events ──────────────────────────────────
  const calendarEvents = useMemo(() => {
    return filteredOrders
      .filter(wo => {
        if (dateMode === 'deadline') return wo.sla_deadline;
        return wo.created_at;
      })
      .map(wo => {
        // P1-1.2: Use date based on mode
        const start = dateMode === 'deadline' ? wo.sla_deadline! : wo.created_at!;
        const techCol = wo.assigned_to && techColorMap[wo.assigned_to];

        // When showing all techs → colour by technician (workload)
        // When filtered → colour by priority
        const useTechColor = techFilter === 'all' && techCol;

        // P1-1.2: Colour coding per date mode
        const modeColor = DATE_MODE_COLORS[dateMode];

        return {
          id: wo.id,
          title: wo.device_name || wo.device_id || 'Untitled',
          start,
          allDay: true,
          extendedProps: { workOrder: wo, dateMode },
          backgroundColor: useTechColor ? techCol + '22' : modeColor.bg,
          borderColor:     useTechColor ? techCol        : modeColor.border,
          textColor:       useTechColor ? techCol        : modeColor.text,
          classNames: [
            `wo-${wo.status}`,
            `wo-prio-${wo.priority}`,
            wo.assigned_to ? 'wo-has-tech' : 'wo-no-tech',
            `wo-date-${dateMode}`,
          ],
        };
      });
  }, [filteredOrders, techColorMap, techFilter, dateMode]);

  // ── Handlers ────────────────────────────────────────────────────────

  const handleEventClick = useCallback((info: EventClickArg) => {
    const wo = info.event.extendedProps.workOrder as WorkOrder | undefined;
    if (wo) onEventClick(wo);
  }, [onEventClick]);

  const handleDateSelect = useCallback((info: DateSelectArg) => {
    onDateClick(info.start);
  }, [onDateClick]);

  const handleEventDrop = useCallback(async (info: EventDropArg) => {
    const wo = info.event.extendedProps.workOrder as WorkOrder | undefined;
    if (!wo || !info.event.start) { info.revert(); return; }
    try {
      await onDateChange(wo.id, info.event.start.toISOString());
    } catch {
      info.revert();
    }
  }, [onDateChange]);

  // ── Custom event rendering ──────────────────────────────────────────
  const renderEventContent = useCallback((info: EventContentArg) => {
    const wo = info.event.extendedProps?.workOrder as WorkOrder | undefined;
    return (
      <div className="fc-custom-event flex items-center gap-1 px-1 py-0.5 text-xs leading-tight overflow-hidden rounded">
        <span
          className="inline-block w-1.5 h-1.5 shrink-0 rounded-full"
          style={{ backgroundColor: (wo && STATUS_DOT[wo.status]) ?? '#9CA3AF' }}
        />
        <span className="truncate font-medium">{info.event.title}</span>
        {wo?.priority === 'critical' && <span className="shrink-0">⚠</span>}
      </div>
    );
  }, []);

  // ── Tooltip on hover (eventDidMount) ────────────────────────────────
  const handleEventDidMount = useCallback((info: EventMountArg) => {
    const wo = info.event.extendedProps?.workOrder as WorkOrder | undefined;
    if (!wo) return;

    const tip = document.createElement('div');
    tip.className = 'wo-cal-tooltip';
    // P1-UX.6: Dual date display with labels
    const deadlineDate = wo.sla_deadline ? new Date(wo.sla_deadline).toLocaleDateString() : '—';
    const creationDate = wo.created_at ? new Date(wo.created_at).toLocaleDateString() : '—';
    tip.innerHTML = `
      <div class="tip-title">${wo.device_name || wo.device_id || 'Untitled'}</div>
      <div class="tip-body">
        <div><span>Status</span><strong>${wo.status.replace(/_/g, ' ')}</strong></div>
        <div><span>Priority</span><strong>${wo.priority}</strong></div>
        ${wo.assignee_name ? `<div><span>Tech</span><strong>${wo.assignee_name}</strong></div>` : ''}
        ${wo.type ? `<div><span>Type</span><strong>${wo.type}</strong></div>` : ''}
        <div class="tip-divider"></div>
        <div><span class="tip-label-red">● Due</span><strong>${deadlineDate}</strong></div>
        <div><span class="tip-label-blue">● Created</span><strong>${creationDate}</strong></div>
      </div>
    `;

    let hideTimeout: ReturnType<typeof setTimeout> | null = null;
    const show = (e: MouseEvent) => {
      if (hideTimeout) clearTimeout(hideTimeout);
      tip.style.display = 'block';
      const rect = info.el.getBoundingClientRect();
      tip.style.left = `${Math.min(e.clientX + 12, window.innerWidth - 260)}px`;
      tip.style.top = `${e.clientY + 12}px`;
    };
    const hide = () => {
      hideTimeout = setTimeout(() => { tip.style.display = 'none'; }, 80);
    };
    info.el.addEventListener('mouseenter', (e) => show(e as MouseEvent));
    info.el.addEventListener('mousemove', (e) => show(e as MouseEvent));
    info.el.addEventListener('mouseleave', hide);
    info.el.appendChild(tip);
  }, []);

  return (
    <div className={`work-order-calendar ${className}`}>
      {/* ── Filter bar + Date mode toggle (P1-UX.6) ────────────────── */}
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <label className="text-sm font-medium text-slate-600 dark:text-slate-400">
          {t('technician') || 'Technician'}:
          <select
            value={techFilter}
            onChange={e => setTechFilter(e.target.value)}
            className="ml-2 border rounded px-2.5 py-1.5 text-sm dark:bg-slate-800 dark:border-slate-600"
            aria-label="Filter by technician"
          >
            <option value="all">{t('all_technicians') || 'All Technicians'}</option>
            <option value="mine">{t('my_orders') || 'My Orders'}</option>
            {technicians.map(t => (
              <option key={t.id} value={t.id}>{t.name || t.username}</option>
            ))}
          </select>
        </label>

        {/* P1-UX.6: Date Mode Toggle */}
        <div className="flex items-center border border-slate-200 dark:border-slate-600 rounded-lg overflow-hidden" role="radiogroup" aria-label={t('date_mode') || 'Date mode'}>
          <button
            onClick={() => setDateMode('deadline')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors ${
              dateMode === 'deadline'
                ? 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'
                : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
            }`}
            role="radio"
            aria-checked={dateMode === 'deadline'}
            title={t('show_by_deadline') || 'Show by deadline'}
          >
            <CalendarClock className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t('by_deadline') || 'Deadline'}</span>
          </button>
          <button
            onClick={() => setDateMode('creation')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium transition-colors ${
              dateMode === 'creation'
                ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300'
                : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
            }`}
            role="radio"
            aria-checked={dateMode === 'creation'}
            title={t('show_by_creation_date') || 'Show by creation date'}
          >
            <CalendarDays className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t('by_creation') || 'Creation'}</span>
          </button>
        </div>

        {/* ── Technician legend (when showing all) ───────────────────── */}
        {techFilter === 'all' && (
          <div className="flex items-center gap-3 text-xs text-slate-500 dark:text-slate-400 flex-wrap">
            {technicians.slice(0, 8).map(t => (
              <span key={t.id} className="flex items-center gap-1">
                <span
                  className="inline-block w-2.5 h-2.5 rounded-full"
                  style={{ backgroundColor: techColorMap[t.id] ?? '#9CA3AF' }}
                />
                {t.name || t.username}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* ── Calendar ───────────────────────────────────────────────── */}
      <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <FullCalendar
          plugins={[dayGridPlugin, interactionPlugin]}
          initialView="dayGridMonth"
          events={calendarEvents}
          eventClick={handleEventClick}
          selectable
          select={handleDateSelect}
          editable
          eventDrop={handleEventDrop}
          eventContent={renderEventContent}
          eventDidMount={handleEventDidMount}
          headerToolbar={{
            left: 'prev,next today',
            center: 'title',
            right: 'dayGridMonth,dayGridWeek,dayGridDay',
          }}
          buttonText={{
            today: t('today') || 'Today',
            month: t('month') || 'Month',
            week: t('week') || 'Week',
            day: t('day') || 'Day',
          }}
          height="auto"
          contentHeight="auto"
          aspectRatio={1.8}
          firstDay={1}
          nowIndicator
          locale="en"
        />
      </div>

      {/* ── Date Mode Legend (P1-UX.6) ──────────────────────────────── */}
      <div className="flex items-center justify-center gap-6 mt-3 text-xs text-slate-500 dark:text-slate-400">
        <span className="flex items-center gap-1.5">
          <span className="inline-block w-3 h-3 rounded" style={{ backgroundColor: '#EF4444' }} />
          <span>{t('deadline_legend') || 'Deadline'}</span>
        </span>
        <span className="flex items-center gap-1.5">
          <span className="inline-block w-3 h-3 rounded" style={{ backgroundColor: '#3B82F6' }} />
          <span>{t('creation_legend') || 'Creation date'}</span>
        </span>
        <span className="flex items-center gap-1.5 text-slate-400 dark:text-slate-500">
          <Info className="w-3 h-3" />
          <span>{t('date_mode_hint') || 'Hover events for both dates'}</span>
        </span>
      </div>
    </div>
  );
});
