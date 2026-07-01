// ═══════════════════════════════════════════════════════════════════════
// UnifiedWorkHub — Объединённая страница Work Orders / Tickets / Requests
//
// UX-1.2: Unified Work Hub
//   - Табы: [My Tasks] [Team] [Requests] с badge-счётчиками
//   - Quick Filters: Overdue, Critical, Unassigned
//   - URL searchParams: ?tab=tasks&filter=critical
//   - Bulk actions toolbar работает во всех WO-табах
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — через RoleProtectedRoute на уровне роутинга)
//   - OWASP ASVS V2.1.1 (Input validation — через WODataGrid / Zod)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import {
  useWorkOrders,
  useTickets,
  useUpdateWorkOrder,
  useUsers,
  queryKeys,
} from '../hooks/useApiQuery';
import { workOrdersApi } from '../services/workOrdersApi';
import { useAuth } from '../hooks/useAuth';
import { useConfirmAction } from '../hooks/useConfirmAction';
import { getArrayData } from '../utils/helpers';
import { Card, Button, useToast, SkeletonTable } from '../components/ui';
import { Plus, Loader2 } from '../components/ui/Icons';
import { WODataGrid, type BulkActionType } from '../components/work-orders/WODataGrid';
import { WorkHubTabs, useActiveTab, type HubTab } from '../components/work-hub/WorkHubTabs';
import { QuickFilters, useHubQuickFilter, type HubQuickFilterKey } from '../components/work-hub/QuickFilters';
import type { WorkOrder } from '../services/workOrdersApi';
import type { Ticket } from '../types';

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function filterWorkOrders(
  workOrders: WorkOrder[],
  tab: HubTab,
  quickFilter: HubQuickFilterKey,
  currentUserId?: string,
): WorkOrder[] {
  return workOrders.filter((wo) => {
    // Tab filter
    if (tab === 'tasks') {
      if (wo.assigned_to !== currentUserId) return false;
    }
    // 'team' tab shows all WOs — no filtering

    // Quick filter
    if (quickFilter === 'overdue') {
      const notFinished = wo.status !== 'completed' && wo.status !== 'cancelled';
      const isOverdue = wo.sla_deadline && new Date(wo.sla_deadline) < new Date();
      if (!notFinished || !isOverdue) return false;
    }
    if (quickFilter === 'unassigned' && wo.assigned_to) return false;
    if (quickFilter === 'critical' && wo.priority !== 'critical') return false;

    return true;
  });
}

// ═══════════════════════════════════════════════════════════════════════
// UnifiedWorkHub Page
// ═══════════════════════════════════════════════════════════════════════

