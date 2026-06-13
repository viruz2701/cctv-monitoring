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
    Activity,
    CheckCircle,
    Server,
    HardDrive,
    Shield,
    Settings
} from 'lucide-react';
import { StatsCard, Card, CardHeader, CardBody, Badge, Button, Select } from '../components/ui';
import { useTickets, useAlerts, useDevicesSites, useSettings } from '../context/DataContext';
import { AlertBanner } from '../components/dashboard/AlertBanner';
import { useTranslation } from 'react-i18next';

export function Dashboard() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { tickets } = useTickets();
    const { alerts } = useAlerts();
    const { devices, sites } = useDevicesSites();
    const { dashboardConfig, updateDashboardConfig } = useSettings();
    const [isConfigOpen, setIsConfigOpen] = React.useState(false);

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
                            <div className="absolute right-0 top-full mt-2 w-64 bg-white dark:bg-slate-800 rounded-lg shadow-xl border border-slate-200 dark:border-slate-700 z-50 p-4">
                                <h3 className="font-medium text-slate-900 dark:text-white mb-3">{t('dashboard_layout')}</h3>
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
                                            <input type="checkbox" className="accent-blue-600 w-4 h-4 rounded" checked={dashboardConfig[item.key as keyof typeof dashboardConfig]} onChange={(e) => updateDashboardConfig({ [item.key]: e.target.checked })} />
                                        </label>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>
                    <Button variant="outline" onClick={() => { setSelectedSite('all'); setSelectedDeviceType('all'); setSelectedStatus('all'); }}>
                        {t('clear_filters')}
                    </Button>
                </div>
            </div>

            {/* Stats Grid */}
            {dashboardConfig.showStatsRow && (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-3 md:gap-4">
                    <StatsCard title={t('total_devices')} value={stats.totalDevices} icon={Camera} iconColor="text-blue-600" iconBgColor="bg-blue-50" trend={{ value: 12, label: t('from_last_month'), direction: 'up' }} />
                    <StatsCard title={t('online_devices')} value={stats.onlineDevices} subtitle={`${stats.totalDevices > 0 ? Math.round((stats.onlineDevices / stats.totalDevices) * 100) : 0}% ${t('uptime')}`} icon={Wifi} iconColor="text-emerald-600" iconBgColor="bg-emerald-50" />
                    <StatsCard title={t('offline_devices')} value={stats.offlineDevices} icon={WifiOff} iconColor="text-red-600" iconBgColor="bg-red-50" />
                    <StatsCard title={t('healthy_cameras')} value={stats.healthyDevices} subtitle={`${stats.faultyDevices} ${t('faulty')}`} icon={Heart} iconColor="text-emerald-600" iconBgColor="bg-emerald-50" />
                    <StatsCard title={t('recording_missing')} value={stats.recordingMissing} subtitle={t('today')} icon={VideoOff} iconColor="text-amber-600" iconBgColor="bg-amber-50" />
                </div>
            )}

            {/* Tickets Summary */}
            {dashboardConfig.showTicketStats && (
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
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
            )}

            {/* Main Content Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {dashboardConfig.showRecentAlerts && (
                    <Card className="lg:col-span-2">
                        <CardHeader action={<Link to="/tickets" className="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium">{t('view_all')}</Link>}>{t('recent_alerts')}</CardHeader>
                        <CardBody>
                            <div className="space-y-3">
                                {recentAlerts.map((alert) => {
                                    const icons = { error: <AlertCircle className="w-5 h-5 text-red-500" />, warning: <AlertTriangle className="w-5 h-5 text-amber-500" />, info: <Info className="w-5 h-5 text-blue-500" /> };
                                    const bgColors = { error: 'bg-red-50 border-red-100', warning: 'bg-amber-50 border-amber-100', info: 'bg-blue-50 border-blue-100' };
                                    return (
                                        <div key={alert.id} className={`flex items-start gap-3 p-3 rounded-lg border ${bgColors[alert.type]} dark:bg-slate-800/50 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors`}>
                                            {icons[alert.type]}
                                            <div className="flex-1 min-w-0"><p className="text-sm font-medium text-slate-900 dark:text-white">{alert.message}</p><p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{alert.deviceName} • {alert.siteName}</p></div>
                                            <span className="text-xs text-slate-400 whitespace-nowrap">{new Date(alert.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                                        </div>
                                    );
                                })}
                            </div>
                        </CardBody>
                    </Card>
                )}
                {dashboardConfig.showLatestTickets && (
                    <Card>
                        <CardHeader action={<Link to="/tickets" className="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 font-medium">{t('view_all')}</Link>}>{t('latest_tickets')}</CardHeader>
                        <CardBody>
                            <div className="space-y-3">
                                {filteredData.tickets.slice(0, 4).map((ticket) => (
                                    <div key={ticket.id} onClick={() => navigate(`/tickets/${ticket.id}`)} className="p-3 bg-slate-50 dark:bg-slate-900/30 border border-transparent dark:border-slate-800 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800/70 cursor-pointer transition-all">
                                        <div className="flex items-start justify-between gap-2 mb-2">
                                            <p className="text-sm font-medium text-slate-900 dark:text-white line-clamp-1">{ticket.title}</p>
                                            <Badge variant={ticket.priority === 'critical' ? 'danger' : ticket.priority === 'high' ? 'warning' : 'neutral'} size="sm">{ticket.priority}</Badge>
                                        </div>
                                        <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400"><span>{ticket.siteName}</span><span>{ticket.id}</span></div>
                                    </div>
                                ))}
                            </div>
                            <div className="mt-4">
                                <Link to="/tickets?action=create"><Button variant="outline" fullWidth size="sm" icon={<ArrowRight className="w-4 h-4" />} iconPosition="right">{t('create_ticket')}</Button></Link>
                            </div>
                        </CardBody>
                    </Card>
                )}
            </div>

            {dashboardConfig.showQuickActions && (
                <Card>
                    <CardHeader>{t('quick_actions')}</CardHeader>
                    <CardBody>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                            <Link to="/devices" className="flex flex-col items-center gap-2 p-4 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all hover:shadow-sm">
                                <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-full"><Camera className="w-5 h-5 text-blue-600 dark:text-blue-400" /></div>
                                <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{t('view_devices')}</span>
                            </Link>
                            <Link to="/tickets" className="flex flex-col items-center gap-2 p-4 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all hover:shadow-sm">
                                <div className="p-3 bg-emerald-100 dark:bg-emerald-900/30 rounded-full"><Ticket className="w-5 h-5 text-emerald-600 dark:text-emerald-400" /></div>
                                <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{t('view_tickets')}</span>
                            </Link>
                            <Link to="/reports" className="flex flex-col items-center gap-2 p-4 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all hover:shadow-sm">
                                <div className="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-full"><TrendingUp className="w-5 h-5 text-purple-600 dark:text-purple-400" /></div>
                                <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{t('run_report')}</span>
                            </Link>
                            <Link to="/alerts" className="flex flex-col items-center gap-2 p-4 bg-slate-50 dark:bg-slate-900/30 border border-slate-200/50 dark:border-slate-700 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-all hover:shadow-sm">
                                <div className="p-3 bg-amber-100 dark:bg-amber-900/30 rounded-full"><HeartCrack className="w-5 h-5 text-amber-600 dark:text-amber-400" /></div>
                                <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{t('system_alerts')}</span>
                            </Link>
                        </div>
                    </CardBody>
                </Card>
            )}
        </div>
    );
}