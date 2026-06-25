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
    Settings,
    GripVertical,
} from 'lucide-react';
import { StatsCard, Card, CardHeader, CardBody, Badge, Button, Select, SkeletonStatsCard, SkeletonCard, SkeletonChart } from '../components/ui';
import { useTickets, useAlerts, useDevicesSites, useSettings } from '../context/DataContext';
import { AlertBanner } from '../components/dashboard/AlertBanner';
import { useTranslation } from 'react-i18next';
import GridLayout from 'react-grid-layout';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';
import {
    LineChart, Line, AreaChart, Area, BarChart, Bar,
    XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts';
import type { Layout } from 'react-grid-layout';

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

const defaultLayout: Layout = [
    { i: 'statsRow', x: 0, y: 0, w: 12, h: 1, minH: 1 },
    { i: 'ticketStats', x: 0, y: 1, w: 12, h: 1, minH: 1 },
    { i: 'deviceHealthChart', x: 0, y: 2, w: 6, h: 2, minH: 1 },
    { i: 'alertTrendChart', x: 6, y: 2, w: 6, h: 2, minH: 1 },
    { i: 'ticketTrendChart', x: 0, y: 4, w: 6, h: 2, minH: 1 },
    { i: 'recentAlerts', x: 6, y: 4, w: 6, h: 2, minH: 1 },
    { i: 'latestTickets', x: 0, y: 6, w: 4, h: 2, minH: 1 },
    { i: 'quickActions', x: 4, y: 6, w: 8, h: 1, minH: 1 },
];

export function Dashboard() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { tickets } = useTickets();
    const { alerts } = useAlerts();
    const { devices, sites } = useDevicesSites();
    const { dashboardConfig, updateDashboardConfig } = useSettings();
    const [isConfigOpen, setIsConfigOpen] = React.useState(false);
    const [pageLoading, setPageLoading] = React.useState(true);

    React.useEffect(() => {
        const timer = setTimeout(() => setPageLoading(false), 300);
        return () => clearTimeout(timer);
    }, []);

    const [selectedSite, setSelectedSite] = React.useState('all');
    const [selectedDeviceType, setSelectedDeviceType] = React.useState('all');
    const [selectedStatus, setSelectedStatus] = React.useState('all');

    // Grid layout state
    const [layout, setLayout] = React.useState<Layout>(() => {
        const saved = localStorage.getItem('dashboardGridLayout');
        if (saved) {
            try {
                const parsed = JSON.parse(saved);
                if (Array.isArray(parsed) && parsed.length > 0) return parsed;
            } catch { /* ignore */ }
        }
        return defaultLayout;
    });

    const onLayoutChange = React.useCallback((newLayout: Layout) => {
        setLayout(newLayout);
        localStorage.setItem('dashboardGridLayout', JSON.stringify(newLayout));
    }, []);

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

    const renderGridItem = (key: string) => {
        // Check visibility from config
        const visible = (() => {
            switch (key) {
                case 'statsRow': return dashboardConfig.showStatsRow;
                case 'ticketStats': return dashboardConfig.showTicketStats;
                case 'recentAlerts': return dashboardConfig.showRecentAlerts;
                case 'latestTickets': return dashboardConfig.showLatestTickets;
                case 'quickActions': return dashboardConfig.showQuickActions;
                default: return true;
            }
        })();

        if (!visible) return <div key={key} className="hidden" />;

        switch (key) {
            case 'statsRow':
                return (
                    <div key="statsRow" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-3 overflow-hidden h-full">
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-3">
                            <StatsCard title={t('total_devices')} value={stats.totalDevices} icon={Camera} iconColor="text-blue-600" iconBgColor="bg-blue-50" trend={{ value: 12, label: t('from_last_month'), direction: 'up' }} />
                            <StatsCard title={t('online_devices')} value={stats.onlineDevices} subtitle={`${stats.totalDevices > 0 ? Math.round((stats.onlineDevices / stats.totalDevices) * 100) : 0}% ${t('uptime')}`} icon={Wifi} iconColor="text-emerald-600" iconBgColor="bg-emerald-50" />
                            <StatsCard title={t('offline_devices')} value={stats.offlineDevices} icon={WifiOff} iconColor="text-red-600" iconBgColor="bg-red-50" />
                            <StatsCard title={t('healthy_cameras')} value={stats.healthyDevices} subtitle={`${stats.faultyDevices} ${t('faulty')}`} icon={Heart} iconColor="text-emerald-600" iconBgColor="bg-emerald-50" />
                            <StatsCard title={t('recording_missing')} value={stats.recordingMissing} subtitle={t('today')} icon={VideoOff} iconColor="text-amber-600" iconBgColor="bg-amber-50" />
                        </div>
                    </div>
                );

            case 'ticketStats':
                return (
                    <div key="ticketStats" className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 overflow-hidden h-full">
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
                );

            case 'deviceHealthChart':
                return (
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
                );

            case 'alertTrendChart':
                return (
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
                );

            case 'ticketTrendChart':
                return (
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
                );

            case 'recentAlerts':
                return (
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
                );

            case 'latestTickets':
                return (
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
                );

            case 'quickActions':
                return (
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
                );

            default:
                return null;
        }
    };

    const visibleKeys = [
        'statsRow', 'ticketStats', 'deviceHealthChart', 'alertTrendChart',
        'ticketTrendChart', 'recentAlerts', 'latestTickets', 'quickActions'
    ];

    if (pageLoading) {
        return (
            <div className="space-y-6">
                {/* Loading Skeleton */}
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
                    <div className="relative">
                        <Button variant="outline" onClick={() => setIsConfigOpen(!isConfigOpen)} icon={<Settings className="w-4 h-4" />}>
                            {t('customize_layout')}
                        </Button>
                        {isConfigOpen && (
                            <div className="absolute right-0 top-full mt-2 w-72 bg-white dark:bg-slate-800 rounded-lg shadow-xl border border-slate-200 dark:border-slate-700 z-50 p-4 max-h-96 overflow-y-auto">
                                <h3 className="font-medium text-slate-900 dark:text-white mb-3">{t('dashboard_layout')}</h3>
                                <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">{t('drag_to_reorder')}</p>
                                <div className="space-y-3">
                                    {[
                                        { key: 'showStatsRow', label: t('show_stats') },
                                        { key: 'showTicketStats', label: t('show_ticket_stats') },
                                        { key: 'showRecentAlerts', label: t('show_recent_alerts') },
                                        { key: 'showLatestTickets', label: t('show_latest_tickets') },
                                        { key: 'showQuickActions', label: t('show_quick_actions') },
                                    ].map(item => (
                                        <label key={item.key} className="flex items-center justify-between cursor-pointer">
                                            <span className="text-sm text-slate-600 dark:text-slate-300">{item.label}</span>
                                            <input
                                                type="checkbox"
                                                className="accent-blue-600 w-4 h-4 rounded"
                                                checked={dashboardConfig[item.key as keyof typeof dashboardConfig] !== false}
                                                onChange={(e) => updateDashboardConfig({ [item.key]: e.target.checked })}
                                            />
                                        </label>
                                    ))}
                                </div>
                                <div className="mt-3 pt-3 border-t border-slate-200 dark:border-slate-700">
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        fullWidth
                                        onClick={() => {
                                            setLayout(defaultLayout);
                                            localStorage.setItem('dashboardGridLayout', JSON.stringify(defaultLayout));
                                        }}
                                    >
                                        {t('reset_layout')}
                                    </Button>
                                </div>
                            </div>
                        )}
                    </div>
                    <Button variant="outline" onClick={() => { setSelectedSite('all'); setSelectedDeviceType('all'); setSelectedStatus('all'); }}>
                        {t('clear_filters')}
                    </Button>
                </div>
            </div>

            {/* Draggable Grid Layout (v2 API) */}
            <div style={{ position: 'relative' }}>
                <GridLayout
                    layout={layout}
                    width={1200}
                    onLayoutChange={onLayoutChange}
                    autoSize={true}
                    className="layout"
                    gridConfig={{
                        cols: 12,
                        rowHeight: 150,
                        margin: [12, 12],
                        containerPadding: [0, 0],
                        maxRows: undefined,
                    }}
                    dragConfig={{
                        handle: '.drag-handle',
                    }}
                    resizeConfig={{}}
                >
                    {visibleKeys.map(key => (
                        <div key={key} className="relative group" style={{ height: '100%' }}>
                            <div className="drag-handle absolute top-2 left-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity cursor-grab active:cursor-grabbing p-1 rounded bg-white/80 dark:bg-slate-700/80 shadow-sm border border-slate-200 dark:border-slate-600">
                                <GripVertical className="w-3.5 h-3.5 text-slate-400" />
                            </div>
                            {renderGridItem(key)}
                        </div>
                    ))}
                </GridLayout>
            </div>
        </div>
    );
}
