// ═══════════════════════════════════════════════════════════════════════
// SpareParts — управление запчастями с Grid/Table toggle, bulk ops, timeline
// P0-3: Редизайн SpareParts (Shelf.nu паттерн)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useMemo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  useSpareParts,
  useLowStockParts,
  useSparePartCategories,
  useCreateSparePart,
  useUpdateSparePart,
  useDeleteSparePart,
  useAdjustStock,
  useCreateSparePartCategory,
  useUpdateSparePartCategory,
  useDeleteSparePartCategory,
} from '../hooks/useApiQuery';
import type { ToastOptions } from '../components/ui';
import {
  Button, Modal, Input, useToast, EmptyState, SkeletonCard, ConfirmModal,
} from '../components/ui';
import { PartsGridView } from '../components/spare-parts/PartsGridView';
import { PartHistoryTimeline } from '../components/spare-parts/PartHistoryTimeline';
import type { PartCardPart } from '../components/spare-parts/PartCard';
import { useConfirmAction } from '../hooks/useConfirmAction';
import {
  Plus, Search, AlertTriangle, RefreshCw, ShoppingCart, Tag,
  Edit, Trash2, Upload, MapPin, Download, FileSpreadsheet,
  History,
} from '../components/ui/Icons';
import type { SparePart, SparePartCategory } from '../services/sparePartsApi';

// ─── Local Types ─────────────────────────────────────────────────────

interface ToastApi {
  success: (options: ToastOptions) => string;
  error: (options: ToastOptions) => string;
  warning: (options: ToastOptions) => string;
  info: (options: ToastOptions) => string;
}

interface LowStockPart extends SparePart {
  suggested_order: number;
}

const CATEGORY_COLORS = [
  '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
  '#ec4899', '#06b6d4', '#f97316', '#6366f1', '#14b8a6',
];

