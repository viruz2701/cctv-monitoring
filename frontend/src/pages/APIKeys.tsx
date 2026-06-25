import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useFormValidation } from '../hooks/useFormValidation';
import { apiKeySchema } from '../lib/validations';
import { api } from '../services/api';
import { Button, Modal, Input, Badge, useToast, EmptyState, ConfirmModal } from '../components/ui';
import { useConfirmAction } from '../hooks/useConfirmAction';
import { Plus, Trash2, Copy, Key, Calendar, Shield } from 'lucide-react';

interface APIKey {
    id: string;
    name: string;
    permissions: string[];
    expires_at: string | null;
    last_used_at: string | null;
    created_at: string;
}

export function APIKeys() {
    const { t } = useTranslation();
    const toast = useToast();
    const { confirm, ConfirmDialog } = useConfirmAction();
    const [keys, setKeys] = useState<APIKey[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreateModal, setShowCreateModal] = useState(false);
    const [showKeyModal, setShowKeyModal] = useState(false);
    const [newKey, setNewKey] = useState<string>('');
    // Zod валидация для формы API ключа
    const { errors: apiKeyErrors, validate: validateApiKey, validateField: validateApiKeyField, touched: apiKeyTouched, reset: resetApiKeyValidation } = useFormValidation(apiKeySchema);
    const [formData, setFormData] = useState({
        name: '',
        permissions: ['read'],
        expires_at: '',
    });

    useEffect(() => {
        loadKeys();
    }, []);

    const loadKeys = async () => {
        try {
            setLoading(true);
            const data = await api.getAPIKeys();
            setKeys(Array.isArray(data) ? data : []);
        } catch (err: any) {
            toast.error(err.message || t('api_keys_load_error'));
            setKeys([]);
        } finally {
            setLoading(false);
        }
    };

    const handleCreate = async () => {
        const validationData = {
            name: formData.name,
            permissions: formData.permissions as Array<'read' | 'write' | 'admin'>,
        };

        if (!validateApiKey(validationData)) return;

        try {
            const data = await api.createAPIKey({
                name: formData.name,
                permissions: formData.permissions,
                expires_at: formData.expires_at || undefined,
            });
            setNewKey(data.api_key);
            setShowCreateModal(false);
            setShowKeyModal(true);
            resetApiKeyValidation();
            setFormData({ name: '', permissions: ['read'], expires_at: '' });
            loadKeys();
        } catch (err: any) {
            toast.error(err.message || t('api_key_create_error'));
        }
    };

    const handleRevoke = async (id: string) => {
        const confirmed = await confirm({
            title: t('revoke_api_key') || 'Revoke API Key',
            message: t('api_key_revoke_confirm'),
            confirmText: t('revoke') || 'Revoke',
            variant: 'danger',
        });
        if (!confirmed) return;
        try {
            await api.revokeAPIKey(id);
            toast.success(t('api_key_revoked'));
            loadKeys();
        } catch (err: any) {
            toast.error(err.message || t('api_key_revoke_error'));
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        toast.success(t('copied_to_clipboard'));
    };

    const permissionOptions = [
        { value: 'read', label: t('permission_read') },
        { value: 'write', label: t('permission_write') },
        { value: 'admin', label: t('permission_admin') },
    ];

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('api_keys')}</h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t('api_keys_description')}</p>
                </div>
                <Button onClick={() => setShowCreateModal(true)} icon={<Plus className="w-4 h-4" />}>
                    {t('create_api_key')}
                </Button>
            </div>

            {loading ? (
                <div className="text-center py-12">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto"></div>
                    <p className="mt-4 text-slate-500 dark:text-slate-400">{t('loading')}</p>
                </div>
            ) : keys.length === 0 ? (
                <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700">
                    <EmptyState
                        icon={<Key className="w-12 h-12" />}
                        title={t('no_api_keys') || 'No API keys'}
                        description={t('api_keys_empty_desc') || 'Create API keys to integrate CCTV Monitor with your external tools and automations'}
                        hint={t('api_keys_hint') || 'Keys support granular permissions and expiration dates'}
                        action={{
                            label: t('create_first_key') || 'Create Key',
                            onClick: () => setShowCreateModal(true),
                        }}
                        size="md"
                    />
                </div>
            ) : (
                <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-slate-200 dark:border-slate-700">
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('name')}
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('permissions')}
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('created')}
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('expires')}
                                </th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('last_used')}
                                </th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('actions')}
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                            {keys.map((key) => (
                                <tr key={key.id} className="hover:bg-slate-50 dark:hover:bg-slate-700/50">
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <div className="flex items-center gap-2">
                                            <Key className="w-4 h-4 text-slate-400" />
                                            <span className="text-sm font-medium text-slate-900 dark:text-white">
                                                {key.name}
                                            </span>
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap">
                                        <div className="flex gap-1">
                                            {Array.isArray(key.permissions) && key.permissions.map((perm) => (
                                                <Badge key={perm} variant={perm === 'admin' ? 'danger' : 'info'}>
                                                    {perm}
                                                </Badge>
                                            ))}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                        <div className="flex items-center gap-1">
                                            <Calendar className="w-3 h-3" />
                                            {new Date(key.created_at).toLocaleDateString()}
                                        </div>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                        {key.expires_at ? (
                                            <div className="flex items-center gap-1">
                                                <Calendar className="w-3 h-3" />
                                                {new Date(key.expires_at).toLocaleDateString()}
                                            </div>
                                        ) : (
                                            <span className="text-slate-400">{t('never')}</span>
                                        )}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                                        {key.last_used_at ? (
                                            new Date(key.last_used_at).toLocaleDateString()
                                        ) : (
                                            <span className="text-slate-400">{t('never')}</span>
                                        )}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => handleRevoke(key.id)}
                                            className="text-red-600 hover:text-red-700 dark:text-red-400"
                                            icon={<Trash2 className="w-4 h-4" />}
                                        >
                                            {t('revoke')}
                                        </Button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {/* Create API Key Modal */}
            <Modal
                isOpen={showCreateModal}
                onClose={() => { setShowCreateModal(false); resetApiKeyValidation(); }}
                title={t('create_api_key')}
                size="md"
            >
                <div className="space-y-4">
                    <Input
                        label={t('name')}
                        value={formData.name}
                        onChange={(e) => {
                            const newData = { ...formData, name: e.target.value };
                            setFormData(newData);
                            validateApiKeyField('name', { name: newData.name, permissions: newData.permissions as any });
                        }}
                        placeholder={t('api_key_name_placeholder')}
                        error={apiKeyTouched.has('name') ? apiKeyErrors.name : undefined}
                    />
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                            {t('permissions')}
                        </label>
                        <div className="space-y-2">
                            {permissionOptions.map((option) => (
                                <label key={option.value} className="flex items-center gap-2">
                                    <input
                                        type="checkbox"
                                        checked={formData.permissions.includes(option.value)}
                                        onChange={(e) => {
                                            const updated = e.target.checked
                                                ? [...formData.permissions, option.value]
                                                : formData.permissions.filter(p => p !== option.value);
                                            const newData = { ...formData, permissions: updated };
                                            setFormData(newData);
                                            validateApiKeyField('permissions', { name: newData.name, permissions: newData.permissions as any });
                                        }}
                                        className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                                    />
                                    <span className="text-sm text-slate-700 dark:text-slate-300">{option.label}</span>
                                </label>
                            ))}
                        </div>
                        {apiKeyTouched.has('permissions') && apiKeyErrors.permissions && (
                            <p className="mt-1.5 text-sm text-red-600">{apiKeyErrors.permissions}</p>
                        )}
                    </div>
                    <Input
                        label={t('expires_at')}
                        type="date"
                        value={formData.expires_at}
                        onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })}
                    />
                    <div className="flex justify-end gap-3 pt-4">
                        <Button variant="ghost" onClick={() => { setShowCreateModal(false); resetApiKeyValidation(); }}>
                            {t('cancel')}
                        </Button>
                        <Button onClick={handleCreate}>
                            {t('create')}
                        </Button>
                    </div>
                </div>
            </Modal>

            {/* Show New Key Modal */}
            <Modal
                isOpen={showKeyModal}
                onClose={() => setShowKeyModal(false)}
                title={t('api_key_created')}
                size="md"
            >
                <div className="space-y-4">
                    <div className="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                        <div className="flex items-start gap-3">
                            <Shield className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                            <div>
                                <p className="text-sm font-medium text-amber-800 dark:text-amber-200">
                                    {t('save_key_warning')}
                                </p>
                                <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
                                    {t('key_wont_be_shown_again')}
                                </p>
                            </div>
                        </div>
                    </div>
                    <div className="p-4 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">{t('your_api_key')}</p>
                        <div className="flex items-center gap-2">
                            <code className="flex-1 p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-600 rounded text-sm font-mono break-all">
                                {newKey}
                            </code>
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => copyToClipboard(newKey)}
                                icon={<Copy className="w-4 h-4" />}
                            />
                        </div>
                    </div>
                    <div className="flex justify-end pt-4">
                        <Button onClick={() => setShowKeyModal(false)}>
                            {t('done')}
                        </Button>
                    </div>
                </div>
            </Modal>

            {ConfirmDialog}
        </div>
    );
}
