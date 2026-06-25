import { generateUUID } from '../utils/uuid';
import { getArrayData } from '../utils/helpers';
import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { useFormValidation } from '../hooks/useFormValidation';
import { ticketSchema } from '../lib/validations';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTickets, useDevices, useSites, useCreateTicket, useDeleteTicket } from '../hooks/useApiQuery';
import type { Ticket as APITicket, Device as APIDevice } from '../services/api';
import {
    Filter,
    Plus,
    Trash2
} from 'lucide-react';
import {
    Card,
    CardBody,
    Button,
    Input,
    Select,
    TicketStatusBadge,
    PriorityBadge,
    VirtualTable,
    Modal,
    ConfirmModal,
    SearchInput,
    SkeletonTable,
    SkeletonCard
} from '../components/ui';
import { SavedViews } from '../components/ui/SavedViews';
import { Ticket as TicketType } from '../types';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';

// ═══ API→UI mapping (migrated from TicketsContext) ═══
function mapAPITicketToUI(t: APITicket): TicketType {
    return {
        id: t.id,
        title: t.title,
        description: t.description,
        deviceId: t.device_id || '',
        deviceName: '',
        siteName: '',
        priority: (t.priority as TicketType['priority']) || 'medium',
        status: (t.status as TicketType['status']) || 'open',
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

function mapAPIDeviceToUI(d: APIDevice): import('../types').Device {
    return {
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: (d as any).site_id || 'site-default',
        siteName: (d as any).location || 'Unknown',
        type: ((d as any).vendor_type === 'camera' ? 'camera' : 'nvr') as import('../types').Device['type'],
        status: (d.status || 'offline').toLowerCase() as import('../types').Device['status'],
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording' as import('../types').Device['recordingStatus'],
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: (d as any).vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    };
}

export function Tickets() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const { data: apiTickets } = useTickets();
    const { data: apiDevices } = useDevices();
    const { data: apiSites } = useSites();
    const apiTicketsData = getArrayData<APITicket>(apiTickets);
    const apiDevicesData = getArrayData<APIDevice>(apiDevices);
    const apiSitesData = getArrayData<{ id: string; name?: string; address?: string; city?: string; status?: string; last_sync?: string; latitude?: number; longitude?: number }>(apiSites);
    const createTicketMut = useCreateTicket();
    const deleteTicketMut = useDeleteTicket();

    const tickets = useMemo(() => apiTicketsData.map(mapAPITicketToUI), [apiTicketsData]);
    const devices = useMemo(() => apiDevicesData.map(mapAPIDeviceToUI), [apiDevicesData]);
    const sites = useMemo(() => apiSitesData.map((s: any) => ({
        id: s.id,
        name: s.name || 'Unnamed',
        address: s.address || '',
        city: s.city || '',
        status: (s.status || 'active') as 'active' | 'inactive' | 'maintenance',
        lastSync: (s as any).last_sync || new Date().toISOString(),
        latitude: (s as any).latitude || 0,
        longitude: (s as any).longitude || 0,
    })), [apiSitesData]);

    const isLoading = tickets.length === 0;
    const [searchTerm, setSearchTerm] = useState('');
    const [statusFilter, setStatusFilter] = useState<string>('all');
    const [priorityFilter, setPriorityFilter] = useState<string>('all');
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showFilters, setShowFilters] = useState(false);
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });

    // UX-14.3.2: Apply saved view
    const handleApplyView = useCallback((view: import('../store/filterStore').SavedView) => {
        const filters = view.filters;
        if (filters.searchTerm) setSearchTerm(filters.searchTerm);
        if (filters.statusFilter) setStatusFilter(filters.statusFilter);
        if (filters.priorityFilter) setPriorityFilter(filters.priorityFilter);
    }, []);

    React.useEffect(() => {
        if (searchParams.get('action') === 'create') {
            setShowCreateModal(true);
        }
    }, [searchParams]);

    // Zod валидация для формы создания тикета
    const { errors: ticketErrors, validate: validateTicket, validateField: validateTicketField, touched: ticketTouched, reset: resetTicketValidation } = useFormValidation(ticketSchema);

    const [newTicket, setNewTicket] = useState({
        title: '',
        description: '',
        priority: 'medium',
        deviceId: '',
        siteId: ''
    });

    React.useEffect(() => {
        if (showCreateModal && !newTicket.siteId && sites.length > 0) {
            setNewTicket(prev => ({ ...prev, siteId: sites[0].id }));
        }
    }, [showCreateModal, sites]);

    const handleCreateTicket = (e: React.FormEvent) => {
        e.preventDefault();
        const selectedDevice = devices.find(d => d.id === newTicket.deviceId);
        if (!selectedDevice) return;

        const validationData = {
            title: newTicket.title,
            description: newTicket.description,
            priority: newTicket.priority as 'critical' | 'high' | 'medium' | 'low',
            deviceId: selectedDevice.id,
            siteId: newTicket.siteId || undefined,
        };

        if (!validateTicket(validationData)) return;

        const ticket: TicketType = {
            id: `TKT-${generateUUID()}`,
            title: newTicket.title,
            description: newTicket.description,
            priority: newTicket.priority as any,
            status: 'open',
            deviceId: selectedDevice.id,
            deviceName: selectedDevice.name,
            siteName: selectedDevice.siteName,
            assignee: 'Unassigned',
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            comments: []
        };
        createTicketMut.mutateAsync({
            title: ticket.title,
            description: ticket.description,
            device_id: ticket.deviceId,
            priority: ticket.priority,
            status: ticket.status,
        });
        setShowCreateModal(false);
        resetTicketValidation();
        setNewTicket({
            title: '',
            description: '',
            priority: 'medium',
            deviceId: '',
            siteId: sites.length > 0 ? sites[0].id : ''
        });
    };

    const handleDeleteTicket = (ticketId: string) => {
        setDeleteConfirm({ isOpen: true, id: ticketId });
    };

    const confirmDelete = () => {
        if (deleteConfirm.id) deleteTicketMut.mutateAsync(deleteConfirm.id);
    };

    const filteredTickets = useMemo(() => {
        return tickets.filter(ticket => {
            const matchesSearch =
                ticket.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
                ticket.id.toLowerCase().includes(searchTerm.toLowerCase());
            const matchesStatus = statusFilter === 'all' || ticket.status === statusFilter;
            const matchesPriority = priorityFilter === 'all' || ticket.priority === priorityFilter;
            return matchesSearch && matchesStatus && matchesPriority;
        });
    }, [tickets, searchTerm, statusFilter, priorityFilter]);

    const columns = [
        {
            key: 'id' as keyof TicketType,
            header: t('ticket_id'),
            render: (ticket: TicketType) => <span className="font-mono text-sm text-slate-600 dark:text-slate-400">{ticket.id}</span>,
        },
        {
            key: 'title' as keyof TicketType,
            header: t('title'),
            render: (ticket: TicketType) => (
                <div>
                    <p className="font-medium text-slate-900 dark:text-white">{ticket.title}</p>
                    <p className="text-xs text-slate-500 dark:text-slate-300 mt-0.5">{ticket.siteName}</p>
                </div>
            ),
        },
        {
            key: 'priority' as keyof TicketType,
            header: t('priority'),
            render: (ticket: TicketType) => <PriorityBadge priority={ticket.priority} />,
        },
        {
            key: 'status' as keyof TicketType,
            header: t('status'),
            render: (ticket: TicketType) => <TicketStatusBadge status={ticket.status} />,
        },
        {
            key: 'assignee' as keyof TicketType,
            header: t('assignee'),
            render: (ticket: TicketType) => (
                <div className="flex items-center gap-2">
                    <div className="w-6 h-6 bg-slate-200 dark:bg-slate-700 rounded-full flex items-center justify-center text-xs font-medium text-slate-600 dark:text-slate-300">
                        {ticket.assignee.split(' ').map(n => n[0]).join('')}
                    </div>
                    <span className="text-sm text-slate-700 dark:text-slate-300">{ticket.assignee}</span>
                </div>
            ),
        },
        {
            key: 'createdAt' as keyof TicketType,
            header: t('created'),
            render: (ticket: TicketType) => <span className="text-sm text-slate-500 dark:text-slate-400">{new Date(ticket.createdAt).toLocaleDateString()}</span>,
        },
        {
            key: 'actions',
            header: '',
            align: 'right' as const,
            render: (ticket: TicketType) => (
                <PermissionGuard requiredRole={['admin', 'manager']}>
                    <button
                        className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors group"
                        onClick={(e) => { e.stopPropagation(); handleDeleteTicket(ticket.id); }}
                        title={t('delete')}
                    >
                        <Trash2 className="w-4 h-4 text-slate-400 group-hover:text-red-500 dark:text-slate-500 dark:group-hover:text-red-400" />
                    </button>
                </PermissionGuard>
            ),
        },
    ];

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('tickets_title')}</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">{t('tickets_subtitle')}</p>
                </div>
                <div className="flex gap-3">
                    <SavedViews
                        page="tickets"
                        currentFilterState={{
                            filters: { searchTerm, statusFilter, priorityFilter },
                            sort: { column: 'createdAt', direction: 'desc' },
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
                    <PermissionGuard requiredRole={['admin', 'manager', 'technician']}>
                        <Button icon={<Plus className="w-4 h-4" />} onClick={() => setShowCreateModal(true)}>
                            {t('create_new_ticket')}
                        </Button>
                    </PermissionGuard>
                </div>
            </div>

            {/* Stats Cards */}
            {isLoading ? (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <SkeletonCard />
                    <SkeletonCard />
                    <SkeletonCard />
                    <SkeletonCard />
                </div>
            ) : (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-slate-900 dark:text-white">{tickets.length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('total_tickets')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-red-600 dark:text-red-500">{tickets.filter((t) => t.status === 'open').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('open')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-blue-600 dark:text-blue-500">{tickets.filter((t) => t.status === 'in_progress').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('in_progress')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-emerald-600 dark:text-emerald-500">{tickets.filter((t) => t.status === 'resolved' || t.status === 'closed').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('resolved')}</p></div></CardBody></Card>
                </div>
            )}

            {showFilters && (
                <div className="flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-in fade-in slide-in-from-top-2">
                    <div className="flex-1 max-w-md">
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('search')}</label>
                        <SearchInput placeholder={t('search_tickets')} onSearch={setSearchTerm} />
                    </div>
                    <div className="flex gap-3">
                        <div className="min-w-[140px]">
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('status')}</label>
                            <Select
                                options={[
                                    { value: 'all', label: t('all_status') },
                                    { value: 'open', label: t('open') },
                                    { value: 'in_progress', label: t('in_progress') },
                                    { value: 'pending', label: t('pending') },
                                    { value: 'resolved', label: t('resolved') },
                                    { value: 'closed', label: t('closed') },
                                ]}
                                value={statusFilter}
                                onChange={(e) => setStatusFilter(e.target.value)}
                            />
                        </div>
                        <div className="min-w-[140px]">
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('priority')}</label>
                            <Select
                                options={[
                                    { value: 'all', label: t('all_priority') },
                                    { value: 'critical', label: t('critical') },
                                    { value: 'high', label: t('high') },
                                    { value: 'medium', label: t('medium') },
                                    { value: 'low', label: t('low') },
                                ]}
                                value={priorityFilter}
                                onChange={(e) => setPriorityFilter(e.target.value)}
                            />
                        </div>
                    </div>
                </div>
            )}

            {isLoading ? (
                <SkeletonTable rows={8} columns={5} />
            ) : (
                <VirtualTable
                    data={filteredTickets}
                    columns={columns}
                    keyExtractor={(ticket: TicketType) => ticket.id}
                    onRowClick={(ticket: TicketType) => navigate(`/tickets/${ticket.id}`)}
                    emptyMessage={t('no_tickets')}
                    maxHeight={600}
                    estimateRowHeight={64}
                />
            )}

            <Modal
                isOpen={showCreateModal}
                onClose={() => { setShowCreateModal(false); resetTicketValidation(); }}
                title={t('create_new_ticket')}
                size="md"
                footer={
                    <div className="flex justify-end gap-3">
                        <Button variant="outline" onClick={() => { setShowCreateModal(false); resetTicketValidation(); }}>{t('cancel')}</Button>
                        <Button variant="primary" onClick={() => { const form = document.getElementById('create-ticket-form') as HTMLFormElement; form?.requestSubmit(); }}>{t('create')}</Button>
                    </div>
                }
            >
                <form id="create-ticket-form" onSubmit={handleCreateTicket} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('title')}</label>
                        <Input
                            value={newTicket.title}
                            onChange={e => {
                                const data = { ...newTicket, title: e.target.value };
                                setNewTicket(data);
                                validateTicketField('title', { title: data.title, description: data.description, priority: data.priority as any });
                            }}
                            placeholder={t('title_placeholder')}
                            error={ticketTouched.has('title') ? ticketErrors.title : undefined}
                            required
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('description')}</label>
                        <Input
                            value={newTicket.description}
                            onChange={e => {
                                const data = { ...newTicket, description: e.target.value };
                                setNewTicket(data);
                                validateTicketField('description', { title: data.title, description: data.description, priority: data.priority as any });
                            }}
                            placeholder={t('description_placeholder')}
                            error={ticketTouched.has('description') ? ticketErrors.description : undefined}
                            required
                        />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                        <div>
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('site')}</label>
                            <Select
                                value={newTicket.siteId}
                                onChange={e => setNewTicket({ ...newTicket, siteId: e.target.value, deviceId: '' })}
                                options={[
                                    { value: '', label: t('select_site') },
                                    ...sites.map(s => ({ value: s.id, label: s.name }))
                                ]}
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('device')}</label>
                            <Select
                                value={newTicket.deviceId}
                                onChange={e => setNewTicket({ ...newTicket, deviceId: e.target.value })}
                                options={[
                                    { value: '', label: t('select_device') },
                                    ...devices.filter(d => !newTicket.siteId || d.siteId === newTicket.siteId).map(d => ({ value: d.id, label: `${d.name}` }))
                                ]}
                                disabled={!newTicket.siteId}
                            />
                        </div>
                    </div>
                    <div>
                        <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('priority')}</label>
                        <Select
                            value={newTicket.priority}
                            onChange={e => setNewTicket({ ...newTicket, priority: e.target.value })}
                            options={[
                                { value: 'low', label: t('low') },
                                { value: 'medium', label: t('medium') },
                                { value: 'high', label: t('high') },
                                { value: 'critical', label: t('critical') },
                            ]}
                        />
                    </div>
                </form>
            </Modal>

            <ConfirmModal
                isOpen={deleteConfirm.isOpen}
                onClose={() => setDeleteConfirm({ isOpen: false, id: '' })}
                onConfirm={confirmDelete}
                title={t('delete_ticket')}
                message={t('delete_ticket_confirm')}
                confirmText={t('delete')}
                cancelText={t('cancel')}
                variant="danger"
            />
        </div>
    );
}