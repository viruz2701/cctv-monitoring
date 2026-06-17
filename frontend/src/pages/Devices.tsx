import { generateUUID } from '../utils/uuid';
import React, { useState, useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
    Search,
    Filter,
    Plus,
    MoreVertical,
    Server,
    Camera,
    Activity,
    Wifi,
    WifiOff,
    AlertTriangle,
    CheckCircle,
    XCircle,
    Clock,
    Monitor,
    HardDrive,
    Edit,
    Trash2,
    Cloud,
} from 'lucide-react';
import {
    Card,
    CardHeader,
    CardBody,
    Badge,
    StatusBadge,
    HealthBadge,
    Button,
    Input,
    Select,
    Table,
    SearchInput,
    Pagination,
    Modal,
    ConfirmModal
} from '../components/ui';
import { useDevicesSites } from '../context/DataContext';
import type { Device } from '../types';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { formatDistanceToNow } from 'date-fns';
import { useTranslation } from 'react-i18next';
import { AddDeviceModal } from '../components/AddDeviceModal';

const ITEMS_PER_PAGE = 10;

const typeIcons = {
    camera: <Camera className="w-5 h-5 text-blue-500" />,
    nvr: <Server className="w-5 h-5 text-purple-500" />,
    dvr: <HardDrive className="w-5 h-5 text-slate-500" />,
    switch: <Activity className="w-5 h-5 text-emerald-500" />,
};

function timeAgo(dateString: string) {
    try {
        return formatDistanceToNow(new Date(dateString), { addSuffix: true });
    } catch (e) {
        return dateString;
    }
}

