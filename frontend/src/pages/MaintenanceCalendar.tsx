// ═══════════════════════════════════════════════════════════════════════
// MaintenanceCalendar — Maintenance schedule with calendar views
// UX-4.3: Maintenance Calendar UI
//   - Month/Week/Day views (via Schedule-X)
//   - Color coding by priority/technician
//   - Drag-n-drop for rescheduling
//   - Conflict detection
//   - iCal export
//   - Google Calendar / Outlook sync
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { useMaintenanceSchedules, useUpdateMaintenanceSchedule } from '../hooks/useApiQuery';
import {
  CalendarDays,
  CalendarClock,
  Download,
  RefreshCw,
  Plus,
  AlertTriangle,
  CheckCircle2,
  Loader2,
  ExternalLink,
  Settings,
} from '../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, Badge, StatsCard, Modal, useToast } from '../components/ui';
import { WorkOrderCalendar } from '../components/work-orders/WorkOrderCalendar';
import type { MaintenanceSchedule } from '../hooks/useApiQuery/shared';
import type { WorkOrder } from '../services/workOrdersApi';

// ─── Mock data helpers (replace with real API) ────────────────────────

function scheduleToWorkOrder(schedule: MaintenanceSchedule): WorkOrder {
  return {
    id: schedule.id || '',
    device_id: schedule.device_id || '',
    type: 'preventive',
    status: 'open',
    priority: (schedule.priority as WorkOrder['priority']) || 'medium',
    sla_deadline: schedule.next_due,
    created_at: schedule.created_at || new Date().toISOString(),
    updated_at: schedule.created_at || new Date().toISOString(),
    checklist: [],
    photos: [],
    parts_used: [],
  };
}

// ─── Conflict Detection ──────────────────────────────────────────────

interface Conflict {
  scheduleId: string;
  conflictingWith: string;
  date: string;
  reason: string;
}

function detectConflicts(schedules: MaintenanceSchedule[]): Conflict[] {
  const conflicts: Conflict[] = [];
  const dateMap = new Map<string, MaintenanceSchedule[]>();

  for (const schedule of schedules) {
    if (!schedule.next_due) continue;
    const date = schedule.next_due.slice(0, 10);
    if (!dateMap.has(date)) dateMap.set(date, []);
    dateMap.get(date)!.push(schedule);
  }

  for (const [date, items] of dateMap.entries()) {
    if (items.length > 1) {
      for (let i = 1; i < items.length; i++) {
        conflicts.push({
          scheduleId: items[i].id || '',
          conflictingWith: items[0].id || '',
          date,
          reason: `Multiple schedules on same day: ${items.map((s) => s.device_name || s.device_id).join(', ')}`,
        });
      }
    }
  }

  return conflicts;
}

// ─── iCal Export ─────────────────────────────────────────────────────

function generateICal(schedules: MaintenanceSchedule[]): string {
  const lines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//CCTV Health Monitor//Maintenance Calendar//EN',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
  ];

  for (const schedule of schedules) {
    if (!schedule.next_due) continue;
    const dtStart = schedule.next_due.replace(/[-:]/g, '').slice(0, 15) + 'Z';
    lines.push('BEGIN:VEVENT');
    lines.push(`UID:${schedule.id}@cctv-monitor`);
    lines.push(`DTSTART:${dtStart}`);
    lines.push(`DTEND:${dtStart}`);
    lines.push(`SUMMARY:${schedule.device_name || schedule.device_id || 'Maintenance'}`);
    lines.push(`DESCRIPTION:${schedule.notes || ''}`);
    lines.push('END:VEVENT');
  }

  lines.push('END:VCALENDAR');
  return lines.join('\r\n');
}

