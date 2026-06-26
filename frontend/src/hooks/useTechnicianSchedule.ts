// ═══════════════════════════════════════════════════════════════════════
// useTechnicianSchedule — Hook for Technician Resource Schedule
// P2-2.3: Resource Planning Calendar
//
// Provides:
//   - Technicians as resources with availability indicators
//   - Work orders mapped to time slots
//   - Conflict detection (overlapping WO for same tech)
//   - Drag & drop handler for reassignment
//
// Compliance:
//   - IEC 62443 SR 5.1 (Workflow — planning integrity)
//   - OWASP ASVS V1.8 (Architecture — stateless design)
// ═══════════════════════════════════════════════════════════════════════

import { useMemo, useCallback } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';
import { workOrdersApi } from '../services/workOrdersApi';
import { queryKeys } from './useApiQuery';
import type { EventDropArg } from '@fullcalendar/core';
import type { EventImpl } from '@fullcalendar/core/internal';
import type { WorkOrder } from '../services/workOrdersApi';
import type { User } from '../services/api';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

/** Availability level for a technician on a given day */
export type AvailabilityLevel = 'green' | 'yellow' | 'red';

/** Week day schedule entry */
export interface ScheduleSlot {
  id: string;
  workOrderId: string;
  technicianId: string;
  title: string;
  start: string;       // ISO datetime
  end: string;         // ISO datetime
  priority: WorkOrder['priority'];
  status: WorkOrder['status'];
  type: WorkOrder['type'];
}

/** Conflict between two work orders for the same technician */
export interface ScheduleConflict {
  workOrderIds: string[];
  technicianId: string;
  date: string;        // YYYY-MM-DD
  message: string;
}

/** Per-day load info for a technician */
export interface DayLoad {
  date: string;        // YYYY-MM-DD
  totalHours: number;
  maxHours: number;    // configurable (default 8)
  availability: AvailabilityLevel;
}

