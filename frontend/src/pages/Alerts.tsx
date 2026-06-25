import React, { useState, useMemo, useCallback } from 'react';
import { useAlarms, useAcknowledgeAlarm, useResolveAlarm } from '../hooks/useApiQuery';
import {
    AlertTriangle,
    AlertCircle,
    Info,
    Filter,
    Clock,
    Trash2,
    Bell,
    Check,
    CheckCircle
} from 'lucide-react';
import { Card, CardBody, DataGrid, Badge, Button, Select, SearchInput, ConfirmModal, SkeletonTable, SkeletonCard } from '../components/ui';
import { SavedViews } from '../components/ui/SavedViews';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';
import type { Alert } from '../types';

export function Alerts() {
    const { t } = useTranslation();
    const { data: apiAlarms = [] } = useAlarms();
    const acknowledgeAlarm = useAcknowledgeAlarm();
    const resolveAlarm = useResolveAlarm();

    // API Alarm → UI Alert mapping (migrated from AlertsContext)
    const alerts = useMemo(() => apiAlarms.map(a => ({
        id: a.device_id + '-' + a.timestamp,
        deviceId: a.device_id,
        deviceName: a.device_id,
        type: (a.priority >= 3 ? 'error' : a.priority >= 2 ? 'warning' : 'info') as 'error' | 'warning' | 'info',
        message: a.description,
        timestamp: a.timestamp,
        status: 'active' as const,
        priority: (a.priority >= 4 ? 'critical' : a.priority >= 3 ? 'high' : a.priority >= 2 ? 'medium' : 'low') as 'critical' | 'high' | 'medium' | 'low',
        source: a.device_id,
        siteName: '',
        acknowledgedBy: undefined as string | undefined,
        resolvedBy: undefined as string | undefined,
        resolvedAt: undefined as string | undefined,
    })), [apiAlarms]);

    const isLoading = alerts.length === 0;

    const updateAlertStatus = (id: string, status: string) => {
        if (status === 'acknowledged') acknowledgeAlarm.mutate(id);
        else if (status === 'resolved') resolveAlarm.mutate(id);
    };
    const deleteAlert = () => {};
    const [searchTerm, setSearchTerm] = useState('');
    const [typeFilter, setTypeFilter] = useState<string>('all');
    const [statusFilter, setStatusFilter] = useState<string>('all');
    const [showFilters, setShowFilters] = useState(false);
    const [sortColumn, setSortColumn] = useState<string>('timestamp');
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });

    // UX-14.3.2: Apply saved view
    const handleApplyView = useCallback((view: import('../store/filterStore').SavedView) => {
        const filters = view.filters;
        if (filters.searchTerm) setSearchTerm(filters.searchTerm);
        if (filters.typeFilter) setTypeFilter(filters.typeFilter);
        if (filters.statusFilter) setStatusFilter(filters.statusFilter);
        if (view.sort.column) {
            setSortColumn(view.sort.column);
            setSortDirection(view.sort.direction);
        }
    }, []);

    const handleAction = (alertId: string, action: 'acknowledge' | 'resolve') => {
        const newStatus = action === 'acknowledge' ? 'acknowledged' : 'resolved';
        updateAlertStatus(alertId, newStatus);
    };

    const filteredAlerts = useMemo(() => {
        let result = alerts.filter(alert => {
            const matchesSearch =
                alert.message.toLowerCase().includes(searchTerm.toLowerCase()) ||
                alert.deviceName.toLowerCase().includes(searchTerm.toLowerCase()) ||
                alert.siteName.toLowerCase().includes(searchTerm.toLowerCase());
            const matchesType = typeFilter === 'all' || alert.type === typeFilter;
            const matchesStatus = statusFilter === 'all' || alert.status === statusFilter;
            return matchesSearch && matchesType && matchesStatus;
        });
        result.sort((a, b) => {
            if (sortColumn === 'timestamp') {
                const aTime = new Date(a.timestamp).getTime();
                const bTime = new Date(b.timestamp).getTime();
                return sortDirection === 'asc' ? aTime - bTime : bTime - aTime;
            }
            const aVal = String(a[sortColumn as keyof Alert] ?? '');
            const bVal = String(b[sortColumn as keyof Alert] ?? '');
            const cmp = aVal.localeCompare(bVal);
            return sortDirection === 'asc' ? cmp : -cmp;
        });
        return result;
    }, [alerts, searchTerm, typeFilter, statusFilter, sortColumn, sortDirection]);

    const handleSort = (column: string) => {
        if (sortColumn === column) {
            setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'));
        } else {
            setSortColumn(column);
            setSortDirection('asc');
        }
    };

    const statusCounts = useMemo(() => {
        const counts = { total: filteredAlerts.length, active: 0, acknowledged: 0, resolved: 0 };
        filteredAlerts.forEach((a) => {
            if (a.status === 'active') counts.active++;
            else if (a.status === 'acknowledged') counts.acknowledged++;
            else if (a.status === 'resolved') counts.resolved++;
        });
        return counts;
    }, [filteredAlerts]);

    const columns = [
        {
            key: 'type' as keyof Alert,
            header: t('severity'),
            sortable: true,
            render: (alert: Alert) => {
                const config = {
                    error: { icon: AlertCircle, color: 'text-red-600', bg: 'bg-red-100 dark:bg-red-900/30', label: t('error') },
                    warning: { icon: AlertTriangle, color: 'text-amber-600', bg: 'bg-amber-100 dark:bg-amber-900/30', label: t('warning') },
                    info: { icon: Info, color: 'text-blue-600', bg: 'bg-blue-100 dark:bg-blue-900/30', label: t('info') }
                }[alert.type];
                const Icon = config.icon;
                return (
                    <div className="flex items-center gap-2 text-slate-700 dark:text-slate-300">
                        <div className={`p-1.5 rounded-lg inline-flex items-center justify-center ${config.bg}`}>
                            <Icon className={`w-4 h-4 ${config.color}`} />
                        </div>
                        <span className="text-sm font-medium">{config.label}</span>
                    </div>
                );
            }
        },
        {
            key: 'message' as keyof Alert,
            header: t('message'),
            sortable: true,
            render: (alert: Alert) => (
                <div>
                    <div className="font-medium text-slate-900 dark:text-slate-100">{alert.message}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                        {alert.deviceName} • {alert.siteName}
                    </div>
                </div>
            )
        },
        {
            key: 'status' as keyof Alert,
            header: t('status'),
            sortable: true,
            render: (alert: Alert) => {
                const styles = {
                    active: 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
                    acknowledged: 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
                    resolved: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300'
                }[alert.status];
                const labels = { active: t('active'), acknowledged: t('acknowledged'), resolved: t('resolved') };
                return <Badge className={styles}>{labels[alert.status]}</Badge>;
            }
        },
        {
            key: 'timestamp' as keyof Alert,
            header: t('time'),
            sortable: true,
            render: (alert: Alert) => (
                <div className="flex items-center gap-1 text-slate-500 dark:text-slate-400 text-sm">
                    <Clock className="w-3.5 h-3.5" />
                    {new Date(alert.timestamp).toLocaleString()}
                </div>
            )
        },
        {
            key: 'actions',
            header: '',
            align: 'right' as const,
            render: (alert: Alert) => (
                <div className="flex justify-end gap-2">
                    {alert.status === 'active' && (
                        <PermissionGuard requiredRole={['admin', 'manager', 'technician']}>
                            <button
                                className="p-2 hover:bg-amber-50 dark:hover:bg-amber-900/20 rounded-lg transition-colors group"
                                onClick={(e) => { e.stopPropagation(); handleAction(alert.id, 'acknowledge'); }}
                                title={t('acknowledge')}
                            >
                                <Check className="w-4 h-4 text-slate-400 group-hover:text-amber-600 dark:text-slate-500 dark:group-hover:text-amber-400" />
                            </button>
                        </PermissionGuard>
                    )}
                    {alert.status !== 'resolved' && (
                        <PermissionGuard requiredRole={['admin', 'manager', 'technician']}>
                            <button
                                className="p-2 hover:bg-emerald-50 dark:hover:bg-emerald-900/20 rounded-lg transition-colors group"
                                onClick={(e) => { e.stopPropagation(); handleAction(alert.id, 'resolve'); }}
                                title={t('resolve')}
                            >
                                <CheckCircle className="w-4 h-4 text-slate-400 group-hover:text-emerald-600 dark:text-slate-500 dark:group-hover:text-emerald-400" />
                            </button>
                        </PermissionGuard>
                    )}
                    <PermissionGuard requiredRole={['admin']}>
                        <button
                            className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors group"
                            onClick={(e) => { e.stopPropagation(); setDeleteConfirm({ isOpen: true, id: alert.id }); }}
                            title={t('delete')}
                        >
                            <Trash2 className="w-4 h-4 text-slate-400 group-hover:text-red-500 dark:text-slate-500 dark:group-hover:text-red-400" />
                        </button>
                    </PermissionGuard>
                </div>
            )
        }
    ];

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('alerts_title')}</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">{t('alerts_subtitle')}</p>
                </div>
                <div className="flex gap-3">
                    <SavedViews
                        page="alerts"
                        currentFilterState={{
                            filters: { searchTerm, typeFilter, statusFilter },
                            sort: { column: sortColumn, direction: sortDirection },
                        }}
                        onApplyView={handleApplyView}
                    />
                    <Button
                        variant={showFilters ? 'primary' : 'outline'}
                        icon={<Filter className="w-4 h-4" />}
                        onClick={() => setShowFilters(!showFilters)}
                    >
                        {t('filter')}
                    </Button>
                </div>
            </div>

            {/* Status Summary Cards */}
            {isLoading ? (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <SkeletonCard />
                    <SkeletonCard />
                    <SkeletonCard />
                    <SkeletonCard />
                </div>
            ) : (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-blue-50 dark:bg-blue-900/30 rounded-xl"><Bell className="w-5 h-5 text-blue-600 dark:text-blue-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('total_alerts')}</p><p className="text-xl font-bold text-slate-900 dark:text-white">{statusCounts.total}</p></div></div></CardBody></Card>
                    <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-red-50 dark:bg-red-900/30 rounded-xl"><AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('active')}</p><p className="text-xl font-bold text-red-600 dark:text-red-400">{statusCounts.active}</p></div></div></CardBody></Card>
                    <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-amber-50 dark:bg-amber-900/30 rounded-xl"><Clock className="w-5 h-5 text-amber-600 dark:text-amber-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('acknowledged')}</p><p className="text-xl font-bold text-amber-600 dark:text-amber-400">{statusCounts.acknowledged}</p></div></div></CardBody></Card>
                    <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-emerald-50 dark:bg-emerald-900/30 rounded-xl"><CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('resolved')}</p><p className="text-xl font-bold text-emerald-600 dark:text-emerald-400">{statusCounts.resolved}</p></div></div></CardBody></Card>
                </div>
            )}

            {showFilters && (
                <div className="flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-in fade-in slide-in-from-top-2">
                    <div className="flex-1 max-w-md"><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('search')}</label><SearchInput placeholder={t('search_alerts')} onSearch={setSearchTerm} /></div>
                    <div className="flex gap-3">
                        <div className="min-w-[140px]"><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('status')}</label><Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} options={[{ value: 'all', label: t('all_status') }, { value: 'active', label: t('active') }, { value: 'acknowledged', label: t('acknowledged') }, { value: 'resolved', label: t('resolved') }]} /></div>
                        <div className="min-w-[140px]"><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('severity')}</label><Select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)} options={[{ value: 'all', label: t('all_severities') }, { value: 'error', label: t('error') }, { value: 'warning', label: t('warning') }, { value: 'info', label: t('info') }]} /></div>
                    </div>
                </div>
            )}

            {isLoading ? (
                <SkeletonTable rows={8} columns={4} />
            ) : (
                <DataGrid
                    data={filteredAlerts}
                    columns={columns}
                    keyExtractor={(item) => item.id}
                    sortColumn={sortColumn}
                    sortDirection={sortDirection}
                    onSort={handleSort}
                    emptyMessage={t('no_alerts')}
                    variant="striped"
                    defaultDensity="standard"
                    pageSize={20}
                    exportFilename="alerts.csv"
                />
            )}

            <ConfirmModal
                isOpen={deleteConfirm.isOpen}
                onClose={() => setDeleteConfirm({ isOpen: false, id: '' })}
                onConfirm={() => { if (deleteConfirm.id) deleteAlert(); }}
                title={t('delete_alert')}
                message={t('delete_alert_confirm')}
                confirmText={t('delete')}
                cancelText={t('cancel')}
                variant="danger"
            />
        </div>
    );
}