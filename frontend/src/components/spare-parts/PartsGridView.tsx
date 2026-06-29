// ═══════════════════════════════════════════════════════════════════════
// PartsGridView — Table/Grid toggle для списка запчастей
// Responsive: 2/3/4 колонки в grid, DataGrid в table
// Low-stock визуальный акцент (красная рамка + иконка ⚠️)
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo, useState } from 'react';
import { DataGrid } from '../ui/DataGrid';
import { PartCard, type PartCardPart } from './PartCard';
import { Badge } from '../ui/Badge';
import {
  LayoutGrid,
  Table as TableIcon,
  AlertTriangle,
  Eye,
  ShoppingCart,
} from '../ui/Icons';

// ── Types ──────────────────────────────────────────────────────────────

type ViewMode = 'grid' | 'table';

interface PartsGridViewProps {
  parts: PartCardPart[];
  /** Выбранные ID для bulk операций */
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
  /** Показать только low-stock */
  lowStockOnly?: boolean;
  onLowStockToggle?: () => void;
  /** Bulk actions */
  bulkActions?: {
    label: string;
    icon?: React.ReactNode;
    variant?: 'primary' | 'secondary' | 'danger';
    onClick: (selectedItems: PartCardPart[]) => void;
  }[];
  onViewPart?: (id: string) => void;
  onOrderPart?: (id: string) => void;
  loading?: boolean;
}

// ── Helpers ────────────────────────────────────────────────────────────

function getStockLevel(stock: number, minStock: number): 'ok' | 'low' | 'critical' | 'out' {
  if (stock <= 0) return 'out';
  if (stock > minStock * 2) return 'ok';
  if (stock > minStock) return 'low';
  return 'critical';
}

const stockBadgeVariant = (level: string) => {
  switch (level) {
    case 'out': return 'danger';
    case 'critical': return 'danger';
    case 'low': return 'warning';
    default: return 'success';
  }
};

const stockLabel: Record<string, string> = {
  ok: '🟢 OK',
  low: '🟡 Low',
  critical: '🔴 Critical',
  out: '🔴 Out',
};

// ── Component ──────────────────────────────────────────────────────────

