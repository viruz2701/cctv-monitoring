import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMaintenance } from '../context/MaintenanceContext';
import { MaintenanceSchedule } from '../services/maintenanceApi';
import { Button, Card, Table, Badge, Modal, Input } from '../components/ui';
import { Plus, Calendar, CheckCircle, AlertCircle } from 'lucide-react';

export const MaintenanceSchedules: React.FC = () => {
  const { t } = useTranslation();
  const { schedules, loading, completeSchedule, deleteSchedule } = useMaintenance();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [filterType, setFilterType] = useState('');
  const [filterPriority, setFilterPriority] = useState('');

  const filteredSchedules = schedules.filter((s) => {
    if (filterType && s.schedule_type !== filterType) return false;
    if (filterPriority && s.priority !== filterPriority) return false;
    return true;
  });

  const getPriorityVariant = (priority: string): 'danger' | 'warning' | 'info' | 'success' => {
    switch (priority) {
      case 'critical': return 'danger';
      case 'high': return 'warning';
      case 'medium': return 'info';
      case 'low': return 'success';
      default: return 'info';
    }
  };

  const isOverdue = (nextDue: string) => new Date(nextDue) < new Date();

  const handleComplete = async (id: string) => {
    if (window.confirm(t('confirm_complete_schedule') || 'Complete this schedule?')) {
      await completeSchedule(id);
    }
  };

  const columns = [
    {
      key: 'device_name',
      header: t('device'),
      render: (item: MaintenanceSchedule) => item.device_name || item.device_id,
    },
    {
      key: 'schedule_type',
      header: t('type'),
      render: (item: MaintenanceSchedule) => t(item.schedule_type),
    },
    {
      key: 'priority',
      header: t('priority'),
      render: (item: MaintenanceSchedule) => (
        <Badge variant={getPriorityVariant(item.priority)}>
          {t(item.priority)}
        </Badge>
      ),
    },
    {
      key: 'next_due',
      header: t('next_due'),
      render: (item: MaintenanceSchedule) => (
        <div className="flex items-center gap-2">
          {isOverdue(item.next_due) ? (
            <AlertCircle className="text-red-500" size={16} />
          ) : (
            <Calendar className="text-blue-500" size={16} />
          )}
          {new Date(item.next_due).toLocaleDateString()}
        </div>
      ),
    },
    {
      key: 'assignee_name',
      header: t('assigned_to'),
      render: (item: MaintenanceSchedule) => item.assignee_name || t('unassigned'),
    },
    {
      key: 'actions',
      header: t('actions'),
      render: (item: MaintenanceSchedule) => (
        <div className="flex gap-2">
          <Button
            size="sm"
            onClick={(e) => { e.stopPropagation(); handleComplete(item.id); }}
            icon={<CheckCircle size={16} />}
          >
            {t('complete')}
          </Button>
          <Button
            size="sm"
            variant="danger"
            onClick={(e) => { e.stopPropagation(); deleteSchedule(item.id); }}
          >
            {t('delete')}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">{t('maintenance_schedules')}</h1>
        <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
          {t('create_schedule')}
        </Button>
      </div>

      <Card>
        <div className="flex gap-4 mb-4">
          <select
            value={filterType}
            onChange={(e) => setFilterType(e.target.value)}
            className="border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
          >
            <option value="">{t('all_types')}</option>
            <option value="daily">{t('daily')}</option>
            <option value="weekly">{t('weekly')}</option>
            <option value="monthly">{t('monthly')}</option>
            <option value="quarterly">{t('quarterly')}</option>
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

        <Table
          data={filteredSchedules}
          columns={columns}
          keyExtractor={(item) => item.id}
          loading={loading}
          emptyMessage={t('no_schedules')}
        />
      </Card>

      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title={t('create_schedule')}
      >
        <CreateScheduleForm onClose={() => setShowCreateModal(false)} />
      </Modal>
    </div>
  );
};

const CreateScheduleForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { t } = useTranslation();
  const { createSchedule } = useMaintenance();

  const [formData, setFormData] = useState<{
    device_id: string;
    schedule_type: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'custom';
    interval_days: number;
    priority: 'critical' | 'high' | 'medium' | 'low';
    next_due: string;
    notes: string;
  }>(() => ({
    device_id: '',
    schedule_type: 'monthly',
    interval_days: 30,
    priority: 'medium',
    next_due: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    notes: '',
  }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await createSchedule(formData);
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
        <label className="block text-sm font-medium mb-1">{t('schedule_type')}</label>
        <select
          value={formData.schedule_type}
          onChange={(e) => setFormData({ ...formData, schedule_type: e.target.value as typeof formData.schedule_type })}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
        >
          <option value="daily">{t('daily')}</option>
          <option value="weekly">{t('weekly')}</option>
          <option value="monthly">{t('monthly')}</option>
          <option value="quarterly">{t('quarterly')}</option>
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
        type="date"
        label={t('next_due')}
        value={formData.next_due}
        onChange={(e) => setFormData({ ...formData, next_due: e.target.value })}
        required
      />
      <Input
        label={t('notes')}
        value={formData.notes}
        onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
      />
      <div className="flex gap-2 justify-end pt-4">
        <Button variant="secondary" onClick={onClose}>
          {t('cancel')}
        </Button>
        <Button type="submit">{t('create')}</Button>
      </div>
    </form>
  );
};
