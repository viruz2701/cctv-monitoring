// ═══════════════════════════════════════════════════════════════════════
// DescriptorForm — форма создания/редактирования дескриптора (PROTO-06)
//
// Содержит поля Vendor, Version, Description, Auth config
// и список endpoint'ов, каждый из которых редактируется через
// EndpointEditor.
//
// Compliance:
//   - WCAG 2.1 AA (labels, aria, semantic form structure)
//   - OWASP ASVS V5 (JSON Schema валидация через Zod)
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Save } from '../../components/ui/Icons';
import { Button, Input, Select, Textarea, Alert } from '../../components/ui';
import { EndpointEditor } from './EndpointEditor';
import {
  useDescriptorStore,
  useCurrentDescriptor,
  useDescriptorDirty,
} from '../../store/descriptorStore';
import type { AuthType } from '../../types/descriptor';

// ─── Auth options ──────────────────────────────────────────────────

const AUTH_OPTIONS: { value: string; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'basic', label: 'Basic Auth' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'digest', label: 'Digest Auth' },
  { value: 'custom', label: 'Custom Header' },
];

const CATEGORY_OPTIONS: { value: string; label: string }[] = [
  { value: 'camera', label: 'Camera' },
  { value: 'nvr', label: 'NVR' },
  { value: 'dvr', label: 'DVR' },
  { value: 'switch', label: 'Switch' },
  { value: 'encoder', label: 'Encoder' },
  { value: 'decoder', label: 'Decoder' },
  { value: 'alarm', label: 'Alarm' },
  { value: 'other', label: 'Other' },
];

// ─── Component ──────────────────────────────────────────────────────

