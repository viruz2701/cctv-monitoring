// ═══════════════════════════════════════════════════════════════════════
// WODataGrid — Snipe-IT inspired DataGrid для Work Orders
//
// P1-UX.1: WorkOrders Redesign (Snipe-IT Pattern)
//   - Bulk actions toolbar (assign, priority, delete, export)
//   - Inline status/priority/assignee change
//   - Quick filters (My, Overdue, Critical, Today, Unassigned)
//   - Column resize + reorder
//   - Export to CSV/Excel
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  DataGrid,
  Button,
  Badge,
  useToast,
} from '../ui';
import type { WorkOrder } from '../../services/workOrdersApi';
import {
  Play, CheckCircle, XCircle, UserCheck, Tags, Trash2,
  Download, AlertTriangle, Clock,
} from '../ui/Icons';
import { useUpdateWorkOrder } from '../../hooks/useApiQuery';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../hooks/useApiQuery';
import { workOrdersApi } from '../../services/workOrdersApi';
import { useConfirmAction } from '../../hooks/useConfirmAction';
import { useAuth } from '../../hooks/useAuth';

// ── Types ─────────────────────────────────────────────────────────────

export type BulkActionType = 'status_change' | 'assign' | 'delete' | 'priority_change' | 'export_csv';

export interface WODataGridProps {
  workOrders: WorkOrder[];
  loading: boolean;
  selectedIds: Set<string>;
  onSelectionChange: (ids: Set<string>) => void;
  onStatusChange: (id: string, newStatus: string) => Promise<void>;
  onBulkAction: (action: BulkActionType, value?: string) => Promise<void>;
  bulkLoading: boolean;
  bulkError: string | null;
  onDismissError: () => void;
}

// ── Inline Edit Select ────────────────────────────────────────────────

interface InlineEditSelectProps {
  value: string;
  options: { value: string; label: string }[];
  onSave: (value: string) => Promise<void>;
  renderDisplay: (value: string) => React.ReactNode;
}

const InlineEditSelect: React.FC<InlineEditSelectProps> = ({
  value, options, onSave, renderDisplay,
}) => {
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (!editing) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setEditing(false);
    };
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setEditing(false);
    };
    const timer = setTimeout(() => document.addEventListener('mousedown', handleClickOutside), 0);
    document.addEventListener('keydown', handleEscape);
    return () => {
      clearTimeout(timer);
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [editing]);

  const handleChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newValue = e.target.value;
    if (newValue === value) { setEditing(false); return; }
    setSaving(true);
    try { await onSave(newValue); setEditing(false); }
    catch { /* stay open */ }
    finally { setSaving(false); }
  };

  if (editing) {
    return (
      <div ref={ref} onClick={(e) => e.stopPropagation()}>
        <select value={value} onChange={handleChange}
          onBlur={() => setEditing(false)} disabled={saving}
          className="border rounded px-2 py-1 text-xs dark:bg-slate-800 dark:border-slate-600 min-w-[130px]" autoFocus>
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
      </div>
    );
  }

  return (
    <div ref={ref} onClick={(e) => e.stopPropagation()}>
      <button onClick={() => setEditing(true)}
        className="cursor-pointer hover:opacity-80 transition-opacity text-left w-full focus:outline-none focus:ring-2 focus:ring-blue-500 rounded">
        {renderDisplay(value)}
      </button>
    </div>
  );
};

// ── Helper functions ──────────────────────────────────────────────────

function getPriorityVariant(p: string): 'danger' | 'warning' | 'info' | 'success' {
  switch (p) {
    case 'critical': return 'danger';
    case 'high': return 'warning';
    case 'medium': return 'info';
    case 'low': return 'success';
    default: return 'info';
  }
}

function getStatusVariant(s: string): 'neutral' | 'primary' | 'warning' | 'success' | 'danger' {
  switch (s) {
    case 'open': return 'neutral';
    case 'in_progress': return 'primary';
    case 'completed': return 'success';
    case 'cancelled': return 'danger';
    default: return 'neutral';
  }
}

function getSLAIcon(slaStatus?: string) {
  switch (slaStatus) {
    case 'breached': return <AlertTriangle className="text-red-500" size={16} />;
    case 'at_risk': return <Clock className="text-orange-500" size={16} />;
    case 'on_track': return <Clock className="text-green-500" size={16} />;
    default: return null;
  }
}

// ── WODataGrid Component ──────────────────────────────────────────────

