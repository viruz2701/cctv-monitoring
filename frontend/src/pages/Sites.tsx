import { generateUUID } from '../utils/uuid';
import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { useFormValidation } from '../hooks/useFormValidation';
import { siteSchema } from '../lib/validations';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
    Building2,
    MapPin,
    Plus,
    Filter,
    Edit,
    Trash2,
    ChevronDown,
    ChevronUp,
    Camera,
    Users,
    Star,
    StarOff,
    X,
    TreePine,
    Table2,
} from 'lucide-react';
import {
    Card,
    CardBody,
    Badge,
    StatusBadge,
    Button,
    Input,
    Select,
    Table,
    Modal,
    ConfirmModal,
    SearchInput,
    SkeletonCard,
    SkeletonTable,
} from '../components/ui';
import { SavedViews } from '../components/ui/SavedViews';
import { useDevices, useSites, useCreateSite, useUpdateSite, useDeleteSite } from '../hooks/useApiQuery';
import { useUsers } from '../hooks/useApiQuery';
import { Breadcrumbs } from '../components/ui/Breadcrumbs';
import { AssetTree } from '../components/organisms/AssetTree';
import { api, TechnicianSiteAssignment } from '../services/api';
import { useToast } from '../components/ui/Toast';
import type { Site, Device } from '../types';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';

