import React, { useState, useMemo } from 'react';
import { getArrayData } from '../utils/helpers';
import { useDevices, useSites, useTickets } from '../hooks/useApiQuery';
import type { Ticket as APITicket, Device as APIDevice, TicketComment } from '../services/api';
import {
    FileText,
    Download,
    Calendar,
    Clock,
    Play,
    Settings,
    TrendingUp,
    Shield,
    Video,
    Ticket,
    AlertTriangle,
    CheckCircle,
    Server
} from 'lucide-react';
import { Card, CardBody, Button, Select } from '../components/ui';
import { ManualDownloadTab, ScheduledReportsTab, ReportHistoryTab } from '../components/reports';
import { useTranslation } from 'react-i18next';

type TabType = 'manual' | 'scheduled' | 'history';

// ═══ API→UI mapping helpers ═══
function mapAPIDeviceToUI(d: APIDevice): import('../types').Device {
    return {
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: (d as any).site_id || 'site-default',
        siteName: (d as any).location || 'Unknown',
        type: ((d as any).vendor_type === 'camera' ? 'camera' : 'nvr') as 'camera' | 'nvr' | 'dvr' | 'switch',
        status: (d.status || 'offline').toLowerCase() as 'online' | 'offline' | 'warning',
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording' as const,
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: (d as any).vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    };
}

function mapAPITicketToUI(t: APITicket): import('../types').Ticket {
    return {
        id: t.id,
        title: t.title,
        description: t.description,
        deviceId: t.device_id || '',
        deviceName: '',
        siteName: '',
        priority: (t.priority as import('../types').Ticket['priority']) || 'medium',
        status: (t.status as import('../types').Ticket['status']) || 'open',
        assignee: t.assignee || '',
        createdAt: t.created_at,
        updatedAt: t.updated_at,
        comments: (t.comments || []).map((c: TicketComment) => ({
            id: c.id,
            ticketId: c.ticket_id,
            userId: c.user_id || '',
            userName: c.user_name || '',
            content: c.content,
            createdAt: c.created_at,
        })),
    };
}

export function Reports() {
    const { t } = useTranslation();
    const [activeTab, setActiveTab] = useState<TabType>('manual');

    const { data: apiDevices } = useDevices();
    const { data: apiTickets } = useTickets();
    const apiDevicesData = getArrayData<APIDevice>(apiDevices);
    const apiTicketsData = getArrayData<APITicket>(apiTickets);

    const devices = useMemo(() => apiDevicesData.map(mapAPIDeviceToUI), [apiDevicesData]);
    const tickets = useMemo(() => apiTicketsData.map(mapAPITicketToUI), [apiTicketsData]);

    const dashboardStats = useMemo(() => ({
        totalDevices: devices.length,
        onlineDevices: devices.filter(d => d.status === 'online').length,
        openTickets: tickets.filter(t => t.status === 'open' || t.status === 'in_progress').length,
        criticalTickets: tickets.filter(t => t.priority === 'critical' && t.status !== 'closed').length,
    }), [devices, tickets]);

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row gap-4 justify-between items-start md:items-center">
                <div>
                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white">{t('reports')}</h2>
                    <p className="text-sm text-slate-500 dark:text-slate-300 mt-1">
                        {t('reports_subtitle') || "Generate and download system reports"}
                    </p>
                </div>
            </div>

            {/* Quick Stats */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card>
                    <CardBody>
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-blue-50 dark:bg-blue-900/30 rounded-lg">
                                <Server className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                            </div>
                            <div>
                                <p className="text-2xl font-bold text-slate-900 dark:text-white">{dashboardStats.totalDevices}</p>
                                <p className="text-sm text-slate-500 dark:text-slate-300">{t('total_devices')}</p>
                            </div>
                        </div>
                    </CardBody>
                </Card>
                <Card>
                    <CardBody>
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-red-50 dark:bg-red-900/30 rounded-lg">
                                <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400" />
                            </div>
                            <div>
                                <p className="text-2xl font-bold text-slate-900 dark:text-white">{dashboardStats.criticalTickets}</p>
                                <p className="text-sm text-slate-500 dark:text-slate-300">{t('critical_issues')}</p>
                            </div>
                        </div>
                    </CardBody>
                </Card>
                <Card>
                    <CardBody>
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-amber-50 dark:bg-amber-900/30 rounded-lg">
                                <Ticket className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                            </div>
                            <div>
                                <p className="text-2xl font-bold text-slate-900 dark:text-white">{dashboardStats.openTickets}</p>
                                <p className="text-sm text-slate-500 dark:text-slate-300">{t('open_tickets')}</p>
                            </div>
                        </div>
                    </CardBody>
                </Card>
                <Card>
                    <CardBody>
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-emerald-50 dark:bg-emerald-900/30 rounded-lg">
                                <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                            </div>
                            <div>
                                <p className="text-2xl font-bold text-slate-900 dark:text-white">{dashboardStats.onlineDevices}</p>
                                <p className="text-sm text-slate-500 dark:text-slate-300">{t('online_devices')}</p>
                            </div>
                        </div>
                    </CardBody>
                </Card>
            </div>

            {/* Navigation Tabs */}
            <div className="border-b border-slate-200 dark:border-slate-700">
                <div className="flex flex-wrap gap-4 md:gap-8">
                    <button
                        className={`pb-4 text-sm font-medium transition-colors relative ${activeTab === 'manual'
                                ? 'text-blue-600 dark:text-blue-400'
                                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                            }`}
                        onClick={() => setActiveTab('manual')}
                    >
                        {t('manual_download')}
                        {activeTab === 'manual' && <div className="absolute bottom-0 left-0 w-full h-0.5 bg-blue-600 dark:bg-blue-400 rounded-t-full" />}
                    </button>
                    <button
                        className={`pb-4 text-sm font-medium transition-colors relative ${activeTab === 'scheduled'
                                ? 'text-blue-600 dark:text-blue-400'
                                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                            }`}
                        onClick={() => setActiveTab('scheduled')}
                    >
                        {t('scheduled_reports')}
                        {activeTab === 'scheduled' && <div className="absolute bottom-0 left-0 w-full h-0.5 bg-blue-600 dark:bg-blue-400 rounded-t-full" />}
                    </button>
                    <button
                        className={`pb-4 text-sm font-medium transition-colors relative ${activeTab === 'history'
                                ? 'text-blue-600 dark:text-blue-400'
                                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                            }`}
                        onClick={() => setActiveTab('history')}
                    >
                        {t('report_history')}
                        {activeTab === 'history' && <div className="absolute bottom-0 left-0 w-full h-0.5 bg-blue-600 dark:bg-blue-400 rounded-t-full" />}
                    </button>
                </div>
            </div>

            <div className="mt-6">
                {activeTab === 'manual' && <ManualDownloadTab />}
                {activeTab === 'scheduled' && <ScheduledReportsTab />}
                {activeTab === 'history' && <ReportHistoryTab />}
            </div>
        </div>
    );
}