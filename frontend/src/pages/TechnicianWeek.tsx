// ═══════════════════════════════════════════════════════════════════════
// TechnicianWeek — Resource Planning Calendar page
// P2-2.3: Resource Planning Calendar
//
//   - Week view using FullCalendar resourceTimelineWeek
//   - Technicians as resources, WO as draggable events
//   - Conflict detection + availability indicators
//   - Print-friendly view via @media print
//
// Compliance:
//   - IEC 62443 SR 5.1 (Workflow — planning integrity)
//   - OWASP ASVS V1.8 (Stateless architecture)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { useTechnicianSchedule } from '../hooks/useTechnicianSchedule';
import { TechnicianCalendar } from '../components/planning/TechnicianCalendar';
import { useUsers } from '../hooks/useApiQuery';
import { useNavigate } from 'react-router-dom';

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function TechnicianWeek() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  // ── Schedules ───────────────────────────────────────────────────
  const {
    slots,
    conflicts,
    dayLoads,
    isLoading,
    error,
    handleEventDrop,
  } = useTechnicianSchedule();

  // ── Technicians (for the calendar) ──────────────────────────────
  const { data: users } = useUsers();
  const technicians = React.useMemo(() =>
    (users ?? []).filter(u => u.role === 'technician' && u.status !== 'inactive'),
    [users],
  );

  // ── Navigation handler ──────────────────────────────────────────
  const handleEventClick = React.useCallback((workOrderId: string) => {
    navigate(`/work-orders/${workOrderId}`);
  }, [navigate]);

  // ── Render ──────────────────────────────────────────────────────
  return (
    <div className="technician-week-page p-4 md:p-6 space-y-4">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
          {t('resource_planning') || 'Resource Planning'}
        </h1>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {t('technician_schedule') || 'Technician weekly schedule'}
        </p>
      </div>

      {/* Error state */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-600 dark:text-red-400">
            {t('schedule_load_error') || 'Failed to load schedule'}: {error.message}
          </p>
        </div>
      )}

      {/* Calendar */}
      <TechnicianCalendar
        technicians={technicians}
        slots={slots}
        conflicts={conflicts}
        dayLoads={dayLoads}
        isLoading={isLoading}
        onEventDrop={handleEventDrop}
        onEventClick={handleEventClick}
      />
    </div>
  );
}
