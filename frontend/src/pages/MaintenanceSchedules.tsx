import React, { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { getArrayData } from '../utils/helpers';
import {
  useMaintenanceSchedules,
  useCompleteMaintenanceSchedule,
  useDeleteMaintenanceSchedule,
  useUpdateMaintenanceSchedule,
  useCreateMaintenanceSchedule,
  useSites,
  useDevices,
  useUsers,
} from '../hooks/useApiQuery';
import { useWorkOrders } from '../hooks/useApiQuery';
import { useNavigate } from 'react-router-dom';
import { MaintenanceSchedule } from '../services/maintenanceApi';
import { Button, Card, DataGrid, Badge, Modal, Input } from '../components/ui';
import { Plus, Calendar, CheckCircle, AlertCircle, Table2, CalendarDays } from '../components/ui/Icons';
import ScheduleXWrapper from '../components/planning/FullCalendarWrapper';

type ViewMode = 'table' | 'calendar';

const PRIORITY_COLORS: Record<string, { bg: string; border: string }> = {
  critical: { bg: '#ef4444', border: '#dc2626' },
  high: { bg: '#f97316', border: '#ea580c' },
  medium: { bg: '#3b82f6', border: '#2563eb' },
  low: { bg: '#22c55e', border: '#16a34a' },
};

export const MaintenanceSchedules: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: schedules = [], isLoading: loading } = useMaintenanceSchedules();
  const { data: workOrders = [] } = useWorkOrders();
  const completeScheduleMut = useCompleteMaintenanceSchedule();
  const deleteScheduleMut = useDeleteMaintenanceSchedule();
  const updateScheduleMut = useUpdateMaintenanceSchedule();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [filterType, setFilterType] = useState('');
  const [filterPriority, setFilterPriority] = useState('');
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [selectedEvent, setSelectedEvent] = useState<MaintenanceSchedule | null>(null);

  const filteredSchedules = useMemo(() => schedules.filter((s) => {
    if (filterType && s.schedule_type !== filterType) return false;
    if (filterPriority && s.priority !== filterPriority) return false;
    return true;
  }), [schedules, filterType, filterPriority]);

  const calendarEvents = useMemo(() => filteredSchedules.map((s) => {
    const colors = PRIORITY_COLORS[s.priority] || PRIORITY_COLORS.medium;
    const dueDate = new Date(s.next_due);
    return {
      id: s.id,
      title: `${s.device_name || s.device_id}: ${t(s.schedule_type)}`,
      start: dueDate.toISOString().split('T')[0],
      backgroundColor: colors.bg,
      borderColor: colors.border,
      textColor: '#ffffff',
      extendedProps: { schedule: s },
    };
  }), [filteredSchedules, t]);

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
      await completeScheduleMut.mutateAsync(id);
      setSelectedEvent(null);
    }
  };

  const handleEventClick = (schedule: MaintenanceSchedule) => {
    setSelectedEvent(schedule);
  };

  const handleEventDrop = async (schedule: MaintenanceSchedule, newDate: string) => {
    await updateScheduleMut.mutateAsync({ id: schedule.id, data: { next_due: newDate } });
  };

  const columns = [
    {
      key: 'device_name',
      header: t('device'),
      sortable: true,
      render: (item: MaintenanceSchedule) => item.device_name || item.device_id,
    },
    {
      key: 'schedule_type',
      header: t('type'),
      sortable: true,
      render: (item: MaintenanceSchedule) => t(item.schedule_type),
    },
    {
      key: 'priority',
      header: t('priority'),
      sortable: true,
      render: (item: MaintenanceSchedule) => (
        <Badge variant={getPriorityVariant(item.priority)}>
          {t(item.priority)}
        </Badge>
      ),
    },
    {
      key: 'next_due',
      header: t('next_due'),
      sortable: true,
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
      sortable: true,
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
            onClick={(e) => { e.stopPropagation(); deleteScheduleMut.mutateAsync(item.id); }}
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
        <div className="flex gap-2">
          <div className="flex bg-slate-100 dark:bg-slate-800 rounded-lg p-0.5">
            <button
              onClick={() => setViewMode('table')}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                viewMode === 'table'
                  ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
              }`}
            >
              <Table2 size={16} />
              {t('view_table')}
            </button>
            <button
              onClick={() => setViewMode('calendar')}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
                viewMode === 'calendar'
                  ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
              }`}
            >
              <CalendarDays size={16} />
              {t('view_calendar')}
            </button>
          </div>
          <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
            {t('create_schedule')}
          </Button>
        </div>
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

        {viewMode === 'table' ? (
          <DataGrid
            data={filteredSchedules}
            columns={columns}
            keyExtractor={(item) => item.id}
            loading={loading}
            emptyMessage={t('no_schedules')}
            variant="striped"
            defaultDensity="standard"
            pageSize={10}
            exportFilename="maintenance-schedules.csv"
          />
        ) : (
          <div className="calendar-container">
            {loading ? (
              <div className="flex items-center justify-center h-96 text-slate-400">{t('loading')}</div>
            ) : filteredSchedules.length === 0 ? (
              <div className="flex items-center justify-center h-96 text-slate-400">{t('no_events')}</div>
            ) : (
              <ScheduleXWrapper
                events={calendarEvents}
                onEventClick={handleEventClick}
                onEventDrop={handleEventDrop}
              />
            )}
          </div>
        )}
      </Card>

      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title={t('create_schedule')}
      >
        <CreateScheduleForm onClose={() => setShowCreateModal(false)} />
      </Modal>

      <Modal
        isOpen={!!selectedEvent}
        onClose={() => setSelectedEvent(null)}
        title={t('schedule_details')}
        size="sm"
        footer={
          <div className="flex gap-2 justify-end">
            <Button variant="secondary" onClick={() => setSelectedEvent(null)}>
              {t('close')}
            </Button>
            <Button
              variant="danger"
              onClick={() => selectedEvent && deleteScheduleMut.mutateAsync(selectedEvent.id)}
            >
              {t('delete')}
            </Button>
            <Button
              onClick={() => selectedEvent && handleComplete(selectedEvent.id)}
              icon={<CheckCircle size={16} />}
            >
              {t('complete')}
            </Button>
          </div>
        }
      >
        {selectedEvent && (
          <div className="space-y-3">
            <div>
              <span className="text-xs text-slate-500 dark:text-slate-400">{t('device')}</span>
              <p className="font-medium">{selectedEvent.device_name || selectedEvent.device_id}</p>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <span className="text-xs text-slate-500 dark:text-slate-400">{t('type')}</span>
                <p>{t(selectedEvent.schedule_type)}</p>
              </div>
              <div>
                <span className="text-xs text-slate-500 dark:text-slate-400">{t('priority')}</span>
                <Badge variant={getPriorityVariant(selectedEvent.priority)}>{t(selectedEvent.priority)}</Badge>
              </div>
              <div>
                <span className="text-xs text-slate-500 dark:text-slate-400">{t('next_due')}</span>
                <p className="flex items-center gap-1">
                  {isOverdue(selectedEvent.next_due) && <AlertCircle className="text-red-500" size={14} />}
                  {new Date(selectedEvent.next_due).toLocaleDateString()}
                </p>
              </div>
              <div>
                <span className="text-xs text-slate-500 dark:text-slate-400">{t('interval_days')}</span>
                <p>{selectedEvent.interval_days}</p>
              </div>
            </div>
            {selectedEvent.last_completed && (
              <div>
                <span className="text-xs text-slate-500 dark:text-slate-400">{t('last_completed')}</span>
                <p>{new Date(selectedEvent.last_completed).toLocaleDateString()}</p>
              </div>
            )}
            <div>
              <span className="text-xs text-slate-500 dark:text-slate-400">{t('assigned_to')}</span>
              <p>{selectedEvent.assignee_name || t('unassigned')}</p>
            </div>
            {selectedEvent.notes && (
              <div>
                <span className="text-xs text-slate-500 dark:text-slate-400">{t('notes')}</span>
                <p className="text-sm">{selectedEvent.notes}</p>
              </div>
            )}

            {/* Related Work Orders */}
            <div className="border-t border-slate-200 dark:border-slate-700 pt-3 mt-3">
              <h4 className="text-sm font-semibold mb-2">{t('work_orders') || 'Work Orders'}</h4>
              {workOrders.filter(wo => wo.schedule_id === selectedEvent.id).length === 0 ? (
                <p className="text-sm text-slate-500">{t('no_work_orders') || 'No work orders'}</p>
              ) : (
                <div className="space-y-2">
                  {workOrders.filter(wo => wo.schedule_id === selectedEvent.id).map(wo => (
                    <div key={wo.id} className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800 rounded-lg text-sm"
                      onClick={() => navigate(`/work-orders/${wo.id}`)} style={{cursor: 'pointer'}}>
                      <span>{wo.device_name || wo.device_id}</span>
                      <Badge variant={wo.status === 'completed' ? 'success' : wo.status === 'in_progress' ? 'primary' : 'neutral'}>
                        {wo.status}
                      </Badge>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

const CreateScheduleForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { t } = useTranslation();
  const createSchedule = useCreateMaintenanceSchedule();
  const { data: rawSites } = useSites();
  const { data: rawDevices } = useDevices();
  const { data: rawUsers = [] } = useUsers();
  const safeSites = getArrayData<{ id: string; name?: string }>(rawSites);
  const safeDevices = getArrayData<{ device_id: string; name?: string; site_id?: string }>(rawDevices);
  const sites = useMemo(() => safeSites.map((s: any) => ({ id: s.id, name: s.name || 'Unnamed' })), [safeSites]);
  const devices = useMemo(() => safeDevices.map((d: any) => ({ id: d.device_id, name: d.name || d.device_id, siteId: d.site_id || 'site-default' })), [safeDevices]);
  const users = useMemo(() => rawUsers.map((u: any) => ({ ...u, name: u.name || u.username })), [rawUsers]);
  const technicians = users.filter((u: any) => u.role === 'technician');

  const [selectedSiteId, setSelectedSiteId] = useState('');
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [selectedTechnicianId, setSelectedTechnicianId] = useState('');
  const [formData, setFormData] = useState({
    schedule_type: 'monthly' as const,
    interval_days: 30,
    priority: 'medium' as const,
    next_due: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    notes: '',
  });

  const siteDevices = devices.filter((d: any) => d.siteId === selectedSiteId);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedDeviceId) return;
    await createSchedule.mutateAsync({
      device_id: selectedDeviceId,
      schedule_type: formData.schedule_type,
      interval_days: formData.interval_days,
      priority: formData.priority,
      next_due: formData.next_due,
      assigned_to: selectedTechnicianId || undefined,
      notes: formData.notes,
    });
    onClose();
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
          {siteDevices.map((dev: any) => <option key={dev.id} value={dev.id}>{dev.name}</option>)}
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

      {/* schedule_type */}
      <div>
        <label className="block text-sm font-medium mb-1">{t('schedule_type')}</label>
        <select value={formData.schedule_type} onChange={e => setFormData({...formData, schedule_type: e.target.value as any})}
          className="w-full border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600">
          <option value="daily">{t('daily')}</option>
          <option value="weekly">{t('weekly')}</option>
          <option value="monthly">{t('monthly')}</option>
          <option value="quarterly">{t('quarterly')}</option>
        </select>
      </div>

      {/* priority */}
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

      <Input type="date" label={t('next_due')} value={formData.next_due}
        onChange={e => setFormData({...formData, next_due: e.target.value})} required />
      <Input label={t('notes')} value={formData.notes}
        onChange={e => setFormData({...formData, notes: e.target.value})} />
      <div className="flex gap-2 justify-end pt-4">
        <Button variant="secondary" onClick={onClose}>{t('cancel')}</Button>
        <Button type="submit">{t('create')}</Button>
      </div>
    </form>
  );
};
