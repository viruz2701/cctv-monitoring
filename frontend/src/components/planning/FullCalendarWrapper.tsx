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

interface FullCalendarWrapperProps {
  /** Массив событий в формате FullCalendar */
  events: Record<string, unknown>[];
  /** Ресурсы для resourceTimeline (опционально) */
  resources?: Resource[];
  onEventClick: (schedule: MaintenanceSchedule) => void;
  onEventDrop: (schedule: MaintenanceSchedule, newDate: string) => Promise<void>;
  /** Включить resourceTimeline views */
  enableResourceView?: boolean;
  className?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Print styles
// ═══════════════════════════════════════════════════════════════════════

const PRINT_STYLE_ID = 'fc-wrapper-print-styles';

function injectPrintStyles(): void {
  if (typeof document === 'undefined' || document.getElementById(PRINT_STYLE_ID)) return;
  const style = document.createElement('style');
  style.id = PRINT_STYLE_ID;
  style.textContent = `
@media print {
  .fc-wrapper-print-header {
    display: block !important;
    text-align: center;
    font-size: 16pt;
    font-weight: 700;
    margin-bottom: 16px;
    color: #1e293b;
  }
  .fc-wrapper .fc-header-toolbar .fc-button {
    display: none !important;
  }
  .fc-wrapper .fc-header-toolbar .fc-toolbar-title {
    font-size: 14pt !important;
    font-weight: 700 !important;
  }
  .fc-wrapper .fc {
    font-size: 9pt !important;
  }
  .fc-wrapper .fc-view-harness {
    page-break-inside: avoid;
  }
  .fc-wrapper .fc-scrollgrid {
    page-break-inside: avoid;
  }
  .fc-wrapper > :not(.fc-wrapper-print-header):not(.fc) {
    display: none !important;
  }
}
  `;
  document.head.appendChild(style);
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

/**
 * P1-PERF.1: Lazy-loaded FullCalendar wrapper.
 * FullCalendar (~500KB) загружается только когда пользователь переключается
 * на календарный режим просмотра.
 *
 * P2-2.3: Поддержка resourceTimeline с drag между ресурсами.
 */
const FullCalendarWrapper: React.FC<FullCalendarWrapperProps> = ({
  events,
  resources,
  onEventClick,
  onEventDrop,
  enableResourceView = false,
  className = '',
}) => {
  const [FCModule, setFCModule] = useState<{
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    default: any;
    dayGridPlugin: any;
    interactionPlugin: any;
    resourceTimelinePlugin?: any;
  } | null>(null);

  // Inject print styles once
  useEffect(() => {
    injectPrintStyles();
  }, []);

  // Lazy-load plugins based on whether resource view is enabled
  useEffect(() => {
    const imports = [
      import('@fullcalendar/react'),
      import('@fullcalendar/daygrid'),
      import('@fullcalendar/interaction'),
    ];

    if (enableResourceView) {
      imports.push(import('@fullcalendar/resource-timeline'));
    }

    Promise.all(imports).then(([fc, dayGrid, interaction, resourceTimeline]) => {
      setFCModule({
        default: fc.default,
        dayGridPlugin: dayGrid.default,
        interactionPlugin: interaction.default,
        resourceTimelinePlugin: resourceTimeline?.default,
      });
    });
  }, [enableResourceView]);

  // Build plugins array based on what's loaded
  const plugins = useMemo(() => {
    if (!FCModule) return [];
    const list = [FCModule.dayGridPlugin, FCModule.interactionPlugin];
    if (FCModule.resourceTimelinePlugin) {
      list.push(FCModule.resourceTimelinePlugin);
    }
    return list;
  }, [FCModule]);

  // Build views config
  const views = useMemo(() => {
    if (!enableResourceView) return undefined;
    return {
      resourceTimelineWeek: {
        type: 'resourceTimeline' as const,
        duration: { weeks: 1 },
        buttonText: 'Week',
      },
      resourceTimelineDay: {
        type: 'resourceTimeline' as const,
        duration: { days: 1 },
        buttonText: 'Day',
      },
    };
  }, [enableResourceView]);

  // Build header toolbar
  const headerToolbar = useMemo(() => {
    if (enableResourceView) {
      return {
        left: 'prev,next today',
        center: 'title',
        right: 'dayGridMonth,dayGridWeek,resourceTimelineWeek,resourceTimelineDay',
      };
    }
    return {
      left: 'prev,next today',
      center: 'title',
      right: 'dayGridMonth,dayGridWeek',
    };
  }, [enableResourceView]);

  if (!FCModule) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-2" />
          <p className="text-sm text-slate-400">Loading calendar...</p>
        </div>
      </div>
    );
  }

  const { default: FullCalendar } = FCModule;

  return (
    <div className={`fc-wrapper ${className}`}>
      {/* Print header — hidden on screen, visible in print */}
      <div className="fc-wrapper-print-header hidden print:block">
        CCTV Health Monitor — Maintenance Schedule
      </div>

      <FullCalendar
        plugins={plugins}
        initialView={enableResourceView ? 'resourceTimelineWeek' : 'dayGridMonth'}
        resources={resources}
        events={events}
        views={views}
        headerToolbar={headerToolbar}
        editable={true}
        droppable={enableResourceView}
        eventClick={(info: any) => {
          const schedule = info.event.extendedProps.schedule as MaintenanceSchedule;
          onEventClick(schedule);
        }}
        eventDrop={async (info: any) => {
          const schedule = info.event.extendedProps.schedule as MaintenanceSchedule;
          const newDate = info.event.startStr;

          // If resource changed (inter-resource drag), update assignment
          if (enableResourceView && info.newResource) {
            // Additional logic for resource change can be added here
            // For now, pass through to the standard handler
          }

          try {
            await onEventDrop(schedule, newDate);
            info.el.style.opacity = '1';
          } catch {
            info.revert();
          }
        }}
        eventReceive={enableResourceView ? async (info: any) => {
          // Handle external drop onto resource
          const schedule = info.event.extendedProps?.schedule as MaintenanceSchedule;
          if (schedule && info.event.startStr) {
            try {
              await onEventDrop(schedule, info.event.startStr);
            } catch {
              info.revert();
            }
          }
        } : undefined}
        height="auto"
      />
    </div>
  );
};

export default FullCalendarWrapper;
