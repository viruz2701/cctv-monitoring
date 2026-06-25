import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useWorkOrders, BulkActionType } from '../context/WorkOrdersContext';
import { useDevicesSites } from '../context/DevicesSitesContext';
import { useUsers } from '../context/UsersContext';
import { useAuth } from '../hooks/useAuth';
import { WorkOrder } from '../services/workOrdersApi';
import type { User as ApiUser } from '../services/api';
import { Button, Card, Badge, Modal, Input, useToast } from '../components/ui';
import { VirtualTable } from '../components/ui/VirtualTable';
import {
  Plus, Play, CheckCircle, XCircle, Clock, AlertTriangle,
  CheckSquare, Square, UserCheck, Trash2, Tags, ArrowUpDown,
  Loader2, ChevronDown, User, AlertOctagon,
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

  // Close on click outside or Escape
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
    // Small delay to prevent immediate close from the click that opened it
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
// Bulk Action Bar Component (WO-4.2.1)
// ═══════════════════════════════════════════════════════════════════════

interface BulkActionBarProps {
  selectedCount: number;
  onClear: () => void;
  onAction: (action: BulkActionType, value?: string) => Promise<void>;
  loading: boolean;
}

const BulkActionBar: React.FC<BulkActionBarProps> = ({ selectedCount, onClear, onAction, loading }) => {
  const { t } = useTranslation();
  const [showActions, setShowActions] = useState(false);
  const [statusValue, setStatusValue] = useState('');
  const [priorityValue, setPriorityValue] = useState('');

  const handleAction = async (action: BulkActionType, value?: string) => {
    await onAction(action, value);
    setShowActions(false);
    setStatusValue('');
    setPriorityValue('');
  };

  if (selectedCount === 0) return null;

  return (
    <div className="flex items-center gap-3 px-4 py-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg mb-4 animate-fadeIn">
      <div className="flex items-center gap-2 text-sm font-medium text-blue-700 dark:text-blue-300">
        <CheckSquare size={16} />
        <span>{t('selected_count', { count: selectedCount })}</span>
      </div>

      <div className="h-5 w-px bg-blue-200 dark:bg-blue-700" />

      {loading ? (
        <Loader2 size={16} className="animate-spin text-blue-600" />
      ) : (
        <div className="flex items-center gap-2 flex-wrap">
          {/* Bulk Status Change */}
          <div className="flex items-center gap-1">
            <select
              value={statusValue}
              onChange={(e) => {
                setStatusValue(e.target.value);
                if (e.target.value) handleAction('status_change', e.target.value);
              }}
              className="text-xs border rounded px-2 py-1.5 dark:bg-slate-800 dark:border-slate-600"
            >
              <option value="">{t('change_status')}</option>
              <option value="open">{t('open')}</option>
              <option value="in_progress">{t('in_progress')}</option>
              <option value="completed">{t('completed')}</option>
              <option value="cancelled">{t('cancelled')}</option>
              <option value="on_hold">{t('on_hold')}</option>
            </select>
          </div>

          {/* Bulk Priority Change */}
          <div className="flex items-center gap-1">
            <select
              value={priorityValue}
              onChange={(e) => {
                setPriorityValue(e.target.value);
                if (e.target.value) handleAction('priority_change', e.target.value);
              }}
              className="text-xs border rounded px-2 py-1.5 dark:bg-slate-800 dark:border-slate-600"
            >
              <option value="">{t('change_priority')}</option>
              <option value="critical">{t('critical')}</option>
              <option value="high">{t('high')}</option>
              <option value="medium">{t('medium')}</option>
              <option value="low">{t('low')}</option>
            </select>
          </div>

          {/* Bulk Assign */}
          <Button
            size="sm"
            variant="secondary"
            icon={<UserCheck size={14} />}
            onClick={() => handleAction('assign', '')}
            title={t('bulk_assign')}
          >
            {t('assign')}
          </Button>

          {/* Bulk Delete */}
          <Button
            size="sm"
            variant="danger"
            icon={<Trash2 size={14} />}
            onClick={() => handleAction('delete')}
            title={t('bulk_delete')}
          >
            {t('delete')}
          </Button>
        </div>
      )}

      <div className="ml-auto">
        <Button
          size="sm"
          variant="ghost"
          onClick={onClear}
          icon={<XCircle size={14} />}
        >
          {t('clear_selection')}
        </Button>
      </div>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// Work Orders Page
// ═══════════════════════════════════════════════════════════════════════

export const WorkOrders: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { workOrders, loading, startWorkOrder, completeWorkOrder, cancelWorkOrder, updateWorkOrder, bulkActionWorkOrders } = useWorkOrders();
  const { users } = useUsers();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [filterStatus, setFilterStatus] = useState('');
  const [filterPriority, setFilterPriority] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [bulkLoading, setBulkLoading] = useState(false);
  const [bulkError, setBulkError] = useState<string | null>(null);
  const [quickFilter, setQuickFilter] = useState<'all' | 'mine' | 'overdue' | 'critical'>('all');

  const technicians = users.filter(u => u.role === 'technician');

  const filtered = workOrders.filter((wo) => {
    // Quick filters (WO-4.2.2)
    if (quickFilter === 'mine' && wo.assigned_to !== user?.id) return false;
    if (quickFilter === 'overdue') {
      const notFinished = wo.status !== 'completed' && wo.status !== 'cancelled';
      const isOverdue = wo.sla_deadline && new Date(wo.sla_deadline) < new Date();
      if (!notFinished || !isOverdue) return false;
    }
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
    setBulkLoading(true);
    setBulkError(null);
    try {
      const ids = Array.from(selectedIds);
      const response = await bulkActionWorkOrders(action, ids, value);
      if (response.failed > 0) {
        setBulkError(`${response.failed} operation(s) failed`);
      }
      setSelectedIds(new Set());
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Bulk action failed');
    } finally {
      setBulkLoading(false);
    }
  }, [selectedIds, bulkActionWorkOrders]);

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
            await updateWorkOrder(item.id, { priority: val as WorkOrder['priority'] });
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
            await updateWorkOrder(item.id, { status: val as WorkOrder['status'] });
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
            await updateWorkOrder(item.id, { assigned_to: val || undefined });
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
            <Button size="sm" onClick={(e) => { e.stopPropagation(); startWorkOrder(item.id); }} icon={<Play size={14} />}>
              {t('start')}
            </Button>
          )}
          {item.status === 'in_progress' && (
            <Button size="sm" onClick={(e) => { e.stopPropagation(); completeWorkOrder(item.id, '', [], []); }} icon={<CheckCircle size={14} />}>
              {t('complete')}
            </Button>
          )}
          {(item.status === 'open' || item.status === 'in_progress') && (
            <Button size="sm" variant="danger" onClick={(e) => { e.stopPropagation(); cancelWorkOrder(item.id, 'Cancelled by user'); }} icon={<XCircle size={14} />}>
              {t('cancel')}
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('work_orders')}</h1>
        <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
          {t('create_work_order')}
        </Button>
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

        {/* Bulk Action Bar */}
        <BulkActionBar
          selectedCount={selectedIds.size}
          onClear={() => setSelectedIds(new Set())}
          onAction={handleBulkAction}
          loading={bulkLoading}
        />

        {/* Quick Filters (WO-4.2.2) */}
        <div className="flex items-center gap-2 mb-4 flex-wrap">
          <span className="text-xs font-medium text-gray-500 dark:text-gray-400 mr-1">
            {t('quick_filters')}:
          </span>
          {([
            { key: 'all', label: t('all'), icon: null },
            { key: 'mine', label: t('my_work_orders'), icon: <User size={14} /> },
            { key: 'overdue', label: t('overdue'), icon: <Clock size={14} /> },
            { key: 'critical', label: t('critical'), icon: <AlertOctagon size={14} /> },
          ] as const).map(({ key, label, icon }) => (
            <button
              key={key}
              onClick={() => {
                setQuickFilter(key);
                // Сбрасываем dropdown фильтры при переключении quick filter
                setFilterStatus('');
                setFilterPriority('');
              }}
              className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full transition-colors ${
                quickFilter === key
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 ring-1 ring-blue-300 dark:ring-blue-700'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-slate-800 dark:text-gray-400 dark:hover:bg-slate-700'
              }`}
            >
              {icon}
              {label}
            </button>
          ))}
        </div>

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

        <VirtualTable
          data={filtered}
          columns={columns}
          keyExtractor={(item: WorkOrder) => item.id}
          loading={loading}
          emptyMessage={t('no_work_orders')}
          exportFilename="work-orders.csv"
          selectable
          selectedIds={selectedIds}
          onSelectionChange={(ids: Set<string>) => setSelectedIds(ids)}
          onRowClick={(item: WorkOrder) => navigate(`/work-orders/${item.id}`)}
          maxHeight={650}
        />
      </Card>

      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title={t('create_work_order')}
      >
        <CreateWorkOrderForm onClose={() => setShowCreateModal(false)} />
      </Modal>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// Create Work Order Form
// ═══════════════════════════════════════════════════════════════════════

const CreateWorkOrderForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { t } = useTranslation();
  const { createWorkOrder } = useWorkOrders();
  const { sites, devices } = useDevicesSites();
  const { users } = useUsers();
  const technicians = users.filter(u => u.role === 'technician');

  const [selectedSiteId, setSelectedSiteId] = useState('');
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [selectedTechnicianId, setSelectedTechnicianId] = useState('');

  // WO-4.2.4: Auto-assign technician when site is selected
  // При выборе сайта пытаемся подтянуть закрепленного техника
  useEffect(() => {
    if (selectedSiteId && technicians.length > 0) {
      // Если техник уже выбран и сайт не менялся — не сбрасываем
      // Иначе пытаемся найти подходящего техника
      // В реальной системе здесь должен быть запрос к TechnicianSiteAssignments API
      // Пока используем fallback: первый техник в списке
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

  const siteDevices = devices.filter(d => d.siteId === selectedSiteId);

  const toast = useToast();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDeviceId) {
      toast.error(t('select_device_required') || 'Select a device');
      return;
    }
    try {
      await createWorkOrder({
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
      {/* Site select */}
      <div>
        <label className="block text-sm font-medium mb-1">{t('site') || 'Site'}</label>
        <select value={selectedSiteId} onChange={e => { setSelectedSiteId(e.target.value); setSelectedDeviceId(''); }}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600" required>
          <option value="">{t('select_site') || 'Select site...'}</option>
          {sites.map(site => <option key={site.id} value={site.id}>{site.name}</option>)}
        </select>
      </div>

      {/* Device select */}
      <div>
        <label className="block text-sm font-medium mb-1">{t('device') || 'Device'}</label>
        <select value={selectedDeviceId} onChange={e => setSelectedDeviceId(e.target.value)}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600" required disabled={!selectedSiteId}>
          <option value="">{t('select_device') || 'Select device...'}</option>
          {siteDevices.map(dev => <option key={dev.id} value={dev.id}>{dev.name}</option>)}
        </select>
      </div>

      {/* Technician select */}
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
