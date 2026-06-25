import React, { useMemo, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { User, Clock, AlertOctagon, Users, List } from 'lucide-react';
import { WorkOrder } from '../../services/workOrdersApi';

// ═══════════════════════════════════════════════════════════════════════
// QuickFilters Component (WO-4.2.2)
// Filter chips with counters synced to URL search params.
// ═══════════════════════════════════════════════════════════════════════

export type QuickFilterKey = 'all' | 'mine' | 'overdue' | 'unassigned' | 'critical';

interface QuickFilterDef {
  key: QuickFilterKey;
  label: string;
  icon: React.ReactNode;
  color: string;
  activeColor: string;
}

const FILTER_DEFS: QuickFilterDef[] = [
  { key: 'all', label: 'All', icon: <List size={14} />, color: '', activeColor: '' },
  { key: 'mine', label: 'My Orders', icon: <User size={14} />, color: '', activeColor: '' },
  { key: 'overdue', label: 'Overdue', icon: <Clock size={14} />, color: '', activeColor: '' },
  { key: 'unassigned', label: 'Unassigned', icon: <Users size={14} />, color: '', activeColor: '' },
  { key: 'critical', label: 'Critical', icon: <AlertOctagon size={14} />, color: '', activeColor: '' },
];

interface QuickFiltersProps {
  /** All work orders (unfiltered) for counter computation */
  workOrders: WorkOrder[];
  /** Current active filter */
  activeFilter: QuickFilterKey;
  /** Current user ID for "My Orders" filter */
  currentUserId?: string;
  /** Callback when filter changes */
  onChange: (filter: QuickFilterKey) => void;
  /** Custom class name */
  className?: string;
}

export function QuickFilters({
  workOrders,
  activeFilter,
  currentUserId,
  onChange,
  className = '',
}: QuickFiltersProps) {
  // ── Counters per filter ────────────────────────────────────────────
  const counters = useMemo(() => {
    const now = new Date();
    return {
      all: workOrders.length,
      mine: currentUserId
        ? workOrders.filter((wo) => wo.assigned_to === currentUserId).length
        : 0,
      overdue: workOrders.filter((wo) => {
        const notFinished = wo.status !== 'completed' && wo.status !== 'cancelled';
        const isOverdue = wo.sla_deadline && new Date(wo.sla_deadline) < now;
        return notFinished && !!isOverdue;
      }).length,
      unassigned: workOrders.filter((wo) => !wo.assigned_to).length,
      critical: workOrders.filter((wo) => wo.priority === 'critical').length,
    };
  }, [workOrders, currentUserId]);

  return (
    <div className={`flex items-center gap-2 flex-wrap ${className}`}>
      {FILTER_DEFS.map((filter) => {
        const isActive = activeFilter === filter.key;
        const count = counters[filter.key];

        return (
          <button
            key={filter.key}
            onClick={() => onChange(filter.key)}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full transition-all ${
              isActive
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 ring-1 ring-blue-300 dark:ring-blue-700 shadow-sm'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-slate-800 dark:text-gray-400 dark:hover:bg-slate-700'
            }`}
            aria-pressed={isActive}
          >
            {filter.icon}
            <span>{filter.label}</span>
            {count > 0 && (
              <span
                className={`inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-[10px] font-semibold leading-none ${
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-200 text-gray-600 dark:bg-slate-700 dark:text-gray-400'
                }`}
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

// ═══════════════════════════════════════════════════════════════════════
// useQuickFilter — hook for URL-synced quick filter state
// ═══════════════════════════════════════════════════════════════════════

export function useQuickFilter(): [QuickFilterKey, (filter: QuickFilterKey) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  const activeFilter = useMemo<QuickFilterKey>(() => {
    const qf = searchParams.get('filter');
    if (qf === 'mine' || qf === 'overdue' || qf === 'unassigned' || qf === 'critical') {
      return qf;
    }
    return 'all';
  }, [searchParams]);

  const setActiveFilter = useCallback(
    (filter: QuickFilterKey) => {
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
