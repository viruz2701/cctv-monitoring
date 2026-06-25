// ═══════════════════════════════════════════════════════════════════════
// Dashboard — Main monitoring dashboard with drag-and-drop widgets
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
    Camera,
    Wifi,
    WifiOff,
    Heart,
    HeartCrack,
    VideoOff,
    Ticket,
    AlertTriangle,
    AlertCircle,
    Info,
    ArrowRight,
    TrendingUp,
    Clock,
    CheckCircle,
    Server,
    Pencil,
} from 'lucide-react';
import { StatsCard, Card, CardHeader, CardBody, Badge, Button, Select, SkeletonStatsCard, SkeletonCard, SkeletonChart } from '../components/ui';
import { useTickets, useAlarms, useDevices, useSites } from '../hooks/useApiQuery';
import { useSettings } from '../context/SettingsContext';
import { AlertBanner } from '../components/dashboard/AlertBanner';
import { DragDropDashboard } from '../components/dashboard/DragDropDashboard';
import type { DashboardWidget } from '../components/dashboard/DragDropDashboard';
import { useTranslation } from 'react-i18next';
import type { Device as APIDevice, Ticket as APITicket, Alarm as APIAlarm } from '../services/api';
import type { Device as UIDevice, Ticket as UITicket, Alert } from '../types';
import { useMemo } from 'react';
import {
    LineChart, Line, AreaChart, Area, BarChart, Bar,
    XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts';

const sparklineData = [
    { name: 'Mon', value: 40 }, { name: 'Tue', value: 55 },
    { name: 'Wed', value: 45 }, { name: 'Thu', value: 70 },
    { name: 'Fri', value: 65 }, { name: 'Sat', value: 80 },
    { name: 'Sun', value: 75 },
];

const ticketTrendData = [
    { name: 'Mon', critical: 5, high: 12, medium: 20, low: 30 },
    { name: 'Tue', critical: 3, high: 10, medium: 18, low: 28 },
    { name: 'Wed', critical: 7, high: 15, medium: 22, low: 35 },
    { name: 'Thu', critical: 4, high: 8, medium: 16, low: 25 },
    { name: 'Fri', critical: 6, high: 11, medium: 19, low: 32 },
    { name: 'Sat', critical: 2, high: 6, medium: 14, low: 22 },
    { name: 'Sun', critical: 8, high: 14, medium: 21, low: 38 },
];

// ═══ API→UI mapping helpers (migrated from DevicesSitesContext/TicketsContext/AlertsContext) ═══
function mapAPIDeviceToUI(d: APIDevice): UIDevice {
    return {
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: (d as any).site_id || 'site-default',
        siteName: (d as any).location || 'Unknown',
        type: (d as any).vendor_type === 'camera' ? 'camera' : 'nvr',
        status: (d.status || 'offline').toLowerCase() as UIDevice['status'],
        health: d.status === 'online' ? 'healthy' as const : 'faulty' as const,
        recordingStatus: 'recording' as const,
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: (d as any).vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    };
}

function mapAPITicketToUI(t: APITicket): UITicket {
    return {
        id: t.id,
        title: t.title,
        description: t.description,
        deviceId: t.device_id || '',
        deviceName: '',
        siteName: '',
        priority: (t.priority as UITicket['priority']) || 'medium',
        status: (t.status as UITicket['status']) || 'open',
        assignee: t.assignee || '',
        createdAt: t.created_at,
        updatedAt: t.updated_at,
        comments: (t.comments || []).map((c: any) => ({
            id: c.id,
            ticketId: c.ticket_id,
            userId: c.user_id,
            userName: c.user_name || '',
            content: c.content,
            createdAt: c.created_at,
        })),
    };
}

function mapAlarmToAlert(alarm: APIAlarm): Alert {
    return {
        id: alarm.device_id + '-' + alarm.timestamp,
        deviceId: alarm.device_id,
        deviceName: alarm.device_id,
        type: alarm.priority >= 3 ? 'error' as const : alarm.priority >= 2 ? 'warning' as const : 'info' as const,
        message: alarm.description,
        timestamp: alarm.timestamp,
        status: 'active' as const,
        priority: alarm.priority >= 4 ? 'critical' as const : alarm.priority >= 3 ? 'high' as const : alarm.priority >= 2 ? 'medium' as const : 'low' as const,
        source: alarm.device_id,
        siteName: '',
    };
}

export function Dashboard() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { data: apiTickets } = useTickets();
    const { data: apiAlarms } = useAlarms();
    const { data: apiDevices } = useDevices();
    const { data: apiSites } = useSites();
    const { dashboardConfig, updateDashboardConfig } = useSettings();
    const [pageLoading, setPageLoading] = React.useState(true);
    const [customizeMode, setCustomizeMode] = React.useState(false);

    // Защита: бэкенд может вернуть { data: [...] } вместо чистого массива
    const getArrayData = <T,>(raw: unknown): T[] => {
      if (Array.isArray(raw)) return raw as T[];
      if (raw && typeof raw === 'object' && 'data' in raw) {
        const nested = (raw as Record<string, unknown>).data;
        if (Array.isArray(nested)) return nested as T[];
      }
      return [];
    };
    const apiTicketsData = getArrayData<APITicket>(apiTickets);
    const apiAlarmsData = getArrayData<APIAlarm>(apiAlarms);
    const apiDevicesData = getArrayData<APIDevice>(apiDevices);
    const apiSitesData = getArrayData<import('../services/api').Site>(apiSites);

    const tickets = useMemo(() => apiTicketsData.map(mapAPITicketToUI), [apiTicketsData]);
    const alerts = useMemo(() => apiAlarmsData.map(mapAlarmToAlert), [apiAlarmsData]);
    const devices = useMemo(() => apiDevicesData.map(mapAPIDeviceToUI), [apiDevicesData]);
    const sites = useMemo(() => apiSitesData.map((s: any) => ({
        id: s.id,
        name: s.name || 'Unnamed',
        address: s.address || '',
        city: s.city || '',
        organization: (s as any).organization || '',
        latitude: (s as any).latitude || 0,
        longitude: (s as any).longitude || 0,
        status: (s.status || 'active') as 'active' | 'inactive' | 'maintenance',
        lastSync: (s as any).last_sync || new Date().toISOString(),
    })), [apiSites]);

    React.useEffect(() => {
        const timer = setTimeout(() => setPageLoading(false), 300);
        return () => clearTimeout(timer);
    }, []);

    const [selectedSite, setSelectedSite] = React.useState('all');
    const [selectedDeviceType, setSelectedDeviceType] = React.useState('all');
    const [selectedStatus, setSelectedStatus] = React.useState('all');

    const filteredData = React.useMemo(() => {
        const filteredDevices = devices.filter(d => {
            const matchSite = selectedSite === 'all' || d.siteId === selectedSite;
            const matchType = selectedDeviceType === 'all' || d.type === selectedDeviceType;
            const matchStatus = selectedStatus === 'all' ||
                (selectedStatus === 'offline' && d.status === 'offline') ||
                (selectedStatus === 'warning' && d.status === 'warning') ||
                (selectedStatus === 'online' && d.status === 'online');
            return matchSite && matchType && matchStatus;
        });
        const filteredTickets = tickets.filter(t => {
            const device = devices.find(d => d.id === t.deviceId);
            const matchSite = selectedSite === 'all' || (device && device.siteId === selectedSite);
            const matchType = selectedDeviceType === 'all' || (device && device.type === selectedDeviceType);
            return matchSite && matchType;
        });
        const stats = {
            totalDevices: filteredDevices.length,
            onlineDevices: filteredDevices.filter(d => d.status === 'online').length,
            offlineDevices: filteredDevices.filter(d => d.status === 'offline').length,
            healthyDevices: filteredDevices.filter(d => d.health === 'healthy').length,
            faultyDevices: filteredDevices.filter(d => d.health === 'faulty').length,
            recordingMissing: filteredDevices.filter(d => d.recordingStatus === 'not_recording' || d.status === 'warning').length,
            openTickets: filteredTickets.filter(t => t.status === 'open' || t.status === 'in_progress').length,
            criticalTickets: filteredTickets.filter(t => t.priority === 'critical' && t.status !== 'closed').length,
            resolutionRate: (() => {
                if (filteredTickets.length === 0) return 0;
                const resolvedCount = filteredTickets.filter(t => t.status === 'resolved' || t.status === 'closed').length;
                return Math.round((resolvedCount / filteredTickets.length) * 100);
            })(),
            avgResponseTime: (() => {
                const respondedTickets = filteredTickets.filter(t => t.comments && t.comments.length > 0);
                if (respondedTickets.length === 0) return 0;
                const totalResponseTime = respondedTickets.reduce((acc, ticket) => {
                    const firstResponse = ticket.comments![0].createdAt;
                    const created = ticket.createdAt;
                    return acc + (new Date(firstResponse).getTime() - new Date(created).getTime());
                }, 0);
                const avgMs = totalResponseTime / respondedTickets.length;
                return Number((avgMs / (1000 * 60 * 60)).toFixed(1));
            })(),
        };
        return { devices: filteredDevices, tickets: filteredTickets, stats };
    }, [devices, tickets, selectedSite, selectedDeviceType, selectedStatus]);

    const stats = filteredData.stats;
    const recentAlerts = alerts
        .filter(a => selectedSite === 'all' || devices.find(d => d.id === a.deviceId)?.siteId === selectedSite)
        .slice(0, 5);

    const deviceHealthChartData = [
        { name: t('healthy'), value: stats.healthyDevices, fill: '#22c55e' },
        { name: t('faulty'), value: stats.faultyDevices, fill: '#ef4444' },
        { name: 'Offline', value: stats.offlineDevices, fill: '#f97316' },
    ];

    // ── Widget definitions for DragDropDashboard ─────────────────────

    const dashboardWidgets = React.useMemo<DashboardWidget[]>(() => [
        {
            id: 'statsOverview',
            content: (
                <div key="statsOverview" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-3 overflow-hidden h-full">
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-3">
                        <StatsCard title={t('total_devices')} value={stats.totalDevices} icon={Camera} iconColor="text-blue-600" iconBgColor="bg-blue-50" trend={{ value: 12, label: t('from_last_month'), direction: 'up' }} />
                        <StatsCard title={t('online_devices')} value={stats.onlineDevices} subtitle={`${stats.totalDevices > 0 ? Math.round((stats.onlineDevices / stats.totalDevices) * 100) : 0}% ${t('uptime')}`} icon={Wifi} iconColor="text-emerald-600" iconBgColor="bg-emerald-50" />
                        <StatsCard title={t('offline_devices')} value={stats.offlineDevices} icon={WifiOff} iconColor="text-red-600" iconBgColor="bg-red-50" />
                        <StatsCard title={t('healthy_cameras')} value={stats.healthyDevices} subtitle={`${stats.faultyDevices} ${t('faulty')}`} icon={Heart} iconColor="text-emerald-600" iconBgColor="bg-emerald-50" />
                        <StatsCard title={t('recording_missing')} value={stats.recordingMissing} subtitle={t('today')} icon={VideoOff} iconColor="text-amber-600" iconBgColor="bg-amber-50" />
                    </div>
                </div>
            ),
        },
        {
            id: 'ticketAnalytics',
            content: (
                <div key="ticketAnalytics" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden h-full">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 h-full">
                        <div className="bg-gradient-to-br from-blue-600 to-blue-700 rounded-xl p-5 text-white shadow-lg shadow-blue-900/20">
                            <div className="flex items-center justify-between">
                                <div>
                                    <p className="text-blue-100 text-sm">{t('open_tickets')}</p>
                                    <p className="text-4xl font-bold mt-1 text-white">{stats.openTickets}</p>
                                </div>
                                <div className="p-3 bg-white/10 rounded-lg"><Ticket className="w-8 h-8 text-blue-100" /></div>
                            </div>
                            <div className="mt-4 flex items-center gap-2">
                                <span className="px-2 py-0.5 bg-red-500 text-white text-xs font-medium rounded">{stats.criticalTickets} {t('critical')}</span>
                                <span className="text-blue-200 text-sm">{t('needs_attention')}</span>
                            </div>
                        </div>
                        <div className="bg-white dark:bg-slate-800 rounded-xl p-5 shadow-sm border border-slate-200 dark:border-slate-700/50">
                            <div className="flex items-center gap-3 mb-4">
                                <div className="p-2 bg-emerald-50 dark:bg-emerald-900/30 rounded-lg"><TrendingUp className="w-5 h-5 text-emerald-600 dark:text-emerald-400" /></div>
                                <div><p className="text-sm font-medium text-slate-900 dark:text-white">{t('resolution_rate')}</p><p className="text-xs text-slate-500 dark:text-slate-400">{t('all_time')}</p></div>
                            </div>
                            <p className="text-3xl font-bold text-slate-900 dark:text-white">{stats.resolutionRate}%</p>
                            <div className="mt-2 h-2 bg-slate-100 dark:bg-slate-700 rounded-full overflow-hidden"><div className="h-full bg-emerald-500 rounded-full" style={{ width: `${stats.resolutionRate}%` }} /></div>
                        </div>
                        <div className="bg-white dark:bg-slate-800 rounded-xl p-5 shadow-sm border border-slate-200 dark:border-slate-700/50">
                            <div className="flex items-center gap-3 mb-4">
                                <div className="p-2 bg-blue-50 dark:bg-blue-900/30 rounded-lg"><Clock className="w-5 h-5 text-blue-600 dark:text-blue-400" /></div>
                                <div><p className="text-sm font-medium text-slate-900 dark:text-white">{t('avg_response_time')}</p><p className="text-xs text-slate-500 dark:text-slate-400">{t('first_response')}</p></div>
                            </div>
                            <p className="text-3xl font-bold text-slate-900 dark:text-white">{stats.avgResponseTime}h</p>
                            <p className="text-sm text-emerald-600 dark:text-emerald-400 mt-2">{t('based_on_activity')}</p>
                        </div>
                    </div>
                </div>
            ),
        },
        {
            id: 'deviceHealthChart',
            content: (
                <div key="deviceHealthChart" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden h-full">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">{t('device_health_distribution') || 'Device Health'}</h3>
                    <ResponsiveContainer width="100%" height="80%">
                        <BarChart data={deviceHealthChartData}>
                            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                            <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                            <YAxis tick={{ fontSize: 11 }} />
                            <Tooltip />
                            <Bar dataKey="value" radius={[4, 4, 0, 0]} />
                        </BarChart>
                    </ResponsiveContainer>
                </div>
            ),
        },
        {
            id: 'alertTrendChart',
            content: (
                <div key="alertTrendChart" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden h-full">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">{t('alert_trend') || 'Alert Trend'}</h3>
                    <ResponsiveContainer width="100%" height="80%">
                        <AreaChart data={sparklineData}>
                            <defs>
                                <linearGradient id="alertGradient" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                </linearGradient>
                            </defs>
                            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                            <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                            <YAxis tick={{ fontSize: 11 }} />
                            <Tooltip />
                            <Area type="monotone" dataKey="value" stroke="#3b82f6" fill="url(#alertGradient)" strokeWidth={2} />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>
            ),
        },
        {
            id: 'ticketTrendChart',
            content: (
                <div key="ticketTrendChart" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden h-full">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">{t('ticket_trend') || 'Ticket Trend'}</h3>
                    <ResponsiveContainer width="100%" height="80%">
                        <LineChart data={ticketTrendData}>
                            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                            <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                            <YAxis tick={{ fontSize: 11 }} />
                            <Tooltip />
                            <Line type="monotone" dataKey="critical" stroke="#ef4444" strokeWidth={2} dot={false} />
                            <Line type="monotone" dataKey="high" stroke="#f97316" strokeWidth={2} dot={false} />
                            <Line type="monotone" dataKey="medium" stroke="#3b82f6" strokeWidth={2} dot={false} />
                            <Line type="monotone" dataKey="low" stroke="#22c55e" strokeWidth={2} dot={false} />
                        </LineChart>
                    </ResponsiveContainer>
                </div>
            ),
        },
        {
            id: 'recentAlerts',
            content: (
                <div key="recentAlerts" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden flex flex-col h-full">
                    <div className="flex items-center justify-between mb-3">
                        <h3 className="text-sm font-semibold text-slate-900 dark:text-white">{t('recent_alerts')}</h3>
                        <Link to="/tickets" className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 font-medium">{t('view_all')}</Link>
                    </div>
                    <div className="flex-1 space-y-2 overflow-y-auto min-h-0">
                        {recentAlerts.map((alert) => {
                            const icons = { error: <AlertCircle className="w-4 h-4 text-red-500" />, warning: <AlertTriangle className="w-4 h-4 text-amber-500" />, info: <Info className="w-4 h-4 text-blue-500" /> };
                            const bgColors = { error: 'bg-red-50 border-red-100', warning: 'bg-amber-50 border-amber-100', info: 'bg-blue-50 border-blue-100' };
                            return (
                                <div key={alert.id} className={`flex items-start gap-2 p-2 rounded-lg border ${bgColors[alert.type]} dark:bg-slate-800/50 dark:border-slate-700`}>
                                    {icons[alert.type]}
                                    <div className="flex-1 min-w-0">
                                        <p className="text-xs font-medium text-slate-900 dark:text-white truncate">{alert.message}</p>
                                        <p className="text-xs text-slate-500 dark:text-slate-400">{alert.deviceName}</p>
                                    </div>
                                    <span className="text-xs text-slate-400 whitespace-nowrap">{new Date(alert.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                                </div>
                            );
                        })}
                    </div>
                </div>
            ),
        },
        {
            id: 'latestTickets',
            content: (
                <div key="latestTickets" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden flex flex-col h-full">
                    <div className="flex items-center justify-between mb-3">
                        <h3 className="text-sm font-semibold text-slate-900 dark:text-white">{t('latest_tickets')}</h3>
                        <Link to="/tickets" className="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 font-medium">{t('view_all')}</Link>
                    </div>
                    <div className="flex-1 space-y-2 overflow-y-auto min-h-0">
                        {filteredData.tickets.slice(0, 4).map((ticket) => (
                            <div key={ticket.id} onClick={() => navigate(`/tickets/${ticket.id}`)} className="p-2 bg-slate-50 dark:bg-slate-900/30 border border-transparent dark:border-slate-800 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800/70 cursor-pointer transition-all">
                                <div className="flex items-start justify-between gap-2 mb-1">
                                    <p className="text-xs font-medium text-slate-900 dark:text-white truncate">{ticket.title}</p>
                                    <Badge variant={ticket.priority === 'critical' ? 'danger' : ticket.priority === 'high' ? 'warning' : 'neutral'} size="sm">{ticket.priority}</Badge>
                                </div>
                                <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
                                    <span>{ticket.siteName}</span>
                                    <span>{ticket.id?.slice(0, 8)}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                    <div className="mt-2 pt-2 border-t border-slate-100 dark:border-slate-700">
                        <Link to="/tickets?action=create">
                            <Button variant="outline" fullWidth size="sm" icon={<ArrowRight className="w-4 h-4" />} iconPosition="right">
                                {t('create_ticket')}
                            </Button>
                        </Link>
                    </div>
                </div>
            ),
        },
        {
            id: 'quickActions',
            content: (
                <div key="quickActions" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden h-full">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">{t('quick_actions')}</h3>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                        <Link to="/devices" className="flex flex-col items-center gap-1 p-3 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all">
                            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-full"><Camera className="w-4 h-4 text-blue-600 dark:text-blue-400" /></div>
                            <span className="text-xs font-medium text-slate-700 dark:text-slate-300">{t('view_devices')}</span>
                        </Link>
                        <Link to="/tickets" className="flex flex-col items-center gap-1 p-3 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all">
                            <div className="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-full"><Ticket className="w-4 h-4 text-emerald-600 dark:text-emerald-400" /></div>
                            <span className="text-xs font-medium text-slate-700 dark:text-slate-300">{t('view_tickets')}</span>
                        </Link>
                        <Link to="/reports" className="flex flex-col items-center gap-1 p-3 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all">
                            <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-full"><TrendingUp className="w-4 h-4 text-purple-600 dark:text-purple-400" /></div>
                            <span className="text-xs font-medium text-slate-700 dark:text-slate-300">{t('run_report')}</span>
                        </Link>
                        <Link to="/alerts" className="flex flex-col items-center gap-1 p-3 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all">
                            <div className="p-2 bg-amber-100 dark:bg-amber-900/30 rounded-full"><HeartCrack className="w-4 h-4 text-amber-600 dark:text-amber-400" /></div>
                            <span className="text-xs font-medium text-slate-700 dark:text-slate-300">{t('system_alerts')}</span>
                        </Link>
                    </div>
                </div>
            ),
        },
    ], [t, stats, recentAlerts, filteredData, navigate]);

    // ── Loading State ────────────────────────────────────────────────

    if (pageLoading) {
        return (
            <div className="space-y-6">
                <div className="flex flex-col sm:flex-row gap-4 p-4 bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700">
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 flex-1">
                        <div className="h-10 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
                        <div className="h-10 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
                        <div className="h-10 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
                    </div>
                    <div className="flex items-end gap-2">
                        <div className="h-10 w-24 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
                        <div className="h-10 w-24 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
                    </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-3">
                    <SkeletonStatsCard />
                    <SkeletonStatsCard />
                    <SkeletonStatsCard />
                    <SkeletonStatsCard />
                    <SkeletonStatsCard />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <SkeletonCard />
                    <SkeletonCard />
                    <SkeletonCard />
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <SkeletonChart />
                    <SkeletonChart />
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <SkeletonChart />
                    <SkeletonCard />
                </div>
            </div>
        );
    }

    // ── Main Render ──────────────────────────────────────────────────

    return (
        <div className="space-y-6">
            <AlertBanner />

            {/* Filter Bar */}
            <div className="flex flex-col sm:flex-row gap-4 p-4 bg-white dark:bg-slate-800 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 flex-1">
                    <Select
                        label={t('site')}
                        value={selectedSite}
                        onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setSelectedSite(e.target.value)}
                        options={[
                            { value: 'all', label: t('all_sites') },
                            ...sites.map(s => ({ value: s.id, label: s.name }))
                        ]}
                    />
                    <Select
                        label={t('device_type')}
                        value={selectedDeviceType}
                        onChange={(e) => setSelectedDeviceType(e.target.value)}
                        options={[
                            { value: 'all', label: t('all_types') },
                            { value: 'camera', label: t('camera') },
                            { value: 'nvr', label: 'NVR' },
                            { value: 'switch', label: t('switch') },
                        ]}
                    />
                    <Select
                        label={t('status')}
                        value={selectedStatus}
                        onChange={(e) => setSelectedStatus(e.target.value)}
                        options={[
                            { value: 'all', label: t('all_statuses') },
                            { value: 'online', label: t('online') },
                            { value: 'offline', label: t('offline') },
                            { value: 'warning', label: t('warning') },
                        ]}
                    />
                </div>
                <div className="flex items-end gap-2">
                    <Button
                        variant={customizeMode ? 'primary' : 'outline'}
                        onClick={() => setCustomizeMode((prev) => !prev)}
                        icon={<Pencil className="w-4 h-4" />}
                    >
                        {customizeMode ? t('done') || 'Done' : t('customize') || 'Customize'}
                    </Button>
                    <Button variant="outline" onClick={() => { setSelectedSite('all'); setSelectedDeviceType('all'); setSelectedStatus('all'); }}>
                        {t('clear_filters')}
                    </Button>
                </div>
            </div>

            {/* Drag-and-Drop Dashboard */}
            <DragDropDashboard
                widgets={dashboardWidgets}
                customizeMode={customizeMode}
            />
        </div>
    );
}