export function WODataGrid({
  workOrders, loading, selectedIds, onSelectionChange,
  onStatusChange, onBulkAction, bulkLoading, bulkError, onDismissError,
}: WODataGridProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user } = useAuth();
  const updateWorkOrderMut = useUpdateWorkOrder();
  const queryClient = useQueryClient();
  const toast = useToast();
  const { confirm, ConfirmDialog } = useConfirmAction();

  const bulkActions = useMemo(() => [
    {
      label: t('assign') || 'Assign',
      icon: <UserCheck size={14} />,
      variant: 'primary' as const,
      onClick: () => onBulkAction('assign'),
    },
    {
      label: t('change_priority') || 'Priority',
      icon: <Tags size={14} />,
      variant: 'secondary' as const,
      onClick: () => onBulkAction('priority_change'),
    },
    {
      label: t('export_csv') || 'Export CSV',
      icon: <Download size={14} />,
      variant: 'secondary' as const,
      onClick: () => onBulkAction('export_csv'),
    },
    {
      label: t('delete') || 'Cancel',
      icon: <Trash2 size={14} />,
      variant: 'danger' as const,
      onClick: () => onBulkAction('delete'),
    },
  ], [t, onBulkAction]);

  const columns = useMemo(() => [
    {
      key: 'device_name',
      header: t('device'),
      sortable: true,
      render: (item: WorkOrder) => item.device_name || item.device_id,
    },
    {
      key: 'type',
      header: t('type'),
      sortable: true,
      render: (item: WorkOrder) => t(item.type),
    },
    {
      key: 'priority',
      header: t('priority'),
      sortable: true,
      width: 140,
      render: (item: WorkOrder) => (
        <InlineEditSelect
          value={item.priority}
          options={[
            { value: 'critical', label: t('critical') },
            { value: 'high', label: t('high') },
            { value: 'medium', label: t('medium') },
            { value: 'low', label: t('low') },
          ]}
          onSave={async (val) => {
            await updateWorkOrderMut.mutateAsync({ id: item.id, data: { priority: val as WorkOrder['priority'] } });
          }}
          renderDisplay={(val) => <Badge variant={getPriorityVariant(val)} size="sm">{t(val)}</Badge>}
        />
      ),
    },
    {
      key: 'status',
      header: t('status'),
      sortable: true,
      width: 160,
      render: (item: WorkOrder) => (
        <InlineEditSelect
          value={item.status}
          options={[
            { value: 'open', label: t('open') },
            { value: 'in_progress', label: t('in_progress') },
            { value: 'completed', label: t('completed') },
            { value: 'cancelled', label: t('cancelled') },
          ]}
          onSave={async (val) => {
            await onStatusChange(item.id, val);
          }}
          renderDisplay={(val) => <Badge variant={getStatusVariant(val)} size="sm">{t(val)}</Badge>}
        />
      ),
    },
    {
      key: 'sla',
      header: 'SLA',
      sortable: true,
      render: (item: WorkOrder) => (
        <div className="flex items-center gap-1">
          {getSLAIcon(item.sla_status)}
          {item.sla_deadline && (
            <span className="text-xs">{new Date(item.sla_deadline).toLocaleString()}</span>
          )}
        </div>
      ),
    },
    {
      key: 'assignee_name',
      header: t('assigned_to'),
      sortable: true,
      render: (item: WorkOrder) => (
        <span className="text-sm">{item.assignee_name || t('unassigned')}</span>
      ),
    },
    {
      key: 'actions',
      header: t('actions'),
      width: 200,
      render: (item: WorkOrder) => (
        <div className="flex gap-1">
          {item.status === 'open' && (
            <Button size="sm" onClick={async (e) => {
              e.stopPropagation();
              await workOrdersApi.startWorkOrder(item.id);
              queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
            }} icon={<Play size={14} />}>{t('start')}</Button>
          )}
          {item.status === 'in_progress' && (
            <Button size="sm" onClick={async (e) => {
              e.stopPropagation();
              await workOrdersApi.completeWorkOrder(item.id, '', [], []);
              queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
            }} icon={<CheckCircle size={14} />}>{t('complete')}</Button>
          )}
          {(item.status === 'open' || item.status === 'in_progress') && (
            <Button size="sm" variant="danger" onClick={async (e) => {
              e.stopPropagation();
              const ok = await confirm({
                title: t('cancel_work_order') || 'Cancel Work Order',
                message: t('cancel_work_order_confirm') || 'Are you sure?',
                confirmText: t('cancel') || 'Cancel',
                variant: 'warning',
              });
              if (ok) {
                await workOrdersApi.cancelWorkOrder(item.id, 'Cancelled by user');
                queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
              }
            }} icon={<XCircle size={14} />}>{t('cancel')}</Button>
          )}
        </div>
      ),
    },
  ], [t, updateWorkOrderMut, onStatusChange, queryClient, confirm]);

  return (
    <>
      {/* Error banner */}
      {bulkError && (
        <div className="px-4 py-2 mb-2 text-sm text-red-700 bg-red-50 dark:bg-red-900/20 dark:text-red-300 border border-red-200 dark:border-red-800 rounded">
          {bulkError}
          <button className="ml-2 font-medium hover:underline" onClick={onDismissError}>
            {t('dismiss')}
          </button>
        </div>
      )}

      <DataGrid
        data={workOrders}
        columns={columns}
        keyExtractor={(item: WorkOrder) => item.id}
        loading={loading}
        emptyMessage={t('no_work_orders')}
        exportFilename="work-orders.csv"
        selectable
        selectedIds={selectedIds}
        onSelectionChange={onSelectionChange}
        onRowClick={(item: WorkOrder) => navigate(`/work-orders/${item.id}`)}
        stickyHeader
        persistId="work-orders"
        bulkActions={bulkActions}
      />

      {ConfirmDialog}
    </>
  );
}

export default WODataGrid;
