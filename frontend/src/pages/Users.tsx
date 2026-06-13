import { generateUUID } from '../utils/uuid';
import React, { useState, useMemo } from 'react';
import { Plus, Search, Edit, Trash2, Shield, User as UserIcon, Filter } from 'lucide-react';
import {
    Card, CardHeader, CardBody, Table, Button, SearchInput,
    Modal, ConfirmModal, Input, Select, Badge, RoleBadge
} from '../components/ui';
import { useUsers } from '../context/DataContext';
import type { User as UserType, UserRole, UserStatus } from '../types';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';

export function Users() {
    const { t } = useTranslation();
    const { users, addUser, updateUser, deleteUser } = useUsers();
    const [searchQuery, setSearchQuery] = useState('');
    const [roleFilter, setRoleFilter] = useState<string>('all');
    const [statusFilter, setStatusFilter] = useState<string>('all');
    const [showAddModal, setShowAddModal] = useState(false);
    const [showFilters, setShowFilters] = useState(false);
    const [selectedUser, setSelectedUser] = useState<UserType | null>(null);
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });
    const [showRoleModal, setShowRoleModal] = useState(false);

    // Form state
    const [formData, setFormData] = useState({
        name: '',
        email: '',
        role: 'viewer' as UserRole,
        status: 'active' as UserStatus
    });

    const resetForm = () => {
        setFormData({
            name: '',
            email: '',
            role: 'viewer',
            status: 'active'
        });
        setSelectedUser(null);
    };

    const handleOpenModal = (user?: UserType) => {
        if (user) {
            setSelectedUser(user);
            setFormData({
                name: user.name,
                email: user.email,
                role: user.role,
                status: user.status
            });
        } else {
            resetForm();
        }
        setShowAddModal(true);
    };

    const handleSubmit = () => {
        if (selectedUser) {
            updateUser(selectedUser.id, formData);
        } else {
            const newUser: UserType = {
                id: `user-${generateUUID()}`,
                name: formData.name,
                email: formData.email,
                role: formData.role,
                status: formData.status,
                avatar: formData.name.split(' ').map(n => n[0]).join('').substring(0, 2).toUpperCase(),
                lastLogin: new Date().toISOString(),
                sites: [] // Default to no sites for refined prototype
            };
            addUser(newUser);
        }
        setShowAddModal(false);
        resetForm();
    };

    const handleDelete = (id: string) => {
        setDeleteConfirm({ isOpen: true, id });
    };

    const confirmDelete = () => {
        if (deleteConfirm.id) deleteUser(deleteConfirm.id);
    };

    const filteredUsers = useMemo(() => {
        return users.filter((user) => {
            const matchesSearch = user.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                user.email.toLowerCase().includes(searchQuery.toLowerCase());
            const matchesRole = roleFilter === 'all' || user.role === roleFilter;
            const matchesStatus = statusFilter === 'all' || user.status === statusFilter;
            return matchesSearch && matchesRole && matchesStatus;
        });
    }, [users, searchQuery, roleFilter, statusFilter]);

    const columns = [
        {
            key: 'name' as keyof UserType,
            header: t('user'),
            render: (user: UserType) => (
                <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-full flex items-center justify-center font-semibold">
                        {user.avatar || <UserIcon className="w-5 h-5" />}
                    </div>
                    <div>
                        <p className="font-medium text-slate-900 dark:text-white">{user.name}</p>
                        <p className="text-sm text-slate-500 dark:text-slate-300">{user.email}</p>
                    </div>
                </div>
            ),
        },
        { key: 'role' as keyof UserType, header: t('role'), render: (user: UserType) => <RoleBadge role={user.role} /> },
        {
            key: 'status' as keyof UserType,
            header: t('status'),
            render: (user: UserType) => <Badge variant={user.status === 'active' ? 'success' : 'neutral'} dot>{t(user.status)}</Badge>,
        },
        { key: 'sites' as keyof UserType, header: t('sites'), render: (user: UserType) => <span className="text-sm text-slate-600 dark:text-slate-300">{user.sites.length} {t('sites')}</span> },
        { key: 'lastLogin' as keyof UserType, header: t('last_login'), render: (user: UserType) => <span className="text-sm text-slate-500 dark:text-slate-300">{new Date(user.lastLogin).toLocaleDateString()}</span> },
        {
            key: 'actions', header: '', align: 'right' as const,
            render: (user: UserType) => (
                <div className="flex justify-end gap-1">
                    <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors" onClick={(e) => { e.stopPropagation(); handleOpenModal(user); }}>
                        <Edit className="w-4 h-4 text-slate-500 dark:text-slate-300" />
                    </button>
                    <button className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors" onClick={(e) => { e.stopPropagation(); handleDelete(user.id); }}>
                        <Trash2 className="w-4 h-4 text-red-500 hover:text-red-600 dark:text-red-400" />
                    </button>
                </div>
            ),
        },
    ];

    const rolePermissions = [
        {
            role: t('admin'),
            desc: t('role_admin_desc'),
            perms: [t('perm_internal_monitoring'), t('perm_user_management'), t('perm_system_config'), t('perm_report_generation')]
        },
        {
            role: t('manager'),
            desc: t('role_manager_desc'),
            perms: [t('perm_team_management'), t('perm_ticket_escalation'), t('perm_advanced_reporting'), t('perm_device_overrides')]
        },
        {
            role: t('technician'),
            desc: t('role_technician_desc'),
            perms: [t('perm_device_config'), t('perm_ticket_resolution'), t('perm_basic_diagnostics'), t('perm_maintenance_logs')]
        },
        {
            role: t('viewer'),
            desc: t('role_viewer_desc'),
            perms: [t('perm_dashboard_viewing'), t('perm_basic_reports'), t('perm_status_monitoring'), t('perm_alert_subscriptions')]
        }
    ];

    return (
        <PermissionGuard
            requiredRole="admin"
            fallback={
                <div className="flex flex-col items-center justify-center h-96">
                    <Shield className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" />
                    <h2 className="text-xl font-bold text-slate-900 dark:text-white">{t('access_denied')}</h2>
                    <p className="text-slate-500 dark:text-slate-400 mt-2">{t('no_permission_users')}</p>
                </div>
            }
        >
            <div className="space-y-6">
                {/* Header */}
                <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('users')}</h1>
                        <p className="text-slate-500 dark:text-slate-400 mt-1">
                            {t('users_subtitle')}
                        </p>
                    </div>
                    <div className="flex gap-3">
                        <Button
                            variant={showFilters ? 'primary' : 'outline'}
                            icon={<Filter className="w-4 h-4" />}
                            onClick={() => setShowFilters(!showFilters)}
                        >
                            {t('filter')}
                        </Button>
                        <Button icon={<Plus className="w-4 h-4" />} onClick={() => handleOpenModal()}>
                            {t('add_user')}
                        </Button>
                    </div>
                </div>

                {/* Stats Cards */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-slate-900 dark:text-white">{users.length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('total_users')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-emerald-600 dark:text-emerald-500">{users.filter(u => u.status === 'active').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('active')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-red-600 dark:text-red-500">{users.filter(u => u.role === 'admin').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('admins')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-blue-600 dark:text-blue-500">{users.filter(u => u.role === 'technician').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('technicians')}</p></div></CardBody></Card>
                </div>

                {/* Filters */}
                {showFilters && (
                    <div className="flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-in fade-in slide-in-from-top-2">
                        <div className="flex-1 max-w-md">
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('search')}</label>
                            <SearchInput
                                placeholder={t('search_users')}
                                onSearch={setSearchQuery}
                            />
                        </div>
                        <div className="flex gap-3">
                            <div className="min-w-[140px]">
                                <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('role')}</label>
                                <Select
                                    options={[
                                        { value: 'all', label: t('all_roles') },
                                        { value: 'admin', label: t('admin') },
                                        { value: 'manager', label: t('manager') },
                                        { value: 'technician', label: t('technician') },
                                        { value: 'viewer', label: t('viewer') }
                                    ]}
                                    value={roleFilter}
                                    onChange={(e) => setRoleFilter(e.target.value)}
                                />
                            </div>
                            <div className="min-w-[140px]">
                                <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('status')}</label>
                                <Select
                                    options={[
                                        { value: 'all', label: t('all_status') },
                                        { value: 'active', label: t('active') },
                                        { value: 'inactive', label: t('inactive') }
                                    ]}
                                    value={statusFilter}
                                    onChange={(e) => setStatusFilter(e.target.value)}
                                />
                            </div>
                        </div>
                    </div>
                )}

                <Table data={filteredUsers} columns={columns} keyExtractor={(user) => user.id} emptyMessage={t('no_users_found')} />

                <Card>
                    <CardHeader action={<Button variant="ghost" size="sm" icon={<Shield className="w-4 h-4" />} onClick={() => setShowRoleModal(true)}>{t('manage_roles')}</Button>}>{t('roles_permissions')}</CardHeader>
                    <CardBody>
                        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                            {rolePermissions.map((r) => (
                                <div key={r.role} className="p-4 bg-slate-50 dark:bg-slate-900/50 rounded-lg border border-slate-100 dark:border-slate-800">
                                    <h4 className="font-semibold text-slate-900 dark:text-white capitalize">{r.role}</h4>
                                    <p className="text-sm text-slate-500 dark:text-slate-300 mt-1 line-clamp-2" title={r.desc}>{r.desc}</p>
                                </div>
                            ))}
                        </div>
                    </CardBody>
                </Card>

                <Modal isOpen={showAddModal} onClose={() => setShowAddModal(false)} title={selectedUser ? t('edit_user') : t('add_user')} size="md"
                    footer={<div className="flex justify-end gap-3"><Button variant="outline" onClick={() => setShowAddModal(false)}>{t('cancel')}</Button><Button onClick={handleSubmit}>{selectedUser ? t('save') : t('add')}</Button></div>}>
                    <div className="space-y-4">
                        <Input
                            label={t('full_name')}
                            placeholder="John Doe"
                            value={formData.name}
                            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                        />
                        <Input
                            label={t('primary_email')}
                            type="email"
                            placeholder="john@company.com"
                            value={formData.email}
                            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                        />
                        <Select
                            label={t('role')}
                            options={[
                                { value: 'admin', label: t('admin') },
                                { value: 'manager', label: t('manager') },
                                { value: 'technician', label: t('technician') },
                                { value: 'viewer', label: t('viewer') }
                            ]}
                            value={formData.role}
                            onChange={(e) => setFormData({ ...formData, role: e.target.value as UserRole })}
                        />
                        <Select
                            label={t('status')}
                            options={[
                                { value: 'active', label: t('active') },
                                { value: 'inactive', label: t('inactive') }
                            ]}
                            value={formData.status}
                            onChange={(e) => setFormData({ ...formData, status: e.target.value as UserStatus })}
                        />
                    </div>
                </Modal>

                <ConfirmModal
                    isOpen={deleteConfirm.isOpen}
                    onClose={() => setDeleteConfirm({ isOpen: false, id: '' })}
                    onConfirm={confirmDelete}
                    title={t('delete_user')}
                    message={t('delete_user_confirm')}
                    confirmText={t('delete')}
                    cancelText={t('cancel')}
                    variant="danger"
                />

                <Modal
                    isOpen={showRoleModal}
                    onClose={() => setShowRoleModal(false)}
                    title={t('roles_permissions')}
                    size="lg"
                    footer={<div className="flex justify-end"><Button onClick={() => setShowRoleModal(false)}>{t('close')}</Button></div>}
                >
                    <div className="space-y-4 pr-2">
                        {rolePermissions.map((r) => (
                            <div key={r.role} className="flex items-start gap-4 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-100 dark:border-slate-700/50">
                                <div className="p-3 bg-indigo-100 dark:bg-indigo-900/30 rounded-xl">
                                    <Shield className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                                </div>
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-2 mb-1">
                                        <h4 className="font-semibold text-slate-900 dark:text-white capitalize">{r.role}</h4>
                                    </div>
                                    <p className="text-sm text-slate-500 dark:text-slate-400 mb-3">
                                        {r.desc}
                                    </p>
                                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[13px]">
                                        {r.perms.map(perm => (
                                            <div key={perm} className="flex items-center gap-2 text-slate-600 dark:text-slate-300">
                                                <div className="w-1.5 h-1.5 rounded-full bg-emerald-500"></div>
                                                {perm}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </Modal>
            </div>
        </PermissionGuard>
    );
}