export function UnifiedWorkHub() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { confirm, ConfirmDialog } = useConfirmAction();
  const toast = useToast();

  // ── Data ─────────────────────────────────────────────────────────
  const { data: workOrders = [], isLoading: woLoading } = useWorkOrders();
  const { data: apiTickets = [] } = useTickets();
  const tickets = useMemo(() => getArrayData<Ticket>(apiTickets as any), [apiTickets]);
  const updateWorkOrderMut = useUpdateWorkOrder();

  // ── Tab state (URL-synced) ──────────────────────────────────────
  const [activeTab, setActiveTab] = useActiveTab();
  const [quickFilter, setQuickFilter] = useHubQuickFilter();

  // ── Selection state ─────────────────────────────────────────────
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkLoading, setBulkLoading] = useState(false);
  const [bulkError, setBulkError] = useState<string | null>(null);

  // ── Filtered work orders ────────────────────────────────────────
  const filtered = useMemo(
    () => filterWorkOrders(workOrders, activeTab, quickFilter, user?.id),
    [workOrders, activeTab, quickFilter, user?.id],
  );

  // ── Tab change handler ──────────────────────────────────────────
  const handleTabChange = useCallback(
    (tab: HubTab) => {
      setActiveTab(tab);
      setSelectedIds(new Set());
      setBulkError(null);
    },
    [setActiveTab],
  );

  // ── Quick filter change handler ─────────────────────────────────
  const handleQuickFilterChange = useCallback(
    (filter: HubQuickFilterKey) => {
      setQuickFilter(filter);
      setSelectedIds(new Set());
    },
    [setQuickFilter],
  );

  // ── Bulk action handler ─────────────────────────────────────────
  const handleBulkAction = useCallback(
    async (action: BulkActionType, value?: string) => {
      if (selectedIds.size === 0) return;

      if (action === 'delete') {
        const ok = await confirm({
          title: t('bulk_delete_title') || 'Delete Work Orders',
          message:
            t('bulk_delete_confirm', { count: selectedIds.size }) ||
            `Are you sure you want to delete ${selectedIds.size} work order(s)?`,
          confirmText: t('delete') || 'Delete',
          variant: 'danger',
        });
        if (!ok) return;
      }

      setBulkLoading(true);
      setBulkError(null);
      try {
        const ids = Array.from(selectedIds);
        const response = await workOrdersApi.bulkActions(action, ids, value);
        if (response.failed > 0) {
          setBulkError(`${response.failed} operation(s) failed`);
        }
        setSelectedIds(new Set());
        queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
      } catch (err) {
        setBulkError(err instanceof Error ? err.message : 'Bulk action failed');
      } finally {
        setBulkLoading(false);
      }
    },
    [selectedIds, confirm, t, queryClient],
  );

  // ── Status change handler (for WODataGrid inline edit) ──────────
  const handleStatusChange = useCallback(
    async (id: string, newStatus: string) => {
      const wo = workOrders.find((w) => w.id === id);
      const oldStatus = wo?.status;
      const woName = wo?.device_name || `WO #${id.slice(0, 8)}`;

      const STATUS_LABELS: Record<string, string> = {
        open: 'Open',
        in_progress: 'In Progress',
        completed: 'Completed',
        cancelled: 'Cancelled',
      };

      try {
        await updateWorkOrderMut.mutateAsync({
          id,
          data: { status: newStatus as WorkOrder['status'] },
        });

        toast.success({
          title: `${woName} moved to ${STATUS_LABELS[newStatus] || newStatus}`,
          duration: 5000,
          undo: {
            label: 'Undo',
            onClick: () => {
              updateWorkOrderMut
                .mutateAsync({ id, data: { status: oldStatus as WorkOrder['status'] } })
                .then(() => toast.success('Status reverted'))
                .catch(() => toast.error('Failed to revert status'));
            },
          },
        });
      } catch {
        toast.error(`Failed to move ${woName}`);
      }
    },
    [updateWorkOrderMut, workOrders, toast],
  );

  // ── Loading state ───────────────────────────────────────────────
  const isLoading = woLoading && workOrders.length === 0;

  // ── Render ──────────────────────────────────────────────────────
  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('unified_work_hub') || 'Work Hub'}</h1>
        <div className="flex gap-3 items-center">
          <Button
            onClick={() => navigate('/work-orders/new')}
            icon={<Plus size={20} />}
          >
            {t('create_work_order')}
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <WorkHubTabs
        workOrders={workOrders}
        currentUserId={user?.id}
        ticketsCount={tickets.length}
        activeTab={activeTab}
        onTabChange={handleTabChange}
      />

      {/* Quick Filters (only for WO tabs) */}
      {activeTab !== 'requests' && (
        <QuickFilters
          workOrders={filtered}
          activeFilter={quickFilter}
          onChange={handleQuickFilterChange}
          className="mb-4"
        />
      )}

      {/* Tab Content */}
      <Card>
        {activeTab === 'requests' ? (
          <RequestsTabContent tickets={tickets} />
        ) : (
          <WorkOrdersTabContent
            workOrders={filtered}
            loading={isLoading}
            selectedIds={selectedIds}
            onSelectionChange={setSelectedIds}
            onStatusChange={handleStatusChange}
            onBulkAction={handleBulkAction}
            bulkLoading={bulkLoading}
            bulkError={bulkError}
            onDismissError={() => setBulkError(null)}
          />
        )}
      </Card>

      {ConfirmDialog}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Work Orders Tab Content
// ═══════════════════════════════════════════════════════════════════════