export function Sites() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const { data: rawSites = [] } = useSites();
    const { data: rawDevices = [] } = useDevices();
    const { data: rawUsers = [] } = useUsers();
    const createSite = useCreateSite();
    const updateSiteMut = useUpdateSite();
    const deleteSiteMut = useDeleteSite();

    const mapAPISiteToUI = useCallback((s: any) => ({
        id: s.id,
        name: s.name || 'Unnamed',
        address: s.address || '',
        city: s.city || '',
        organization: s.organization || '',
        latitude: s.latitude || 0,
        longitude: s.longitude || 0,
        status: s.status || 'active',
        lastSync: s.last_sync || new Date().toISOString(),
    }), []);

    const sites = useMemo(() => rawSites.map(mapAPISiteToUI), [rawSites, mapAPISiteToUI]);

    const mapAPIDeviceToUI = useCallback((d: any) => ({
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: d.site_id || 'site-default',
        siteName: d.location || 'Unknown',
        type: d.vendor_type === 'camera' ? 'camera' : 'nvr',
        status: (d.status || 'offline').toLowerCase(),
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording',
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: d.vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    }), []);

    const devices = useMemo(() => {
        const devsArray = Array.isArray(rawDevices) ? rawDevices : (rawDevices && typeof rawDevices === 'object' && 'devices' in rawDevices ? (rawDevices as any).devices : []);
        return devsArray.map(mapAPIDeviceToUI);
    }, [rawDevices, mapAPIDeviceToUI]);

    const users = useMemo(() => rawUsers.map(u => ({ ...u, name: (u as any).name || u.username })), [rawUsers]);

    const isLoading = sites.length === 0 && devices.length === 0;
    const toast = useToast();

    const [searchQuery, setSearchQuery] = useState(searchParams.get('search') || '');
    const [statusFilter, setStatusFilter] = useState('all');
    const [showFilters, setShowFilters] = useState(!!searchParams.get('search'));
    const [expandedSiteId, setExpandedSiteId] = useState<string | null>(null);
    const [showAddModal, setShowAddModal] = useState(false);
    const [selectedSite, setSelectedSite] = useState<Site | null>(null);
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });
    const [viewMode, setViewMode] = useState<'table' | 'tree'>('table');
    const [treeBreadcrumbs, setTreeBreadcrumbs] = useState<Array<{ id: string; name: string; type: string }>>([]);

    // UX-14.3.2: Apply saved view
    const handleApplyView = useCallback((view: import('../store/filterStore').SavedView) => {
        const filters = view.filters;
        if (filters.searchQuery !== undefined) {
            setSearchQuery(filters.searchQuery);
            if (filters.searchQuery) {
                setSearchParams({ search: filters.searchQuery });
                setShowFilters(true);
            } else {
                setSearchParams({});
            }
        }
        if (filters.statusFilter) setStatusFilter(filters.statusFilter);
    }, [setSearchParams]);

    // Zod валидация для формы сайта
    const { errors: siteErrors, validate: validateSite, validateField: validateSiteField, touched: siteTouched, reset: resetSiteValidation } = useFormValidation(siteSchema);

    const [formData, setFormData] = useState({
        name: '',
        address: '',
        city: '',
        organization: '',
        latitude: 0,
        longitude: 0,
        status: 'active'
    });

    // Technician assignment state
    const [siteAssignments, setSiteAssignments] = useState<TechnicianSiteAssignment[]>([]);
    const [assignmentsLoading, setAssignmentsLoading] = useState(false);
    const [assignForm, setAssignForm] = useState({ technician_id: '', is_primary: false });
    const [deleteAssignConfirm, setDeleteAssignConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });

    const technicians = users.filter(u => u.role === 'technician');

    const loadSiteAssignments = useCallback(async (siteId: string) => {
        try {
            setAssignmentsLoading(true);
            const data = await api.getTechnicianSiteAssignments({ site_id: siteId });
            setSiteAssignments(data);
        } catch {
            // silent fail
        } finally {
            setAssignmentsLoading(false);
        }
    }, []);

    const resetForm = () => {
        setFormData({ name: '', address: '', city: '', organization: '', latitude: 0, longitude: 0, status: 'active' });
        setSelectedSite(null);
        setSiteAssignments([]);
        setAssignForm({ technician_id: '', is_primary: false });
        resetSiteValidation();
    };

    const handleSearch = (query: string) => {
        setSearchQuery(query);
        if (query) {
            setSearchParams({ search: query });
            setShowFilters(true);
        } else {
            setSearchParams({});
        }
    };

    useEffect(() => {
        const query = searchParams.get('search') || '';
        setSearchQuery(query);
        if (query) setShowFilters(true);
    }, [searchParams]);

    const handleOpenModal = (site?: Site) => {
        if (site) {
            setSelectedSite(site);
            setFormData({
                name: site.name,
                address: site.address,
                city: site.city,
                organization: site.organization || '',
                latitude: site.latitude || 0,
                longitude: site.longitude || 0,
                status: site.status as string
            });
            loadSiteAssignments(site.id);
        } else {
            resetForm();
        }
        setShowAddModal(true);
    };

    const handleAddAssignment = async () => {
        if (!assignForm.technician_id || !selectedSite) return;
        try {
            await api.createTechnicianSiteAssignment({
                technician_id: assignForm.technician_id,
                site_id: selectedSite.id,
                is_primary: assignForm.is_primary,
            });
            toast.success(t('assignment_created') || 'Assignment created');
            setAssignForm({ technician_id: '', is_primary: false });
            loadSiteAssignments(selectedSite.id);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed';
            toast.error(message);
        }
    };

    const handleDeleteAssignment = async () => {
        try {
            await api.deleteTechnicianSiteAssignment(deleteAssignConfirm.id);
            toast.success(t('assignment_deleted') || 'Assignment deleted');
            setDeleteAssignConfirm({ isOpen: false, id: '' });
            if (selectedSite) loadSiteAssignments(selectedSite.id);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed';
            toast.error(message);
        }
    };

    const handleTogglePrimary = async (assignment: TechnicianSiteAssignment) => {
        try {
            await api.updateTechnicianSiteAssignment(assignment.id, { is_primary: !assignment.is_primary });
            if (selectedSite) loadSiteAssignments(selectedSite.id);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed';
            toast.error(message);
        }
    };

    const handleSaveSite = async (e: React.FormEvent) => {
        e.preventDefault();

        const validationData = {
            name: formData.name,
            address: formData.address,
            city: formData.city,
            status: formData.status as 'active' | 'inactive' | 'maintenance',
            organization: formData.organization || undefined,
        };

        if (!validateSite(validationData)) return;

        try {
            if (selectedSite) {
                await updateSiteMut.mutateAsync({ id: selectedSite.id, updates: {
                    name: formData.name,
                    address: formData.address,
                    city: formData.city,
                    organization: formData.organization || undefined,
                    latitude: formData.latitude || undefined,
                    longitude: formData.longitude || undefined,
                    status: formData.status,
                } as any });
            } else {
                const newSite: Partial<Site> = {
                    name: formData.name,
                    address: formData.address,
                    city: formData.city,
                    organization: formData.organization || undefined,
                    latitude: formData.latitude || undefined,
                    longitude: formData.longitude || undefined,
                    status: formData.status as any,
                };
                await createSite.mutateAsync(newSite as any);
            }
            setShowAddModal(false);
            resetForm();
        } catch (err) {
            console.error('Failed to save site:', err);
            toast.error(t('save_failed') || 'Failed to save site');
        }
    };

    const handleDeleteSite = (siteId: string) => {
        setDeleteConfirm({ isOpen: true, id: siteId });
    };

    const confirmDeleteSite = () => {
        if (deleteConfirm.id) deleteSiteMut.mutateAsync(deleteConfirm.id);
    };

    const toggleExpand = (siteId: string) => {
        setExpandedSiteId(expandedSiteId === siteId ? null : siteId);
    };

    const filteredSites = useMemo(() => {
        let result = [...sites];
        if (searchQuery) {
            const q = searchQuery.toLowerCase();
            result = result.filter(s => s.name.toLowerCase().includes(q) || s.city.toLowerCase().includes(q) || s.address.toLowerCase().includes(q) || (s.organization || '').toLowerCase().includes(q));
        }
        if (statusFilter !== 'all') {
            result = result.filter(s => s.status === statusFilter);
        }
        return result;
    }, [sites, searchQuery, statusFilter]);

    const getSiteDevices = (siteId: string) => devices.filter(d => d.siteId === siteId);

    const getTechnicianName = (techId: string) => {
        const tech = technicians.find(t => t.id === techId);
        return tech?.name || tech?.username || techId;
    };

    const columns = [
        {
            key: 'name' as keyof Site,
            header: t('site_name'),
            render: (site: Site) => (
                <div className="flex items-center gap-3">
                    <div className="p-2 bg-slate-100 dark:bg-slate-700/50 rounded-lg text-slate-600 dark:text-slate-300">
                        <Building2 className="w-5 h-5" />
                    </div>
                    <div>
                        <p className="font-medium text-slate-900 dark:text-white">{site.name}</p>
                        <p className="text-xs text-slate-500 dark:text-slate-400">ID: {site.id}</p>
                    </div>
                </div>
            ),
        },
        {
            key: 'organization' as keyof Site,
            header: t('organization'),
            render: (site: Site) => (
                <span className="text-slate-600 dark:text-slate-300">{site.organization || '-'}</span>
            ),
        },
        {
            key: 'address' as keyof Site,
            header: t('location'),
            render: (site: Site) => (
                <div className="flex items-center gap-2 text-slate-600 dark:text-slate-300">
                    <MapPin className="w-4 h-4 text-slate-400" />
                    <span>{site.address}, {site.city}</span>
                </div>
            ),
        },
        {
            key: 'status' as keyof Site,
            header: t('status'),
            render: (site: Site) => <StatusBadge status={site.status} />,
        },
        {
            key: 'devices' as keyof Site,
            header: t('devices'),
            render: (site: Site) => {
                const siteDevices = getSiteDevices(site.id);
                return (
                    <div className="flex flex-col gap-1">
                        <Badge variant="neutral">{siteDevices.length} {t('devices')}</Badge>
                        <span className="text-xs text-slate-500 dark:text-slate-400">{siteDevices.filter(d => d.status === 'online').length} {t('online')}</span>
                    </div>
                );
            },
        },
        {
            key: 'actions',
            header: '',
            align: 'right' as const,
            render: (site: Site) => (
                <div className="flex justify-end gap-2">
                    <button
                        className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors text-slate-400 hover:text-blue-600 dark:hover:text-blue-400"
                        onClick={(e) => { e.stopPropagation(); toggleExpand(site.id); }}
                    >
                        {expandedSiteId === site.id ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                    </button>
                    <PermissionGuard requiredRole={['admin', 'manager']}>
                        <button
                            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors text-slate-400 hover:text-blue-600 dark:hover:text-blue-400"
                            onClick={(e) => { e.stopPropagation(); handleOpenModal(site); }}
                        >
                            <Edit className="w-4 h-4" />
                        </button>
                    </PermissionGuard>
                    <PermissionGuard requiredRole={['admin']}>
                        <button
                            className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors group"
                            onClick={(e) => { e.stopPropagation(); handleDeleteSite(site.id); }}
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
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('sites')}</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">{t('sites_subtitle')}</p>
                </div>
                <div className="flex gap-3">
                    {/* View Toggle: Table ↔ Tree */}
                    <div className="flex items-center bg-slate-100 dark:bg-slate-800 rounded-lg p-0.5">
                        <button
                            onClick={() => setViewMode('table')}
                            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                                viewMode === 'table'
                                    ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                            }`}
                            title={t('table_view') || 'Table view'}
                        >
                            <Table2 className="w-3.5 h-3.5" />
                            <span className="hidden sm:inline">{t('table') || 'Table'}</span>
                        </button>
                        <button
                            onClick={() => setViewMode('tree')}
                            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                                viewMode === 'tree'
                                    ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                            }`}
                            title={t('tree_view') || 'Tree view'}
                        >
                            <TreePine className="w-3.5 h-3.5" />
                            <span className="hidden sm:inline">{t('tree') || 'Tree'}</span>
                        </button>
                    </div>
                    <SavedViews
                        page="sites"
                        currentFilterState={{
                            filters: { searchQuery, statusFilter },
                            sort: { column: 'name', direction: 'asc' },
                        }}
                        onApplyView={handleApplyView}
                    />
                    <Button variant={showFilters ? 'primary' : 'outline'} icon={<Filter className="w-4 h-4" />} onClick={() => setShowFilters(!showFilters)}>{t('filter')}</Button>
                    {viewMode === 'table' && (
                        <PermissionGuard requiredRole={['admin']}>
                            <Button icon={<Plus className="w-4 h-4" />} onClick={() => handleOpenModal()}>{t('add_site')}</Button>
                        </PermissionGuard>
                    )}
                </div>
            </div>

            {showFilters && (
                <div className="flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-in fade-in slide-in-from-top-2">
                    <div className="flex-1 max-w-md">
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('search')}</label>
                        <SearchInput placeholder={t('search_sites')} value={searchQuery} onSearch={handleSearch} />
                    </div>
                    <div className="w-48">
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('status')}</label>
                        <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} options={[
                            { value: 'all', label: t('all_status') },
                            { value: 'active', label: t('active') },
                            { value: 'inactive', label: t('inactive') },
                            { value: 'maintenance', label: t('maintenance') }
                        ]} />
                    </div>
                </div>
            )}

            {viewMode === 'tree' ? (
                <div className="animate-in fade-in slide-in-from-top-2 duration-200">
                    {treeBreadcrumbs.length > 0 && (
                        <div className="mb-3">
                            <Breadcrumbs
                                items={treeBreadcrumbs.map((crumb, idx) => ({
                                    label: crumb.name,
                                    href: idx < treeBreadcrumbs.length - 1 ? undefined : undefined,
                                }))}
                            />
                        </div>
                    )}
                    <AssetTree
                        hideStats
                        hideSearch={false}
                        onNodeSelect={(node, crumbs) =>
                            setTreeBreadcrumbs(crumbs.map((c) => ({ id: c.id, name: c.name, type: c.type })))
                        }
                    />
                </div>
            ) : isLoading ? (
                <SkeletonTable rows={5} columns={5} />
            ) : (
                <Table<Site>
                    data={filteredSites}
                    columns={columns}
                    keyExtractor={(s) => s.id}
                    onRowClick={(s) => toggleExpand(s.id)}
                    expandable={(site) => expandedSiteId === site.id && (
                        <div className="p-4 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-200 dark:border-slate-700">
                            <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 flex items-center gap-2">
                                <Camera className="w-4 h-4" /> {t('connected_devices')}
                            </h4>
                            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                                {getSiteDevices(site.id).map(device => (
                                    <div key={device.id} className="flex items-center gap-3 p-3 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
                                        <div className={`w-2 h-2 rounded-full ${device.status === 'online' ? 'bg-emerald-500' : 'bg-red-500'}`} />
                                        <div><p className="text-sm font-medium text-slate-900 dark:text-white">{device.name}</p><p className="text-xs text-slate-500 dark:text-slate-400">{device.ipAddress}</p></div>
                                    </div>
                                ))}
                                {getSiteDevices(site.id).length === 0 && <p className="text-sm text-slate-500 dark:text-slate-400 italic">{t('no_devices_site')}</p>}
                            </div>
                            <div className="mt-4 flex justify-end">
                                <Button size="sm" variant="outline" onClick={(e) => { e.stopPropagation(); navigate(`/devices?site=${site.id}`); }}>{t('view_all_devices')}</Button>
                            </div>
                        </div>
                    )}
                    emptyMessage={t('no_sites')}
                />
            )}

            {/* Edit/Create Site Modal */}
            <Modal
                isOpen={showAddModal}
                onClose={() => { setShowAddModal(false); resetForm(); }}
                title={selectedSite ? t('edit_site') : t('add_site')}
                size="lg"
                footer={
                    <div className="flex justify-end gap-3">
                        <Button variant="outline" onClick={() => { setShowAddModal(false); resetForm(); }}>{t('cancel')}</Button>
                        <Button variant="primary" onClick={() => { const form = document.getElementById('site-form') as HTMLFormElement; form?.requestSubmit(); }}>{selectedSite ? t('save') : t('add_site')}</Button>
                    </div>
                }
            >
                <form id="site-form" onSubmit={handleSaveSite} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('site_name')}</label>
                        <Input
                            value={formData.name}
                            onChange={e => {
                                const newData = { ...formData, name: e.target.value };
                                setFormData(newData);
                                validateSiteField('name', { name: newData.name, address: newData.address, city: newData.city, status: newData.status as any });
                            }}
                            placeholder={t('site_name_placeholder')}
                            error={siteTouched.has('name') ? siteErrors.name : undefined}
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('organization') || 'Organization'}</label>
                        <Input value={formData.organization} onChange={e => setFormData({ ...formData, organization: e.target.value })} placeholder={t('organization_placeholder') || 'Organization name'} />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('latitude') || 'Latitude'}</label>
                            <Input type="number" step="any" value={formData.latitude} onChange={e => setFormData({ ...formData, latitude: parseFloat(e.target.value) || 0 })} placeholder="0.0" />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('longitude') || 'Longitude'}</label>
                            <Input type="number" step="any" value={formData.longitude} onChange={e => setFormData({ ...formData, longitude: parseFloat(e.target.value) || 0 })} placeholder="0.0" />
                        </div>
                    </div>
                    <div className="flex gap-2">
                        <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={async () => {
                                const query = [formData.address, formData.city].filter(Boolean).join(', ');
                                if (!query) {
                                    toast.error(t('enter_address_first') || 'Enter address first');
                                    return;
                                }
                                try {
                                    const res = await fetch(`https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(query)}&limit=1`);
                                    const data = await res.json();
                                    if (data && data[0]) {
                                        setFormData(prev => ({
                                            ...prev,
                                            latitude: parseFloat(data[0].lat),
                                            longitude: parseFloat(data[0].lon),
                                        }));
                                        toast.success(t('coordinates_found') || 'Coordinates found');
                                    } else {
                                        toast.error(t('coordinates_not_found') || 'Coordinates not found for this address');
                                    }
                                } catch {
                                    toast.error(t('geocoding_error') || 'Failed to get coordinates');
                                }
                            }}
                            icon={<MapPin className="w-4 h-4" />}
                        >
                            {t('get_coordinates') || 'Get coordinates'}
                        </Button>
                        {formData.latitude !== 0 && formData.longitude !== 0 && (
                            <Button
                                type="button"
                                size="sm"
                                variant="ghost"
                                onClick={() => {
                                    window.open(`https://www.openstreetmap.org/?mlat=${formData.latitude}&mlon=${formData.longitude}#map=15/${formData.latitude}/${formData.longitude}`, '_blank');
                                }}
                                icon={<MapPin className="w-4 h-4" />}
                            >
                                {t('view_on_map') || 'View on map'}
                            </Button>
                        )}
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('address')}</label>
                        <Input
                            value={formData.address}
                            onChange={e => {
                                const newData = { ...formData, address: e.target.value };
                                setFormData(newData);
                                validateSiteField('address', { name: newData.name, address: newData.address, city: newData.city, status: newData.status as any });
                            }}
                            placeholder={t('address_placeholder')}
                            error={siteTouched.has('address') ? siteErrors.address : undefined}
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('city')}</label>
                        <Input
                            value={formData.city}
                            onChange={e => {
                                const newData = { ...formData, city: e.target.value };
                                setFormData(newData);
                                validateSiteField('city', { name: newData.name, address: newData.address, city: newData.city, status: newData.status as any });
                            }}
                            placeholder={t('city_placeholder')}
                            error={siteTouched.has('city') ? siteErrors.city : undefined}
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('status')}</label>
                        <Select
                            value={formData.status}
                            onChange={e => setFormData({ ...formData, status: e.target.value })}
                            options={[{ value: 'active', label: t('active') }, { value: 'inactive', label: t('inactive') }, { value: 'maintenance', label: t('maintenance') }]}
                        />
                    </div>
                </form>

                {/* Technician Assignments Section (only when editing) */}
                {selectedSite && (
                    <div className="mt-6 pt-6 border-t border-slate-200 dark:border-slate-700">
                        <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 flex items-center gap-2">
                            <Users className="w-4 h-4" />
                            {t('assigned_technicians') || 'Assigned Technicians'}
                        </h4>

                        {/* Current assignments */}
                        {assignmentsLoading ? (
                            <div className="text-center py-4">
                                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600 mx-auto"></div>
                            </div>
                        ) : siteAssignments.length === 0 ? (
                            <p className="text-sm text-slate-500 dark:text-slate-400 py-2">
                                {t('no_technicians_assigned') || 'No technicians assigned to this site'}
                            </p>
                        ) : (
                            <div className="space-y-2 mb-4">
                                {siteAssignments.map((assignment) => (
                                    <div key={assignment.id} className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                                        <div className="flex items-center gap-3">
                                            <div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                                                <Users className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                                            </div>
                                            <div>
                                                <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                    {assignment.technician_name || getTechnicianName(assignment.technician_id)}
                                                </p>
                                                <p className="text-xs text-slate-500 dark:text-slate-400">
                                                    {t('assigned_date') || 'Assigned'} {new Date(assignment.assigned_at).toLocaleDateString()}
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-1">
                                            <button
                                                onClick={() => handleTogglePrimary(assignment)}
                                                className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                                                title={assignment.is_primary ? (t('unset_primary') || 'Unset as primary') : (t('set_primary') || 'Set as primary')}
                                            >
                                                {assignment.is_primary ? (
                                                    <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" />
                                                ) : (
                                                    <StarOff className="w-4 h-4 text-slate-400" />
                                                )}
                                            </button>
                                            <button
                                                onClick={() => setDeleteAssignConfirm({ isOpen: true, id: assignment.id })}
                                                className="p-1.5 rounded hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors text-slate-400 hover:text-red-500"
                                            >
                                                <X className="w-4 h-4" />
                                            </button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}

                        {/* Add assignment form */}
                        <div className="flex items-end gap-3">
                            <div className="flex-1">
                                <label className="block text-xs font-medium mb-1 text-slate-600 dark:text-slate-400">
                                    {t('technician') || 'Technician'}
                                </label>
                                <Select
                                    value={assignForm.technician_id}
                                    onChange={(e) => setAssignForm({ ...assignForm, technician_id: e.target.value })}
                                    options={[
                                        { value: '', label: t('select_technician') || 'Select technician...' },
                                        ...technicians
                                            .filter(t => !siteAssignments.some(a => a.technician_id === t.id))
                                            .map((tech) => ({
                                                value: tech.id,
                                                label: tech.name || tech.username,
                                            })),
                                    ]}
                                />
                            </div>
                            <div className="flex items-center gap-2 pb-0.5">
                                <label className="flex items-center gap-1.5 text-xs text-slate-600 dark:text-slate-400 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={assignForm.is_primary}
                                        onChange={(e) => setAssignForm({ ...assignForm, is_primary: e.target.checked })}
                                        className="w-3.5 h-3.5 text-blue-600 border-slate-300 rounded focus:ring-blue-500"
                                    />
                                    {t('primary_technician') || 'Primary'}
                                </label>
                            </div>
                            <Button
                                size="sm"
                                onClick={handleAddAssignment}
                                disabled={!assignForm.technician_id}
                            >
                                <Plus className="w-3.5 h-3.5 mr-1" />
                                {t('add') || 'Add'}
                            </Button>
                        </div>
                    </div>
                )}
            </Modal>

            <ConfirmModal
                isOpen={deleteConfirm.isOpen}
                onClose={() => setDeleteConfirm({ isOpen: false, id: '' })}
                onConfirm={confirmDeleteSite}
                title={t('delete_site')}
                message={t('delete_site_confirm')}
                confirmText={t('delete')}
                cancelText={t('cancel')}
                variant="danger"
            />

            {/* Delete Assignment Confirm */}
            <ConfirmModal
                isOpen={deleteAssignConfirm.isOpen}
                onClose={() => setDeleteAssignConfirm({ isOpen: false, id: '' })}
                onConfirm={handleDeleteAssignment}
                title={t('remove_assignment') || 'Remove Assignment'}
                message={t('remove_assignment_confirm') || 'Are you sure you want to remove this technician assignment?'}
                confirmText={t('remove') || 'Remove'}
                cancelText={t('cancel')}
                variant="danger"
            />
        </div>
    );
}