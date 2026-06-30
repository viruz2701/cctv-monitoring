// ═══════════════════════════════════════════════════════════════════════
// DescriptorEditor — Web UI для создания/редактирования Protocol
// Descriptor'ов (PROTO-06, P2-EDGE).
//
// Режимы:
//   - list: список всех дескрипторов
//   - create: создание нового
//   - edit: редактирование существующего
//
// Вкладки редактора:
//   - form: форма с полями
//   - preview: JSON preview
//   - test: тестирование
//
// Compliance:
//   - WCAG 2.1 AA
//   - i18n (RU/EN)
//   - OWASP ASVS V5 (JSON Schema validation)
//   - IEC 62443 SR 7.1 (Resource availability)
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import {
  Plus,
  ArrowLeft,
  Edit2,
  Trash2,
  Eye,
  Beaker,
  RefreshCw,
  FileCode,
} from '../components/ui/Icons';
import {
  Button,
  Card,
  Badge,
  Table,
  useToast,
  EmptyState,
  ConfirmModal,
} from '../components/ui';
import {
  useDescriptorStore,
  useDescriptorMode,
  useDescriptorTab,
  useDescriptorList,
  useCurrentDescriptor,
  useDescriptorLoading,
} from '../store/descriptorStore';
import { DescriptorForm } from '../components/descriptors/DescriptorForm';
import { DescriptorPreview } from '../components/descriptors/DescriptorPreview';
import { DescriptorTester } from '../components/descriptors/DescriptorTester';
import type { EditorMode, DescriptorViewTab } from '../types/descriptor';

// ─── Tab configuration ─────────────────────────────────────────────

interface TabItem {
  key: DescriptorViewTab;
  label: string;
  icon: React.ReactNode;
}

// ─── Component ──────────────────────────────────────────────────────