interface WorkOrdersTabContentProps {
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

function WorkOrdersTabContent(props: WorkOrdersTabContentProps) {
  if (props.loading) {
    return <SkeletonTable rows={8} columns={6} />;
  }

  return (
    <WODataGrid
      workOrders={props.workOrders}
      loading={props.loading}
      selectedIds={props.selectedIds}
      onSelectionChange={props.onSelectionChange}
      onStatusChange={props.onStatusChange}
      onBulkAction={props.onBulkAction}
      bulkLoading={props.bulkLoading}
      bulkError={props.bulkError}
      onDismissError={props.onDismissError}
    />
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Requests Tab Content
// ═══════════════════════════════════════════════════════════════════════

interface RequestsTabContentProps {
  tickets: Ticket[];
}

function RequestsTabContent({ tickets }: RequestsTabContentProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isLoading = tickets.length === 0;

  if (isLoading) {
    return <SkeletonTable rows={8} columns={5} />;
  }

  return (
    <div className="space-y-3">
      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
        <SummaryCard
          label={t('total_tickets') || 'Total'}
          value={tickets.length}
          color="text-slate-900 dark:text-white"
        />
        <SummaryCard
          label={t('open') || 'Open'}
          value={tickets.filter((t) => t.status === 'open').length}
          color="text-red-600 dark:text-red-500"
        />
        <SummaryCard
          label={t('in_progress') || 'In Progress'}
          value={tickets.filter((t) => t.status === 'in_progress').length}
          color="text-blue-600 dark:text-blue-500"
        />
        <SummaryCard
          label={t('resolved') || 'Resolved'}
          value={tickets.filter((t) => t.status === 'resolved' || t.status === 'closed').length}
          color="text-emerald-600 dark:text-emerald-500"
        />
      </div>

      {/* Tickets list */}
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 dark:border-slate-700">
            <th className="text-left py-2 px-3 font-medium text-slate-500 dark:text-slate-400">{t('title') || 'Title'}</th>
            <th className="text-left py-2 px-3 font-medium text-slate-500 dark:text-slate-400">{t('priority') || 'Priority'}</th>
            <th className="text-left py-2 px-3 font-medium text-slate-500 dark:text-slate-400">{t('status') || 'Status'}</th>
            <th className="text-left py-2 px-3 font-medium text-slate-500 dark:text-slate-400">{t('assignee') || 'Assignee'}</th>
            <th className="text-left py-2 px-3 font-medium text-slate-500 dark:text-slate-400">{t('created') || 'Created'}</th>
          </tr>
        </thead>
        <tbody>
          {tickets.length === 0 ? (
            <tr>
              <td colSpan={5} className="text-center py-8 text-slate-400">
                {t('no_tickets') || 'No tickets found'}
              </td>
            </tr>
          ) : (
            tickets.map((ticket) => (
              <tr
                key={ticket.id}
                onClick={() => navigate(`/tickets/${ticket.id}`)}
                className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
              >
                <td className="py-2.5 px-3">
                  <p className="font-medium text-slate-900 dark:text-white">{ticket.title}</p>
                  <p className="text-xs text-slate-400">{ticket.id}</p>
                </td>
                <td className="py-2.5 px-3">
                  <PriorityBadge priority={ticket.priority} />
                </td>
                <td className="py-2.5 px-3">
                  <StatusBadge status={ticket.status} />
                </td>
                <td className="py-2.5 px-3 text-slate-600 dark:text-slate-400">
                  {ticket.assignee || '—'}
                </td>
                <td className="py-2.5 px-3 text-slate-500 dark:text-slate-400 text-xs">
                  {new Date(ticket.createdAt).toLocaleDateString()}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

function SummaryCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 text-center">
      <p className={`text-2xl font-bold ${color}`}>{value}</p>
      <p className="text-sm text-slate-500 dark:text-slate-400">{label}</p>
    </div>
  );
}

function PriorityBadge({ priority }: { priority: string }) {
  const colors: Record<string, string> = {
    critical: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
    high: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
    medium: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    low: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  };

  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${colors[priority] || colors.medium}`}>
      {priority}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    open: 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300',
    in_progress: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300',
    resolved: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300',
    closed: 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-300',
    pending: 'bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30 dark:text-yellow-300',
  };

  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${colors[status] || colors.open}`}>
      {status.replace('_', ' ')}
    </span>
  );
}

export default UnifiedWorkHub;
