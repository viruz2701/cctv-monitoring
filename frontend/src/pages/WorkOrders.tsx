import React, { useState, useCallback, useRef, useEffect, useMemo } from 'react';
import { useFormValidation } from '../hooks/useFormValidation';
import { workOrderSchema } from '../lib/validations';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import {
    prefetchWorkOrder,
    useWorkOrders,
    useCreateWorkOrder,
    useUpdateWorkOrder,
    useDeleteWorkOrder,
    useSites,
    useDevices,
    useUsers,
    queryKeys,
} from '../hooks/useApiQuery';
import { workOrdersApi, WorkOrder } from '../services/workOrdersApi';
import { useAuth } from '../hooks/useAuth';
import type { User as ApiUser } from '../services/api';
import { Button, Card, Badge, Modal, Input, useToast, SkeletonTable, DataGrid } from '../components/ui';
import { SavedViews } from '../components/ui/SavedViews';
import { useConfirmAction } from '../hooks/useConfirmAction';
import { QuickFilters, useQuickFilter, type QuickFilterKey } from '../components/work-orders/QuickFilters';
import { WOKanbanBoard } from '../components/work-orders/WOKanbanBoard';
import { WorkOrderCalendar } from '../components/work-orders/WorkOrderCalendar';
import {
  Plus, Play, CheckCircle, XCircle, Clock, AlertTriangle,
  CheckSquare, Square, UserCheck, Trash2, Tags, ArrowUpDown,
  Loader2, ChevronDown, User, AlertOctagon, LayoutGrid, List, Calendar,
} from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// Inline Edit Select Component (WO-4.2.3)
// ═══════════════════════════════════════════════════════════════════════

interface InlineEditSelectProps {
  value: string;
  options: { value: string; label: string }[];
  onSave: (value: string) => Promise<void>;
  renderDisplay: (value: string) => React.ReactNode;
}

