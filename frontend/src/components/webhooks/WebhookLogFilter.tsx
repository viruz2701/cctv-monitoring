// ═══════════════════════════════════════════════════════════════════════
// WebhookLogFilter — фильтрация логов доставки вебхуков (P2-3.1)
//
// Features:
//   - Filter by status (success/failed/all)
//   - Filter by date range
//   - Filter by event type
//   - Search by URL
//
// Compliance:
//   - OWASP ASVS V5 (Input validation — controlled filter inputs)
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Filter, Search, Calendar, X } from '../ui/Icons';
import { Badge } from '../ui';
import { EVENT_GROUPS } from '../../hooks/useWebhooks';
import type { WebhookLogEntry } from '../../hooks/useWebhooks';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface LogFilterState {
  status: 'all' | 'success' | 'failed';
  dateFrom: string;
  dateTo: string;
  eventType: string;
  urlSearch: string;
}

export const DEFAULT_LOG_FILTER: LogFilterState = {
  status: 'all',
  dateFrom: '',
  dateTo: '',
  eventType: '',
  urlSearch: '',
};

interface WebhookLogFilterProps {
  logs?: WebhookLogEntry[];
  filter: LogFilterState;
  onFilterChange: (filter: LogFilterState) => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function filterLogs(logs: WebhookLogEntry[], filter: LogFilterState): WebhookLogEntry[] {
  return logs.filter((log) => {
    // Status filter
    if (filter.status !== 'all' && log.status !== filter.status) return false;

    // Event type filter
    if (filter.eventType && log.event_type !== filter.eventType) return false;

    // URL search
    if (filter.urlSearch) {
      const query = filter.urlSearch.toLowerCase();
      if (!log.request_url?.toLowerCase().includes(query)) return false;
    }

    // Date range
    if (filter.dateFrom && new Date(log.created_at) < new Date(filter.dateFrom)) return false;
    if (filter.dateTo) {
      const endDate = new Date(filter.dateTo);
      endDate.setHours(23, 59, 59, 999);
      if (new Date(log.created_at) > endDate) return false;
    }

    return true;
  });
}

// Derive unique event types from logs
function getEventTypes(logs: WebhookLogEntry[]): string[] {
  const types = new Set(logs.map((l) => l.event_type).filter(Boolean));
  return Array.from(types).sort();
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function WebhookLogFilter({
  logs,
  filter,
  onFilterChange,
}: WebhookLogFilterProps) {
  const { t } = useTranslation();

  const eventTypes = useMemo(() => getEventTypes(logs || []), [logs]);

  const filteredCount = useMemo(
    () => (logs ? filterLogs(logs, filter).length : 0),
    [logs, filter]
  );

  const totalCount = logs?.length || 0;

  const hasActiveFilters = filter.status !== 'all' || filter.dateFrom || filter.dateTo || filter.eventType || filter.urlSearch;

  const resetFilters = useCallback(() => {
    onFilterChange(DEFAULT_LOG_FILTER);
  }, [onFilterChange]);

  const updateFilter = useCallback(
    (partial: Partial<LogFilterState>) => {
      onFilterChange({ ...filter, ...partial });
    },
    [filter, onFilterChange]
  );

  return (
    <div className="space-y-3">
      {/* Filter Bar */}
      <div className="flex flex-wrap items-center gap-2">
        {/* Status Filter */}
        <div className="flex items-center gap-0.5 bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg p-0.5">
          {(['all', 'success', 'failed'] as const).map((status) => (
            <button
              key={status}
              type="button"
              onClick={() => updateFilter({ status })}
              className={`px-2.5 py-1 text-xs font-medium rounded-md transition-colors ${
                filter.status === status
                  ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                  : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'
              }`}
            >
              {status === 'all'
                ? (t('all') || 'All')
                : status === 'success'
                  ? (t('success') || 'Success')
                  : (t('failed') || 'Failed')}
            </button>
          ))}
        </div>

        {/* Event Type Select */}
        <select
          value={filter.eventType}
          onChange={(e) => updateFilter({ eventType: e.target.value })}
          className="text-xs px-2.5 py-1.5 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">
            {t('all_events') || 'All Events'}
          </option>
          {eventTypes.map((type) => (
            <option key={type} value={type}>
              {type}
            </option>
          ))}
        </select>

        {/* Date From */}
        <div className="flex items-center gap-1">
          <Calendar className="w-3 h-3 text-slate-400 shrink-0" />
          <input
            type="date"
            value={filter.dateFrom}
            onChange={(e) => updateFilter({ dateFrom: e.target.value })}
            className="text-xs px-2 py-1.5 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500 w-36"
            title={t('from_date') || 'From date'}
          />
        </div>

        {/* Date To */}
        <div className="flex items-center gap-1">
          <Calendar className="w-3 h-3 text-slate-400 shrink-0" />
          <input
            type="date"
            value={filter.dateTo}
            onChange={(e) => updateFilter({ dateTo: e.target.value })}
            className="text-xs px-2 py-1.5 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500 w-36"
            title={t('to_date') || 'To date'}
          />
        </div>

        {/* URL Search */}
        <div className="relative flex-1 min-w-[180px] max-w-xs">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3 h-3 text-slate-400" />
          <input
            type="text"
            value={filter.urlSearch}
            onChange={(e) => updateFilter({ urlSearch: e.target.value })}
            placeholder={t('search_url') || 'Search URL...'}
            className="w-full text-xs pl-7 pr-2.5 py-1.5 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {/* Reset */}
        {hasActiveFilters && (
          <button
            type="button"
            onClick={resetFilters}
            className="flex items-center gap-1 text-xs px-2.5 py-1.5 rounded-lg text-slate-500 hover:text-slate-700 hover:bg-slate-100 dark:hover:bg-slate-700 dark:hover:text-slate-300 transition-colors"
          >
            <X className="w-3 h-3" />
            {t('reset') || 'Reset'}
          </button>
        )}
      </div>

      {/* Results Count */}
      <div className="flex items-center gap-2">
        <Filter className="w-3 h-3 text-slate-400" />
        <p className="text-[10px] text-slate-500">
          {hasActiveFilters
            ? (t('filtered_results') || 'Filtered') + `: ${filteredCount} / ${totalCount}`
            : (t('total_logs') || 'Total') + `: ${totalCount} ${t('deliveries') || 'deliveries'}`}
        </p>
        {hasActiveFilters && (
          <Badge variant="info" size="sm">
            {t('filtered') || 'Filtered'}
          </Badge>
        )}
      </div>
    </div>
  );
}

// Re-export the filter function for use in parent components
export { filterLogs };