export const SpareParts: React.FC = () => {
  const { t } = useTranslation();
  const toast = useToast();
  const { confirm, ConfirmDialog } = useConfirmAction();
  const { data: spareParts = [], isLoading: loading } = useSpareParts();
  const { data: lowStockParts = [] } = useLowStockParts();
  const { data: categories = [] } = useSparePartCategories();
  const createCategoryMut = useCreateSparePartCategory();
  const updateCategoryMut = useUpdateSparePartCategory();
  const deleteCategoryMut = useDeleteSparePartCategory();
  const updateSparePartMut = useUpdateSparePart();
  const deleteSparePartMut = useDeleteSparePart();
  const adjustStockMut = useAdjustStock();

  // ── UI State ──────────────────────────────────────────────────────
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showAlertsModal, setShowAlertsModal] = useState(false);
  const [showCategoryModal, setShowCategoryModal] = useState(false);
  const [showBulkStockModal, setShowBulkStockModal] = useState(false);
  const [showBulkLocationModal, setShowBulkLocationModal] = useState(false);
  const [showHistoryModal, setShowHistoryModal] = useState(false);
  const [historyPartId, setHistoryPartId] = useState<string | null>(null);
  const [editingCategory, setEditingCategory] = useState<SparePartCategory | null>(null);
  const [search, setSearch] = useState('');
  const [lowStockOnly, setLowStockOnly] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Bulk operation forms
  const [bulkStockValue, setBulkStockValue] = useState(0);
  const [bulkLocationValue, setBulkLocationValue] = useState('');

  const [catForm, setCatForm] = useState({ name: '', description: '', color: '#3b82f6' });

  // ── Derived Data ──────────────────────────────────────────────────
  const filtered = useMemo(() => {
    let list = spareParts;
    if (search) {
      const term = search.toLowerCase();
      list = list.filter(
        (p) => p.name.toLowerCase().includes(term) || p.sku.toLowerCase().includes(term),
      );
    }
    if (lowStockOnly) {
      list = list.filter((p) => p.stock <= p.min_stock);
    }
    return list;
  }, [spareParts, search, lowStockOnly]);

  const lowStockCount = useMemo(
    () => spareParts.filter((p) => p.stock <= p.min_stock).length,
    [spareParts],
  );

  // Map SparePart → PartCardPart
  const mapToPartCard = useCallback((p: typeof spareParts[0]) => ({
    id: p.id,
    name: p.name,
    sku: p.sku,
    category: categories.find((c) => c.id === p.category)?.name || p.category,
    stock: p.stock,
    min_stock: p.min_stock,
    location: p.location,
    cost: p.cost,
    supplier: p.supplier,
  }), [categories]);

  const mappedParts = useMemo(() => filtered.map(mapToPartCard), [filtered, mapToPartCard]);

  // ── Bulk Actions ──────────────────────────────────────────────────
  const selectedParts = useMemo(
    () => spareParts.filter((p) => selectedIds.has(p.id)),
    [spareParts, selectedIds],
  );

  const handleBulkUpdateStock = useCallback(async () => {
    try {
      for (const part of selectedParts) {
        const diff = bulkStockValue - part.stock;
        await adjustStockMut.mutateAsync({ id: part.id, quantity: diff });
      }
      toast.success(`Stock updated for ${selectedParts.length} parts`);
      setShowBulkStockModal(false);
      setSelectedIds(new Set());
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to update stock';
      toast.error(message);
    }
  }, [selectedParts, bulkStockValue, adjustStockMut, toast]);

  const handleBulkChangeLocation = useCallback(async () => {
    try {
      for (const part of selectedParts) {
        await updateSparePartMut.mutateAsync({ id: part.id, data: { location: bulkLocationValue } });
      }
      toast.success(`Location updated for ${selectedParts.length} parts`);
      setShowBulkLocationModal(false);
      setSelectedIds(new Set());
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to update location';
      toast.error(message);
    }
  }, [selectedParts, bulkLocationValue, updateSparePartMut, toast]);

  const handleExportSelectedCSV = useCallback(() => {
    const header = 'Name,SKU,Category,Stock,Min Stock,Location,Cost,Supplier';
    const rows = selectedParts.map((p) => {
      const catName = categories.find((c) => c.id === p.category)?.name || p.category || '';
      return [
        `"${p.name}"`,
        p.sku,
        `"${catName}"`,
        p.stock,
        p.min_stock,
        `"${p.location || ''}"`,
        p.cost,
        `"${p.supplier || ''}"`,
      ].join(',');
    });
    const csv = [header, ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `spare-parts-selected-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    toast.success(`Exported ${selectedParts.length} parts`);
  }, [selectedParts, categories, toast]);

  const bulkActions = useMemo(() => [
    {
      label: 'Update Stock',
      icon: <RefreshCw size={14} />,
      variant: 'primary' as const,
      onClick: (_items: PartCardPart[]) => {
        setShowBulkStockModal(true);
      },
    },
    {
      label: 'Change Location',
      icon: <MapPin size={14} />,
      variant: 'secondary' as const,
      onClick: (_items: PartCardPart[]) => {
        setShowBulkLocationModal(true);
      },
    },
    {
      label: 'Export CSV',
      icon: <Download size={14} />,
      variant: 'secondary' as const,
      onClick: () => handleExportSelectedCSV(),
    },
  ], [handleExportSelectedCSV]);

  // ── Handlers ──────────────────────────────────────────────────────
  const handleViewPart = useCallback((id: string) => {
    setHistoryPartId(id);
    setShowHistoryModal(true);
  }, []);

  const handleOrderPart = useCallback((id: string) => {
    const part = spareParts.find((p) => p.id === id);
    if (part) {
      const suggested = Math.max(part.min_stock * 2 - part.stock, part.min_stock);
      adjustStockMut.mutateAsync({ id, quantity: suggested });
      toast.success(`Ordered ${suggested} units of ${part.name}`);
    }
  }, [spareParts, adjustStockMut, toast]);

  // ── Render ────────────────────────────────────────────────────────
  return (
    <div className="p-6">
      {/* ── Header ────────────────────────────────────────────────── */}
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{t('spare_parts')}</h1>
          {lowStockCount > 0 && (
            <button
              onClick={() => setShowAlertsModal(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-amber-50 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-700 rounded-lg text-amber-700 dark:text-amber-400 text-sm font-medium hover:bg-amber-100 dark:hover:bg-amber-900/50 transition-colors"
            >
              <AlertTriangle className="w-4 h-4" />
              {lowStockCount} {t('low_stock_alerts')}
            </button>
          )}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => {
              setEditingCategory(null);
              setCatForm({ name: '', description: '', color: CATEGORY_COLORS[0] });
              setShowCategoryModal(true);
            }}
            icon={<Tag size={20} />}
          >
            {t('manage_categories') || 'Categories'}
          </Button>
          <Button onClick={() => setShowCreateModal(true)} icon={<Plus size={20} />}>
            {t('add_part')}
          </Button>
        </div>
      </div>

      {/* ── Search ────────────────────────────────────────────────── */}
      <div className="mb-6">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder={t('search_parts')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 border border-slate-300 dark:border-slate-600 rounded-xl bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      {/* ── Parts Grid/Table ──────────────────────────────────────── */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <SkeletonCard count={6} />
        </div>
      ) : mappedParts.length === 0 && !lowStockOnly ? (
        <EmptyState
          icon={search ? <Search className="w-12 h-12" /> : <ShoppingCart className="w-12 h-12" />}
          title={search ? (t('no_search_results') || 'No results') : (t('no_parts') || 'No spare parts')}
          description={search ? (t('try_different_search') || 'Try a different search term') : (t('add_first_part_desc') || 'Add your first spare part to start tracking inventory')}
          hint={search ? undefined : (t('spare_parts_hint') || 'Track stock levels, reorder points, and costs')}
          action={search ? undefined : {
            label: t('add_part') || 'Add Part',
            onClick: () => setShowCreateModal(true),
          }}
          size="md"
        />
      ) : (
        <PartsGridView
          parts={mappedParts}
          selectedIds={selectedIds}
          onSelectionChange={setSelectedIds}
          lowStockOnly={lowStockOnly}
          onLowStockToggle={() => setLowStockOnly(!lowStockOnly)}
          bulkActions={bulkActions}
          onViewPart={handleViewPart}
          onOrderPart={handleOrderPart}
          loading={loading}
        />
      )}

      {/* ── Low Stock Parts Category Management ────────────────────── */}
      {/* (kept from original - moved below grid) */}

      {/* ── Category Management Modal ──────────────────────────────── */}
      <Modal isOpen={showCategoryModal} onClose={() => setShowCategoryModal(false)} title={editingCategory ? t('edit_category') || 'Edit Category' : t('add_category') || 'Add Category'} size="md">
        <div className="space-y-4">
          <div className="space-y-2 max-h-60 overflow-y-auto">
            {categories.map((cat: SparePartCategory) => (
              <div key={cat.id} className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-2">
                  <div className="w-4 h-4 rounded-full" style={{ backgroundColor: cat.color || '#3b82f6' }} />
                  <span className="font-medium text-sm">{cat.name}</span>
                  {cat.description && <span className="text-xs text-slate-500">{cat.description}</span>}
                </div>
                <div className="flex gap-1">
                  <button
                    onClick={() => {
                      setEditingCategory(cat);
                      setCatForm({ name: cat.name, description: cat.description || '', color: cat.color || '#3b82f6' });
                    }}
                    className="p-1 hover:bg-slate-200 dark:hover:bg-slate-700 rounded"
                  >
                    <Edit className="w-4 h-4 text-slate-500" />
                  </button>
                  <button
                    onClick={async () => {
                      const ok = await confirm({ title: t('delete_category') || 'Delete Category', message: t('delete_category_confirm') || 'Are you sure you want to delete this category?', variant: 'danger' });
                      if (ok) deleteCategoryMut.mutateAsync(cat.id);
                    }}
                    className="p-1 hover:bg-red-100 dark:hover:bg-red-900/20 rounded"
                  >
                    <Trash2 className="w-4 h-4 text-red-500" />
                  </button>
                </div>
              </div>
            ))}
            {categories.length === 0 && (
              <p className="text-sm text-slate-500 text-center py-4">{t('no_categories') || 'No categories yet'}</p>
            )}
          </div>

          <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
            <h4 className="text-sm font-semibold mb-3">
              {editingCategory ? t('edit_category') || 'Edit Category' : t('new_category') || 'New Category'}
            </h4>
            <form onSubmit={async (e) => {
              e.preventDefault();
              if (!catForm.name.trim()) return;
              try {
                if (editingCategory) {
                  await updateCategoryMut.mutateAsync({ id: editingCategory.id, data: catForm });
                  toast.success(t('category_updated') || 'Category updated');
                } else {
                  await createCategoryMut.mutateAsync(catForm);
                  toast.success(t('category_created') || 'Category created');
                }
                setCatForm({ name: '', description: '', color: CATEGORY_COLORS[0] });
                setEditingCategory(null);
              } catch (err: unknown) {
                const message = err instanceof Error ? err.message : (t('operation_failed') || 'Operation failed');
                toast.error(message);
              }
            }} className="space-y-3">
              <Input
                label={t('name') || 'Name'}
                value={catForm.name}
                onChange={e => setCatForm({ ...catForm, name: e.target.value })}
                required
              />
              <Input
                label={t('description') || 'Description'}
                value={catForm.description}
                onChange={e => setCatForm({ ...catForm, description: e.target.value })}
              />
              <div>
                <label className="block text-sm font-medium mb-1">{t('color') || 'Color'}</label>
                <div className="flex gap-2 flex-wrap">
                  {CATEGORY_COLORS.map(color => (
                    <button
                      key={color}
                      type="button"
                      onClick={() => setCatForm({ ...catForm, color })}
                      className={`w-8 h-8 rounded-full border-2 transition-all ${
                        catForm.color === color ? 'border-slate-900 dark:border-white scale-110' : 'border-transparent'
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>
              <div className="flex gap-2 justify-end pt-2">
                <Button variant="secondary" type="button" onClick={() => setShowCategoryModal(false)}>
                  {t('close') || 'Close'}
                </Button>
                <Button type="submit">
                  {editingCategory ? t('save') || 'Save' : t('add') || 'Add'}
                </Button>
              </div>
            </form>
          </div>
        </div>
      </Modal>

      {/* ── Create Part Modal ──────────────────────────────────────── */}
      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title={t('add_part')}>
        <CreatePartForm onClose={() => setShowCreateModal(false)} categories={categories} />
      </Modal>

      {/* ── Stock Alerts Modal ─────────────────────────────────────── */}
      <Modal isOpen={showAlertsModal} onClose={() => setShowAlertsModal(false)} title={t('stock_alerts')} size="lg">
        <LowStockAlertsContent
          lowStockParts={lowStockParts}
          adjustStock={async (id, qty) => { await adjustStockMut.mutateAsync({ id, quantity: qty }); }}
          toast={toast}
          t={t}
        />
      </Modal>

      {/* ── Bulk Stock Update Modal ────────────────────────────────── */}
      <Modal
        isOpen={showBulkStockModal}
        onClose={() => setShowBulkStockModal(false)}
        title={`Update Stock (${selectedParts.length} parts)`}
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Set new stock value for all {selectedParts.length} selected parts.
          </p>
          <Input
            type="number"
            label="New Stock Value"
            value={String(bulkStockValue)}
            onChange={(e) => setBulkStockValue(parseInt(e.target.value) || 0)}
          />
          <div className="flex gap-2 justify-end pt-2">
            <Button variant="secondary" onClick={() => setShowBulkStockModal(false)}>
              Cancel
            </Button>
            <Button onClick={handleBulkUpdateStock}>
              Update All
            </Button>
          </div>
        </div>
      </Modal>

      {/* ── Bulk Location Change Modal ─────────────────────────────── */}
      <Modal
        isOpen={showBulkLocationModal}
        onClose={() => setShowBulkLocationModal(false)}
        title={`Change Location (${selectedParts.length} parts)`}
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Set new location for all {selectedParts.length} selected parts.
          </p>
          <Input
            label="New Location"
            value={bulkLocationValue}
            onChange={(e) => setBulkLocationValue(e.target.value)}
            placeholder="e.g. Warehouse A, Shelf 12"
          />
          <div className="flex gap-2 justify-end pt-2">
            <Button variant="secondary" onClick={() => setShowBulkLocationModal(false)}>
              Cancel
            </Button>
            <Button onClick={handleBulkChangeLocation}>
              Update All
            </Button>
          </div>
        </div>
      </Modal>

      {/* ── Part History Modal ─────────────────────────────────────── */}
      <Modal
        isOpen={showHistoryModal}
        onClose={() => setShowHistoryModal(false)}
        title="Part History"
        size="lg"
      >
        {historyPartId && (
          <PartHistoryTimeline partId={historyPartId} maxEvents={50} />
        )}
      </Modal>

      {ConfirmDialog}
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// CreatePartForm — форма создания новой запчасти
// ═══════════════════════════════════════════════════════════════════════

const CreatePartForm: React.FC<{ onClose: () => void; categories: SparePartCategory[] }> = ({ onClose, categories }) => {
  const { t } = useTranslation();
  const createSparePart = useCreateSparePart();
  const createCategory = useCreateSparePartCategory();
  const [formData, setFormData] = useState({
    name: '', sku: '', category: '', stock: 0, min_stock: 5,
    location: '', cost: 0, supplier: '',
  });
  const [showNewCategory, setShowNewCategory] = useState(false);
  const [newCategoryName, setNewCategoryName] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await createSparePart.mutateAsync(formData);
    onClose();
  };

  const handleAddCategory = async () => {
    if (!newCategoryName) return;
    const cat = await createCategory.mutateAsync({ name: newCategoryName });
    setFormData({ ...formData, category: cat.id });
    setShowNewCategory(false);
    setNewCategoryName('');
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <Input label={t('name')} value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} required />
      <Input label="SKU" value={formData.sku} onChange={(e) => setFormData({ ...formData, sku: e.target.value })} />

      <div>
        <label className="block text-sm font-medium mb-1">{t('category')}</label>
        <div className="flex gap-2">
          <select
            value={formData.category}
            onChange={(e) => {
              if (e.target.value === '__new__') {
                setShowNewCategory(true);
              } else {
                setFormData({ ...formData, category: e.target.value });
              }
            }}
            className="flex-1 border rounded px-3 py-2 dark:bg-slate-800 dark:border-slate-600"
          >
            <option value="">{t('select_category') || 'Select category...'}</option>
            {categories.map((cat: SparePartCategory) => (
              <option key={cat.id} value={cat.id}>{cat.name}</option>
            ))}
            <option value="__new__">+ {t('create_new_category') || 'Create new category'}</option>
          </select>
        </div>
        {showNewCategory && (
          <div className="flex gap-2 mt-2">
            <Input
              value={newCategoryName}
              onChange={e => setNewCategoryName(e.target.value)}
              placeholder={t('category_name') || 'Category name'}
            />
            <Button size="sm" onClick={handleAddCategory}>{t('add') || 'Add'}</Button>
            <Button size="sm" variant="secondary" onClick={() => setShowNewCategory(false)}>
              {t('cancel') || 'Cancel'}
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <Input type="number" label={t('stock')} value={String(formData.stock)} onChange={(e) => setFormData({ ...formData, stock: parseInt(e.target.value) || 0 })} />
        <Input type="number" label={t('min_stock')} value={String(formData.min_stock)} onChange={(e) => setFormData({ ...formData, min_stock: parseInt(e.target.value) || 0 })} />
      </div>
      <Input type="number" label={t('cost')} value={String(formData.cost)} onChange={(e) => setFormData({ ...formData, cost: parseFloat(e.target.value) || 0 })} />
      <Input label={t('location')} value={formData.location} onChange={(e) => setFormData({ ...formData, location: e.target.value })} />
      <Input label={t('supplier')} value={formData.supplier} onChange={(e) => setFormData({ ...formData, supplier: e.target.value })} />
      <div className="flex gap-2 justify-end pt-4">
        <Button variant="secondary" onClick={onClose}>{t('cancel')}</Button>
        <Button type="submit">{t('create')}</Button>
      </div>
    </form>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// LowStockAlertsContent — содержимое модалки low-stock алертов
// ═══════════════════════════════════════════════════════════════════════

const LowStockAlertsContent: React.FC<{
  lowStockParts: SparePart[];
  adjustStock: (id: string, qty: number) => Promise<void>;
  toast: ToastApi;
  t: (key: string) => string;
}> = ({ lowStockParts, adjustStock, toast, t }) => {
  const autoReorderSuggestions = useMemo((): LowStockPart[] => {
    return lowStockParts
      .map((p) => ({
        ...p,
        suggested_order: Math.max(p.min_stock * 2 - p.stock, p.min_stock),
      }))
      .filter((p) => p.suggested_order > 0);
  }, [lowStockParts]);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 text-amber-600 dark:text-amber-400">
        <AlertTriangle className="w-5 h-5" />
        <span className="font-medium">
          {lowStockParts.length} {t('parts_below_minimum')}
        </span>
      </div>

      <div className="space-y-3">
        {lowStockParts.map((part: SparePart) => (
          <div
            key={part.id}
            className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-xl border border-slate-200 dark:border-slate-700"
          >
            <div className="flex-1">
              <p className="font-medium text-slate-900 dark:text-white">{part.name}</p>
              <p className="text-sm text-slate-500 dark:text-slate-400">SKU: {part.sku}</p>
              <div className="flex items-center gap-3 mt-1">
                <span className="text-sm">
                  {t('stock')}: <span className="font-mono font-bold text-red-600 dark:text-red-400">{part.stock}</span>
                  {' / '}
                  <span className="font-mono text-slate-500">{part.min_stock} min</span>
                </span>
                {part.supplier && (
                  <span className="text-xs text-slate-400">{t('supplier')}: {part.supplier}</span>
                )}
              </div>
            </div>

            <div className="text-right ml-4">
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">{t('reorder')}</p>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => {
                    const suggested = Math.max(part.min_stock * 2 - part.stock, part.min_stock);
                    adjustStock(part.id, suggested);
                    toast.success(`Ordered +${suggested} of ${part.name}`);
                  }}
                  className="px-3 py-1 bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-medium rounded-lg transition-colors"
                >
                  <ShoppingCart className="w-3 h-3 inline mr-1" />
                  +{Math.max(part.min_stock * 2 - part.stock, part.min_stock)}
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {autoReorderSuggestions.length > 0 && (
        <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400 mb-3">
            <RefreshCw className="w-4 h-4" />
            <span className="font-medium text-sm">{t('auto_reorder_suggestions')}</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {autoReorderSuggestions.map((part) => (
              <span key={part.id} className="inline-flex items-center gap-1 px-3 py-1.5 bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-300 rounded-lg text-sm border border-emerald-200 dark:border-emerald-800 cursor-pointer hover:bg-emerald-100 dark:hover:bg-emerald-900/40"
                onClick={() => {
                  adjustStock(part.id, part.suggested_order);
                  toast.success(`Ordered +${part.suggested_order} of ${part.name}`);
                }}
              >
                {part.name} <strong>+{part.suggested_order}</strong>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
