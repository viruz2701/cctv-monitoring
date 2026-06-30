// ═══════════════════════════════════════════════════════════════════════
// P2-API: API Versioning — Admin Page
//
// Отображает:
//   - Список зарегистрированных версий API
//   - Статус deprecated/sunset
//   - Даты релиза/deprecation/sunset
//   - Changelog для каждой версии
//   - Форма создания новой версии (admin)
//   - Форма обновления метаданных версии (admin)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../services/api';
import type { VersionInfo, ChangelogEntry } from '../services/api';
import { Button, Modal, Input, Badge, useToast } from '../components/ui';
import { Plus, Shield, Calendar, Clock, AlertTriangle } from '../components/ui/Icons';

// ─── Types ──────────────────────────────────────────────────────────

import { useConfirmAction } from '../hooks/useConfirmAction';

interface FormData {
  version: string;
  changelog: string;
}

interface UpdateFormData {
  deprecated: boolean;
  sunset: string;
  changelog: string;
}

const VERSION_REGEX = /^v[0-9]+$/;

// ─── Component ──────────────────────────────────────────────────────

export function APIVersioning() {
    const { t } = useTranslation();
    const toast = useToast();

  const [versions, setVersions] = useState<VersionInfo[]>([]);
  const [changelog, setChangelog] = useState<ChangelogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create modal
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState<FormData>({
    version: '',
    changelog: '',
  });
  const [createError, setCreateError] = useState<string | null>(null);

  // Update modal
  const [editingVersion, setEditingVersion] = useState<VersionInfo | null>(null);
  const [updateForm, setUpdateForm] = useState<UpdateFormData>({
    deprecated: false,
    sunset: '',
    changelog: '',
  });
  const [updateError, setUpdateError] = useState<string | null>(null);

  // ── Load data ─────────────────────────────────────────────────────

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [versionsData, changelogData] = await Promise.all([
        api.listVersions(),
        api.getChangelog(),
      ]);
      setVersions(versionsData.versions ?? []);
      setChangelog(changelogData.changelog ?? []);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      toast.error(message || t('versions_load_error'));
    } finally {
      setLoading(false);
    }
  }, [toast, t]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // ── Create version ────────────────────────────────────────────────

  const handleCreate = async () => {
    setCreateError(null);

    // Validation
    const ver = createForm.version.trim().toLowerCase();
    if (!VERSION_REGEX.test(ver)) {
      setCreateError(t('versions_invalid_format') || 'Invalid version format. Use v{number}, e.g. v2');
      return;
    }

    try {
      await api.createVersion(ver, createForm.changelog.trim());
      toast.success(t('versions_created') || `Version ${ver} created`);
      setShowCreateModal(false);
      setCreateForm({ version: '', changelog: '' });
      await loadData();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setCreateError(message);
    }
  };

  // ── Update version ────────────────────────────────────────────────

  const openUpdateModal = (version: VersionInfo) => {
    setEditingVersion(version);
    setUpdateForm({
      deprecated: version.deprecated,
      sunset: version.sunset || '',
      changelog: version.changelog,
    });
    setUpdateError(null);
  };

  const handleUpdate = async () => {
    if (!editingVersion) return;
    setUpdateError(null);

    try {
      await api.updateVersion(editingVersion.version, {
        deprecated: updateForm.deprecated,
        sunset: updateForm.sunset || undefined,
        changelog: updateForm.changelog,
      });
      toast.success(t('versions_updated') || `Version ${editingVersion.version} updated`);
      setEditingVersion(null);
      await loadData();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setUpdateError(message);
    }
  };

  // ── Render helpers ────────────────────────────────────────────────

  const formatDate = (dateStr: string | undefined | null): string => {
    if (!dateStr) return '—';
    try {
      return new Date(dateStr).toLocaleDateString('ru-RU', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  // ── Render ────────────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Shield className="h-6 w-6" />
            {t('api_versioning') || 'API Versioning'}
          </h1>
          <p className="text-gray-500 mt-1">
            {t('api_versioning_description') || 'Manage API versions, deprecation schedule, and changelog'}
          </p>
        </div>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="h-4 w-4 mr-2" />
          {t('versions_new') || 'New Version'}
        </Button>
      </div>

      {/* Versions Table */}
      <div className="bg-white rounded-lg shadow-sm border">
        <div className="px-4 py-3 border-b bg-gray-50">
          <h2 className="font-semibold">{t('versions_list') || 'API Versions'}</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-500 border-b">
                <th className="px-4 py-3">{t('versions_version') || 'Version'}</th>
                <th className="px-4 py-3">{t('versions_status') || 'Status'}</th>
                <th className="px-4 py-3">{t('versions_released') || 'Released'}</th>
                <th className="px-4 py-3">{t('versions_sunset') || 'Sunset'}</th>
                <th className="px-4 py-3">{t('versions_changelog') || 'Changelog'}</th>
                <th className="px-4 py-3">{t('actions') || 'Actions'}</th>
              </tr>
            </thead>
            <tbody>
              {versions.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                    {t('versions_no_data') || 'No versions registered'}
                  </td>
                </tr>
              ) : (
                versions.map((v) => (
                  <tr key={v.version} className="border-b last:border-0 hover:bg-gray-50">
                    <td className="px-4 py-3 font-mono font-medium">{v.version}</td>
                    <td className="px-4 py-3">
                      {v.deprecated ? (
                        <Badge variant="warning">
                          <AlertTriangle className="h-3 w-3 mr-1" />
                          {t('versions_deprecated') || 'Deprecated'}
                        </Badge>
                      ) : (
                        <Badge variant="success">
                          {t('versions_active') || 'Active'}
                        </Badge>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {formatDate(v.released_at)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      {v.sunset ? (
                        <span className="flex items-center gap-1 text-orange-600">
                          <Clock className="h-3 w-3" />
                          {formatDate(v.sunset)}
                        </span>
                      ) : (
                        <span className="text-gray-400">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 max-w-xs truncate">
                      {v.changelog || '—'}
                    </td>
                    <td className="px-4 py-3">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openUpdateModal(v)}
                      >
                        {t('versions_edit') || 'Edit'}
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Changelog */}
      <div className="bg-white rounded-lg shadow-sm border">
        <div className="px-4 py-3 border-b bg-gray-50">
          <h2 className="font-semibold">{t('versions_changelog') || 'Changelog'}</h2>
        </div>
        <div className="divide-y">
          {changelog.length === 0 ? (
            <div className="px-4 py-8 text-center text-gray-400">
              {t('versions_changelog_empty') || 'No changelog entries'}
            </div>
          ) : (
            changelog.map((entry, idx) => (
              <div key={idx} className="px-4 py-3 flex items-start gap-3">
                <Badge variant={entry.deprecated ? 'warning' : 'neutral'}>
                  {entry.version}
                </Badge>
                <div className="flex-1 min-w-0">
                  <p className="text-sm">{entry.change}</p>
                  <p className="text-xs text-gray-400 mt-1">
                    {formatDate(entry.date)}
                    {entry.sunset && (
                      <span className="ml-2 text-orange-500">
                        · Sunset: {formatDate(entry.sunset)}
                      </span>
                    )}
                  </p>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Create Modal */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title={t('versions_create_title') || 'Create New API Version'}
      >
        <div className="space-y-4">
          <Input
            label={t('versions_version') || 'Version'}
            placeholder="v2"
            value={createForm.version}
            onChange={(e) => setCreateForm({ ...createForm, version: e.target.value })}
          />
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {t('versions_changelog') || 'Changelog'}
            </label>
            <textarea
              className="w-full border rounded-lg px-3 py-2 text-sm min-h-[100px]"
              placeholder="Describe what's new in this version"
              value={createForm.changelog}
              onChange={(e) => setCreateForm({ ...createForm, changelog: e.target.value })}
            />
          </div>
          {createError && (
            <p className="text-sm text-red-600">{createError}</p>
          )}
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setShowCreateModal(false)}>
              {t('cancel') || 'Cancel'}
            </Button>
            <Button onClick={handleCreate}>
              {t('versions_create') || 'Create'}
            </Button>
          </div>
        </div>
      </Modal>

      {/* Update Modal */}
      <Modal
        isOpen={!!editingVersion}
        onClose={() => setEditingVersion(null)}
        title={`${t('versions_edit_title') || 'Edit Version'} ${editingVersion?.version || ''}`}
      >
        <div className="space-y-4">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={updateForm.deprecated}
              onChange={(e) => setUpdateForm({ ...updateForm, deprecated: e.target.checked })}
              className="rounded border-gray-300"
            />
            <span className="text-sm">{t('versions_mark_deprecated') || 'Mark as deprecated'}</span>
          </label>
          {updateForm.deprecated && (
            <Input
              label={t('versions_sunset_date') || 'Sunset Date (RFC 3339)'}
              type="text"
              placeholder="2027-06-30T00:00:00Z"
              value={updateForm.sunset}
              onChange={(e) => setUpdateForm({ ...updateForm, sunset: e.target.value })}
            />
          )}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {t('versions_changelog') || 'Changelog'}
            </label>
            <textarea
              className="w-full border rounded-lg px-3 py-2 text-sm min-h-[100px]"
              value={updateForm.changelog}
              onChange={(e) => setUpdateForm({ ...updateForm, changelog: e.target.value })}
            />
          </div>
          {updateError && (
            <p className="text-sm text-red-600">{updateError}</p>
          )}
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setEditingVersion(null)}>
              {t('cancel') || 'Cancel'}
            </Button>
            <Button onClick={handleUpdate}>
              {t('versions_save') || 'Save'}
            </Button>
          </div>
        </div>
      </Modal>

    </div>
  );
}