function downloadICal(content: string, filename = 'maintenance-calendar.ics') {
  const blob = new Blob([content], { type: 'text/calendar;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// ─── Main Component ──────────────────────────────────────────────────

export function MaintenanceCalendar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const [searchParams, setSearchParams] = useSearchParams();

  const { data: schedules, isLoading } = useMaintenanceSchedules();
  const updateSchedule = useUpdateMaintenanceSchedule();

  const [viewMode, setViewMode] = useState<'month' | 'week' | 'day'>(
    (searchParams.get('view') as 'month' | 'week' | 'day') || 'month',
  );
  const [showICalModal, setShowICalModal] = useState(false);
  const [exporting, setExporting] = useState(false);

  const scheduleList = useMemo(() => {
    if (!schedules) return [];
    return Array.isArray(schedules) ? (schedules as MaintenanceSchedule[]) : [];
  }, [schedules]);

  const workOrders = useMemo(() => scheduleList.map(scheduleToWorkOrder), [scheduleList]);

  const conflicts = useMemo(() => detectConflicts(scheduleList), [scheduleList]);

  const overdueCount = useMemo(() => {
    const now = new Date();
    return scheduleList.filter((s) => {
      if (!s.next_due) return false;
      return new Date(s.next_due) < now;
    }).length;
  }, [scheduleList]);

  const dueThisWeek = useMemo(() => {
    const now = new Date();
    const weekEnd = new Date(now);
    weekEnd.setDate(weekEnd.getDate() + 7);
    return scheduleList.filter((s) => {
      if (!s.next_due) return false;
      const due = new Date(s.next_due);
      return due >= now && due <= weekEnd;
    }).length;
  }, [scheduleList]);

  // ── Handlers ─────────────────────────────────────────────────────

  const handleViewModeChange = useCallback(
    (mode: 'month' | 'week' | 'day') => {
      setViewMode(mode);
      setSearchParams({ view: mode }, { replace: true });
    },
    [setSearchParams],
  );

  const handleDateChange = useCallback(
    async (id: string, newDate: string) => {
      try {
        await updateSchedule.mutateAsync({
          id,
          data: { next_due: newDate },
        });
        toast.success(t('schedule_updated') || 'Schedule updated');
      } catch {
        toast.error(t('failed_to_update') || 'Failed to update schedule');
      }
    },
    [updateSchedule, toast, t],
  );

  const handleEventClick = useCallback(
    (workOrder: WorkOrder) => {
      navigate(`/maintenance/${workOrder.id}`);
    },
    [navigate],
  );

  const handleDateClick = useCallback(
    (date: Date) => {
      // Navigate to create maintenance on selected date
      navigate(`/maintenance?date=${date.toISOString().slice(0, 10)}`);
    },
    [navigate],
  );

  const handleICalExport = useCallback(() => {
    try {
      const ical = generateICal(scheduleList);
      downloadICal(ical);
      toast.success(t('ical_exported') || 'Calendar exported');
    } catch {
      toast.error(t('export_failed') || 'Export failed');
    }
  }, [scheduleList, toast, t]);

  const handleGoogleSync = useCallback(() => {
    // Open Google Calendar with import URL
    const ical = generateICal(scheduleList);
    const encoded = encodeURIComponent(ical);
    window.open(
      `https://calendar.google.com/calendar/u/0/r/week?cid=${encoded}`,
      '_blank',
    );
  }, [scheduleList]);

  // ── Render ───────────────────────────────────────────────────────

  return (
    <div className="p-4 md:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('maintenance_calendar') || 'Maintenance Calendar'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('maintenance_calendar_description') || 'View and manage maintenance schedules'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            icon={<Download className="w-4 h-4" />}
            onClick={() => setShowICalModal(true)}
          >
            {t('export') || 'Export'}
          </Button>
          <Button
            variant="primary"
            size="sm"
            icon={<Plus className="w-4 h-4" />}
            onClick={() => navigate('/maintenance?create=true')}
          >
            {t('new_schedule') || 'New Schedule'}
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('total_schedules') || 'Total Schedules'}
          value={scheduleList.length}
          icon={CalendarDays}
          iconBgColor="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('overdue') || 'Overdue'}
          value={overdueCount}
          icon={AlertTriangle}
          iconBgColor={overdueCount > 0 ? 'bg-red-50' : 'bg-emerald-50'}
          iconColor={overdueCount > 0 ? 'text-red-600' : 'text-emerald-600'}
        />
        <StatsCard
          title={t('due_this_week') || 'Due This Week'}
          value={dueThisWeek}
          icon={CalendarClock}
          iconBgColor="bg-amber-50"
          iconColor="text-amber-600"
        />
        <StatsCard
          title={t('conflicts') || 'Conflicts'}
          value={conflicts.length}
          icon={AlertTriangle}
          iconBgColor={conflicts.length > 0 ? 'bg-red-50' : 'bg-slate-50'}
          iconColor={conflicts.length > 0 ? 'text-red-600' : 'text-slate-400'}
        />
      </div>

      {/* Conflict Alerts */}
      {conflicts.length > 0 && (
        <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800/50 rounded-lg">
          <div className="flex items-center gap-2 mb-1">
            <AlertTriangle className="w-4 h-4 text-red-500" />
            <span className="text-sm font-medium text-red-700 dark:text-red-400">
              {t('conflicts_detected') || 'Conflicts Detected'}
            </span>
          </div>
          <ul className="space-y-1">
            {conflicts.slice(0, 3).map((conflict) => (
              <li key={`${conflict.scheduleId}-${conflict.date}`} className="text-xs text-red-600 dark:text-red-400">
                {conflict.date}: {conflict.reason}
              </li>
            ))}
            {conflicts.length > 3 && (
              <li className="text-xs text-slate-500">
                +{conflicts.length - 3} {t('more_conflicts') || 'more conflicts'}
              </li>
            )}
          </ul>
        </div>
      )}

      {/* View Mode Toggle */}
      <div className="flex items-center gap-2 border-b border-slate-200 dark:border-slate-700 pb-2">
        {(['month', 'week', 'day'] as const).map((mode) => (
          <button
            key={mode}
            onClick={() => handleViewModeChange(mode)}
            className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
              viewMode === mode
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
            }`}
          >
            {mode === 'month'
              ? (t('month') || 'Month')
              : mode === 'week'
                ? (t('week') || 'Week')
                : (t('day') || 'Day')}
          </button>
        ))}
      </div>

      {/* Calendar */}
      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      ) : scheduleList.length === 0 ? (
        <Card>
          <CardBody>
            <div className="flex flex-col items-center justify-center py-16">
              <CalendarDays className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" />
              <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300">
                {t('no_maintenance_schedules') || 'No Maintenance Schedules'}
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                {t('no_schedules_description') || 'Create your first maintenance schedule'}
              </p>
              <Button
                variant="primary"
                className="mt-4"
                icon={<Plus className="w-4 h-4" />}
                onClick={() => navigate('/maintenance?create=true')}
              >
                {t('create_schedule') || 'Create Schedule'}
              </Button>
            </div>
          </CardBody>
        </Card>
      ) : (
        <WorkOrderCalendar
          workOrders={workOrders}
          technicians={[]}
          onDateChange={handleDateChange}
          onEventClick={handleEventClick}
          onDateClick={handleDateClick}
        />
      )}

      {/* Export Modal */}
      <Modal
        isOpen={showICalModal}
        onClose={() => setShowICalModal(false)}
        title={t('export_calendar') || 'Export Calendar'}
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setShowICalModal(false)}>
              {t('cancel') || 'Cancel'}
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            {t('export_description') || 'Export your maintenance schedule to use with your favorite calendar app'}
          </p>

          <Button
            variant="outline"
            fullWidth
            icon={<Download className="w-4 h-4" />}
            onClick={() => {
              handleICalExport();
              setShowICalModal(false);
            }}
          >
            {t('download_ical') || 'Download iCal (.ics)'}
          </Button>

          <Button
            variant="outline"
            fullWidth
            icon={<ExternalLink className="w-4 h-4" />}
            onClick={() => {
              handleGoogleSync();
              setShowICalModal(false);
            }}
          >
            {t('sync_google_calendar') || 'Sync with Google Calendar'}
          </Button>

          <Button
            variant="outline"
            fullWidth
            icon={<ExternalLink className="w-4 h-4" />}
            onClick={() => {
              // Outlook web calendar
              const ical = generateICal(scheduleList);
              const encoded = encodeURIComponent(ical);
              window.open(
                `https://outlook.live.com/calendar/0/calendar/import?url=${encoded}`,
                '_blank',
              );
              setShowICalModal(false);
            }}
          >
            {t('sync_outlook') || 'Sync with Outlook'}
          </Button>
        </div>
      </Modal>
    </div>
  );
}

export default MaintenanceCalendar;