export function DescriptorEditor() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { vendor: vendorParam } = useParams<{ vendor: string }>();
  const location = useLocation();
  const toast = useToast();

  // Store
  const mode = useDescriptorMode();
  const tab = useDescriptorTab();
  const descriptors = useDescriptorList();
  const descriptor = useCurrentDescriptor();
  const loading = useDescriptorLoading();
  const setMode = useDescriptorStore((s) => s.setMode);
  const setTab = useDescriptorStore((s) => s.setTab);
  const fetchDescriptors = useDescriptorStore((s) => s.fetchDescriptors);
  const loadDescriptor = useDescriptorStore((s) => s.loadDescriptor);
  const startNew = useDescriptorStore((s) => s.startNew);
  const deleteDescriptor = useDescriptorStore((s) => s.deleteDescriptor);

  // ─── Delete confirmation ───────────────────────────────────
  const [deleteTarget, setDeleteTarget] = React.useState<string | null>(null);

  // ─── Load list on mount ────────────────────────────────────
  useEffect(() => {
    fetchDescriptors();
  }, [fetchDescriptors]);

  // ─── Handle vendor param for editing ───────────────────────
  useEffect(() => {
    const isNew = location.pathname.endsWith('/new');
    if (isNew) {
      startNew();
    } else if (vendorParam) {
      loadDescriptor(vendorParam);
    }
  }, [vendorParam, location.pathname, loadDescriptor, startNew]);

  // ─── Navigate to create ────────────────────────────────────
  const handleCreate = useCallback(() => {
    startNew();
    navigate('/admin/descriptors/new');
  }, [navigate, startNew]);

  // ─── Navigate to edit ──────────────────────────────────────
  const handleEdit = useCallback(
    (vendor: string) => {
      navigate(`/admin/descriptors/${encodeURIComponent(vendor)}/edit`);
    },
    [navigate],
  );

  // ─── Handle delete ─────────────────────────────────────────
  const handleDeleteConfirm = useCallback(async () => {
    if (!deleteTarget) return;
    const ok = await deleteDescriptor(deleteTarget);
    if (ok) {
      toast.success(t('descriptors.deleteSuccess'));
    } else {
      toast.error(t('descriptors.deleteError'));
    }
    setDeleteTarget(null);
  }, [deleteTarget, deleteDescriptor, toast, t]);

  // ─── Back to list ──────────────────────────────────────────
  const handleBack = useCallback(() => {
    setMode('list');
    navigate('/admin/descriptors');
  }, [navigate, setMode]);

  // ─── Tabs for editor mode ──────────────────────────────────
  const tabs: TabItem[] = [
    { key: 'form', label: t('descriptors.tabForm'), icon: <Edit2 className="w-4 h-4" /> },
    { key: 'preview', label: t('descriptors.tabPreview'), icon: <Eye className="w-4 h-4" /> },
    { key: 'test', label: t('descriptors.tabTest'), icon: <Beaker className="w-4 h-4" /> },
  ];

  // ─── Columns for list view ─────────────────────────────────
  const columns = [
    {
      key: 'vendor',
      header: t('descriptors.vendor'),
      render: (item: typeof descriptors[0]) => (
        <span className="font-medium text-slate-800 dark:text-slate-100">
          {item.vendor}
        </span>
      ),
      sortable: true,
    },
    {
      key: 'version',
      header: t('descriptors.version'),
      render: (item: typeof descriptors[0]) => (
        <Badge variant="neutral">{item.version}</Badge>
      ),
      sortable: true,
    },
    {
      key: 'category',
      header: t('descriptors.category'),
      render: (item: typeof descriptors[0]) =>
        item.category ? (
          <span className="text-xs text-slate-500 dark:text-slate-400 capitalize">
            {item.category}
          </span>
        ) : null,
    },
    {
      key: 'endpointCount',
      header: t('descriptors.endpoints'),
      render: (item: typeof descriptors[0]) => (
        <span className="text-xs text-slate-500">{item.endpointCount}</span>
      ),
    },
    {
      key: 'updatedAt',
      header: t('descriptors.updated'),
      render: (item: typeof descriptors[0]) =>
        item.updatedAt ? (
          <span className="text-xs text-slate-400">
            {new Date(item.updatedAt).toLocaleDateString()}
          </span>
        ) : null,
    },
    {
      key: 'actions',
      header: t('descriptors.actions'),
      render: (item: typeof descriptors[0]) => (
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => handleEdit(item.vendor)}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-400"
            title={t('descriptors.edit')}
            aria-label={`${t('descriptors.edit')} ${item.vendor}`}
          >
            <Edit2 className="w-4 h-4" />
          </button>
          <button
            type="button"
            onClick={() => setDeleteTarget(item.vendor)}
            className="p-1.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-400"
            title={t('descriptors.deleteDescriptor')}
            aria-label={`${t('descriptors.deleteDescriptor')} ${item.vendor}`}
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      ),
    },
  ];

  // ─── Render: Loading ───────────────────────────────────────
  if (loading && mode === 'list') {
    return (
      <div className="flex items-center justify-center py-20">
        <RefreshCw className="w-6 h-6 text-blue-500 animate-spin" />
      </div>
    );
  }

  // ─── Render: List View ─────────────────────────────────────
  if (mode === 'list') {
    return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-slate-900 dark:text-white">
              {t('descriptors.pageTitle')}
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
              {t('descriptors.pageDescription')}
            </p>
          </div>
          <Button variant="primary" onClick={handleCreate}>
            <Plus className="w-4 h-4 mr-1" />
            {t('descriptors.createDescriptor')}
          </Button>
        </div>

        {/* Table */}
        <Card>
          {descriptors.length === 0 ? (
            <EmptyState
              icon={<FileCode className="w-12 h-12" />}
              title={t('descriptors.noDescriptors')}
              description={t('descriptors.noDescriptorsDesc')}
              action={{
                label: t('descriptors.createDescriptor'),
                onClick: handleCreate,
              }}
            />
          ) : (
            <Table
              data={descriptors}
              columns={columns}
              keyExtractor={(item) => item.vendor}
              emptyMessage={t('descriptors.noDescriptors')}
            />
          )}
        </Card>

        {/* Delete Confirmation */}
        <ConfirmModal
          isOpen={!!deleteTarget}
          onClose={() => setDeleteTarget(null)}
          onConfirm={handleDeleteConfirm}
          title={t('descriptors.deleteConfirmTitle')}
          message={t('descriptors.deleteConfirmMessage', { vendor: deleteTarget || '' })}
          confirmText={t('descriptors.deleteConfirm')}
          variant="danger"
        />
      </div>
    );
  }

  // ─── Render: Create / Edit View ────────────────────────────
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            type="button"
            onClick={handleBack}
            className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-400 transition-colors"
            aria-label={t('descriptors.back')}
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-lg font-semibold text-slate-900 dark:text-white">
              {mode === 'create'
                ? t('descriptors.createTitle')
                : t('descriptors.editTitle', { vendor: descriptor.vendor })}
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
              {mode === 'create'
                ? t('descriptors.createDescription')
                : t('descriptors.editDescription', { vendor: descriptor.vendor })}
            </p>
          </div>
        </div>

        {/* Vendor badge */}
        {descriptor.vendor && (
          <Badge variant="primary" className="text-xs">
            <FileCode className="w-3 h-3 mr-1" />
            {descriptor.vendor} {descriptor.version}
          </Badge>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-200 dark:border-slate-700">
        {tabs.map((tItem) => (
          <button
            key={tItem.key}
            type="button"
            onClick={() => setTab(tItem.key)}
            className={`
              flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2
              transition-colors -mb-px
              ${tab === tItem.key
                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 hover:border-slate-300'
              }
            `}
            role="tab"
            aria-selected={tab === tItem.key}
            aria-controls={`panel-${tItem.key}`}
          >
            {tItem.icon}
            {tItem.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div role="tabpanel" id={`panel-${tab}`} aria-label={t('descriptors.editorPanel')}>
        {tab === 'form' && <DescriptorForm />}
        {tab === 'preview' && <DescriptorPreview />}
        {tab === 'test' && <DescriptorTester />}
      </div>
    </div>
  );
}
