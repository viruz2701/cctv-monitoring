import React, { useState, useCallback, useMemo } from 'react';
import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd';
import { Clock, AlertTriangle, User } from '../ui/Icons';
import { Badge } from '../ui/Badge';
import { ProgressBar } from '../ui/ProgressBar';
import type { WorkOrder } from '../../services/workOrdersApi';

// ═══════════════════════════════════════════════════════════════════════
// WOKanbanBoard — Drag-and-Drop Kanban Board for Work Orders
// Columns: New → In Progress → Completed → Cancelled
// Uses @hello-pangea/dnd for drag-and-drop
// ═══════════════════════════════════════════════════════════════════════

type KanbanColumn = 'open' | 'in_progress' | 'completed' | 'cancelled';

interface ColumnDef {
  key: KanbanColumn;
  title: string;
  color: string;
}

const COLUMNS: ColumnDef[] = [
  { key: 'open', title: 'New', color: 'border-t-sky-500' },
  { key: 'in_progress', title: 'In Progress', color: 'border-t-amber-500' },
  { key: 'completed', title: 'Completed', color: 'border-t-emerald-500' },
  { key: 'cancelled', title: 'Cancelled', color: 'border-t-slate-500' },
];

const PRIORITY_CONFIG: Record<string, { variant: 'danger' | 'warning' | 'info' | 'success'; label: string }> = {
  critical: { variant: 'danger', label: 'Critical' },
  high: { variant: 'warning', label: 'High' },
  medium: { variant: 'info', label: 'Medium' },
  low: { variant: 'success', label: 'Low' },
};

interface WOKanbanBoardProps {
  workOrders: WorkOrder[];
  onStatusChange: (id: string, newStatus: KanbanColumn) => Promise<void>;
  onCardClick?: (workOrder: WorkOrder) => void;
  /** Current user ID to show "My Orders" highlight */
  currentUserId?: string;
  className?: string;
}