const InlineEditSelect: React.FC<InlineEditSelectProps> = ({
  value,
  options,
  onSave,
  renderDisplay,
}) => {
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!editing) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setEditing(false);
      }
    };
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setEditing(false);
    };
    const timer = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside);
    }, 0);
    document.addEventListener('keydown', handleEscape);
    return () => {
      clearTimeout(timer);
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [editing]);

  const handleChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newValue = e.target.value;
    if (newValue === value) {
      setEditing(false);
      return;
    }
    setSaving(true);
    try {
      await onSave(newValue);
      setEditing(false);
    } catch {
      // Stay open on error so user can retry
    } finally {
      setSaving(false);
    }
  };

  if (editing) {
    return (
      <div ref={containerRef} onClick={(e) => e.stopPropagation()}>
        <select
          value={value}
          onChange={handleChange}
          onBlur={() => setEditing(false)}
          disabled={saving}
          className="border rounded px-2 py-1 text-xs dark:bg-slate-800 dark:border-slate-600 min-w-[130px]"
          autoFocus
        >
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>
      </div>
    );
  }

  return (
    <div ref={containerRef} onClick={(e) => e.stopPropagation()}>
      <button
        onClick={() => setEditing(true)}
        className="cursor-pointer hover:opacity-80 transition-opacity text-left w-full focus:outline-none focus:ring-2 focus:ring-blue-500 rounded"
      >
        {renderDisplay(value)}
      </button>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// Work Orders Page
// ═══════════════════════════════════════════════════════════════════════

type ViewMode = 'table' | 'kanban' | 'calendar';

type BulkActionType = 'status_change' | 'assign' | 'delete' | 'priority_change';

export const WorkOrders: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { confirm, ConfirmDialog } = useConfirmAction();
  const { data: workOrders = [], isLoading: loading } = useWorkOrders();
  const { data: rawUsers = [] } = useUsers();
  const updateWorkOrderMut = useUpdateWorkOrder();

  const users = useMemo(() => rawUsers.map(u => ({ ...u, name: (u as any).name || u.username })), [rawUsers]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [filterStatus, setFilterStatus] = useState('');
  const [filterPriority, setFilterPriority] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkLoading, setBulkLoading] = useState(false);
  const [bulkError, setBulkError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('table');

  // Quick filter synced with URL
  const [quickFilter, setQuickFilter] = useQuickFilter();

  // UX-14.3.2: Apply saved view
  const handleApplyView = useCallback((view: import('../store/filterStore').SavedView) => {
    const filters = view.filters;
    if (filters.filterStatus !== undefined) setFilterStatus(filters.filterStatus);
    if (filters.filterPriority !== undefined) setFilterPriority(filters.filterPriority);
    if (filters.quickFilter) setQuickFilter(filters.quickFilter as QuickFilterKey);
  }, [setQuickFilter]);

  const technicians = users.filter(u => u.role === 'technician');

  const filtered = workOrders.filter((wo) => {
    // Quick filters (WO-4.2.2)
    if (quickFilter === 'mine' && wo.assigned_to !== user?.id) return false;
    if (quickFilter === 'overdue') {
      const notFinished = wo.status !== 'completed' && wo.status !== 'cancelled';
      const isOverdue = wo.sla_deadline && new Date(wo.sla_deadline) < new Date();
      if (!notFinished || !isOverdue) return false;
    }
    if (quickFilter === 'unassigned' && wo.assigned_to) return false;
    if (quickFilter === 'critical' && wo.priority !== 'critical') return false;

    // Dropdown filters
    if (filterStatus && wo.status !== filterStatus) return false;
    if (filterPriority && wo.priority !== filterPriority) return false;
    return true;
  });

  const getPriorityVariant = (p: string): 'danger' | 'warning' | 'info' | 'success' => {
    switch (p) {
      case 'critical': return 'danger';
      case 'high': return 'warning';
      case 'medium': return 'info';
      case 'low': return 'success';
      default: return 'info';
    }
  };

  const getStatusVariant = (s: string): 'neutral' | 'primary' | 'warning' | 'success' | 'danger' => {
    switch (s) {
      case 'open': return 'neutral';
      case 'in_progress': return 'primary';
      case 'completed': return 'success';
      case 'cancelled': return 'danger';
      default: return 'neutral';
    }
  };

  const getSLAIcon = (slaStatus?: string) => {
    switch (slaStatus) {
      case 'breached': return <AlertTriangle className="text-red-500" size={16} />;
      case 'at_risk': return <Clock className="text-orange-500" size={16} />;
      case 'on_track': return <Clock className="text-green-500" size={16} />;
      default: return null;
    }
  };

  // ── Bulk Action Handler ──────────────────────────────────────────

  const handleBulkAction = useCallback(async (action: BulkActionType, value?: string) => {
    if (selectedIds.size === 0) return;

    if (action === 'delete') {
      const ok = await confirm({
        title: t('bulk_delete_title') || 'Delete Work Orders',
        message: t('bulk_delete_confirm', { count: selectedIds.size }) || `Are you sure you want to delete ${selectedIds.size} work order(s)?`,
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
  }, [selectedIds, confirm, t, queryClient]);

  // ── Bulk Actions for DataGrid ────────────────────────────────────
  const bulkActions = [
    {
      label: t('assign') || 'Assign',
      icon: <UserCheck size={14} />,
      variant: 'primary' as const,
      onClick: (items: WorkOrder[]) => {
        handleBulkAction('assign', '');
      },
    },
    {
      label: t('change_priority') || 'Priority',
      icon: <Tags size={14} />,
      variant: 'secondary' as const,
      onClick: (items: WorkOrder[]) => {
        handleBulkAction('status_change', '');
      },
    },
    {
      label: t('delete') || 'Cancel',
      icon: <Trash2 size={14} />,
      variant: 'danger' as const,
      onClick: (items: WorkOrder[]) => {
        handleBulkAction('delete');
      },
    },
  ];

  // ── Kanban status change handler ─────────────────────────────────
  const handleKanbanStatusChange = useCallback(async (id: string, newStatus: string) => {
    await updateWorkOrderMut.mutateAsync({ id, data: { status: newStatus as WorkOrder['status'] } });
  }, [updateWorkOrderMut]);

  // ── Calendar date change handler (drag-and-drop) ─────────────────
  const handleCalendarDateChange = useCallback(async (id: string, newDate: string) => {
    await updateWorkOrderMut.mutateAsync({ id, data: { sla_deadline: newDate } });
  }, [updateWorkOrderMut]);

  // ── Calendar date click handler (create WO on date) ──────────────
  const handleCalendarDateClick = useCallback((date: Date) => {
    setShowCreateModal(true);
  }, []);

  // ── Table columns ────────────────────────────────────────────────
  const columns = [
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
          renderDisplay={(val) => (
            <Badge variant={getPriorityVariant(val)} size="sm">{t(val)}</Badge>
          )}
        />
      ),
    },
    {
      key: 'status',
      header: t('status'),
      sortable: true,
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
            await updateWorkOrderMut.mutateAsync({ id: item.id, data: { status: val as WorkOrder['status'] } });
          }}
          renderDisplay={(val) => (
            <Badge variant={getStatusVariant(val)} size="sm">{t(val)}</Badge>
          )}
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
        <InlineEditSelect
          value={item.assigned_to || ''}
          options={[
            { value: '', label: t('unassigned') },
            ...technicians.map((tech: ApiUser) => ({
              value: tech.id,
              label: tech.name || tech.username,
            })),
          ]}
          onSave={async (val) => {
            await updateWorkOrderMut.mutateAsync({ id: item.id, data: { assigned_to: val || undefined } });
          }}
          renderDisplay={() => (
            <span className="text-sm">{item.assignee_name || t('unassigned')}</span>
          )}
        />
      ),
    },
    {
      key: 'actions',
      header: t('actions'),
      render: (item: WorkOrder) => (
        <div className="flex gap-1">
          {item.status === 'open' && (
            <Button size="sm" onClick={async (e) => { e.stopPropagation(); await workOrdersApi.startWorkOrder(item.id); queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all }); }} icon={<Play size={14} />}>
              {t('start')}
            </Button>
          )}
          {item.status === 'in_progress' && (
            <Button size="sm" onClick={async (e) => { e.stopPropagation(); await workOrdersApi.completeWorkOrder(item.id, '', [], []); queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all }); }} icon={<CheckCircle size={14} />}>
              {t('complete')}
            </Button>
          )}
          {(item.status === 'open' || item.status === 'in_progress') && (
            <Button size="sm" variant="danger" onClick={async (e) => {
              e.stopPropagation();
              const ok = await confirm({
                title: t('cancel_work_order') || 'Cancel Work Order',
                message: t('cancel_work_order_confirm') || 'Are you sure you want to cancel this work order?',
                confirmText: t('cancel') || 'Cancel',
                variant: 'warning',
              });
              if (ok) { await workOrdersApi.cancelWorkOrder(item.id, 'Cancelled by user'); queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all }); }
            }} icon={<XCircle size={14} />}>
              {t('cancel')}
            </Button>
          )}
        </div>
      ),
    },
  ];

  if (loading && workOrders.length === 0) {
    return (
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <div className="h-8 w-48 bg-gray-200 dark:bg-slate-700 rounded animate-pulse" />
          <div className="h-10 w-40 bg-gray-200 dark:bg-slate-700 rounded animate-pulse" />
        </div>
        <Card>
          <SkeletonTable rows={8} columns={6} />
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('work_orders')}</h1>
        <div className="flex gap-3 items-center">
          {/* View Toggle: Table ↔ Kanban ↔ Calendar */}
          <div className="flex items-center border border-slate-200 dark:border-slate-600 rounded-lg overflow-hidden">
            <button
              onClick={() => setViewMode('table')}
              className={`p-2 transition-colors ${
                viewMode === 'table'
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
              }`}
              title={t('table_view') || 'Table View'}
              aria-label={t('table_view') || 'Table View'}
              aria-pressed={viewMode === 'table'}
            >
              <List size={18} />
            </button>
            <button
              onClick={() => setViewMode('kanban')}
              className={`p-2 transition-colors ${
                viewMode === 'kanban'
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
              }`}
              title={t('kanban_view') || 'Kanban View'}
              aria-label={t('kanban_view') || 'Kanban View'}
              aria-pressed={viewMode === 'kanban'}
            >
              <LayoutGrid size={18} />
            </button>
            <button
              onClick={() => setViewMode('calendar')}
              className={`p-2 transition-colors ${
                viewMode === 'calendar'
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
              }`}
              title={t('calendar_view') || 'Calendar View'}
              aria-label={t('calendar_view') || 'Calendar View'}
              aria-pressed={viewMode === 'calendar'}
            >
              <Calendar size={18} />
            </button>
          </div>

          <SavedViews
            page="work-orders"
            currentFilterState={{
              filters: { filterStatus, filterPriority, quickFilter },
              sort: { column: '', direction: 'asc' },
            }}
            onApplyView={handleApplyView}
          />
          <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
            {t('create_work_order')}
          </Button>
        </div>
      </div>

      <Card>
        {/* Error banner */}
        {bulkError && (
          <div className="px-4 py-2 mb-2 text-sm text-red-700 bg-red-50 dark:bg-red-900/20 dark:text-red-300 border border-red-200 dark:border-red-800 rounded">
            {bulkError}
            <button className="ml-2 font-medium hover:underline" onClick={() => setBulkError(null)}>
              {t('dismiss')}
            </button>
          </div>
        )}

        {/* Quick Filters (WO-4.2.2) */}
        <QuickFilters
          workOrders={workOrders}
          activeFilter={quickFilter}
          currentUserId={user?.id}
          onChange={(filter) => {
            setQuickFilter(filter);
            setFilterStatus('');
            setFilterPriority('');
          }}
          className="mb-4"
        />

        {/* Filters */}
        <div className="flex gap-4 mb-4">
          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
            className="border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
          >
            <option value="">{t('all_statuses')}</option>
            <option value="open">{t('open')}</option>
            <option value="in_progress">{t('in_progress')}</option>
            <option value="completed">{t('completed')}</option>
            <option value="cancelled">{t('cancelled')}</option>
          </select>
          <select
            value={filterPriority}
            onChange={(e) => setFilterPriority(e.target.value)}
            className="border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
          >
            <option value="">{t('all_priorities')}</option>
            <option value="critical">{t('critical')}</option>
            <option value="high">{t('high')}</option>
            <option value="medium">{t('medium')}</option>
            <option value="low">{t('low')}</option>
          </select>
        </div>

        {/* Table / Kanban View */}
        {viewMode === 'table' ? (
          <DataGrid
            data={filtered}
            columns={columns}
            keyExtractor={(item: WorkOrder) => item.id}
            loading={loading}
            emptyMessage={t('no_work_orders')}
            exportFilename="work-orders.csv"
            selectable
            selectedIds={selectedIds}
            onSelectionChange={(ids: Set<string>) => setSelectedIds(ids)}
            onRowHover={(item: WorkOrder) => prefetchWorkOrder(queryClient, item.id)}
            onRowClick={(item: WorkOrder) => navigate(`/work-orders/${item.id}`)}
            stickyHeader
            persistId="work-orders"
            bulkActions={bulkActions}
          />
        ) : viewMode === 'kanban' ? (
          <WOKanbanBoard
            workOrders={filtered}
            onStatusChange={handleKanbanStatusChange}
            onCardClick={(wo) => navigate(`/work-orders/${wo.id}`)}
            currentUserId={user?.id}
          />
        ) : (
          <WorkOrderCalendar
            workOrders={filtered}
            technicians={technicians}
            currentUserId={user?.id}
            onDateChange={handleCalendarDateChange}
            onEventClick={(wo) => navigate(`/work-orders/${wo.id}`)}
            onDateClick={handleCalendarDateClick}
          />
        )}
      </Card>

      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title={t('create_work_order')}
      >
        <CreateWorkOrderForm onClose={() => setShowCreateModal(false)} />
      </Modal>

      {ConfirmDialog}
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// Create Work Order Form
// ═══════════════════════════════════════════════════════════════════════

const CreateWorkOrderForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { t } = useTranslation();
  const createWorkOrder = useCreateWorkOrder();
  const { data: rawSites = [] } = useSites();
  const { data: rawDevices = [] } = useDevices();
  const { data: rawUsers = [] } = useUsers();

  const sites = useMemo(() => rawSites.map((s: any) => ({ id: s.id, name: s.name || 'Unnamed' })), [rawSites]);
  const devsArray = Array.isArray(rawDevices) ? rawDevices : (rawDevices && typeof rawDevices === 'object' && 'devices' in rawDevices ? (rawDevices as any).devices : []);
  const devices = useMemo(() => devsArray.map((d: any) => ({ id: d.device_id, name: d.name || d.device_id, siteId: d.site_id || 'site-default' })), [devsArray]);
  const users = useMemo(() => rawUsers.map((u: any) => ({ ...u, name: u.name || u.username })), [rawUsers]);
  const technicians = users.filter((u: any) => u.role === 'technician');

  const { errors: woErrors, validate: validateWO, validateField: validateWOField, touched: woTouched, reset: resetWOValidation } = useFormValidation(workOrderSchema);

  const [selectedSiteId, setSelectedSiteId] = useState('');
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [selectedTechnicianId, setSelectedTechnicianId] = useState('');

  useEffect(() => {
    if (selectedSiteId && technicians.length > 0) {
      setSelectedTechnicianId(technicians[0].id);
    } else {
      setSelectedTechnicianId('');
    }
  }, [selectedSiteId, technicians]);

  const [formData, setFormData] = useState({
    type: 'corrective' as 'preventive' | 'corrective' | 'emergency',
    priority: 'medium' as 'critical' | 'high' | 'medium' | 'low',
    notes: '',
  });

  const siteDevices = devices.filter((d: any) => d.siteId === selectedSiteId);

  const toast = useToast();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDeviceId) {
      toast.error(t('select_device_required') || 'Select a device');
      return;
    }

    const validationData = {
      title: formData.notes || 'Work Order',
      description: formData.notes || 'Work order description',
      deviceId: selectedDeviceId,
      type: formData.type,
      priority: formData.priority,
      assignedTo: selectedTechnicianId || undefined,
    };

    if (!validateWO(validationData)) return;

    try {
      await createWorkOrder.mutateAsync({
        device_id: selectedDeviceId,
        type: formData.type,
        priority: formData.priority,
        assigned_to: selectedTechnicianId || undefined,
        notes: formData.notes,
      });
      onClose();
      toast.success(t('work_order_created') || 'Work order created');
    } catch (err) {
      const message = err instanceof Error ? err.message : (t('create_failed') || 'Failed to create work order');
      toast.error(message);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium mb-1">{t('site') || 'Site'}</label>
        <select value={selectedSiteId} onChange={e => { setSelectedSiteId(e.target.value); setSelectedDeviceId(''); }}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600" required>
          <option value="">{t('select_site') || 'Select site...'}</option>
          {sites.map(site => <option key={site.id} value={site.id}>{site.name}</option>)}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">{t('device') || 'Device'}</label>
        <select
          value={selectedDeviceId}
          onChange={e => {
            setSelectedDeviceId(e.target.value);
            const validationData = {
              title: formData.notes || 'Work Order',
              description: formData.notes || 'Work order description',
              deviceId: e.target.value,
              type: formData.type,
              priority: formData.priority,
              assignedTo: selectedTechnicianId || undefined,
            };
            validateWOField('deviceId', validationData);
          }}
          className={`w-full border rounded px-3 py-2 dark:bg-slate-800 ${woTouched.has('deviceId') && woErrors.deviceId ? 'border-red-500' : 'dark:border-slate-600'}`}
          required disabled={!selectedSiteId}
        >
          <option value="">{t('select_device') || 'Select device...'}</option>
          {siteDevices.map((dev: any) => <option key={dev.id} value={dev.id}>{dev.name}</option>)}
        </select>
        {woTouched.has('deviceId') && woErrors.deviceId && (
          <p className="mt-1 text-sm text-red-600">{woErrors.deviceId}</p>
        )}
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">{t('assigned_to') || 'Assigned to'}</label>
        <select value={selectedTechnicianId} onChange={e => setSelectedTechnicianId(e.target.value)}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600">
          <option value="">{t('unassigned') || 'Unassigned'}</option>
          {technicians.map(tech => <option key={tech.id} value={tech.id}>{tech.name || tech.username}</option>)}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">{t('type')}</label>
        <select value={formData.type} onChange={e => setFormData({...formData, type: e.target.value as any})}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600">
          <option value="preventive">{t('preventive')}</option>
          <option value="corrective">{t('corrective')}</option>
          <option value="emergency">{t('emergency')}</option>
        </select>
      </div>
      <div>
        <label className="block text-sm font-medium mb-1">{t('priority')}</label>
        <select value={formData.priority} onChange={e => setFormData({...formData, priority: e.target.value as any})}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600">
          <option value="critical">{t('critical')}</option>
          <option value="high">{t('high')}</option>
          <option value="medium">{t('medium')}</option>
          <option value="low">{t('low')}</option>
        </select>
      </div>
      <Input label={t('notes')} value={formData.notes}
        onChange={e => setFormData({...formData, notes: e.target.value})} />
      <div className="flex gap-2 justify-end pt-4">
        <Button variant="secondary" onClick={onClose}>{t('cancel')}</Button>
        <Button type="submit">{t('create')}</Button>
      </div>
    </form>
  );
};
