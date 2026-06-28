// ═══════════════════════════════════════════════════════════════════════
// ScheduleXWrapper — Lazy-loaded Schedule-X calendar wrapper
// P1-PERF-BUNDLE.1: Schedule-X (~80KB) replaces FullCalendar (~328KB)
//
// Schedule-X загружается лениво (dynamic import) при первом рендере.
// Поддержка: month view, week view, resource events, drag & drop.
// Print-friendly CSS, dark mode (CSS variables).
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect, useMemo } from 'react';
import type { MaintenanceSchedule } from '../../services/maintenanceApi';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface Resource {
  id: string;
  title: string;
  eventColor?: string;
}

export interface ScheduleXEvent {
  id: string;
  start: string;
  end?: string;
  title?: string;
  calendarId?: string;
  /** Custom data passed through extended props */
  [key: string]: unknown;
}

interface CalendarColors {
  colorName: string;
  lightColors: { main: string; container: string; onContainer: string };
  darkColors: { main: string; container: string; onContainer: string };
}

interface ScheduleXWrapperProps {
  /** Массив событий в формате Schedule-X */
  events: ScheduleXEvent[];
  /** Ресурсы для color-coded calendars (опционально) */
  resources?: Resource[];
  /** Map resource.id → calendar config for color coding */
  calendars?: Record<string, CalendarColors>;
  onEventClick: (schedule: MaintenanceSchedule) => void;
  onEventDrop: (schedule: MaintenanceSchedule, newDate: string) => Promise<void>;
  /** Включить resource views */
  enableResourceView?: boolean;
  className?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Print styles
// ═══════════════════════════════════════════════════════════════════════

const PRINT_STYLE_ID = 'sx-wrapper-print-styles';

function injectPrintStyles(): void {
  if (typeof document === 'undefined' || document.getElementById(PRINT_STYLE_ID)) return;
  const style = document.createElement('style');
  style.id = PRINT_STYLE_ID;
  style.textContent = `
@media print {
  .sx-wrapper-print-header {
    display: block !important;
    text-align: center;
    font-size: 16pt;
    font-weight: 700;
    margin-bottom: 16px;
    color: #1e293b;
  }
  .sx-wrapper .sx__calendar-header .sx__button {
    display: none !important;
  }
  .sx-wrapper .sx__calendar-header .sx__title {
    font-size: 14pt !important;
    font-weight: 700 !important;
  }
  .sx-wrapper .sx__calendar {
    font-size: 9pt !important;
  }
  .sx-wrapper > :not(.sx-wrapper-print-header):not(.sx__calendar) {
    display: none !important;
  }
}
  `;
  document.head.appendChild(style);
}

// ═══════════════════════════════════════════════════════════════════════
// Build calendars config from resources
// ═══════════════════════════════════════════════════════════════════════

function buildCalendarsFromResources(resources: Resource[]): Record<string, CalendarColors> {
  const calendars: Record<string, CalendarColors> = {};
  const palette: CalendarColors[] = [
    { colorName: 'blue', lightColors: { main: '#3B82F6', container: '#DBEAFE', onContainer: '#1E40AF' }, darkColors: { main: '#60A5FA', container: '#1E3A5F', onContainer: '#BFDBFE' } },
    { colorName: 'green', lightColors: { main: '#22C55E', container: '#DCFCE7', onContainer: '#166534' }, darkColors: { main: '#4ADE80', container: '#14532D', onContainer: '#BBF7D0' } },
    { colorName: 'orange', lightColors: { main: '#F97316', container: '#FED7AA', onContainer: '#9A3412' }, darkColors: { main: '#FB923C', container: '#7C2D12', onContainer: '#FED7AA' } },
    { colorName: 'purple', lightColors: { main: '#A855F7', container: '#F3E8FF', onContainer: '#6B21A8' }, darkColors: { main: '#C084FC', container: '#4C1D95', onContainer: '#E9D5FF' } },
    { colorName: 'red', lightColors: { main: '#EF4444', container: '#FEE2E2', onContainer: '#991B1B' }, darkColors: { main: '#F87171', container: '#7F1D1D', onContainer: '#FECACA' } },
    { colorName: 'teal', lightColors: { main: '#14B8A6', container: '#CCFBF1', onContainer: '#115E59' }, darkColors: { main: '#2DD4BF', container: '#134E4A', onContainer: '#CCFBF1' } },
    { colorName: 'pink', lightColors: { main: '#EC4899', container: '#FCE7F3', onContainer: '#9D174D' }, darkColors: { main: '#F472B6', container: '#831843', onContainer: '#FBCFE8' } },
    { colorName: 'yellow', lightColors: { main: '#EAB308', container: '#FEF9C3', onContainer: '#854D0E' }, darkColors: { main: '#FACC15', container: '#713F12', onContainer: '#FEF08A' } },
  ];

  resources.forEach((r, i) => {
    calendars[r.id] = palette[i % palette.length];
  });

  return calendars;
}

// ═══════════════════════════════════════════════════════════════════════
// ScheduleXCalendarApp — renders the actual calendar using hooks
// ═══════════════════════════════════════════════════════════════════════

interface CalendarAppProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  sxModules: any;
  events: ScheduleXEvent[];
  calendars: Record<string, CalendarColors>;
  onEventClick: (schedule: MaintenanceSchedule) => void;
  onEventDrop: (schedule: MaintenanceSchedule, newDate: string) => Promise<void>;
  enableResourceView: boolean;
}

