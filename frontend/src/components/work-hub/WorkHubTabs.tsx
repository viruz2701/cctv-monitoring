// ═══════════════════════════════════════════════════════════════════════
// WorkHubTabs — табы для Unified Work Hub (UX-1.2)
//
// Табы: [My Tasks] [Team] [Requests] с badge-счётчиками.
// Состояние синхронизируется с URL searchParams (?tab=tasks).
//
// Compliance:
//   - WCAG 2.1 AA: ARIA tabpanel, aria-selected, keyboard navigation
//   - ISO 27001 A.12.4 (Audit trail — через URL state, не теряем контекст)
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { WorkOrder } from '../../services/workOrdersApi';

// ── Types ─────────────────────────────────────────────────────────────

export type HubTab = 'tasks' | 'team' | 'requests';

export interface WorkHubTabsProps {
  /** Все work orders (unfiltered) для вычисления badge-счётчиков */
  workOrders: WorkOrder[];
  /** Текущий ID пользователя для "My Tasks" счётчика */
  currentUserId?: string;
  /** Кол-во тикетов (Requests tab) */
  ticketsCount?: number;
  /** Active tab (controlled) */
  activeTab: HubTab;
  /** Tab change handler */
  onTabChange: (tab: HubTab) => void;
}

// ── Constants ─────────────────────────────────────────────────────────

export const VALID_TABS: ReadonlySet<string> = new Set<HubTab>(['tasks', 'team', 'requests']);

const TAB_DEFS: { id: HubTab; labelKey: string; defaultLabel: string }[] = [
  { id: 'tasks', labelKey: 'hub_tab_my_tasks', defaultLabel: 'My Tasks' },
  { id: 'team', labelKey: 'hub_tab_team', defaultLabel: 'Team' },
  { id: 'requests', labelKey: 'hub_tab_requests', defaultLabel: 'Requests' },
];

// ── Hook: URL-synced active tab ───────────────────────────────────────

export function useActiveTab(): [HubTab, (tab: HubTab) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  const activeTab = useMemo<HubTab>(() => {
    const tab = searchParams.get('tab');
    if (tab && VALID_TABS.has(tab)) {
      return tab as HubTab;
    }
    return 'tasks';
  }, [searchParams]);

  const setActiveTab = useCallback(
    (tab: HubTab) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set('tab', tab);
        return next;
      }, { replace: true });
    },
    [setSearchParams],
  );

  return [activeTab, setActiveTab];
}

// ── Component ─────────────────────────────────────────────────────────

export function WorkHubTabs({
  workOrders,
  currentUserId,
  ticketsCount = 0,
  activeTab,
  onTabChange,
}: WorkHubTabsProps) {
  const { t } = useTranslation();

  // ── Badge counters ───────────────────────────────────────────────
  const badges = useMemo(() => {
    const now = new Date();
    const activeWOs = workOrders.filter(
      (wo) => wo.status !== 'completed' && wo.status !== 'cancelled',
    );

    return {
      tasks: currentUserId
        ? activeWOs.filter((wo) => wo.assigned_to === currentUserId).length
        : 0,
      team: activeWOs.length,
      requests: ticketsCount,
    };
  }, [workOrders, currentUserId, ticketsCount]);

  // ── Render ───────────────────────────────────────────────────────
  return (
    <div className="flex border-b border-slate-200 dark:border-slate-700 mb-6" role="tablist">
      {TAB_DEFS.map((tabDef) => {
        const isActive = activeTab === tabDef.id;
        const count = badges[tabDef.id];

        return (
          <button
            key={tabDef.id}
            role="tab"
            aria-selected={isActive}
            aria-controls={`hub-panel-${tabDef.id}`}
            id={`hub-tab-${tabDef.id}`}
            onClick={() => onTabChange(tabDef.id)}
            className={`
              px-4 py-2.5 text-sm font-medium border-b-2 transition-colors cursor-pointer
              ${
                isActive
                  ? 'border-blue-600 text-blue-600 dark:text-blue-400 dark:border-blue-400'
                  : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 hover:border-slate-300 dark:hover:border-slate-600'
              }
            `}
          >
            <span className="flex items-center gap-1.5">
              {t(tabDef.labelKey) || tabDef.defaultLabel}
              {count > 0 && (
                <span
                  className={`
                    inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 
                    text-xs font-semibold rounded-full 
                    ${
                      isActive
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-400'
                    }
                  `}
                >
                  {count}
                </span>
              )}
            </span>
          </button>
        );
      })}
    </div>
  );
}

export default WorkHubTabs;
