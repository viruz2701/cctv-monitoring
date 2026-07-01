// ═══════════════════════════════════════════════════════════════════════
// QuickFilters (Hub) — быстрые фильтры для Unified Work Hub (UX-1.2)
//
// Фильтры: Overdue, Critical, Unassigned — синхронизируются с URL (?filter=).
// В отличие от QuickFilters в work-orders, здесь нет "All" и "My Orders",
// т.к. это покрывается табами.
//
// Compliance:
//   - WCAG 2.1 AA: aria-pressed для кнопок фильтрации
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Clock, AlertOctagon, Users } from '../ui/Icons';
import type { WorkOrder } from '../../services/workOrdersApi';

// ── Types ─────────────────────────────────────────────────────────────

export type HubQuickFilterKey = 'all' | 'overdue' | 'critical' | 'unassigned';

export interface QuickFiltersProps {
  /** Все work orders текущего таба (unfiltered) для вычисления счётчиков */
  workOrders: WorkOrder[];
  /** Current active filter */
  activeFilter: HubQuickFilterKey;
  /** Callback when filter changes */
  onChange: (filter: HubQuickFilterKey) => void;
  /** Custom class name */
  className?: string;
}

// ── Filter definitions ────────────────────────────────────────────────

const FILTER_DEFS: {
  key: HubQuickFilterKey;
  label: string;
  icon: React.ReactNode;
}[] = [
  { key: 'all', label: 'All', icon: null },
  { key: 'overdue', label: 'Overdue', icon: <Clock size={14} /> },
  { key: 'critical', label: 'Critical', icon: <AlertOctagon size={14} /> },
  { key: 'unassigned', label: 'Unassigned', icon: <Users size={14} /> },
];

// ── Hook: URL-synced quick filter ─────────────────────────────────────

export function useHubQuickFilter(): [HubQuickFilterKey, (filter: HubQuickFilterKey) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  const activeFilter = useMemo<HubQuickFilterKey>(() => {
    const qf = searchParams.get('filter');
    if (qf === 'overdue' || qf === 'critical' || qf === 'unassigned') {
      return qf;
    }
    return 'all';
  }, [searchParams]);

  const setActiveFilter = useCallback(
    (filter: HubQuickFilterKey) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (filter === 'all') {
          next.delete('filter');
        } else {
          next.set('filter', filter);
        }
        return next;
      }, { replace: true });
    },
    [setSearchParams],
  );

  return [activeFilter, setActiveFilter];
}

// ── Component ─────────────────────────────────────────────────────────

export function QuickFilters({
  workOrders,
  activeFilter,
  onChange,
  className = '',
}: QuickFiltersProps) {
  // ── Counters per filter ─────────────────────────────────────────
  const counters = useMemo(() => {
    const now = new Date();
    return {
      all: workOrders.length,
      overdue: workOrders.filter((wo) => {
        const notFinished = wo.status !== 'completed' && wo.status !== 'cancelled';
        const isOverdue = wo.sla_deadline && new Date(wo.sla_deadline) < now;
        return notFinished && !!isOverdue;
      }).length,
      critical: workOrders.filter((wo) => wo.priority === 'critical').length,
      unassigned: workOrders.filter((wo) => !wo.assigned_to).length,
    };
  }, [workOrders]);

  return (
    <div className={`flex items-center gap-2 flex-wrap ${className}`}>
      {FILTER_DEFS.map((filter) => {
        const isActive = activeFilter === filter.key;
        const count = counters[filter.key];

        return (
          <button
            key={filter.key}
            onClick={() => onChange(filter.key)}
            className={`
              inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full transition-all cursor-pointer
              ${
                isActive
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 ring-1 ring-blue-300 dark:ring-blue-700 shadow-sm'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-slate-800 dark:text-gray-400 dark:hover:bg-slate-700'
              }
            `}
            aria-pressed={isActive}
          >
            {filter.icon}
            <span>{filter.label}</span>
            {count > 0 && (
              <span
                className={`
                  inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-[10px] font-semibold leading-none
                  ${
                    isActive
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-200 text-gray-600 dark:bg-slate-700 dark:text-gray-400'
                  }
                `}
              >
                {count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

export default QuickFilters;
