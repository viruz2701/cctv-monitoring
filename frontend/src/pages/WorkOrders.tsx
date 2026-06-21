import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useWorkOrders } from '../context/WorkOrdersContext';
import { WorkOrder } from '../services/workOrdersApi';
import { Button, Card, VirtualTable, Badge, Modal, Input } from '../components/ui';
import { Plus, Play, CheckCircle, XCircle, Clock, AlertTriangle } from 'lucide-react';

export const WorkOrders: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { workOrders, loading, startWorkOrder, completeWorkOrder, cancelWorkOrder } = useWorkOrders();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [filterStatus, setFilterStatus] = useState('');
  const [filterPriority, setFilterPriority] = useState('');

  const filtered = workOrders.filter((wo) => {
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

  const columns = [
    {
      key: 'device_name',
      header: t('device'),
      render: (item: WorkOrder) => item.device_name || item.device_id,
    },
    {
      key: 'type',
      header: t('type'),
      render: (item: WorkOrder) => t(item.type),
    },
    {
      key: 'priority',
      header: t('priority'),
      render: (item: WorkOrder) => (
        <Badge variant={getPriorityVariant(item.priority)}>{t(item.priority)}</Badge>
      ),
    },
    {
      key: 'status',
      header: t('status'),
      render: (item: WorkOrder) => (
        <Badge variant={getStatusVariant(item.status)}>{t(item.status)}</Badge>
      ),
    },
    {
      key: 'sla',
      header: 'SLA',
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
      render: (item: WorkOrder) => item.assignee_name || t('unassigned'),
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
          keyExtractor={(item) => item.id}
          loading={loading}
          emptyMessage={t('no_work_orders')}
          exportable
          exportFilename="work-orders.csv"
          selectable
          onRowClick={(item) => navigate(`/work-orders/${item.id}`)}
          maxHeight={600}
          estimateRowHeight={64}
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

const CreateWorkOrderForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { t } = useTranslation();
  const { createWorkOrder } = useWorkOrders();
  const [formData, setFormData] = useState<{
    device_id: string;
    type: 'preventive' | 'corrective' | 'emergency';
    priority: 'critical' | 'high' | 'medium' | 'low';
    notes: string;
  }>({
    device_id: '',
    type: 'corrective',
    priority: 'medium',
    notes: '',
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await createWorkOrder(formData);
    onClose();
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input
        label={t('device_id')}
        value={formData.device_id}
        onChange={(e) => setFormData({ ...formData, device_id: e.target.value })}
        required
      />
      <div>
        <label className="block text-sm font-medium mb-1">{t('type')}</label>
        <select
          value={formData.type}
          onChange={(e) => setFormData({ ...formData, type: e.target.value as typeof formData.type })}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
        >
          <option value="preventive">{t('preventive')}</option>
          <option value="corrective">{t('corrective')}</option>
          <option value="emergency">{t('emergency')}</option>
        </select>
      </div>
      <div>
        <label className="block text-sm font-medium mb-1">{t('priority')}</label>
        <select
          value={formData.priority}
          onChange={(e) => setFormData({ ...formData, priority: e.target.value as typeof formData.priority })}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
        >
          <option value="critical">{t('critical')}</option>
          <option value="high">{t('high')}</option>
          <option value="medium">{t('medium')}</option>
          <option value="low">{t('low')}</option>
        </select>
      </div>
      <Input
        label={t('notes')}
        value={formData.notes}
        onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
      />
      <div className="flex gap-2 justify-end pt-4">
        <Button variant="secondary" onClick={onClose}>{t('cancel')}</Button>
        <Button type="submit">{t('create')}</Button>
      </div>
    </form>
  );
};