/** Combined result from the hook */
export interface TechnicianSchedule {
  slots: ScheduleSlot[];
  conflicts: ScheduleConflict[];
  dayLoads: Map<string, DayLoad[]>;   // techId → day loads
  isLoading: boolean;
  error: Error | null;
  handleEventDrop: (info: EventDropArg) => Promise<void>;
  refetch: () => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

const MAX_HOURS_PER_DAY = 8;
const OVERLOAD_THRESHOLD = 1.0;   // >100% → red
const WARNING_THRESHOLD = 0.75;   // >75%  → yellow

/**
 * Compute availability level from load ratio.
 */
function computeAvailability(usedHours: number, maxHours: number): AvailabilityLevel {
  const ratio = maxHours > 0 ? usedHours / maxHours : 0;
  if (ratio > OVERLOAD_THRESHOLD) return 'red';
  if (ratio >= WARNING_THRESHOLD) return 'yellow';
  return 'green';
}

/**
 * Build day loads for a technician from slots.
 */
function computeDayLoads(slots: ScheduleSlot[]): Map<string, DayLoad[]> {
  const techMap = new Map<string, Map<string, DayLoad>>();

  for (const slot of slots) {
    const day = slot.start.slice(0, 10); // YYYY-MM-DD
    if (!techMap.has(slot.technicianId)) {
      techMap.set(slot.technicianId, new Map());
    }
    const dayMap = techMap.get(slot.technicianId)!;
    const startMs = new Date(slot.start).getTime();
    const endMs = new Date(slot.end).getTime();
    const hours = (endMs - startMs) / (1000 * 60 * 60);

    if (dayMap.has(day)) {
      dayMap.get(day)!.totalHours += hours;
    } else {
      dayMap.set(day, {
        date: day,
        totalHours: hours,
        maxHours: MAX_HOURS_PER_DAY,
        availability: 'green',
      });
    }
  }

  // Finalize availability
  const result = new Map<string, DayLoad[]>();
  for (const [techId, dayMap] of techMap) {
    const loads: DayLoad[] = [];
    for (const [_, dl] of dayMap) {
      dl.availability = computeAvailability(dl.totalHours, dl.maxHours);
      loads.push(dl);
    }
    loads.sort((a, b) => a.date.localeCompare(b.date));
    result.set(techId, loads);
  }
  return result;
}

/**
 * Detect overlapping slots for the same technician on the same day.
 */
function detectConflicts(slots: ScheduleSlot[]): ScheduleConflict[] {
  const conflicts: ScheduleConflict[] = [];
  const byTechAndDay = new Map<string, ScheduleSlot[]>();

  for (const slot of slots) {
    const key = `${slot.technicianId}::${slot.start.slice(0, 10)}`;
    if (!byTechAndDay.has(key)) {
      byTechAndDay.set(key, []);
    }
    byTechAndDay.get(key)!.push(slot);
  }

  for (const [key, daySlots] of byTechAndDay) {
    if (daySlots.length < 2) continue;

    // Sort by start time
    daySlots.sort((a, b) => a.start.localeCompare(b.start));

    for (let i = 0; i < daySlots.length; i++) {
      for (let j = i + 1; j < daySlots.length; j++) {
        if (daySlots[i].end > daySlots[j].start) {
          const [techId, date] = key.split('::');
          conflicts.push({
            workOrderIds: [daySlots[i].workOrderId, daySlots[j].workOrderId],
            technicianId: techId,
            date,
            message: `Overlap: "${daySlots[i].title}" conflicts with "${daySlots[j].title}"`,
          });
        }
      }
    }
  }

  return conflicts;
}

/**
 * Transform a WorkOrder[] into ScheduleSlot[].
 * Uses `sla_deadline` or `created_at` to infer the day,
 * and `type` to infer duration (preventive=2h, corrective=3h, emergency=4h).
 */
function toSlots(workOrders: WorkOrder[], technicians: User[]): ScheduleSlot[] {
  const techSet = new Set(technicians.map(t => t.id));
  const slots: ScheduleSlot[] = [];

  for (const wo of workOrders) {
    if (!wo.assigned_to || !techSet.has(wo.assigned_to)) continue;

    // Determine start: use started_at or sla_deadline or created_at
    const baseDate = wo.started_at || wo.sla_deadline || wo.created_at;
    if (!baseDate) continue;

    const start = new Date(baseDate);
    // Default to 08:00 if no time component
    if (start.getHours() === 0 && start.getMinutes() === 0) {
      start.setHours(8, 0, 0, 0);
    }

    // Infer duration from type
    const durationHours =
      wo.type === 'emergency' ? 4 :
      wo.type === 'corrective' ? 3 : 2;

    const end = new Date(start.getTime() + durationHours * 60 * 60 * 1000);

    slots.push({
      id: `slot-${wo.id}`,
      workOrderId: wo.id,
      technicianId: wo.assigned_to,
      title: wo.device_name || wo.device_id || `WO #${wo.id.slice(0, 8)}`,
      start: start.toISOString(),
      end: end.toISOString(),
      priority: wo.priority,
      status: wo.status,
      type: wo.type,
    });
  }

  return slots;
}

// ═══════════════════════════════════════════════════════════════════════
// Hook
// ═══════════════════════════════════════════════════════════════════════

export function useTechnicianSchedule(): TechnicianSchedule {
  const queryClient = useQueryClient();

  // ── Fetch technicians (users with role 'technician') ─────────────
  const { data: users, isLoading: usersLoading, error: usersError } = useQuery({
    queryKey: queryKeys.users.all,
    queryFn: () => api.getUsers(),
    staleTime: 300_000,  // 5 min — reference data
    gcTime: 3_600_000,
  });

  const technicians = useMemo(() =>
    (users ?? []).filter((u): u is User =>
      u.role === 'technician' && u.status !== 'inactive'
    ),
    [users],
  );

  // ── Fetch work orders ───────────────────────────────────────────
  const { data: workOrders, isLoading: woLoading, error: woError, refetch } = useQuery({
    queryKey: [...queryKeys.workOrders.all, 'scheduled'],
    queryFn: () => workOrdersApi.getWorkOrders({
      status: 'open,in_progress',
      sort: 'sla_deadline:asc',
    }),
    staleTime: 30_000,
    gcTime: 300_000,
    refetchInterval: 120_000,
  });

  // ── Compute derived data ────────────────────────────────────────
  const slots = useMemo(() =>
    toSlots(workOrders ?? [], technicians),
    [workOrders, technicians],
  );

  const conflicts = useMemo(() =>
    detectConflicts(slots),
    [slots],
  );

  const dayLoads = useMemo(() =>
    computeDayLoads(slots),
    [slots],
  );

  // ── Drag & drop handler ─────────────────────────────────────────
  const handleEventDrop = useCallback(async (info: EventDropArg) => {
    const woId = info.event.extendedProps?.workOrderId as string | undefined;
    if (!woId) { info.revert(); return; }

    // Extract new resource ID from the dropped event.
    // FullCalendar resource-timeline attaches resourceIds to the event after drop.
    // Using type assertion for resource plugin properties.
    const event = info.event as EventImpl & { getResources?: () => Array<{ id: string }> };
    const drop = info as EventDropArg & { newResource?: { id: string } };
    const newTechId = event.getResources?.()?.[0]?.id ?? drop.newResource?.id;

    if (!newTechId) { info.revert(); return; }

    try {
      await workOrdersApi.assignWorkOrder(woId, newTechId);
      // Invalidate queries to refresh data
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    } catch {
      info.revert();
    }
  }, [queryClient]);

  const isLoading = usersLoading || woLoading;
  const error = usersError ?? woError ?? null;

  return {
    slots,
    conflicts,
    dayLoads,
    isLoading,
    error: error as Error | null,
    handleEventDrop,
    refetch,
  };
}
