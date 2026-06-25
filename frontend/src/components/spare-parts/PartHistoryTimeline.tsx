// ═══════════════════════════════════════════════════════════════════════
// PartHistoryTimeline — история перемещений и использований запчасти
// Привязка к WorkOrders через существующий Timeline компонент
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Timeline, type TimelineEvent } from '../ui/Timeline';
import { Skeleton } from '../ui/Skeleton';
import { request } from '../../services/api';
import { Package, ClipboardList, AlertCircle } from 'lucide-react';

// ── Types ──────────────────────────────────────────────────────────────

export interface PartHistoryEntry {
  id: string;
  part_id: string;
  part_name: string;
  part_sku: string;
  event_type: 'stock_in' | 'stock_out' | 'usage' | 'transfer' | 'adjustment' | 'order';
  quantity: number;
  stock_before: number;
  stock_after: number;
  work_order_id?: string;
  work_order_title?: string;
  user_id?: string;
  user_name?: string;
  notes?: string;
  created_at: string;
}

interface PartHistoryTimelineProps {
  partId: string;
  maxEvents?: number;
  className?: string;
}

// ── Helpers ────────────────────────────────────────────────────────────

const eventTypeConfig: Record<string, { type: TimelineEvent['type']; label: string }> = {
  stock_in: { type: 'part', label: 'Stock In' },
  stock_out: { type: 'part', label: 'Stock Out' },
  usage: { type: 'maintenance', label: 'Used in WO' },
  transfer: { type: 'note', label: 'Transfer' },
  adjustment: { type: 'system', label: 'Adjustment' },
  order: { type: 'assignment', label: 'Ordered' },
};

function formatQuantity(quantity: number): string {
  return quantity > 0 ? `+${quantity}` : `${quantity}`;
}

// ── Hook ───────────────────────────────────────────────────────────────

function usePartHistory(partId: string, maxEvents: number) {
  return useQuery<PartHistoryEntry[]>({
    queryKey: ['part-history', partId],
    queryFn: () => request<PartHistoryEntry[]>(`/spare-parts/${partId}/history?limit=${maxEvents}`),
    enabled: !!partId,
    staleTime: 30_000,
  });
}

// ── Component ──────────────────────────────────────────────────────────

export function PartHistoryTimeline({
  partId,
  maxEvents = 20,
  className = '',
}: PartHistoryTimelineProps) {
  const { data: history, isLoading, error } = usePartHistory(partId, maxEvents);

  const events: TimelineEvent[] = useMemo(() => {
    if (!history) return [];

    return history.map((entry) => {
      const config = eventTypeConfig[entry.event_type] || eventTypeConfig.adjustment;
      const qtyDisplay = formatQuantity(entry.quantity);

      let description = '';
      if (entry.event_type === 'usage' && entry.work_order_title) {
        description = `Used in: ${entry.work_order_title}`;
      } else if (entry.event_type === 'transfer') {
        description = entry.notes || 'Transferred to another location';
      } else if (entry.event_type === 'adjustment') {
        description = entry.notes || 'Manual adjustment';
      } else if (entry.event_type === 'order') {
        description = entry.notes || 'Order placed';
      } else {
        description = entry.notes || '';
      }

      const title = `${config.label}: ${qtyDisplay} (${entry.stock_before} → ${entry.stock_after})`;

      return {
        id: entry.id,
        timestamp: entry.created_at,
        type: config.type,
        title,
        description,
        user: entry.user_name || undefined,
      };
    });
  }, [history]);

  if (isLoading) {
    return (
      <div className={`space-y-3 ${className}`}>
        <Skeleton className="h-16 w-full rounded-lg" />
        <Skeleton className="h-16 w-full rounded-lg" />
        <Skeleton className="h-16 w-full rounded-lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div className={`flex items-center gap-2 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg ${className}`}>
        <AlertCircle size={16} className="text-red-500" />
        <p className="text-sm text-red-600 dark:text-red-400">
          Failed to load history: {(error as Error).message}
        </p>
      </div>
    );
  }

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
          История перемещений
        </h3>
        {history && history.length > 1 && (
          <span className="text-xs text-slate-400">
            {history.length} событий
          </span>
        )}
      </div>
      <Timeline events={events} />
    </div>
  );
}
