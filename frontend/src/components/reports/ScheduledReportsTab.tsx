import React, { useState } from 'react';
import { Card, CardHeader, CardBody, Button, Badge, Modal, Input, Select } from '../ui';
import { Plus, Clock, Users, Mail, Settings, Play, Trash2, Pause, X } from 'lucide-react';
import { useToast } from '../ui';
import { useReports } from '../../context/DataContext';
import { useTranslation } from 'react-i18next';

interface ScheduledReport {
    id: string;
    name: string;
    type: string;
    frequency: string;
    recipients: string[];
    status: 'active' | 'paused';
    nextRun: string;
    filters?: Record<string, string>;
}

interface CreateScheduleFormData {
    name: string;
    type: string;
    frequency: string;
    recipients: string;
}

export function ScheduledReportsTab() {
    const { t } = useTranslation();
    const toast = useToast();
    const { addGeneratedReport } = useReports();

    const [scheduledReports, setScheduledReports] = useState<ScheduledReport[]>([
        { id: 'sr-001', name: 'Weekly Management Overview', type: 'Consolidated Health', frequency: 'Weekly (Monday)', recipients: ['management@company.com', 'admin@company.com'], status: 'active', nextRun: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString() },
        { id: 'sr-002', name: 'Daily Region 1 Camera Status', type: 'Camera Health', frequency: 'Daily', recipients: ['region1-tech@company.com'], status: 'active', nextRun: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString() },
        { id: 'sr-003', name: 'Monthly Storage Compliance', type: 'Recording Availability', frequency: 'Monthly (1st)', recipients: ['compliance@company.com'], status: 'paused', nextRun: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString() }
    ]);
    const [showCreateSchedule, setShowCreateSchedule] = useState(false);
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editForm, setEditForm] = useState<Partial<ScheduledReport>>({});
    const [createForm, setCreateForm] = useState<CreateScheduleFormData>({
        name: '',
        type: 'Consolidated Health',
        frequency: 'Weekly (Monday)',
        recipients: ''
    });

    const handleRunNow = (report: ScheduledReport) => {
        const generatedAt = new Date().toISOString();
        addGeneratedReport({
            id: `sch-${Date.now()}`,
            name: `${report.name} (scheduled run)`,
            type: report.type,
            format: 'xlsx',
            dateRange: 'Last 7 days',
            generatedAt,
            generatedBy: 'Scheduled Report',
            status: 'ready',
            size: '1.2 MB',
        });
        toast.success(`Report "${report.name}" generated successfully`);
    };

    const handleTogglePause = (id: string) => {
        setScheduledReports(prev => prev.map(r => {
            if (r.id === id) {
                const newStatus = r.status === 'active' ? 'paused' : 'active';
                toast.info(`Schedule ${newStatus === 'active' ? 'resumed' : 'paused'}`);
                return { ...r, status: newStatus as 'active' | 'paused' };
            }
            return r;
        }));
    };

    const handleDelete = (id: string) => {
        setScheduledReports(prev => prev.filter(r => r.id !== id));
        toast.success('Schedule deleted');
    };

    const handleEdit = (report: ScheduledReport) => {
        setEditingId(report.id);
        setEditForm({
            name: report.name,
            type: report.type,
            frequency: report.frequency,
            recipients: report.recipients,
        });
    };

    const handleSaveEdit = () => {
        if (!editingId) return;
        setScheduledReports(prev => prev.map(r => {
            if (r.id === editingId && editForm.name && editForm.type && editForm.frequency) {
                return {
                    ...r,
                    name: editForm.name || r.name,
                    type: editForm.type || r.type,
                    frequency: editForm.frequency || r.frequency,
                    recipients: editForm.recipients?.filter(Boolean) || r.recipients,
                };
            }
            return r;
        }));
        setEditingId(null);
        setEditForm({});
        toast.success('Schedule updated');
    };

    const handleCreate = () => {
        if (!createForm.name || !createForm.recipients) {
            toast.error('Please fill in all required fields');
            return;
        }
        const newReport: ScheduledReport = {
            id: `sr-${Date.now()}`,
            name: createForm.name,
            type: createForm.type,
            frequency: createForm.frequency,
            recipients: createForm.recipients.split(',').map(r => r.trim()).filter(Boolean),
            status: 'active',
            nextRun: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
        };
        setScheduledReports(prev => [...prev, newReport]);
        setShowCreateSchedule(false);
        setCreateForm({ name: '', type: 'Consolidated Health', frequency: 'Weekly (Monday)', recipients: '' });
        toast.success('Schedule created successfully');
    };

    const typeOptions = [
        { value: 'Consolidated Health', label: t('consolidated_health') },
        { value: 'Camera Health', label: t('camera_health_report') },
        { value: 'Recording Availability', label: t('recording_availability_report') },
        { value: 'HDD/Storage Health', label: t('hdd_health_report') },
    ];

    const frequencyOptions = [
        { value: 'Daily', label: t('daily') },
        { value: 'Weekly (Monday)', label: t('weekly') },
        { value: 'Monthly (1st)', label: t('monthly') },
        { value: 'Quarterly', label: t('quarterly') },
    ];

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h3 className="text-lg font-medium text-slate-900 dark:text-white">{t('scheduled_reports')}</h3>
                    <p className="text-sm text-slate-500 dark:text-slate-400">{t('scheduled_reports_desc')}</p>
                </div>
                <Button icon={<Plus className="w-4 h-4" />} onClick={() => setShowCreateSchedule(true)}>{t('new_schedule')}</Button>
            </div>
            <div className="grid grid-cols-1 gap-4">
                {scheduledReports.map((report) => (
                    <Card key={report.id} variant="bordered" className="hover:border-blue-500/50 transition-colors">
                        <CardBody className="p-5">
                            <div className="flex flex-col md:flex-row gap-4 justify-between items-start md:items-center">
                                <div className="space-y-1">
                                    {editingId === report.id ? (
                                        <div className="space-y-3 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
                                            <Input label={t('name')} value={editForm.name || ''} onChange={(e) => setEditForm(f => ({ ...f, name: e.target.value }))} />
                                            <Select label={t('type')} options={typeOptions} value={editForm.type || ''} onChange={(e) => setEditForm(f => ({ ...f, type: e.target.value }))} />
                                            <Select label={t('frequency')} options={frequencyOptions} value={editForm.frequency || ''} onChange={(e) => setEditForm(f => ({ ...f, frequency: e.target.value }))} />
                                            <Input label={t('recipients')} value={(editForm.recipients || []).join(', ')} onChange={(e) => setEditForm(f => ({ ...f, recipients: e.target.value.split(',').map(r => r.trim()) }))} placeholder="email1@example.com, email2@example.com" />
                                            <div className="flex gap-2 pt-2">
                                                <Button size="sm" onClick={handleSaveEdit}>{t('save')}</Button>
                                                <Button variant="secondary" size="sm" onClick={() => setEditingId(null)}>{t('cancel')}</Button>
                                            </div>
                                        </div>
                                    ) : (
                                        <>
                                            <div className="flex items-center gap-3">
                                                <h4 className="font-semibold text-slate-900 dark:text-white text-base">{report.name}</h4>
                                                <Badge variant={report.status === 'active' ? 'success' : 'neutral'} size="sm">{report.status === 'active' ? t('active') : t('paused')}</Badge>
                                            </div>
                                            <div className="flex flex-wrap items-center gap-4 text-sm text-slate-500 dark:text-slate-400">
                                                <span className="flex items-center gap-1.5"><Settings className="w-4 h-4" />{report.type}</span>
                                                <span className="flex items-center gap-1.5"><Clock className="w-4 h-4" />{report.frequency}</span>
                                                <span className="flex items-center gap-1.5"><Users className="w-4 h-4" />{report.recipients.length} {t('recipients')}</span>
                                            </div>
                                        </>
                                    )}
                                </div>
                                {editingId !== report.id && (
                                    <div className="flex items-center gap-6 w-full md:w-auto mt-4 md:mt-0 pt-4 md:pt-0 border-t md:border-t-0 border-slate-200 dark:border-slate-700/50">
                                        <div className="hidden md:flex md:flex-col md:items-end">
                                            <span className="text-xs text-slate-500 dark:text-slate-400">{t('next_run')}</span>
                                            <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{new Date(report.nextRun).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
                                        </div>
                                        <div className="flex gap-2 w-full md:w-auto justify-end">
                                            <Button variant="ghost" size="sm" icon={<Play className="w-4 h-4" />} onClick={() => handleRunNow(report)}>{t('run_now')}</Button>
                                            <Button variant="ghost" size="sm" icon={report.status === 'active' ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />} onClick={() => handleTogglePause(report.id)}>{report.status === 'active' ? t('pause') || 'Pause' : t('resume') || 'Resume'}</Button>
                                            <Button variant="ghost" size="sm" onClick={() => handleEdit(report)}>{t('edit')}</Button>
                                            <Button variant="ghost" size="sm" className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20" icon={<Trash2 className="w-4 h-4" />} onClick={() => handleDelete(report.id)}>{t('delete')}</Button>
                                        </div>
                                    </div>
                                )}
                            </div>
                        </CardBody>
                    </Card>
                ))}
                {scheduledReports.length === 0 && (
                    <div className="text-center py-12 bg-slate-50 dark:bg-slate-800/50 rounded-xl border border-dashed border-slate-200 dark:border-slate-700">
                        <Clock className="w-12 h-12 text-slate-400 mx-auto mb-4" />
                        <h4 className="text-base font-medium text-slate-900 dark:text-white mb-1">{t('no_schedules') || 'No scheduled reports'}</h4>
                        <p className="text-sm text-slate-500 dark:text-slate-400">{t('scheduled_reports_desc')}</p>
                    </div>
                )}
            </div>

            {/* Create Schedule Modal */}
            <Modal isOpen={showCreateSchedule} onClose={() => setShowCreateSchedule(false)} title={t('new_schedule')} size="md">
                <div className="space-y-4">
                    <Input label={t('name')} value={createForm.name} onChange={(e) => setCreateForm(f => ({ ...f, name: e.target.value }))} placeholder="e.g. Weekly Report" required />
                    <Select label={t('type')} options={typeOptions} value={createForm.type} onChange={(e) => setCreateForm(f => ({ ...f, type: e.target.value }))} />
                    <Select label={t('frequency')} options={frequencyOptions} value={createForm.frequency} onChange={(e) => setCreateForm(f => ({ ...f, frequency: e.target.value }))} />
                    <Input label={t('recipients')} value={createForm.recipients} onChange={(e) => setCreateForm(f => ({ ...f, recipients: e.target.value }))} placeholder="email1@example.com, email2@example.com" required />
                    <div className="flex gap-2 justify-end pt-4">
                        <Button variant="secondary" onClick={() => setShowCreateSchedule(false)}>{t('cancel')}</Button>
                        <Button onClick={handleCreate}>{t('create')}</Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}