const ScheduleXCalendarApp: React.FC<CalendarAppProps> = ({
  sxModules,
  events,
  calendars,
  onEventClick,
  onEventDrop,
  enableResourceView,
}) => {
  const { useCalendarApp, ScheduleXCalendar, viewMonthGrid, viewWeek, viewDay } = sxModules;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const dndPlugin = (sxModules as any).dndPlugin;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const timePlugin = (sxModules as any).timePlugin;

  // Convert events to match Schedule-X format
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const calendarEvents = useMemo<any[]>(() => {
    return events.map(evt => {
      // Copy only custom props, not the reserved ones
      const { id, title, start, end, calendarId, ...customProps } = evt;
      return {
        id,
        title: title || '',
        start,
        end: end || start,
        calendarId,
        ...customProps,
      };
    });
  }, [events]);

  // Determine dark mode
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark');

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const calendar = (useCalendarApp as any)({
    views: enableResourceView ? [viewMonthGrid, viewWeek, viewDay] : [viewMonthGrid, viewWeek],
    defaultView: enableResourceView ? 'week' : 'month-grid',
    events: calendarEvents,
    calendars,
    plugins: [dndPlugin, timePlugin].filter(Boolean),
    isDark,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    callbacks: {
      onEventClick: (event: any) => {
        const schedule = event.schedule as MaintenanceSchedule | undefined;
        if (schedule) onEventClick(schedule);
      },
      onEventUpdate: (event: any) => {
        const schedule = event.schedule as MaintenanceSchedule | undefined;
        if (schedule && event.start) {
          const newDate = typeof event.start === 'string' ? event.start : event.start.toString();
          onEventDrop(schedule, newDate);
        }
      },
    },
    firstDayOfWeek: 1,
  });

  return <ScheduleXCalendar calendarApp={calendar} />;
};

// ═══════════════════════════════════════════════════════════════════════
// ScheduleXWrapper — main component
// ═══════════════════════════════════════════════════════════════════════

const ScheduleXWrapper: React.FC<ScheduleXWrapperProps> = ({
  events,
  resources,
  calendars: calendarsProp,
  onEventClick,
  onEventDrop,
  enableResourceView = false,
  className = '',
}) => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [sxModules, setSxModules] = useState<any>(null);

  // Inject print styles once
  useEffect(() => {
    injectPrintStyles();
  }, []);

  // Lazy-load Schedule-X modules
  useEffect(() => {
    Promise.all([
      import('@schedule-x/react'),
      import('@schedule-x/calendar'),
      import('@schedule-x/drag-and-drop'),
      import('@schedule-x/current-time'),
    ]).then(([sxReact, sxCalendar, sxDnD, sxTime]) => {
      setSxModules({
        useCalendarApp: sxReact.useCalendarApp,
        ScheduleXCalendar: sxReact.ScheduleXCalendar,
        viewMonthGrid: sxCalendar.viewMonthGrid,
        viewWeek: sxCalendar.viewWeek,
        viewDay: sxCalendar.viewDay,
        dndPlugin: sxDnD.createDragAndDropPlugin(),
        timePlugin: sxTime.createCurrentTimePlugin(),
      });
    });
  }, []);

  // Build calendars config
  const calendars = useMemo(() => {
    if (calendarsProp) return calendarsProp;
    if (resources && enableResourceView) return buildCalendarsFromResources(resources);
    return {
      default: {
        colorName: 'blue' as const,
        lightColors: { main: '#3B82F6', container: '#DBEAFE', onContainer: '#1E40AF' },
        darkColors: { main: '#60A5FA', container: '#1E3A5F', onContainer: '#BFDBFE' },
      },
    };
  }, [calendarsProp, resources, enableResourceView]);

  if (!sxModules) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-2" />
          <p className="text-sm text-slate-400">Loading calendar...</p>
        </div>
      </div>
    );
  }

  return (
    <div className={`sx-wrapper ${className}`}>
      {/* Print header — hidden on screen, visible in print */}
      <div className="sx-wrapper-print-header hidden print:block">
        CCTV Health Monitor — Maintenance Schedule
      </div>

      <ScheduleXCalendarApp
        sxModules={sxModules}
        events={events}
        calendars={calendars}
        onEventClick={onEventClick}
        onEventDrop={onEventDrop}
        enableResourceView={enableResourceView}
      />
    </div>
  );
};

export default ScheduleXWrapper;