export function Devices() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { devices, sites, addDevice, updateDevice, deleteDevice } = useDevicesSites();
    const [searchQuery, setSearchQuery] = useState('');
    const [searchParams] = useSearchParams();
    const [statusFilter, setStatusFilter] = useState('all');
    const [siteFilter, setSiteFilter] = useState(searchParams.get('site') || 'all');
    const [currentPage, setCurrentPage] = useState(1);
    const [sortColumn, setSortColumn] = useState<string>('name');
    const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
    const [showAddDeviceModal, setShowAddDeviceModal] = useState(false);
    const [showFilters, setShowFilters] = useState(false);
    const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });

    // Форма для редактирования (только для обычных устройств, без P2P)
    const [editFormData, setEditFormData] = useState({
        name: '',
        type: 'camera' as Device['type'],
        siteId: '',
        ipAddress: '',
        model: '',
        siteName: ''
    });

    const resetEditForm = () => {
        setEditFormData({
            name: '',
            type: 'camera',
            siteId: sites[0]?.id || '',
            ipAddress: '',
            model: '',
            siteName: ''
        });
        setSelectedDevice(null);
    };

    const handleOpenEditModal = (device: Device) => {
        if (device.p2p_brand) {
            alert('Editing P2P devices is not yet implemented');
            return;
        }
        setSelectedDevice(device);
        setEditFormData({
            name: device.name,
            type: device.type,
            siteId: device.siteId,
            ipAddress: device.ipAddress,
            model: device.model,
            siteName: device.siteName
        });
    };

    const handleSaveEdit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!selectedDevice) return;
        const selectedSite = sites.find(s => s.id === editFormData.siteId);
        const siteName = selectedSite?.name || 'Unknown';
        updateDevice(selectedDevice.id, {
            name: editFormData.name,
            type: editFormData.type,
            siteId: editFormData.siteId,
            siteName: siteName,
            ipAddress: editFormData.ipAddress,
            model: editFormData.model
        });
        setSelectedDevice(null);
        resetEditForm();
    };

    const handleDeleteDevice = (deviceId: string) => {
        setDeleteConfirm({ isOpen: true, id: deviceId });
    };

    const confirmDeleteDevice = () => {
        if (deleteConfirm.id) deleteDevice(deleteConfirm.id);
    };

    const handleAddDeviceSuccess = () => {
        // Не вызываем refresh, чтобы не потерять только что добавленное P2P устройство
        // Если нужно синхронизировать с бэкендом, реализуйте сохранение через API
        // Просто закрываем модальное окно
        setShowAddDeviceModal(false);
    };

    const filteredDevices = useMemo(() => {
        let result = [...devices];
        if (searchQuery) {
            const q = searchQuery.toLowerCase();
            result = result.filter(
                (d) =>
                    d.name.toLowerCase().includes(q) ||
                    d.ipAddress.toLowerCase().includes(q) ||
                    d.siteName.toLowerCase().includes(q)
            );
        }
        if (siteFilter !== 'all') {
            result = result.filter((d) => d.siteId === siteFilter);
        }
        if (statusFilter !== 'all') {
            result = result.filter((d) => d.status === statusFilter);
        }
        result.sort((a, b) => {
            const aVal = String(a[sortColumn as keyof Device] ?? '');
            const bVal = String(b[sortColumn as keyof Device] ?? '');
            const cmp = aVal.localeCompare(bVal);
            return sortDirection === 'asc' ? cmp : -cmp;
        });
        return result;
    }, [devices, searchQuery, siteFilter, statusFilter, sortColumn, sortDirection]);

    const totalPages = Math.ceil(filteredDevices.length / ITEMS_PER_PAGE);
    const paginatedDevices = filteredDevices.slice(
        (currentPage - 1) * ITEMS_PER_PAGE,
        currentPage * ITEMS_PER_PAGE
    );

    const handleSort = (column: string) => {
        if (sortColumn === column) {
            setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'));
        } else {
            setSortColumn(column);
            setSortDirection('asc');
        }
    };

    const statusCounts = useMemo(() => {
        const counts = { total: filteredDevices.length, online: 0, offline: 0, warning: 0 };
        filteredDevices.forEach((d) => {
            if (d.status === 'online') counts.online++;
            else if (d.status === 'offline') counts.offline++;
            else counts.warning++;
        });
        return counts;
    }, [filteredDevices]);

    const columns = [
        {
            key: 'name' as keyof Device,
            header: t('device_name'),
            sortable: true,
            render: (device: Device) => (
                <div className="flex items-center gap-3">
                    <div className="p-2 bg-slate-100 dark:bg-slate-700/50 rounded-lg">
                        {typeIcons[device.type]}
                    </div>
                    <div>
                        <p className="font-medium text-slate-900 dark:text-white flex items-center gap-2">
                            {device.name}
                            {device.p2p_brand && (
                                <Cloud className="w-4 h-4 text-blue-500" />
                            )}
                        </p>
                        <p className="text-xs text-slate-500 dark:text-slate-400 uppercase">
                            {device.type} • {device.model}
                        </p>
                    </div>
                </div>
            ),
        },
        {
            key: 'siteName' as keyof Device,
            header: t('site'),
            sortable: true,
            render: (device: Device) => (
                <span className="text-slate-700 dark:text-slate-300">{device.siteName}</span>
            ),
        },
        {
            key: 'ipAddress' as keyof Device,
            header: t('ip_address'),
            sortable: true,
            render: (device: Device) => (
                <span className="font-mono text-slate-600 dark:text-slate-400 text-xs">
                    {device.ipAddress}
                </span>
            ),
        },
        {
            key: 'status' as keyof Device,
            header: t('status'),
            sortable: true,
            render: (device: Device) => <StatusBadge status={device.status} />,
        },
        {
            key: 'health' as keyof Device,
            header: t('health'),
            sortable: true,
            render: (device: Device) => <HealthBadge health={device.health} />,
        },
        {
            key: 'lastSeen' as keyof Device,
            header: t('last_seen'),
            sortable: true,
            render: (device: Device) => (
                <span className="text-slate-500 dark:text-slate-400 text-sm">
                    {timeAgo(device.lastSeen)}
                </span>
            ),
        },
        {
            key: 'actions',
            header: '',
            align: 'right' as const,
            render: (device: Device) => (
                <div className="flex justify-end gap-2">
                    <PermissionGuard requiredRole={['admin', 'manager']}>
                        <button
                            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors text-slate-400 hover:text-blue-600 dark:hover:text-blue-400"
                            onClick={(e) => {
                                e.stopPropagation();
                                handleOpenEditModal(device);
                            }}
                        >
                            <Edit className="w-4 h-4" />
                        </button>
                    </PermissionGuard>
                    <PermissionGuard requiredRole={['admin']}>
                        <button
                            className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors group"
                            onClick={(e) => {
                                e.stopPropagation();
                                handleDeleteDevice(device.id);
                            }}
                            title={t('delete')}
                        >
                            <Trash2 className="w-4 h-4 text-slate-400 group-hover:text-red-500 dark:text-slate-500 dark:group-hover:text-red-400" />
                        </button>
                    </PermissionGuard>
                </div>
            ),
        },
    ];

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('devices_title')}</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">{t('devices_subtitle')}</p>
                </div>
                <div className="flex gap-3">
                    <Button
                        variant={showFilters ? 'primary' : 'outline'}
                        icon={<Filter className="w-4 h-4" />}
                        onClick={() => setShowFilters(!showFilters)}
                    >
                        {t('filter')}
                    </Button>
                    <PermissionGuard requiredRole={['admin', 'manager']}>
                        <Button icon={<Plus className="w-4 h-4" />} onClick={() => setShowAddDeviceModal(true)}>
                            {t('add_device')}
                        </Button>
                    </PermissionGuard>
                </div>
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-blue-50 dark:bg-blue-900/30 rounded-xl"><HardDrive className="w-5 h-5 text-blue-600 dark:text-blue-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('total_devices')}</p><p className="text-xl font-bold text-slate-900 dark:text-white">{statusCounts.total}</p></div></div></CardBody></Card>
                <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-emerald-50 dark:bg-emerald-900/30 rounded-xl"><div className="w-5 h-5 flex items-center justify-center"><span className="w-2.5 h-2.5 bg-emerald-500 rounded-full animate-pulse" /></div></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('online')}</p><p className="text-xl font-bold text-emerald-600 dark:text-emerald-400">{statusCounts.online}</p></div></div></CardBody></Card>
                <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-red-50 dark:bg-red-900/30 rounded-xl"><div className="w-5 h-5 flex items-center justify-center"><span className="w-2.5 h-2.5 bg-red-500 rounded-full" /></div></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('offline')}</p><p className="text-xl font-bold text-red-600 dark:text-red-400">{statusCounts.offline}</p></div></div></CardBody></Card>
                <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-amber-50 dark:bg-amber-900/30 rounded-xl"><div className="w-5 h-5 flex items-center justify-center"><span className="w-2.5 h-2.5 bg-amber-500 rounded-full" /></div></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('warning')}</p><p className="text-xl font-bold text-amber-600 dark:text-amber-400">{statusCounts.warning}</p></div></div></CardBody></Card>
            </div>

            {showFilters && (
                <div className="flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-in fade-in slide-in-from-top-2">
                    <div className="flex-1 max-w-md">
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('search')}</label>
                        <SearchInput placeholder={t('search_devices')} onSearch={(q) => { setSearchQuery(q); setCurrentPage(1); }} />
                    </div>
                    <div className="flex gap-3">
                        <div className="min-w-[140px]"><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('site')}</label><Select value={siteFilter} onChange={(e) => { setSiteFilter(e.target.value); setCurrentPage(1); }} options={[{ value: 'all', label: t('all_sites') }, ...sites.map(s => ({ value: s.id, label: s.name }))]} /></div>
                        <div className="min-w-[140px]"><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('status')}</label><Select value={statusFilter} onChange={(e) => { setStatusFilter(e.target.value); setCurrentPage(1); }} options={[{ value: 'all', label: t('all_status') }, { value: 'online', label: t('online') }, { value: 'offline', label: t('offline') }, { value: 'warning', label: t('warning') }]} /></div>
                    </div>
                </div>
            )}

            <Table
                data={paginatedDevices}
                columns={columns}
                keyExtractor={(device) => device.id}
                onRowClick={(device) => navigate(`/devices/${device.id}`)}
                sortColumn={sortColumn}
                sortDirection={sortDirection}
                onSort={handleSort}
                emptyMessage={t('no_devices')}
            />

            {totalPages > 1 && (
                <div className="mt-4 flex justify-center">
                    <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={setCurrentPage} totalItems={filteredDevices.length} itemsPerPage={ITEMS_PER_PAGE} />
                </div>
            )}

            <AddDeviceModal
                isOpen={showAddDeviceModal}
                onClose={() => setShowAddDeviceModal(false)}
                onSuccess={handleAddDeviceSuccess}
            />

            {/* Модальное окно редактирования (только для обычных устройств) */}
            {selectedDevice && !selectedDevice.p2p_brand && (
                <Modal
                    isOpen={!!selectedDevice}
                    onClose={() => setSelectedDevice(null)}
                    title={t('edit_device')}
                >
                    <form onSubmit={handleSaveEdit} className="space-y-4">
                        <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">{t('device_name')}</label><Input value={editFormData.name} onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })} required /></div>
                        <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">{t('device_type')}</label><Select value={editFormData.type} onChange={(e) => setEditFormData({ ...editFormData, type: e.target.value as any })} options={[{ value: 'camera', label: t('camera') }, { value: 'nvr', label: 'NVR' }, { value: 'dvr', label: 'DVR' }, { value: 'switch', label: t('switch') }]} /></div>
                        <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">{t('site')}</label><Select value={editFormData.siteId} onChange={(e) => { const site = sites.find(s => s.id === e.target.value); setEditFormData({ ...editFormData, siteId: e.target.value, siteName: site?.name || '' }); }} options={sites.map(site => ({ value: site.id, label: site.name }))} /></div>
                        <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">{t('ip_address')}</label><Input value={editFormData.ipAddress} onChange={(e) => setEditFormData({ ...editFormData, ipAddress: e.target.value })} required /></div>
                        <div className="flex justify-end gap-3 mt-6"><Button type="button" variant="outline" onClick={() => setSelectedDevice(null)}>{t('cancel')}</Button><Button type="submit" variant="primary">{t('save')}</Button></div>
                    </form>
                </Modal>
            )}

            <ConfirmModal
                isOpen={deleteConfirm.isOpen}
                onClose={() => setDeleteConfirm({ isOpen: false, id: '' })}
                onConfirm={confirmDeleteDevice}
                title={t('delete_device')}
                message={t('delete_device_confirm')}
                confirmText={t('delete')}
                cancelText={t('cancel')}
                variant="danger"
            />
        </div>
    );
}