export function PartsGridView({
  parts,
  selectedIds,
  onSelectionChange,
  lowStockOnly = false,
  onLowStockToggle,
  bulkActions,
  onViewPart,
  onOrderPart,
  loading = false,
}: PartsGridViewProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('grid');

  // Pre-compute stock levels for table rendering
  const tableData = useMemo(() => parts.map(p => ({
    ...p,
    _stockLevel: getStockLevel(p.stock, p.min_stock),
  })), [parts]);

  const lowStockCount = useMemo(
    () => parts.filter(p => getStockLevel(p.stock, p.min_stock) !== 'ok').length,
    [parts],
  );

  const columns = useMemo(() => [
    {
      key: 'sku' as const,
      header: 'SKU',
      sortable: true,
      width: 130,
      render: (item: typeof tableData[0]) => (
        <span className="font-mono text-xs text-slate-500 dark:text-slate-400">{item.sku}</span>
      ),
    },
    {
      key: 'name' as const,
      header: 'Name',
      sortable: true,
      width: 200,
      render: (item: typeof tableData[0]) => (
        <div>
          <p className="font-medium text-slate-900 dark:text-white">{item.name}</p>
          {item.category && (
            <p className="text-xs text-slate-400">{item.category}</p>
          )}
        </div>
      ),
    },
    {
      key: 'stock' as const,
      header: 'Stock',
      sortable: true,
      width: 120,
      align: 'right' as const,
      render: (item: typeof tableData[0]) => {
        const level = getStockLevel(item.stock, item.min_stock);
        return (
          <div className="flex items-center justify-end gap-2">
            <Badge variant={stockBadgeVariant(level)} size="sm">
              {stockLabel[level]}
            </Badge>
            <span className={`font-mono text-sm font-bold ${
              level === 'out' || level === 'critical'
                ? 'text-red-600 dark:text-red-400'
                : level === 'low'
                ? 'text-amber-600 dark:text-amber-400'
                : 'text-emerald-600 dark:text-emerald-400'
            }`}>
              {item.stock}
            </span>
          </div>
        );
      },
    },
    {
      key: 'min_stock' as const,
      header: 'Min',
      sortable: true,
      width: 80,
      align: 'right' as const,
    },
    {
      key: 'location' as const,
      header: 'Location',
      width: 150,
      render: (item: typeof tableData[0]) => (
        <span className="text-sm text-slate-600 dark:text-slate-300">{item.location || '—'}</span>
      ),
    },
    {
      key: 'supplier' as const,
      header: 'Supplier',
      width: 150,
      render: (item: typeof tableData[0]) => (
        <span className="text-sm text-slate-600 dark:text-slate-300">{item.supplier || '—'}</span>
      ),
    },
    {
      key: 'cost' as const,
      header: 'Cost',
      sortable: true,
      width: 100,
      align: 'right' as const,
      render: (item: typeof tableData[0]) => (
        <span className="font-mono text-sm">
          {item.cost != null ? `$${item.cost.toFixed(2)}` : '—'}
        </span>
      ),
    },
    {
      key: 'actions' as const,
      header: '',
      width: 80,
      align: 'center' as const,
      render: (item: typeof tableData[0]) => (
        <div className="flex items-center justify-center gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); onViewPart?.(item.id); }}
            className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
            title="View"
          >
            <Eye size={14} />
          </button>
          {getStockLevel(item.stock, item.min_stock) !== 'ok' && (
            <button
              onClick={(e) => { e.stopPropagation(); onOrderPart?.(item.id); }}
              className="p-1.5 text-slate-400 hover:text-amber-600 hover:bg-amber-50 dark:hover:bg-amber-900/20 rounded transition-colors"
              title="Order"
            >
              <ShoppingCart size={14} />
            </button>
          )}
        </div>
      ),
    },
  ], [onViewPart, onOrderPart]);

  return (
    <div>
      {/* ── View Toggle Bar ────────────────────────────────────────── */}
      <div className="flex items-center justify-between mb-4 gap-3">
        <div className="flex items-center gap-3">
          {/* View mode toggle */}
          <div className="flex items-center bg-slate-100 dark:bg-slate-800 rounded-lg p-0.5 border border-slate-200 dark:border-slate-700">
            <button
              onClick={() => setViewMode('grid')}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                viewMode === 'grid'
                  ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
              }`}
            >
              <LayoutGrid size={16} />
              Grid
            </button>
            <button
              onClick={() => setViewMode('table')}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                viewMode === 'table'
                  ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
              }`}
            >
              <TableIcon size={16} />
              Table
            </button>
          </div>

          {/* Low stock quick filter */}
          {onLowStockToggle && (
            <button
              onClick={onLowStockToggle}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
                lowStockOnly
                  ? 'bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800'
                  : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-300 border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
              }`}
            >
              <AlertTriangle size={16} />
              Low Stock
              {lowStockCount > 0 && (
                <span className={`ml-1 px-1.5 py-0.5 text-xs rounded-full ${
                  lowStockOnly
                    ? 'bg-red-200 dark:bg-red-800 text-red-800 dark:text-red-200'
                    : 'bg-slate-200 dark:bg-slate-600 text-slate-600 dark:text-slate-300'
                }`}>
                  {lowStockCount}
                </span>
              )}
            </button>
          )}
        </div>

        <p className="text-sm text-slate-500 dark:text-slate-400">
          {parts.length} {parts.length === 1 ? 'part' : 'parts'}
        </p>
      </div>

      {/* ── Grid View ──────────────────────────────────────────────── */}
      {viewMode === 'grid' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {parts.map((part) => {
            const level = getStockLevel(part.stock, part.min_stock);
            const isLow = level !== 'ok';
            return (
              <PartCard
                key={part.id}
                part={part}
                onView={onViewPart}
                onOrder={onOrderPart}
                lowStockAccent={isLow}
              />
            );
          })}
          {parts.length === 0 && !loading && (
            <div className="col-span-full flex items-center justify-center py-12">
              <p className="text-slate-400 dark:text-slate-500 text-sm">No parts found</p>
            </div>
          )}
        </div>
      )}

      {/* ── Table View ─────────────────────────────────────────────── */}
      {viewMode === 'table' && (
        <DataGrid
          data={tableData}
          columns={columns}
          keyExtractor={(item) => item.id}
          selectable={true}
          selectedIds={selectedIds}
          onSelectionChange={onSelectionChange}
          bulkActions={bulkActions}
          onRowClick={(item) => onViewPart?.(item.id)}
          exportable={true}
          exportFilename="spare-parts-export.csv"
          loading={loading}
          variant="striped"
          defaultDensity="compact"
          persistId="spare-parts"
          pageSize={25}
          rowClassName={(item) => {
            const level = getStockLevel(item.stock, item.min_stock);
            if (level === 'out' || level === 'critical') {
              return 'bg-red-50/50 dark:bg-red-900/10';
            }
            return '';
          }}
        />
      )}
    </div>
  );
}
