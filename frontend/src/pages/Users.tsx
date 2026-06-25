import React, { useState, useMemo } from 'react';
import { Plus, Edit, Trash2, Shield, User as UserIcon, Filter, Key } from 'lucide-react';
import { useFormValidation } from '../hooks/useFormValidation';
import { userSchema } from '../lib/validations';
import {
    Card, CardHeader, CardBody, Table, Button, SearchInput,
    Modal, ConfirmModal, Input, Select, Badge, RoleBadge, useToast
} from '../components/ui';
import { useUsers } from '../context/DataContext';
import type { User } from '../services/api';
import { api } from '../services/api';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';

type UserRole = User['role'];
type UserStatus = NonNullable<User['status']>;

export function Users() {
    const { t } = useTranslation();
    const toast = useToast();
    const { users, addUser, updateUser, deleteUser } = useUsers();
    const [searchQuery, setSearchQuery] = useState('');
    const [roleFilter, setRoleFilter] = useState<string>('all');
    const [statusFilter, setStatusFilter] = useState<string>('all');
    const [showAddModal, setShowAddModal] = useState(false);
    const [showFilters, setShowFilters] = useState(false);
    const [selectedUser, setSelectedUser] = useState<User | null>(null);
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });
    const [showRoleModal, setShowRoleModal] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    const ALL_PERMISSIONS = [
        'devices_read', 'devices_write', 'devices_delete',
        'users_read', 'users_write', 'users_delete',
        'sites_read', 'sites_write', 'sites_delete',
        'reports_generate', 'reports_schedule',
        'settings_read', 'settings_write',
        'maintenance_read', 'maintenance_write',
        'tickets_read', 'tickets_write',
    ] as const;

    const DEFAULT_ROLE_PERMISSIONS: Record<string, string[]> = {
        admin: [...ALL_PERMISSIONS],
        manager: ['devices_read', 'devices_write', 'sites_read', 'sites_write', 'reports_generate', 'reports_schedule', 'maintenance_read', 'maintenance_write', 'tickets_read', 'tickets_write'],
        technician: ['devices_read', 'maintenance_read', 'maintenance_write', 'tickets_read', 'tickets_write'],
        viewer: ['devices_read', 'sites_read', 'reports_generate', 'maintenance_read', 'tickets_read'],
    };

    const [rolePermissions, setRolePermissions] = useState<Record<string, string[]>>(DEFAULT_ROLE_PERMISSIONS);
    const [savingRole, setSavingRole] = useState<string | null>(null);

    // ═══ Reset Password States ═══
    const [showResetPasswordModal, setShowResetPasswordModal] = useState(false);
    const [resetPasswordForm, setResetPasswordForm] = useState({ userId: '', newPassword: '', confirm: '' });
    const [resetPasswordLoading, setResetPasswordLoading] = useState(false);
    const [resetPasswordError, setResetPasswordError] = useState('');

    // Zod валидация для формы пользователя
    const { errors: userErrors, validate: validateUser, validateField: validateUserField, touched: userTouched, reset: resetUserValidation } = useFormValidation(userSchema);

    const [formData, setFormData] = useState({
        username: '',
        name: '',
        email: '',
        password: '',
        role: 'viewer' as UserRole,
        status: 'active' as UserStatus
    });

    // ═══ Password Strength ═══
    const [passwordStrength, setPasswordStrength] = useState<'weak' | 'medium' | 'strong' | ''>('');

    const calculatePasswordStrength = (password: string): 'weak' | 'medium' | 'strong' | '' => {
        if (!password) return '';
        if (password.length < 8) return 'weak';

        const hasUpper = /[A-Z]/.test(password);
        const hasLower = /[a-z]/.test(password);
        const hasDigit = /\d/.test(password);
        const hasSpecial = /[!@#$%^&*(),.?":{}|<>_\-~`\[\]\\';\/]/.test(password);

        if (!hasUpper || !hasLower || !hasDigit || !hasSpecial) return 'weak';

        let score = 0;
        if (password.length >= 12) score++;
        if (hasUpper && hasLower && hasDigit && hasSpecial) score++;
        if (password.length >= 16) score++;

        if (score >= 3) return 'strong';
        return 'medium';
    };

    const handlePasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const pwd = e.target.value;
        setFormData({ ...formData, password: pwd });
        setPasswordStrength(calculatePasswordStrength(pwd));
    };

    const resetForm = () => {
        setFormData({
            username: '',
            name: '',
            email: '',
            password: '',
            role: 'viewer',
            status: 'active'
        });
        setSelectedUser(null);
    };

    const handleOpenModal = (user?: User) => {
        if (user) {
            setSelectedUser(user);
            setFormData({
                username: user.username,
                name: user.name || user.username,
                email: user.email || '',
                password: '',
                role: user.role,
                status: user.status || 'active'
            });
        } else {
            resetForm();
        }
        setShowAddModal(true);
    };

    // ═══ Reset Password Handlers ═══
    const handleOpenResetPassword = (userId: string) => {
        setResetPasswordForm({ userId, newPassword: '', confirm: '' });
        setResetPasswordError('');
        setShowResetPasswordModal(true);
    };

    const handleResetPassword = async () => {
        setResetPasswordError('');
        if (!resetPasswordForm.newPassword || !resetPasswordForm.confirm) {
            setResetPasswordError(t('all_fields_required') || 'All fields are required');
            return;
        }
        if (resetPasswordForm.newPassword.length < 6) {
            setResetPasswordError(t('password_min_length') || 'Password must be at least 6 characters');
            return;
        }
        if (resetPasswordForm.newPassword !== resetPasswordForm.confirm) {
            setResetPasswordError(t('passwords_do_not_match') || 'Passwords do not match');
            return;
        }

        setResetPasswordLoading(true);
        try {
            await api.resetUserPassword(resetPasswordForm.userId, resetPasswordForm.newPassword);
            toast.success(t('password_reset_success') || 'Password reset successfully');
            setShowResetPasswordModal(false);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : (t('password_reset_failed') || 'Failed to reset password');
            setResetPasswordError(message);
        } finally {
            setResetPasswordLoading(false);
        }
    };

    const handleSubmit = async () => {
        setSubmitting(true);

        // Zod валидация
        const validationData = {
            username: formData.username,
            email: formData.email,
            role: formData.role as 'admin' | 'manager' | 'technician' | 'viewer',
            password: formData.password || undefined,
        };

        if (!validateUser(validationData)) {
            setSubmitting(false);
            return;
        }

        try {
            if (selectedUser) {
                await updateUser(selectedUser.id, {
                    name: formData.name,
                    email: formData.email,
                    role: formData.role,
                    status: formData.status,
                });
                toast.success(t('user_updated') || 'User updated successfully');
            } else {
                if (!formData.password) {
                    toast.error(t('password_required') || 'Password is required for new users');
                    setSubmitting(false);
                    return;
                }
                if (!formData.username) {
                    toast.error(t('username_required') || 'Username is required for login');
                    setSubmitting(false);
                    return;
                }
                // OWASP ASVS V2: Password strength validation — блокируем слабые пароли
                const strength = calculatePasswordStrength(formData.password);
                if (strength === 'weak') {
                    toast.error(t('password_too_weak') || 'Password is too weak. Minimum 8 chars with uppercase, lowercase, digit, and special character.');
                    setSubmitting(false);
                    return;
                }
                const newUser = {
                    username: formData.username,
                    name: formData.name || formData.username,
                    email: formData.email,
                    password: formData.password,
                    role: formData.role,
                    status: formData.status,
                    avatar: (formData.name || formData.username).split(' ').map(n => n[0]).join('').substring(0, 2).toUpperCase(),
                    lastLogin: new Date().toISOString(),
                    sites: []
                };
                await addUser(newUser);
                toast.success(t('user_created') || 'User created successfully');
            }
            setShowAddModal(false);
            resetForm();
        } catch (err: unknown) {
            console.error('Submit error:', err);
            const message = err instanceof Error ? err.message : (t('operation_failed') || 'Operation failed');
            toast.error(message);
        } finally {
            setSubmitting(false);
        }
    };

    const handleDelete = (id: string) => {
        setDeleteConfirm({ isOpen: true, id });
    };

    const confirmDelete = async () => {
        if (deleteConfirm.id) {
            try {
                await deleteUser(deleteConfirm.id);
                toast.success(t('user_deleted') || 'User deleted successfully');
            } catch (err: unknown) {
                const message = err instanceof Error ? err.message : (t('delete_failed') || 'Failed to delete user');
                toast.error(message);
            }
        }
    };

    const filteredUsers = useMemo(() => {
        return users.filter((user) => {
            const matchesSearch = (user.name || user.username).toLowerCase().includes(searchQuery.toLowerCase()) ||
                (user.email || '').toLowerCase().includes(searchQuery.toLowerCase());
            const matchesRole = roleFilter === 'all' || user.role === roleFilter;
            const matchesStatus = statusFilter === 'all' || user.status === statusFilter;
            return matchesSearch && matchesRole && matchesStatus;
        });
    }, [users, searchQuery, roleFilter, statusFilter]);

    const columns = [
        {
            key: 'name' as keyof User,
            header: t('user'),
            render: (user: User) => (
                <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-full flex items-center justify-center font-semibold">
                        {user.avatar || <UserIcon className="w-5 h-5" />}
                    </div>
                    <div>
                        <p className="font-medium text-slate-900 dark:text-white">{user.name || user.username}</p>
                        <p className="text-sm text-slate-500 dark:text-slate-300">{user.email || ''}</p>
                    </div>
                </div>
            ),
        },
        { key: 'role' as keyof User, header: t('role'), render: (user: User) => <RoleBadge role={user.role} /> },
        {
            key: 'status' as keyof User,
            header: t('status'),
            render: (user: User) => <Badge variant={user.status === 'active' ? 'success' : 'neutral'} dot>{t(user.status || 'inactive')}</Badge>,
        },
        { key: 'sites' as keyof User, header: t('sites'), render: (user: User) => <span className="text-sm text-slate-600 dark:text-slate-300">{(user.sites || []).length} {t('sites')}</span> },
        { key: 'lastLogin' as keyof User, header: t('last_login'), render: (user: User) => <span className="text-sm text-slate-500 dark:text-slate-300">{user.lastLogin ? new Date(user.lastLogin).toLocaleDateString() : '-'}</span> },
        {
            key: 'actions', header: '', align: 'right' as const,
            render: (user: User) => (
                <div className="flex justify-end gap-1">
                    <button className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors" onClick={(e) => { e.stopPropagation(); handleOpenModal(user); }} title={t('edit_user')}>
                        <Edit className="w-4 h-4 text-slate-500 dark:text-slate-300" />
                    </button>
                    {/* ═══ Reset Password Button ═══ */}
                    <button className="p-2 hover:bg-amber-50 dark:hover:bg-amber-900/20 rounded-lg transition-colors" onClick={(e) => { e.stopPropagation(); handleOpenResetPassword(user.id); }} title={t('reset_password')}>
                        <Key className="w-4 h-4 text-amber-500 hover:text-amber-600 dark:text-amber-400" />
                    </button>
                    <button className="p-2 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors" onClick={(e) => { e.stopPropagation(); handleDelete(user.id); }} title={t('delete_user')}>
                        <Trash2 className="w-4 h-4 text-red-500 hover:text-red-600 dark:text-red-400" />
                    </button>
                </div>
            ),
        },
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
                <div className="flex flex-col sm:flex-row gap-4 justify-between items-start sm:items-center">
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('users')}</h1>
                        <p className="text-slate-500 dark:text-slate-400 mt-1">{t('users_subtitle')}</p>
                    </div>
                    <div className="flex gap-3">
                        <Button variant={showFilters ? 'primary' : 'outline'} icon={<Filter className="w-4 h-4" />} onClick={() => setShowFilters(!showFilters)}>
                            {t('filter')}
                        </Button>
                        <Button icon={<Plus className="w-4 h-4" />} onClick={() => handleOpenModal()}>
                            {t('add_user')}
                        </Button>
                    </div>
                </div>

                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-slate-900 dark:text-white">{users.length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('total_users')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-emerald-600 dark:text-emerald-500">{users.filter(u => u.status === 'active').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('active')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-red-600 dark:text-red-500">{users.filter(u => u.role === 'admin').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('admins')}</p></div></CardBody></Card>
                    <Card padding="sm"><CardBody><div className="text-center"><p className="text-2xl font-bold text-blue-600 dark:text-blue-500">{users.filter(u => u.role === 'technician').length}</p><p className="text-sm text-slate-500 dark:text-slate-300">{t('technicians')}</p></div></CardBody></Card>
                </div>

                {showFilters && (
                    <div className="flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 animate-in fade-in slide-in-from-top-2">
                        <div className="flex-1 max-w-md">
                            <label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-300">{t('search')}</label>
                            <SearchInput placeholder={t('search_users')} onSearch={setSearchQuery} />
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
                            {Object.entries(rolePermissions).map(([role, perms]) => (
                                <div key={role} className="p-4 bg-slate-50 dark:bg-slate-900/50 rounded-lg border border-slate-100 dark:border-slate-800">
                                    <h4 className="font-semibold text-slate-900 dark:text-white capitalize">{role}</h4>
                                    <p className="text-sm text-slate-500 dark:text-slate-300 mt-1">{perms.length} {t('permissions') || 'permissions'}</p>
                                </div>
                            ))}
                        </div>
                    </CardBody>
                </Card>

                <Modal isOpen={showAddModal} onClose={() => { setShowAddModal(false); resetUserValidation(); }} title={selectedUser ? t('edit_user') : t('add_user')} size="md"
                    footer={<div className="flex justify-end gap-3"><Button variant="outline" onClick={() => { setShowAddModal(false); resetUserValidation(); }} disabled={submitting}>{t('cancel')}</Button><Button onClick={handleSubmit} disabled={submitting}>{submitting ? t('saving') : (selectedUser ? t('save') : t('add'))}</Button></div>}>
                    <div className="space-y-4">
                        {!selectedUser && (
                            <Input
                                label={t('username') || 'Username'}
                                placeholder="johndoe"
                                value={formData.username}
                                onChange={(e) => {
                                    const newData = { ...formData, username: e.target.value };
                                    setFormData(newData);
                                    validateUserField('username', { username: newData.username, email: newData.email, role: newData.role as any, password: newData.password || undefined });
                                }}
                                error={userTouched.has('username') ? userErrors.username : undefined}
                                required
                            />
                        )}
                        <Input label={t('full_name')} placeholder="John Doe" value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} />
                        <Input
                            label={t('primary_email')}
                            type="email"
                            placeholder="john@company.com"
                            value={formData.email}
                            onChange={(e) => {
                                const newData = { ...formData, email: e.target.value };
                                setFormData(newData);
                                validateUserField('email', { username: newData.username, email: newData.email, role: newData.role as any, password: newData.password || undefined });
                            }}
                            error={userTouched.has('email') ? userErrors.email : undefined}
                        />
                        {!selectedUser && (
                            <div>
                                <Input
                                    label={t('password')}
                                    type="password"
                                    placeholder="••••••••"
                                    value={formData.password}
                                    onChange={(e) => {
                                        handlePasswordChange(e);
                                        const newData = { ...formData, password: e.target.value };
                                        validateUserField('password', { username: newData.username, email: newData.email, role: newData.role as any, password: newData.password || undefined });
                                    }}
                                    error={userTouched.has('password') ? userErrors.password : undefined}
                                    required
                                />
                                {passwordStrength && (
                                    <div className="mt-2">
                                        <div className="flex items-center gap-2">
                                            <div className="flex-1 h-1.5 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
                                                <div className={`h-full rounded-full transition-all duration-300 ${
                                                    passwordStrength === 'weak' ? 'w-1/3 bg-red-500' :
                                                    passwordStrength === 'medium' ? 'w-2/3 bg-amber-500' :
                                                    'w-full bg-emerald-500'
                                                }`} />
                                            </div>
                                            <span className={`text-xs font-medium min-w-[48px] text-right ${
                                                passwordStrength === 'weak' ? 'text-red-500' :
                                                passwordStrength === 'medium' ? 'text-amber-500' :
                                                'text-emerald-500'
                                            }`}>
                                                {passwordStrength === 'weak' ? (t('weak') || 'Weak') :
                                                 passwordStrength === 'medium' ? (t('medium') || 'Medium') :
                                                 (t('strong') || 'Strong')}
                                            </span>
                                        </div>
                                    </div>
                                )}
                            </div>
                        )}
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

                {/* ═══ Reset Password Modal ═══ */}
                <Modal
                    isOpen={showResetPasswordModal}
                    onClose={() => setShowResetPasswordModal(false)}
                    title={t('reset_password')}
                    size="sm"
                    footer={
                        <div className="flex justify-end gap-3">
                            <Button variant="outline" onClick={() => setShowResetPasswordModal(false)}>{t('cancel')}</Button>
                            <Button onClick={handleResetPassword} loading={resetPasswordLoading}>{t('save')}</Button>
                        </div>
                    }
                >
                    <div className="space-y-4">
                        {resetPasswordError && (
                            <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
                                {resetPasswordError}
                            </div>
                        )}
                        <Input
                            label={t('new_password')}
                            type="password"
                            placeholder="••••••••"
                            value={resetPasswordForm.newPassword}
                            onChange={(e) => setResetPasswordForm({ ...resetPasswordForm, newPassword: e.target.value })}
                            autoComplete="new-password"
                        />
                        <Input
                            label={t('confirm_password')}
                            type="password"
                            placeholder="••••••••"
                            value={resetPasswordForm.confirm}
                            onChange={(e) => setResetPasswordForm({ ...resetPasswordForm, confirm: e.target.value })}
                            autoComplete="new-password"
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

                <Modal isOpen={showRoleModal} onClose={() => setShowRoleModal(false)} title={t('roles_permissions')} size="xl"
                    footer={<div className="flex justify-end"><Button onClick={() => setShowRoleModal(false)}>{t('close')}</Button></div>}>
                    <div className="space-y-6 pr-2">
                        {Object.entries(rolePermissions).map(([role, permissions]) => {
                            return (
                                <div key={role} className="p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-100 dark:border-slate-700/50">
                                    <div className="flex items-center justify-between mb-3">
                                        <div className="flex items-center gap-2">
                                            <Shield className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                                            <h4 className="font-semibold text-slate-900 dark:text-white capitalize">{role}</h4>
                                            <span className="text-xs text-slate-400">({permissions.length}/{ALL_PERMISSIONS.length})</span>
                                        </div>
                                        <Button
                                            size="sm"
                                            loading={savingRole === role}
                                            onClick={async () => {
                                                setSavingRole(role);
                                                try {
                                                    // Simulate API save - in production, call api.updateRolePermissions(role, permissions)
                                                    await new Promise(resolve => setTimeout(resolve, 300));
                                                    toast.success(`${role} ${t('permissions_saved') || 'permissions saved'}`);
                                                } catch (err: any) {
                                                    toast.error(err?.message || (t('operation_failed') || 'Operation failed'));
                                                } finally {
                                                    setSavingRole(null);
                                                }
                                            }}
                                        >
                                            {t('save') || 'Save'}
                                        </Button>
                                    </div>
                                    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
                                        {ALL_PERMISSIONS.map(perm => {
                                            const isChecked = permissions.includes(perm);
                                            const group = perm.split('_')[0];
                                            const groupColors: Record<string, string> = {
                                                devices: 'bg-blue-100 dark:bg-blue-900/30 border-blue-200 dark:border-blue-700 text-blue-700 dark:text-blue-300',
                                                users: 'bg-purple-100 dark:bg-purple-900/30 border-purple-200 dark:border-purple-700 text-purple-700 dark:text-purple-300',
                                                sites: 'bg-green-100 dark:bg-green-900/30 border-green-200 dark:border-green-700 text-green-700 dark:text-green-300',
                                                reports: 'bg-amber-100 dark:bg-amber-900/30 border-amber-200 dark:border-amber-700 text-amber-700 dark:text-amber-300',
                                                settings: 'bg-rose-100 dark:bg-rose-900/30 border-rose-200 dark:border-rose-700 text-rose-700 dark:text-rose-300',
                                                maintenance: 'bg-cyan-100 dark:bg-cyan-900/30 border-cyan-200 dark:border-cyan-700 text-cyan-700 dark:text-cyan-300',
                                                tickets: 'bg-orange-100 dark:bg-orange-900/30 border-orange-200 dark:border-orange-700 text-orange-700 dark:text-orange-300',
                                            };
                                            return (
                                                <label
                                                    key={perm}
                                                    className={`
                                                        flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer transition-all text-[13px]
                                                        ${isChecked
                                                            ? (groupColors[group] || 'bg-slate-100 dark:bg-slate-700 border-slate-300 dark:border-slate-500')
                                                            : 'bg-white dark:bg-slate-800 border-slate-200 dark:border-slate-600 opacity-70 hover:opacity-100'
                                                        }
                                                    `}
                                                >
                                                    <input
                                                        type="checkbox"
                                                        checked={isChecked}
                                                        onChange={() => {
                                                            const updated = isChecked
                                                                ? permissions.filter(p => p !== perm)
                                                                : [...permissions, perm];
                                                            setRolePermissions(prev => ({ ...prev, [role]: updated }));
                                                        }}
                                                        className="rounded border-slate-300 dark:border-slate-600 text-indigo-600 focus:ring-indigo-500"
                                                    />
                                                    <span className="font-medium">{perm.replace(/_/g, ' ')}</span>
                                                </label>
                                            );
                                        })}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </Modal>
            </div>
        </PermissionGuard>
    );
}