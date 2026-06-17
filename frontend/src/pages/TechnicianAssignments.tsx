import React, { useState, useEffect, useCallback } from 'react';
import { Users, MapPin, Plus, Trash2, Star, StarOff } from 'lucide-react';
import { Card, CardBody, Button, Modal, Select, ConfirmModal } from '../components/ui';
import { api, TechnicianSiteAssignment } from '../services/api';
import { useUsers } from '../context/UsersContext';
import { useDevicesSites } from '../context/DevicesSitesContext';
import { useToast } from '../components/ui/Toast';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../hooks/useAuth';

export function TechnicianAssignments() {
    const { t } = useTranslation();
    const { users } = useUsers();
    const { sites } = useDevicesSites();
    const toast = useToast();
    const { token } = useAuth();

    const [assignments, setAssignments] = useState<TechnicianSiteAssignment[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; id: string }>({ isOpen: false, id: '' });
    
    const [formData, setFormData] = useState({
        technician_id: '',
        site_id: '',
        is_primary: false,
    });

    // Фильтруем только техников
    const technicians = users.filter(u => u.role === 'technician');

    const loadAssignments = useCallback(async () => {
        try {
            setLoading(true);
            const data = await api.getTechnicianSiteAssignments();
            setAssignments(data);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed to load assignments';
            toast.error(message);
        } finally {
            setLoading(false);
        }
    }, [toast]);

    useEffect(() => {
        if (!token) return;
        loadAssignments();
    }, [token, loadAssignments]);

    const handleAdd = async () => {
        if (!formData.technician_id || !formData.site_id) {
            toast.error('Please select both technician and site');
            return;
        }

        try {
            await api.createTechnicianSiteAssignment({
                technician_id: formData.technician_id,
                site_id: formData.site_id,
                is_primary: formData.is_primary,
            });
            toast.success('Assignment created successfully');
            setShowAddModal(false);
            setFormData({ technician_id: '', site_id: '', is_primary: false });
            loadAssignments();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed to create assignment';
            toast.error(message);
        }
    };

    const handleDelete = async () => {
        try {
            await api.deleteTechnicianSiteAssignment(deleteConfirm.id);
            toast.success('Assignment deleted successfully');
            setDeleteConfirm({ isOpen: false, id: '' });
            loadAssignments();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed to delete assignment';
            toast.error(message);
        }
    };

    const handleTogglePrimary = async (assignment: TechnicianSiteAssignment) => {
        try {
            await api.updateTechnicianSiteAssignment(assignment.id, {
                is_primary: !assignment.is_primary,
            });
            toast.success('Assignment updated successfully');
            loadAssignments();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Failed to update assignment';
            toast.error(message);
        }
    };

    const getTechnicianName = (technicianId: string) => {
        const tech = technicians.find(t => t.id === technicianId);
        return tech?.name || tech?.username || technicianId;
    };

    const getSiteName = (siteId: string) => {
        const site = sites.find(s => s.id === siteId);
        return site?.name || siteId;
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                        {t('technician_assignments') || 'Technician Site Assignments'}
                    </h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">
                        {t('technician_assignments_desc') || 'Manage technician assignments to sites'}
                    </p>
                </div>
                <Button onClick={() => setShowAddModal(true)}>
                    <Plus className="w-4 h-4 mr-2" />
                    {t('add_assignment') || 'Add Assignment'}
                </Button>
            </div>

            <Card>
                <CardBody>
                    {assignments.length === 0 ? (
                        <div className="text-center py-12">
                            <Users className="w-12 h-12 text-slate-400 mx-auto mb-4" />
                            <p className="text-slate-500 dark:text-slate-400">
                                {t('no_assignments') || 'No technician assignments yet'}
                            </p>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="border-b border-slate-200 dark:border-slate-700">
                                        <th className="text-left py-3 px-4 text-sm font-medium text-slate-700 dark:text-slate-300">
                                            {t('technician') || 'Technician'}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm font-medium text-slate-700 dark:text-slate-300">
                                            {t('site') || 'Site'}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm font-medium text-slate-700 dark:text-slate-300">
                                            {t('primary') || 'Primary'}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm font-medium text-slate-700 dark:text-slate-300">
                                            {t('assigned_date') || 'Assigned Date'}
                                        </th>
                                        <th className="text-right py-3 px-4 text-sm font-medium text-slate-700 dark:text-slate-300">
                                            {t('actions') || 'Actions'}
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {assignments.map((assignment) => (
                                        <tr
                                            key={assignment.id}
                                            className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50"
                                        >
                                            <td className="py-3 px-4">
                                                <div className="flex items-center gap-3">
                                                    <div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                                                        <Users className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                                                    </div>
                                                    <span className="text-sm font-medium text-slate-900 dark:text-white">
                                                        {assignment.technician_name || getTechnicianName(assignment.technician_id)}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="py-3 px-4">
                                                <div className="flex items-center gap-2">
                                                    <MapPin className="w-4 h-4 text-slate-400" />
                                                    <span className="text-sm text-slate-700 dark:text-slate-300">
                                                        {assignment.site_name || getSiteName(assignment.site_id)}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="py-3 px-4">
                                                <button
                                                    onClick={() => handleTogglePrimary(assignment)}
                                                    className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                                                    title={assignment.is_primary ? 'Unset as primary' : 'Set as primary'}
                                                >
                                                    {assignment.is_primary ? (
                                                        <Star className="w-5 h-5 text-yellow-500 fill-yellow-500" />
                                                    ) : (
                                                        <StarOff className="w-5 h-5 text-slate-400" />
                                                    )}
                                                </button>
                                            </td>
                                            <td className="py-3 px-4">
                                                <span className="text-sm text-slate-500 dark:text-slate-400">
                                                    {new Date(assignment.assigned_at).toLocaleDateString()}
                                                </span>
                                            </td>
                                            <td className="py-3 px-4 text-right">
                                                <button
                                                    onClick={() => setDeleteConfirm({ isOpen: true, id: assignment.id })}
                                                    className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                                                    title="Delete assignment"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </CardBody>
            </Card>

            {/* Add Assignment Modal */}
            <Modal
                isOpen={showAddModal}
                onClose={() => setShowAddModal(false)}
                title={t('add_assignment') || 'Add Technician Assignment'}
            >
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            {t('technician') || 'Technician'}
                        </label>
                        <Select
                            value={formData.technician_id}
                            onChange={(e) => setFormData({ ...formData, technician_id: e.target.value })}
                            options={[
                                { value: '', label: 'Select technician...' },
                                ...technicians.map((tech) => ({
                                    value: tech.id,
                                    label: tech.name || tech.username,
                                })),
                            ]}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                            {t('site') || 'Site'}
                        </label>
                        <Select
                            value={formData.site_id}
                            onChange={(e) => setFormData({ ...formData, site_id: e.target.value })}
                            options={[
                                { value: '', label: 'Select site...' },
                                ...sites.map((site) => ({
                                    value: site.id,
                                    label: site.name,
                                })),
                            ]}
                        />
                    </div>

                    <div className="flex items-center gap-2">
                        <input
                            type="checkbox"
                            id="is_primary"
                            checked={formData.is_primary}
                            onChange={(e) => setFormData({ ...formData, is_primary: e.target.checked })}
                            className="w-4 h-4 text-blue-600 border-slate-300 rounded focus:ring-blue-500"
                        />
                        <label htmlFor="is_primary" className="text-sm text-slate-700 dark:text-slate-300">
                            {t('primary_technician') || 'Primary technician (responsible)'}
                        </label>
                    </div>

                    <div className="flex gap-3 pt-4">
                        <Button onClick={handleAdd} className="flex-1">
                            {t('create') || 'Create'}
                        </Button>
                        <Button onClick={() => setShowAddModal(false)} variant="secondary" className="flex-1">
                            {t('cancel') || 'Cancel'}
                        </Button>
                    </div>
                </div>
            </Modal>

            {/* Delete Confirmation Modal */}
            <ConfirmModal
                isOpen={deleteConfirm.isOpen}
                onClose={() => setDeleteConfirm({ isOpen: false, id: '' })}
                onConfirm={handleDelete}
                title={t('delete_assignment') || 'Delete Assignment'}
                message={t('delete_assignment_confirm') || 'Are you sure you want to delete this assignment?'}
                confirmText={t('delete') || 'Delete'}
                cancelText={t('cancel') || 'Cancel'}
            />
        </div>
    );
}