export function DescriptorForm() {
  const { t } = useTranslation();
  const descriptor = useCurrentDescriptor();
  const dirty = useDescriptorDirty();

  const updateDescriptor = useDescriptorStore((s) => s.updateDescriptor);
  const addEndpoint = useDescriptorStore((s) => s.addEndpoint);
  const updateEndpoint = useDescriptorStore((s) => s.updateEndpoint);
  const removeEndpoint = useDescriptorStore((s) => s.removeEndpoint);
  const saveDescriptor = useDescriptorStore((s) => s.saveDescriptor);
  const loading = useDescriptorStore((s) => s.loading);
  const error = useDescriptorStore((s) => s.error);
  const clearError = useDescriptorStore((s) => s.clearError);

  // ─── Collapse state per endpoint ────────────────────────────
  const [collapsed, setCollapsed] = React.useState<Record<number, boolean>>({});

  const toggleCollapse = useCallback((index: number) => {
    setCollapsed((prev) => ({ ...prev, [index]: !prev[index] }));
  }, []);

  const handleDuplicate = useCallback(
    (index: number) => {
      const original = descriptor.endpoints[index];
      if (!original) return;
      const clone = {
        ...original,
        id: `endpoint_${Date.now()}`,
        name: `${original.name} (copy)`,
      };
      const endpoints = [...descriptor.endpoints];
      endpoints.splice(index + 1, 0, clone);
      updateDescriptor({ endpoints });
    },
    [descriptor.endpoints, updateDescriptor],
  );

  const handleSave = useCallback(async () => {
    await saveDescriptor();
  }, [saveDescriptor]);

  return (
    <div className="space-y-6">
      {error && (
        <Alert variant="error" onClose={clearError}>
          {error}
        </Alert>
      )}

      {/* ── Basic Info Section ──────────────────────────────── */}
      <section className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 space-y-4">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">
          {t('descriptors.basicInfo')}
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label
              htmlFor="vendor"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.vendor')} <span className="text-red-500">*</span>
            </label>
            <Input
              id="vendor"
              value={descriptor.vendor}
              onChange={(e) => updateDescriptor({ vendor: e.target.value })}
              placeholder="Hikvision"
              required
              aria-required="true"
            />
          </div>
          <div>
            <label
              htmlFor="version"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.version')} <span className="text-red-500">*</span>
            </label>
            <Input
              id="version"
              value={descriptor.version}
              onChange={(e) => updateDescriptor({ version: e.target.value })}
              placeholder="1.0.0"
              required
              aria-required="true"
            />
          </div>
          <div>
            <label
              htmlFor="category"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.category')}
            </label>
            <Select
              id="category"
              value={descriptor.category || 'camera'}
              onChange={(e) => updateDescriptor({ category: e.target.value })}
              options={CATEGORY_OPTIONS}
            />
          </div>
        </div>

        <div>
          <label
            htmlFor="description"
            className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
          >
            {t('descriptors.description')}
          </label>
          <Textarea
            id="description"
            value={descriptor.description || ''}
            onChange={(e) => updateDescriptor({ description: e.target.value })}
            placeholder={t('descriptors.descPlaceholder')}
            rows={2}
          />
        </div>

        <div>
          <label
            htmlFor="minFirmware"
            className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
          >
            {t('descriptors.minFirmware')}
          </label>
          <Input
            id="minFirmware"
            value={descriptor.minFirmware || ''}
            onChange={(e) => updateDescriptor({ minFirmware: e.target.value })}
            placeholder="V5.6.0"
          />
        </div>
      </section>

      {/* ── Auth Section ────────────────────────────────────── */}
      <section className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 space-y-4">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">
          {t('descriptors.authConfig')}
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="authType"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.authType')}
            </label>
            <Select
              id="authType"
              value={descriptor.auth?.type || 'none'}
              onChange={(e) =>
                updateDescriptor({
                  auth: {
                    ...(descriptor.auth || {}),
                    type: e.target.value as AuthType,
                  },
                })
              }
              options={AUTH_OPTIONS}
            />
          </div>

          {descriptor.auth?.type === 'bearer' && (
            <div>
              <label
                htmlFor="tokenField"
                className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
              >
                {t('descriptors.tokenField')}
              </label>
              <Input
                id="tokenField"
                value={descriptor.auth.tokenField || ''}
                onChange={(e) =>
                  updateDescriptor({
                    auth: { ...descriptor.auth!, tokenField: e.target.value },
                  })
                }
                placeholder="access_token"
              />
            </div>
          )}

          {descriptor.auth?.type === 'custom' && (
            <div>
              <label
                htmlFor="headerName"
                className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
              >
                {t('descriptors.headerName')}
              </label>
              <Input
                id="headerName"
                value={descriptor.auth.headerName || ''}
                onChange={(e) =>
                  updateDescriptor({
                    auth: { ...descriptor.auth!, headerName: e.target.value },
                  })
                }
                placeholder="X-API-Key"
              />
            </div>
          )}
        </div>

        {descriptor.auth?.type !== 'none' && (
          <div>
            <label
              htmlFor="authEndpoint"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.authEndpoint')}
            </label>
            <Input
              id="authEndpoint"
              value={descriptor.auth?.authEndpoint || ''}
              onChange={(e) =>
                updateDescriptor({
                  auth: {
                    ...descriptor.auth!,
                    authEndpoint: e.target.value,
                  },
                })
              }
              placeholder="/api/auth/login"
            />
          </div>
        )}
      </section>

      {/* ── Endpoints Section ───────────────────────────────── */}
      <section className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">
            {t('descriptors.endpoints')}
            <span className="ml-2 text-xs font-normal text-slate-400">
              ({descriptor.endpoints.length})
            </span>
          </h3>
          <Button variant="outline" size="sm" onClick={addEndpoint}>
            <Plus className="w-4 h-4 mr-1" />
            {t('descriptors.addEndpoint')}
          </Button>
        </div>

        {descriptor.endpoints.length === 0 && (
          <p className="text-sm text-slate-400 italic py-4 text-center">
            {t('descriptors.noEndpoints')}
          </p>
        )}

        <div className="space-y-2">
          {descriptor.endpoints.map((ep, i) => (
            <EndpointEditor
              key={ep.id || i}
              endpoint={ep}
              index={i}
              onChange={updateEndpoint}
              onRemove={removeEndpoint}
              onDuplicate={handleDuplicate}
              collapsed={collapsed[i] ?? true}
              onToggleCollapse={toggleCollapse}
            />
          ))}
        </div>
      </section>

      {/* ── Actions ─────────────────────────────────────────── */}
      <div className="flex items-center justify-end gap-3 pt-2">
        <Button
          variant="primary"
          onClick={handleSave}
          disabled={loading || !descriptor.vendor || !descriptor.version}
        >
          <Save className="w-4 h-4 mr-1" />
          {loading ? t('descriptors.saving') : t('descriptors.save')}
        </Button>
      </div>
    </div>
  );
}
