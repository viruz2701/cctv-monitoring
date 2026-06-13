import React from 'react';
import { Card, CardHeader, CardBody, Button, Badge } from '../ui';
import { Plus, Clock, Users, Mail, Settings, Play, Trash2 } from 'lucide-react';
import { useToast } from '../ui';
import { useReports } from '../../context/DataContext';
import { useTranslation } from 'react-i18next';

const mockScheduledReports = [
    { id: 'sr-001', name: 'Weekly Management Overview', type: 'Consolidated Health', frequency: 'Weekly (Monday)', recipients: ['management@company.com', 'admin@company.com'], status: 'active', nextRun: '2026-02-23T08:00:00Z' },
    { id: 'sr-002', name: 'Daily Region 1 Camera Status', type: 'Camera Health', frequency: 'Daily', recipients: ['region1-tech@company.com'], status: 'active', nextRun: '2026-02-21T06:00:00Z' },
    { id: 'sr-003', name: 'Monthly Storage Compliance', type: 'Recording Availability', frequency: 'Monthly (1st)', recipients: ['compliance@company.com'], status: 'paused', nextRun: '2026-03-01T00:00:00Z' }
];

export function ScheduledReportsTab() {
    const { t } = useTranslation();
    const toast = useToast();

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h3 className="text-lg font-medium text-slate-900 dark:text-white">{t('scheduled_reports')}</h3>
                    <p className="text-sm text-slate-500 dark:text-slate-400">{t('scheduled_reports_desc')}</p>
                </div>
                <Button icon={<Plus className="w-4 h-4" />} onClick={() => toast.info(t('create_schedule_soon'))}>{t('new_schedule')}</Button>
            </div>
            <div className="grid grid-cols-1 gap-4">
                {mockScheduledReports.map((report) => (
                    <Card key={report.id} variant="bordered" className="hover:border-blue-500/50 transition-colors">
                        <CardBody className="p-5">
                            <div className="flex flex-col md:flex-row gap-4 justify-between items-start md:items-center">
                                <div className="space-y-1">
                                    <div className="flex items-center gap-3">
                                        <h4 className="font-semibold text-slate-900 dark:text-white text-base">{report.name}</h4>
                                        <Badge variant={report.status === 'active' ? 'success' : 'neutral'} size="sm">{report.status === 'active' ? t('active') : t('paused')}</Badge>
                                    </div>
                                    <div className="flex flex-wrap items-center gap-4 text-sm text-slate-500 dark:text-slate-400">
                                        <span className="flex items-center gap-1.5"><Settings className="w-4 h-4" />{report.type}</span>
                                        <span className="flex items-center gap-1.5"><Clock className="w-4 h-4" />{report.frequency}</span>
                                        <span className="flex items-center gap-1.5"><Users className="w-4 h-4" />{report.recipients.length} {t('recipients')}</span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-6 w-full md:w-auto mt-4 md:mt-0 pt-4 md:pt-0 border-t md:border-t-0 border-slate-200 dark:border-slate-700/50">
                                    <div className="hidden md:flex md:flex-col md:items-end">
                                        <span className="text-xs text-slate-500 dark:text-slate-400">{t('next_run')}</span>
                                        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{new Date(report.nextRun).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
                                    </div>
                                    <div className="flex gap-2 w-full md:w-auto justify-end">
                                        <Button variant="ghost" size="sm" onClick={() => toast.info(t('run_now_soon'))}>{t('run_now')}</Button>
                                        <Button variant="ghost" size="sm" onClick={() => toast.info(t('edit_schedule_soon'))}>{t('edit')}</Button>
                                        <Button variant="ghost" size="sm" className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20" onClick={() => toast.info(t('delete_schedule_soon'))}>{t('delete')}</Button>
                                    </div>
                                </div>
                            </div>
                        </CardBody>
                    </Card>
                ))}
            </div>
        </div>
    );
}