export function WOKanbanBoard({
  workOrders,
  onStatusChange,
  onCardClick,
  currentUserId,
  className = '',
}: WOKanbanBoardProps) {
  const [draggingId, setDraggingId] = useState<string | null>(null);

  // ── Group work orders by status ────────────────────────────────────
  const columns = useMemo(() => {
    const grouped: Record<KanbanColumn, WorkOrder[]> = {
      open: [],
      in_progress: [],
      completed: [],
      cancelled: [],
    };
    for (const wo of workOrders) {
      const status = wo.status as KanbanColumn;
      if (grouped[status]) {
        grouped[status].push(wo);
      } else {
        grouped.open.push(wo); // fallback для неизвестных статусов
      }
    }
    return grouped;
  }, [workOrders]);

  // ── Drag end handler ───────────────────────────────────────────────
  const onDragEnd = useCallback(
    async (result: DropResult) => {
      setDraggingId(null);
      const { draggableId, destination, source } = result;
      if (!destination) return;
      if (destination.droppableId === source.droppableId) return;

      const newStatus = destination.droppableId as KanbanColumn;
      if (newStatus === source.droppableId as KanbanColumn) return;

      try {
        await onStatusChange(draggableId, newStatus);
      } catch {
        // Ошибка обрабатывается родителем
      }
    },
    [onStatusChange],
  );

  // ── Render assignee avatar (initials) ──────────────────────────────
  const renderAssignee = (wo: WorkOrder) => {
    if (!wo.assignee_name) return null;
    const initials = wo.assignee_name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
    const isCurrentUser = currentUserId && wo.assigned_to === currentUserId;
    return (
      <div
        className={`w-7 h-7 rounded-full flex items-center justify-center text-[10px] font-bold ${
          isCurrentUser
            ? 'bg-blue-500 text-white ring-2 ring-blue-300'
            : 'bg-slate-200 text-slate-600 dark:bg-slate-600 dark:text-slate-300'
        }`}
        title={wo.assignee_name}
        aria-label={`Assigned to ${wo.assignee_name}`}
      >
        {initials}
      </div>
    );
  };

  // ── Compute SLA percentage ─────────────────────────────────────────
  const getSLAProgress = (wo: WorkOrder): { value: number; variant: 'success' | 'warning' | 'danger'; max: number } => {
    if (!wo.sla_deadline) return { value: 0, variant: 'info' as any, max: 100 };
    const now = Date.now();
    const deadline = new Date(wo.sla_deadline).getTime();
    const created = wo.created_at ? new Date(wo.created_at).getTime() : deadline - 86400000;
    const total = deadline - created;
    const elapsed = now - created;
    const pct = total > 0 ? (elapsed / total) * 100 : 100;
    const clamped = Math.min(100, Math.max(0, pct));

    let variant: 'success' | 'warning' | 'danger';
    if (clamped >= 100) variant = 'danger';
    else if (clamped >= 75) variant = 'warning';
    else variant = 'success';

    return { value: clamped, variant, max: 100 };
  };

  return (
    <DragDropContext onDragEnd={onDragEnd} onDragStart={(result) => setDraggingId(result.draggableId)}>
      <div
        className={`grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 ${className}`}
        role="list"
        aria-label="Kanban board — work orders grouped by status"
        aria-live="polite"
      >
        {COLUMNS.map((col) => {
          const items = columns[col.key];
          return (
            <div
              key={col.key}
              role="listitem"
              aria-label={`${col.title} column, ${items.length} items`}
              className={`bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 border-t-2 ${col.color} flex flex-col min-h-[400px]`}
            >
              {/* Column Header */}
              <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700">
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">
                  {col.title}
                </h3>
                <span className="inline-flex items-center justify-center min-w-[22px] h-[22px] px-1.5 rounded-full bg-slate-200 dark:bg-slate-700 text-xs font-medium text-slate-600 dark:text-slate-400">
                  {items.length}
                </span>
              </div>

              {/* Droppable Area */}
              <Droppable droppableId={col.key}>
                {(provided, snapshot) => (
                  <div
                    ref={provided.innerRef}
                    {...provided.droppableProps}
                    className={`flex-1 p-3 space-y-3 overflow-y-auto min-h-[200px] transition-all duration-200 ${
                      snapshot.isDraggingOver
                        ? 'bg-blue-50/50 dark:bg-blue-900/10 scale-[1.01]'
                        : ''
                    }`}
                  >
                    {items.length === 0 && !snapshot.isDraggingOver && (
                      <div className="flex items-center justify-center h-24 text-xs text-slate-400 dark:text-slate-500 italic">
                        No items
                      </div>
                    )}
                    {items.map((wo, idx) => {
                      const priority = PRIORITY_CONFIG[wo.priority] || PRIORITY_CONFIG.medium;
                      const sla = getSLAProgress(wo);
                      const isDragging = draggingId === wo.id;

                      return (
                        <Draggable key={wo.id} draggableId={wo.id} index={idx}>
                          {(provided, snapshot) => (
                            <div
                              ref={provided.innerRef}
                              {...provided.draggableProps}
                              {...provided.dragHandleProps}
                              onClick={() => onCardClick?.(wo)}
                              role="listitem"
                              aria-roledescription="draggable work order card"
                              className={`bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-3 space-y-2 cursor-pointer transition-all duration-200 ease-out transform-gpu will-change-transform ${
                                snapshot.isDragging
                                  ? 'shadow-xl ring-2 ring-blue-400 rotate-[2deg] scale-105'
                                  : 'hover:shadow-md hover:border-slate-300 dark:hover:border-slate-600 active:scale-[0.98]'
                              } ${isDragging ? 'opacity-90' : ''}`}
                              style={{
                                ...provided.draggableProps.style,
                              }}
                            >
                              {/* Priority + Assignee Row */}
                              <div className="flex items-center justify-between gap-2">
                                <Badge variant={priority.variant} size="sm">
                                  {priority.label}
                                </Badge>
                                {renderAssignee(wo)}
                              </div>

                              {/* Title */}
                              <p className="text-sm font-medium text-slate-800 dark:text-slate-200 line-clamp-2">
                                {wo.device_name || wo.device_id || 'Untitled'}
                              </p>

                              {/* Device info */}
                              {wo.device_name && (
                                <p className="text-xs text-slate-500 dark:text-slate-400">
                                  {wo.type && (
                                    <span className="capitalize">{wo.type.replace('_', ' ')}</span>
                                  )}
                                </p>
                              )}

                              {/* SLA Progress */}
                              {wo.sla_deadline && wo.status !== 'completed' && wo.status !== 'cancelled' && (
                                <div className="pt-1">
                                  <div className="flex items-center gap-1 mb-1">
                                    {sla.variant === 'danger' ? (
                                      <AlertTriangle size={10} className="text-red-500" />
                                    ) : sla.variant === 'warning' ? (
                                      <Clock size={10} className="text-amber-500" />
                                    ) : (
                                      <Clock size={10} className="text-emerald-500" />
                                    )}
                                    <span className="text-[10px] text-slate-400 dark:text-slate-500">
                                      {new Date(wo.sla_deadline).toLocaleDateString()}
                                    </span>
                                  </div>
                                  <ProgressBar
                                    value={sla.value}
                                    max={sla.max}
                                    variant={sla.variant}
                                    size="sm"
                                  />
                                </div>
                              )}
                            </div>
                          )}
                        </Draggable>
                      );
                    })}
                    {provided.placeholder}
                  </div>
                )}
              </Droppable>
            </div>
          );
        })}
      </div>
    </DragDropContext>
  );
}
