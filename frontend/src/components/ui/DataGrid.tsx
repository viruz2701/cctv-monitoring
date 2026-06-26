import React, { useState, useRef, useCallback, useMemo, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  ChevronUp, ChevronDown, ChevronsUpDown, Download, Columns,
  Search, CheckSquare, Square, FileDown, GripVertical,
  ChevronLeft, ChevronRight, Maximize2, Minus, Type,
  Settings2, RotateCcw, Filter, X, Bookmark, Save,
} from 'lucide-react';
import { DragDropContext, Droppable, Draggable, type DropResult } from '@hello-pangea/dnd';
import { useFilterStore } from '../../store/filterStore';

// ═══════════════════════════════════════════════════════════════════════
// UX-14.3.3: Advanced DataGrid
// Features:
//   - Column reorder via @hello-pangea/dnd (drag-n-drop заголовков)
//   - Column resize (перетаскивание границ)
//   - Column visibility (чекбоксы, gear icon)
//   - Pivot zone (группировка по колонке)
//   - CSV export с учётом скрытых колонок
//   - Persist в localStorage (datagrid:<page>:columns)
// ═══════════════════════════════════════════════════════════════════════

interface Column<T> {
  key: keyof T | string;
  header: string;
  render?: (item: T) => React.ReactNode;
  sortable?: boolean;
  width?: number;
  minWidth?: number;
  align?: 'left' | 'center' | 'right';
  hideable?: boolean;
  /** WO-4.2.3: Inline Editing */
  editable?: boolean;
  editType?: 'text' | 'select' | 'number';
  editOptions?: { value: string; label: string }[];
}

type DensityMode = 'compact' | 'standard' | 'comfortable';
type DataGridVariant = 'default' | 'striped' | 'bordered' | 'minimal';

const densityStyles: Record<DensityMode, { cell: string; header: string; iconSize: number }> = {
  compact: { cell: 'px-3 py-2 text-xs', header: 'px-3 py-2 text-[10px]', iconSize: 14 },
  standard: { cell: 'px-4 py-3 text-sm', header: 'px-4 py-3 text-xs', iconSize: 16 },
  comfortable: { cell: 'px-5 py-4 text-sm', header: 'px-5 py-4 text-xs', iconSize: 18 },
};

const variantStyles: Record<DataGridVariant, {
  container: string;
  headerRow: string;
  bodyRow: string;
  evenRow: string;
  border: string;
}> = {
  default: {
    container: 'bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm',
    headerRow: 'bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700',
    bodyRow: 'bg-white dark:bg-slate-800',
    evenRow: '',
    border: 'divide-y divide-slate-100 dark:divide-slate-700/50',
  },
  striped: {
    container: 'bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-sm',
    headerRow: 'bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700',
    bodyRow: 'bg-white dark:bg-slate-800',
    evenRow: 'bg-slate-50/50 dark:bg-slate-800/50',
    border: 'divide-y divide-slate-100 dark:divide-slate-700/50',
  },
  bordered: {
    container: 'bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 shadow-sm',
    headerRow: 'bg-slate-100 dark:bg-slate-800 border-b-2 border-slate-300 dark:border-slate-600',
    bodyRow: 'bg-white dark:bg-slate-800',
    evenRow: '',
    border: '',
  },
  minimal: {
    container: 'bg-transparent',
    headerRow: 'border-b border-slate-200 dark:border-slate-700',
    bodyRow: 'bg-transparent',
    evenRow: '',
    border: 'divide-y divide-slate-100/50 dark:divide-slate-700/30',
  },
};

interface BulkAction<T> {
  label: string;
  icon?: React.ReactNode;
  variant?: 'primary' | 'secondary' | 'danger';
  onClick: (selectedItems: T[]) => void;
}

interface DataGridProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string;
  onRowClick?: (item: T) => void;
  sortColumn?: string;
  sortDirection?: 'asc' | 'desc';
  onSort?: (column: string) => void;
  emptyMessage?: string;
  emptyIcon?: React.ReactNode;
  loading?: boolean;
  exportable?: boolean;
  exportFilename?: string;
  selectable?: boolean;
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
  toolbar?: React.ReactNode;
  /** Bulk action buttons shown when items are selected */
  bulkActions?: BulkAction<T>[];
  pageSize?: number;
  defaultDensity?: DensityMode;
  variant?: DataGridVariant;
  stickyHeader?: boolean;
  rowClassName?: (item: T) => string;
  onCellEdit?: (item: T, key: string, value: any) => Promise<void>;
  onColumnReorder?: (columns: Column<T>[]) => void;
  /** Уникальный ID для persistence (например, 'devices', 'work-orders') */
  persistId?: string;
  /** Включить pivot mode */
  pivotable?: boolean;
  /** Текущие pivot колонки */
  pivotColumns?: string[];
  /** Колбэк при изменении pivot */
  onPivotChange?: (columns: string[]) => void;
  /** P1-3.3: Prefetch on row hover */
  onRowHover?: (item: T) => void;
  /** P1-1.5: Enable saved filter presets */
  filterPresets?: boolean;
  /** P1-1.5: Current filter state for save/share */
  filterState?: { filters: Record<string, string>; sort: { column: string; direction: 'asc' | 'desc' } };
  /** P1-1.5: Apply a saved filter preset */
  onApplyFilterPreset?: (filterState: { filters: Record<string, string>; sort: { column: string; direction: 'asc' | 'desc' } }) => void;
}

function getValue<T>(item: T, key: keyof T | string): unknown {
  if (typeof key === 'string' && key.includes('.')) {
    return key.split('.').reduce<unknown>((acc, part) => {
      if (acc && typeof acc === 'object') {
        return (acc as Record<string, unknown>)[part];
      }
      return undefined;
    }, item);
  }
  return item[key as keyof T];
}

// ── RenderTable sub-component (React.memo for row rendering optimization) ──
interface RenderTableProps<T> {
  columns: Column<T>[];
  data: T[];
  keyExtractor: (item: T) => string;
  selectable?: boolean;
  selectedIds?: Set<string>;
  onRowClick?: (item: T) => void;
  onRowHover?: (item: T) => void;
  vs: any;
  ds: any;
  columnWidths: Record<string, number>;
  search: string;
  emptyMessage: string;
  emptyIcon?: React.ReactNode;
  onCellEdit?: (item: T, key: string, value: any) => Promise<void>;
  editingCell: { rowId: string; colKey: string } | null;
  editValue: string;
  editSaving: boolean;
  editInputRef: React.RefObject<HTMLInputElement | HTMLSelectElement | null>;
  setEditingCell: (cell: { rowId: string; colKey: string } | null) => void;
  setEditValue: (v: string) => void;
  setEditSaving: (v: boolean) => void;
  toggleSelect: (id: string) => void;
  rowClassName?: (item: T) => string;
  colCount: number;
  alignClasses: Record<string, string>;
  variant: DataGridVariant;
}

const RenderTable = React.memo(function RenderTableInner<T>({
  columns,
  data,
  keyExtractor,
  selectable,
  selectedIds,
  onRowClick,
  onRowHover,
  vs,
  ds,
  columnWidths,
  search,
  emptyMessage,
  emptyIcon,
  onCellEdit,
  editingCell,
  editValue,
  editSaving,
  editInputRef,
  setEditingCell,
  setEditValue,
  setEditSaving,
  toggleSelect,
  rowClassName,
  colCount,
  alignClasses,
  variant,
}: RenderTableProps<T>) {
  return (
    <table className="w-full" role="grid">
      <thead className={vs.headerRow}>
        <tr>
          {selectable && <th className="w-10 px-3 py-3" scope="col" />}
          {columns.map((column) => {
            const colKey = String(column.key);
            const width = columnWidths[colKey] || column.width;
            return (
              <th
                key={colKey}
                className={`${ds.header} font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${alignClasses[column.align || 'left']}`}
                style={{ width: width ? `${width}px` : undefined, minWidth: column.minWidth }}
                scope="col"
              >
                <div className="flex items-center gap-1">
                  {column.header}
                </div>
              </th>
            );
          })}
        </tr>
      </thead>
      <tbody className={vs.border}>
        {data.length === 0 ? (
          <tr>
            <td colSpan={colCount} className="px-4 py-12 text-center text-slate-500 dark:text-slate-300" role="status">
              <span>{search ? 'No results match your search' : emptyMessage}</span>
            </td>
          </tr>
        ) : (
          data.map((item, rowIdx) => {
            const id = keyExtractor(item);
            const isSelected = selectedIds?.has(id);
            const customRowClass = rowClassName?.(item) || '';
            const isEvenRow = rowIdx % 2 === 1;
            const rowBgClass = isSelected
              ? 'bg-blue-50 dark:bg-blue-900/20'
              : isEvenRow && vs.evenRow ? vs.evenRow : vs.bodyRow;

            return (
              <tr
                key={id}
                className={`group ${rowBgClass} ${onRowClick ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors' : ''} ${customRowClass}`}
                onMouseEnter={() => onRowHover?.(item)}
                onClick={() => onRowClick?.(item)}
                aria-selected={isSelected}
              >
                {selectable && (
                  <td className="w-10 px-3 py-4" onClick={(e) => e.stopPropagation()}>
                    <button
                      onClick={() => toggleSelect(id)}
                      className="text-slate-500 hover:text-blue-600"
                      aria-label={isSelected ? `Deselect row ${rowIdx + 1}` : `Select row ${rowIdx + 1}`}
                    >
                      {isSelected ? <CheckSquare size={16} className="text-blue-600" aria-hidden="true" /> : <Square size={16} aria-hidden="true" />}
                    </button>
                  </td>
                )}
                {columns.map((column) => {
                  const colKey = String(column.key);
                  const isEditing = editingCell?.rowId === id && editingCell?.colKey === colKey;

                  const startEditing = (e: React.MouseEvent) => {
                    e.stopPropagation();
                    if (!column.editable || !onCellEdit) return;
                    const currentVal = String(getValue(item, colKey) ?? '');
                    setEditValue(currentVal);
                    setEditingCell({ rowId: id, colKey });
                    setTimeout(() => {
                      if (editInputRef.current) editInputRef.current.focus();
                    }, 50);
                  };

                  const saveEdit = async () => {
                    if (!editingCell || !onCellEdit) return;
                    setEditSaving(true);
                    try {
                      const parsedValue = column.editType === 'number' ? Number(editValue) : editValue;
                      await onCellEdit(item, colKey, parsedValue);
                      setEditingCell(null);
                    } catch {
                      setEditingCell(null);
                    } finally {
                      setEditSaving(false);
                    }
                  };

                  const cancelEdit = () => setEditingCell(null);

                  const handleKeyDown = async (e: React.KeyboardEvent) => {
                    if (e.key === 'Enter') await saveEdit();
                    if (e.key === 'Escape') cancelEdit();
                  };

                  return (
                    <td
                      key={colKey}
                      className={`${ds.cell} text-slate-700 dark:text-slate-300 ${alignClasses[column.align || 'left']} ${variant === 'bordered' ? 'border border-slate-200 dark:border-slate-700' : ''} ${
                        column.editable && onCellEdit ? 'group relative cursor-pointer' : ''
                      }`}
                      onDoubleClick={startEditing}
                    >
                      {isEditing ? (
                        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                          {column.editType === 'select' && column.editOptions ? (
                            <select
                              ref={editInputRef as any}
                              className="w-full px-2 py-1 text-sm border border-blue-400 rounded bg-white focus:outline-none focus:ring-2 focus:ring-blue-400"
                              value={editValue}
                              onChange={(e) => setEditValue(e.target.value)}
                              onKeyDown={handleKeyDown}
                              onBlur={saveEdit}
                              disabled={editSaving}
                            >
                              {column.editOptions.map((opt) => (
                                <option key={opt.value} value={opt.value}>{opt.label}</option>
                              ))}
                            </select>
                          ) : (
                            <input
                              ref={editInputRef as any}
                              type={column.editType === 'number' ? 'number' : 'text'}
                              className="w-full px-2 py-1 text-sm border border-blue-400 rounded bg-white focus:outline-none focus:ring-2 focus:ring-blue-400"
                              value={editValue}
                              onChange={(e) => setEditValue(e.target.value)}
                              onKeyDown={handleKeyDown}
                              onBlur={saveEdit}
                              disabled={editSaving}
                            />
                          )}
                          {editSaving && <span className="text-xs text-blue-500">...</span>}
                        </div>
                      ) : (
                        <div className="flex items-center gap-1">
                          {column.render ? column.render(item) : String(getValue(item, column.key) ?? '')}
                          {column.editable && onCellEdit && (
                            <span className="opacity-0 group-hover:opacity-100 text-[10px] text-blue-400 ml-1 transition-opacity">✎</span>
                          )}
                        </div>
                      )}
                    </td>
                  );
                })}
              </tr>
            );
          })
        )}
      </tbody>
    </table>
  );
}) as any;

// ═══════════════════════════════════════════════════════════════════════
// Main DataGrid component
// ═══════════════════════════════════════════════════════════════════════

export function DataGrid<T>({
  data,
  columns,
  keyExtractor,
  onRowClick,
  sortColumn,
  sortDirection,
  onSort,
  emptyMessage = 'No data available',
  emptyIcon,
  loading = false,
  exportable = true,
  exportFilename = 'export.csv',
  selectable = false,
  selectedIds,
  onSelectionChange,
  toolbar,
  bulkActions,
  pageSize,
  defaultDensity = 'standard',
  variant = 'default',
  stickyHeader = false,
  rowClassName,
  onCellEdit,
  onColumnReorder,
  persistId,
  pivotable = false,
  pivotColumns = [],
  onPivotChange,
  onRowHover,
  filterPresets = false,
  filterState,
  onApplyFilterPreset,
}: DataGridProps<T>) {
  const [search, setSearch] = useState('');
  const [hiddenColumns, setHiddenColumns] = useState<Set<string>>(new Set());
  const [showColumnMenu, setShowColumnMenu] = useState(false);
  const [showDensityMenu, setShowDensityMenu] = useState(false);
  const [showSettingsMenu, setShowSettingsMenu] = useState(false);
  const [density, setDensity] = useState<DensityMode>(defaultDensity);
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({});
  const [currentPage, setCurrentPage] = useState(0);
  const resizingRef = useRef<{ key: string; startX: number; startWidth: number } | null>(null);
  const tableRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  // P1-1.5: Saved filter presets
  const [searchParams, setSearchParams] = useSearchParams();
  const { saveView, savedViews, getViewsForPage, deleteView } = useFilterStore();
  const [showFilterMenu, setShowFilterMenu] = useState(false);
  const [filterSaveName, setFilterSaveName] = useState('');

  // P1-1.5: URL sync for search/filter state
  useEffect(() => {
    const urlSearch = searchParams.get('search');
    if (urlSearch && urlSearch !== search) {
      setSearch(urlSearch);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const syncSearchToUrl = useCallback((value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) {
        next.set('search', value);
      } else {
        next.delete('search');
      }
      return next;
    }, { replace: true });
  }, [setSearchParams]);

  const alignClasses = {
    left: 'text-left',
    center: 'text-center',
    right: 'text-right',
  } as const;

  const [orderedColumnKeys, setOrderedColumnKeys] = useState<string[] | null>(null);
  // Inline editing state
  const [editingCell, setEditingCell] = useState<{ rowId: string; colKey: string } | null>(null);
  const [editValue, setEditValue] = useState<string>('');
  const [editSaving, setEditSaving] = useState(false);
  const editInputRef = useRef<HTMLInputElement | HTMLSelectElement>(null);

  // ── Persistence: загружаем конфигурацию из localStorage ──────────
  const storageKey = persistId ? `datagrid:${persistId}:columns` : null;

  useEffect(() => {
    if (!storageKey) return;
    try {
      const saved = localStorage.getItem(storageKey);
      if (!saved) return;
      const config = JSON.parse(saved);
      if (config.orderedColumnKeys) setOrderedColumnKeys(config.orderedColumnKeys);
      if (config.hiddenColumns) setHiddenColumns(new Set(config.hiddenColumns));
      if (config.columnWidths) setColumnWidths(config.columnWidths);
      if (config.density) setDensity(config.density);
    } catch {
      // игнорируем ошибки парсинга
    }
  }, [storageKey]);

  // ── Persistence: сохраняем конфигурацию ──────────────────────────
  const persistConfig = useCallback(() => {
    if (!storageKey) return;
    try {
      localStorage.setItem(storageKey, JSON.stringify({
        orderedColumnKeys,
        hiddenColumns: Array.from(hiddenColumns),
        columnWidths,
        density,
      }));
    } catch {
      // localStorage может быть переполнен
    }
  }, [storageKey, orderedColumnKeys, hiddenColumns, columnWidths, density]);

  useEffect(() => {
    persistConfig();
  }, [persistConfig]);

  // ── Pivot zone state ─────────────────────────────────────────────
  const [dragToPivot, setDragToPivot] = useState<string | null>(null);

  const visibleColumns = useMemo(() => {
    const visible = columns.filter((c) => !hiddenColumns.has(String(c.key)));
    if (orderedColumnKeys) {
      visible.sort((a, b) => {
        const ai = orderedColumnKeys.indexOf(String(a.key));
        const bi = orderedColumnKeys.indexOf(String(b.key));
        return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi);
      });
    }
    return visible;
  }, [columns, hiddenColumns, orderedColumnKeys]);

  const vs = variantStyles[variant];
  const ds = densityStyles[density];
  const pageSizeVal = pageSize || 0;

  // ── Pivot grouping ───────────────────────────────────────────────
  const pivotGroupedData = useMemo(() => {
    if (!pivotable || pivotColumns.length === 0) return null;
    const groups = new Map<string, T[]>();
    for (const item of data) {
      const key = pivotColumns.map((pc) => String(getValue(item, pc) ?? '')).join(' › ');
      const existing = groups.get(key) ?? [];
      existing.push(item);
      groups.set(key, existing);
    }
    return groups;
  }, [data, pivotColumns, pivotable]);

  const filteredData = useMemo(() => {
    if (!search.trim()) return data;
    const term = search.toLowerCase();
    return data.filter((item) =>
      columns.some((col) => {
        const raw = getValue(item, col.key);
        return String(raw ?? '').toLowerCase().includes(term);
      }),
    );
  }, [data, search, columns]);

  const totalPages = pageSizeVal > 0 ? Math.max(1, Math.ceil(filteredData.length / pageSizeVal)) : 1;
  const safePage = Math.min(currentPage, totalPages - 1);
  const pagedData = pageSizeVal > 0
    ? filteredData.slice(safePage * pageSizeVal, (safePage + 1) * pageSizeVal)
    : filteredData;
  const displayData = pagedData;

  const allSelected = useMemo(() => {
    if (!selectable || !selectedIds) return false;
    return displayData.length > 0 && displayData.every((item) => selectedIds.has(keyExtractor(item)));
  }, [displayData, selectedIds, selectable, keyExtractor]);

  const toggleSelectAll = useCallback(() => {
    if (!onSelectionChange) return;
    if (allSelected) {
      onSelectionChange(new Set());
    } else {
      onSelectionChange(new Set(displayData.map(keyExtractor)));
    }
  }, [allSelected, displayData, keyExtractor, onSelectionChange]);

  const toggleSelect = useCallback((id: string) => {
    if (!onSelectionChange || !selectedIds) return;
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onSelectionChange(next);
  }, [onSelectionChange, selectedIds]);

  // ── Column resize ────────────────────────────────────────────────
  const handleMouseDown = useCallback((e: React.MouseEvent, colKey: string) => {
    e.preventDefault();
    const th = (e.target as HTMLElement).closest('th');
    if (!th) return;
    resizingRef.current = {
      key: colKey,
      startX: e.clientX,
      startWidth: th.getBoundingClientRect().width,
    };

    const handleMouseMove = (ev: MouseEvent) => {
      if (!resizingRef.current) return;
      const delta = ev.clientX - resizingRef.current.startX;
      const newWidth = Math.max(60, resizingRef.current.startWidth + delta);
      setColumnWidths((prev) => ({ ...prev, [resizingRef.current!.key]: newWidth }));
    };

    const handleMouseUp = () => {
      resizingRef.current = null;
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, []);

  // ── CSV Export ───────────────────────────────────────────────────
  const exportCSV = useCallback(() => {
    const header = visibleColumns.map((c) => c.header).join(',');
    const exportData = pageSizeVal > 0 ? filteredData : displayData;
    const rows = exportData.map((item) =>
      visibleColumns
        .map((c) => {
          const val = getValue(item, c.key);
          const str = String(val ?? '');
          return str.includes(',') ? `"${str}"` : str;
        })
        .join(','),
    );
    const csv = [header, ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = exportFilename.replace(/\.csv$/i, '') + '.csv';
    a.click();
    URL.revokeObjectURL(url);
  }, [visibleColumns, pageSizeVal, filteredData, displayData, exportFilename]);

  // ── @hello-pangea/dnd: Column reorder ────────────────────────────
  const onDragEnd = useCallback((result: DropResult) => {
    const { source, destination, draggableId } = result;

    // Drop в pivot zone
    if (destination && destination.droppableId === 'pivot-zone') {
      const colKey = draggableId.replace('col-', '');
      if (onPivotChange && !pivotColumns.includes(colKey)) {
        onPivotChange([...pivotColumns, colKey]);
      }
      setDragToPivot(null);
      return;
    }

    // Drop в заголовки
    if (!destination || destination.droppableId !== 'columns') {
      setDragToPivot(null);
      return;
    }

    const colKeys = visibleColumns.map((c) => String(c.key));
    const srcIdx = colKeys.indexOf(draggableId.replace('col-', ''));
    const destIdx = destination.index;

    if (srcIdx === -1 || srcIdx === destIdx) {
      setDragToPivot(null);
      return;
    }

    const newOrder = [...colKeys];
    const [moved] = newOrder.splice(srcIdx, 1);
    newOrder.splice(destIdx, 0, moved);
    setOrderedColumnKeys(newOrder);
    setDragToPivot(null);
  }, [visibleColumns, onPivotChange, pivotColumns]);

  // ── Pivot: удалить колонку из pivot ──────────────────────────────
  const removePivotColumn = useCallback((colKey: string) => {
    if (!onPivotChange) return;
    onPivotChange(pivotColumns.filter((pc) => pc !== colKey));
  }, [pivotColumns, onPivotChange]);

  // ── Pivot: очистить все ──────────────────────────────────────────
  const clearPivot = useCallback(() => {
    if (!onPivotChange) return;
    onPivotChange([]);
  }, [onPivotChange]);

  // ── Сбросить конфигурацию колонок ───────────────────────────────
  const resetColumnConfig = useCallback(() => {
    setOrderedColumnKeys(null);
    setHiddenColumns(new Set());
    setColumnWidths({});
    setDensity(defaultDensity);
    if (storageKey) {
      localStorage.removeItem(storageKey);
    }
  }, [defaultDensity, storageKey]);

  // ── Gear menu: закрытие по клику вне ─────────────────────────────
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowSettingsMenu(false);
      }
    };
    if (showSettingsMenu) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [showSettingsMenu]);

  const renderSortIcon = useCallback((column: Column<T>) => {
    if (!column.sortable) return null;
    const colKey = String(column.key);
    if (sortColumn !== colKey) return <ChevronsUpDown className="w-4 h-4 text-slate-500" aria-hidden="true" />;
    return sortDirection === 'asc' ? (
      <ChevronUp className="w-4 h-4 text-blue-600" aria-hidden="true" />
    ) : (
      <ChevronDown className="w-4 h-4 text-blue-600" aria-hidden="true" />
    );
  }, [sortColumn, sortDirection]);

  const tableId = useRef(`datagrid-${Math.random().toString(36).slice(2, 9)}`).current;

  if (loading) {
    return (
      <div
        className={`rounded-xl overflow-hidden ${vs.container}`}
        role="region"
        aria-label="Loading data grid"
        aria-busy="true"
      >
        <div className="animate-pulse">
          <div className="h-12 bg-slate-100 dark:bg-slate-700/50 border-b border-slate-200 dark:border-slate-700" />
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-16 bg-white dark:bg-slate-800 border-b border-slate-100 dark:border-slate-700/50">
              <div className="flex items-center h-full px-4 gap-4">
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/4" />
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/6" />
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/5" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  const colCount = visibleColumns.length + (selectable ? 1 : 0);

  const renderTableHeader = () => (
    <thead className={`${vs.headerRow} ${stickyHeader ? 'sticky top-0 z-10' : ''}`}>
      <tr>
        {selectable && (
          <th className="w-10 px-3 py-3" scope="col" aria-label="Select all rows">
            <button
              onClick={toggleSelectAll}
              className="text-slate-500 hover:text-blue-600"
              aria-label={allSelected ? 'Deselect all rows' : 'Select all rows'}
            >
              {allSelected ? <CheckSquare size={16} className="text-blue-600" aria-hidden="true" /> : <Square size={16} aria-hidden="true" />}
            </button>
          </th>
        )}
        {visibleColumns.map((column, colIdx) => {
          const colKey = String(column.key);
          const width = columnWidths[colKey] || column.width;
          const draggingId = `col-${colKey}`;
          return (
            <Draggable key={colKey} draggableId={draggingId} index={colIdx}>
              {(provided, snapshot) => (
                <th
                  ref={provided.innerRef}
                  {...provided.draggableProps}
                  className={`relative ${ds.header} font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${alignClasses[column.align || 'left']} ${
                    column.sortable ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800' : ''
                  } ${snapshot.isDragging ? 'opacity-50 bg-blue-50 dark:bg-blue-900/20 shadow-lg z-50' : ''}`}
                  style={{
                    width: width ? `${width}px` : undefined,
                    minWidth: column.minWidth,
                    ...provided.draggableProps.style,
                  }}
                  onClick={() => column.sortable && onSort?.(colKey)}
                  scope="col"
                  aria-label={`${column.header}${column.sortable ? ', sortable' : ''}`}
                  aria-sort={
                    sortColumn === colKey
                      ? sortDirection === 'asc' ? 'ascending' : 'descending'
                      : undefined
                  }
                  tabIndex={column.sortable ? 0 : undefined}
                  onKeyDown={(e) => {
                    if (column.sortable && (e.key === 'Enter' || e.key === ' ')) {
                      e.preventDefault();
                      onSort?.(colKey);
                    }
                  }}
                >
                  <div className="flex items-center gap-1">
                    <span {...provided.dragHandleProps} className="cursor-grab active:cursor-grabbing flex-shrink-0">
                      <GripVertical size={12} className="text-slate-300 dark:text-slate-600" aria-hidden="true" />
                    </span>
                    {column.header}
                    {renderSortIcon(column)}
                  </div>
                  {/* Resize handle */}
                  <div
                    className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-blue-500 active:bg-blue-600"
                    onMouseDown={(e) => handleMouseDown(e, colKey)}
                    role="separator"
                    aria-label={`Resize ${column.header} column`}
                    tabIndex={0}
                    onKeyDown={(e) => {
                      if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
                        e.preventDefault();
                        const currentWidth = columnWidths[colKey] || column.width || 150;
                        const delta = e.key === 'ArrowRight' ? 10 : -10;
                        setColumnWidths((prev) => ({
                          ...prev,
                          [colKey]: Math.max(60, currentWidth + delta),
                        }));
                      }
                    }}
                  />
                </th>
              )}
            </Draggable>
          );
        })}
      </tr>
    </thead>
  );

  return (
    <div className={`rounded-xl overflow-hidden ${vs.container}`} role="region" aria-label="Data grid">
      {/* ── Pivot Zone ──────────────────────────────────────────────── */}
      {pivotable && (
        <div className="px-4 py-2 border-b border-dashed border-slate-200 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/50">
          <div className="flex items-center gap-2 flex-wrap">
            <Filter size={14} className="text-slate-400" aria-hidden="true" />
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">Pivot:</span>
            <Droppable droppableId="pivot-zone" direction="horizontal">
              {(provided, snapshot) => (
                <div
                  ref={provided.innerRef}
                  {...provided.droppableProps}
                  className={`flex items-center gap-1 flex-wrap min-h-[28px] px-2 rounded-md transition-colors ${
                    snapshot.isDraggingOver
                      ? 'bg-blue-50 dark:bg-blue-900/20 border border-blue-300 dark:border-blue-700'
                      : 'bg-transparent'
                  }`}
                >
                  {pivotColumns.length === 0 && !snapshot.isDraggingOver && (
                    <span className="text-xs text-slate-400 dark:text-slate-500 italic">
                      Drop columns here to group
                    </span>
                  )}
                  {pivotColumns.map((pc, idx) => {
                    const col = columns.find((c) => String(c.key) === pc);
                    return (
                      <span
                        key={pc}
                        className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300 rounded-md"
                      >
                        {col?.header || pc}
                        <button
                          onClick={() => removePivotColumn(pc)}
                          className="hover:text-indigo-900 dark:hover:text-indigo-100"
                          aria-label={`Remove ${col?.header || pc} from pivot`}
                        >
                          <X size={12} />
                        </button>
                      </span>
                    );
                  })}
                  {provided.placeholder}
                </div>
              )}
            </Droppable>
            {pivotColumns.length > 0 && (
              <button
                onClick={clearPivot}
                className="text-xs text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                aria-label="Clear pivot"
              >
                <RotateCcw size={12} />
              </button>
            )}
          </div>
        </div>
      )}

      {/* ── Bulk Action Bar ────────────────────────────────────────── */}
      {selectable && selectedIds && selectedIds.size > 0 && bulkActions && bulkActions.length > 0 && (
        <div className="flex items-center gap-3 px-4 py-3 bg-blue-50 dark:bg-blue-900/20 border-b border-blue-200 dark:border-blue-800">
          <div className="flex items-center gap-2 text-sm font-medium text-blue-700 dark:text-blue-300">
            <CheckSquare size={16} />
            <span>{selectedIds.size} selected</span>
          </div>
          <div className="h-5 w-px bg-blue-200 dark:bg-blue-700" />
          <div className="flex items-center gap-2 flex-wrap">
            {(() => {
              const selectedItems = data.filter((item) => selectedIds.has(keyExtractor(item)));
              return bulkActions.map((action, idx) => {
                const btnVariant = action.variant === 'danger'
                  ? 'bg-red-50 text-red-700 border-red-200 hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/40'
                  : action.variant === 'primary'
                    ? 'bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800 dark:hover:bg-blue-900/40'
                    : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-50 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-600 dark:hover:bg-slate-700';
                return (
                  <button
                    key={idx}
                    onClick={() => action.onClick(selectedItems)}
                    className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${btnVariant}`}
                  >
                    {action.icon}
                    {action.label}
                  </button>
                );
              });
            })()}
          </div>
          <div className="ml-auto">
            <button
              onClick={() => onSelectionChange?.(new Set())}
              className="inline-flex items-center gap-1 px-2 py-1 text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 transition-colors"
            >
              <X size={14} />
              Clear
            </button>
          </div>
        </div>
      )}

      {/* ── Toolbar ────────────────────────────────────────────────── */}
      {(exportable || selectable || toolbar || persistId || filterPresets) && (
        <div
          className="flex items-center justify-between px-4 py-2 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 gap-2 flex-wrap"
          role="toolbar"
          aria-label="Data grid toolbar"
        >
          <div className="flex items-center gap-2">
            {selectable && selectedIds && selectedIds.size > 0 && (
              <span className="text-sm text-slate-600 dark:text-slate-300" aria-live="polite">
                {selectedIds.size} selected
              </span>
            )}
            {toolbar}
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" aria-hidden="true" />
              <input
                type="text"
                placeholder="Search..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  syncSearchToUrl(e.target.value);
                  setCurrentPage(0);
                }}
                className="pl-8 pr-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 w-48 focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Search data grid"
              />
            </div>

            {/* P1-1.5: Saved Filter Presets */}
            {filterPresets && filterState && (
              <div className="relative">
                <button
                  onClick={() => {
                    setShowFilterMenu(!showFilterMenu);
                    setShowColumnMenu(false);
                    setShowDensityMenu(false);
                    setShowSettingsMenu(false);
                  }}
                  className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                  title="Saved filters"
                  aria-label="Saved filter presets"
                  aria-expanded={showFilterMenu}
                  aria-haspopup="true"
                >
                  <Bookmark size={16} aria-hidden="true" />
                </button>
                {showFilterMenu && (
                  <div
                    className="absolute right-0 top-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-2 z-50 min-w-[220px]"
                    role="menu"
                    aria-label="Filter presets menu"
                  >
                    <div className="px-2 py-1 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-1">
                      Saved Filters
                    </div>
                    {getViewsForPage(persistId || 'default').length === 0 && (
                      <div className="px-2 py-3 text-center text-xs text-slate-400">
                        No saved filters yet
                      </div>
                    )}
                    {getViewsForPage(persistId || 'default').map((view) => (
                      <button
                        key={view.id}
                        onClick={() => {
                          if (onApplyFilterPreset) {
                            onApplyFilterPreset({ filters: view.filters, sort: view.sort });
                          }
                          setShowFilterMenu(false);
                        }}
                        className="w-full text-left flex items-center gap-2 px-2 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded"
                      >
                        <Bookmark size={12} className="text-blue-500 shrink-0" />
                        <span className="flex-1 truncate">{view.name}</span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            deleteView(view.id);
                          }}
                          className="text-slate-400 hover:text-red-500"
                        >
                          <X size={12} />
                        </button>
                      </button>
                    ))}
                    <div className="border-t border-slate-100 dark:border-slate-700 mt-1 pt-1">
                      <div className="flex items-center gap-1">
                        <input
                          type="text"
                          value={filterSaveName}
                          onChange={(e) => setFilterSaveName(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && filterSaveName.trim()) {
                              saveView(
                                filterSaveName.trim(),
                                persistId || 'default',
                                { search },
                                { column: sortColumn || '', direction: sortDirection || 'asc' }
                              );
                              setFilterSaveName('');
                              setShowFilterMenu(false);
                            }
                          }}
                          placeholder="Save current filter..."
                          className="flex-1 text-xs px-2 py-1 border border-slate-200 dark:border-slate-700 rounded bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder:text-slate-400 outline-none focus:ring-1 focus:ring-blue-500"
                          maxLength={32}
                        />
                        <button
                          onClick={() => {
                            if (filterSaveName.trim()) {
                              saveView(
                                filterSaveName.trim(),
                                persistId || 'default',
                                { search },
                                { column: sortColumn || '', direction: sortDirection || 'asc' }
                              );
                              setFilterSaveName('');
                              setShowFilterMenu(false);
                            }
                          }}
                          disabled={!filterSaveName.trim()}
                          className="p-1 rounded bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
                        >
                          <Save size={12} />
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Column visibility */}
            <div className="relative">
              <button
                onClick={() => {
                  setShowColumnMenu(!showColumnMenu);
                  setShowDensityMenu(false);
                  setShowSettingsMenu(false);
                }}
                className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                title="Toggle columns"
                aria-label="Toggle column visibility"
                aria-expanded={showColumnMenu}
                aria-haspopup="true"
              >
                <Columns size={16} aria-hidden="true" />
              </button>
              {showColumnMenu && (
                <div
                  className="absolute right-0 top-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-2 z-50 min-w-[180px]"
                  role="menu"
                  aria-label="Column visibility menu"
                >
                  {columns.filter((c) => c.hideable !== false).map((col) => (
                    <label
                      key={String(col.key)}
                      className="flex items-center gap-2 px-2 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded cursor-pointer"
                      role="menuitemcheckbox"
                      aria-checked={!hiddenColumns.has(String(col.key))}
                    >
                      <input
                        type="checkbox"
                        checked={!hiddenColumns.has(String(col.key))}
                        onChange={() => {
                          setHiddenColumns((prev) => {
                            const next = new Set(prev);
                            if (next.has(String(col.key))) next.delete(String(col.key));
                            else next.add(String(col.key));
                            return next;
                          });
                        }}
                        className="rounded"
                        aria-label={`Toggle ${col.header} column`}
                      />
                      {col.header}
                    </label>
                  ))}
                </div>
              )}
            </div>

            {/* Density control */}
            <div className="relative">
              <button
                onClick={() => {
                  setShowDensityMenu(!showDensityMenu);
                  setShowColumnMenu(false);
                  setShowSettingsMenu(false);
                }}
                className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                title={`Density: ${density}`}
                aria-label={`Row density: ${density}. Click to change.`}
                aria-expanded={showDensityMenu}
                aria-haspopup="true"
              >
                {density === 'compact' ? (
                  <Minus size={16} aria-hidden="true" />
                ) : density === 'comfortable' ? (
                  <Type size={16} aria-hidden="true" />
                ) : (
                  <Maximize2 size={16} aria-hidden="true" />
                )}
              </button>
              {showDensityMenu && (
                <div
                  className="absolute right-0 top-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-1 z-50 min-w-[140px]"
                  role="menu"
                  aria-label="Density menu"
                >
                  {(['compact', 'standard', 'comfortable'] as DensityMode[]).map((m) => (
                    <button
                      key={m}
                      onClick={() => { setDensity(m); setShowDensityMenu(false); }}
                      className={`w-full text-left px-3 py-1.5 text-sm rounded ${
                        density === m
                          ? 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                          : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
                      }`}
                      role="menuitemradio"
                      aria-checked={density === m}
                    >
                      {m === 'compact' ? '🟢 Compact' : m === 'standard' ? '🟡 Standard' : '🔵 Comfortable'}
                    </button>
                  ))}
                </div>
              )}
            </div>

            {/* Settings gear (persist + reset) */}
            {persistId && (
              <div className="relative" ref={menuRef}>
                <button
                  onClick={() => {
                    setShowSettingsMenu(!showSettingsMenu);
                    setShowColumnMenu(false);
                    setShowDensityMenu(false);
                  }}
                  className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                  title="Column settings"
                  aria-label="Column settings"
                  aria-expanded={showSettingsMenu}
                  aria-haspopup="true"
                >
                  <Settings2 size={16} aria-hidden="true" />
                </button>
                {showSettingsMenu && (
                  <div
                    className="absolute right-0 top-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-1 z-50 min-w-[160px]"
                    role="menu"
                    aria-label="Column settings menu"
                  >
                    <button
                      onClick={() => { resetColumnConfig(); setShowSettingsMenu(false); }}
                      className="w-full flex items-center gap-2 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded"
                      role="menuitem"
                    >
                      <RotateCcw size={14} />
                      Reset columns
                    </button>
                  </div>
                )}
              </div>
            )}

            {/* CSV Export */}
            {exportable && (
              <button
                onClick={exportCSV}
                className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                title="Export CSV"
                aria-label="Export data as CSV"
              >
                <Download size={16} aria-hidden="true" />
              </button>
            )}
          </div>
        </div>
      )}

      {/* ── Pivot Grouped Display ──────────────────────────────────── */}
      {pivotGroupedData && (
        <div className="divide-y divide-slate-200 dark:divide-slate-700">
          {Array.from(pivotGroupedData.entries()).map(([groupKey, groupItems]) => (
            <div key={groupKey}>
              <div className="px-4 py-2 bg-slate-100 dark:bg-slate-800/50 font-medium text-sm text-slate-700 dark:text-slate-300 sticky top-0 z-10">
                {groupKey}
                <span className="ml-2 text-xs text-slate-400 dark:text-slate-500 font-normal">
                  ({groupItems.length} items)
                </span>
              </div>
              <RenderTable
                columns={visibleColumns}
                data={groupItems}
                keyExtractor={keyExtractor}
                selectable={selectable}
                selectedIds={selectedIds}
                onRowClick={onRowClick}
                onRowHover={onRowHover}
                vs={vs}
                ds={ds}
                columnWidths={columnWidths}
                search={search}
                emptyMessage={emptyMessage}
                emptyIcon={emptyIcon}
                onCellEdit={onCellEdit}
                editingCell={editingCell}
                editValue={editValue}
                editSaving={editSaving}
                editInputRef={editInputRef}
                setEditingCell={setEditingCell}
                setEditValue={setEditValue}
                setEditSaving={setEditSaving}
                toggleSelect={toggleSelect}
                rowClassName={rowClassName}
                colCount={colCount}
                alignClasses={alignClasses}
                variant={variant}
              />
            </div>
          ))}
        </div>
      )}

      {/* ── Main Table (DragDropContext for column reorder) ────────── */}
      {!pivotGroupedData && (
        <DragDropContext onDragEnd={onDragEnd}>
          <div className="overflow-x-auto" ref={tableRef}>
            <table
              className="w-full"
              role="grid"
              aria-label="Data grid table"
              aria-rowcount={displayData.length}
              aria-colcount={colCount}
            >
              <Droppable droppableId="columns" direction="horizontal">
                {(provided) => (
                  <thead
                    ref={provided.innerRef}
                    {...provided.droppableProps}
                    className={`${vs.headerRow} ${stickyHeader ? 'sticky top-0 z-10' : ''}`}
                  >
                    <tr>
                      {selectable && (
                        <th className="w-10 px-3 py-3" scope="col" aria-label="Select all rows">
                          <button
                            onClick={toggleSelectAll}
                            className="text-slate-500 hover:text-blue-600"
                            aria-label={allSelected ? 'Deselect all rows' : 'Select all rows'}
                          >
                            {allSelected ? <CheckSquare size={16} className="text-blue-600" aria-hidden="true" /> : <Square size={16} aria-hidden="true" />}
                          </button>
                        </th>
                      )}
                      {visibleColumns.map((column, colIdx) => {
                        const colKey = String(column.key);
                        const width = columnWidths[colKey] || column.width;
                        return (
                          <Draggable key={colKey} draggableId={`col-${colKey}`} index={colIdx}>
                            {(provided, snapshot) => (
                              <th
                                ref={provided.innerRef}
                                {...provided.draggableProps}
                                className={`relative ${ds.header} font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${alignClasses[column.align || 'left']} ${
                                  column.sortable ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800' : ''
                                } ${snapshot.isDragging ? 'opacity-50 bg-blue-50 dark:bg-blue-900/20 shadow-lg z-50' : ''}`}
                                style={{
                                  width: width ? `${width}px` : undefined,
                                  minWidth: column.minWidth,
                                  ...provided.draggableProps.style,
                                }}
                                onClick={() => column.sortable && onSort?.(colKey)}
                                scope="col"
                                aria-label={`${column.header}${column.sortable ? ', sortable' : ''}`}
                                aria-sort={
                                  sortColumn === colKey
                                    ? sortDirection === 'asc' ? 'ascending' : 'descending'
                                    : undefined
                                }
                                tabIndex={column.sortable ? 0 : undefined}
                                onKeyDown={(e) => {
                                  if (column.sortable && (e.key === 'Enter' || e.key === ' ')) {
                                    e.preventDefault();
                                    onSort?.(colKey);
                                  }
                                }}
                              >
                                <div className="flex items-center gap-1">
                                  <span {...provided.dragHandleProps} className="cursor-grab active:cursor-grabbing flex-shrink-0">
                                    <GripVertical size={12} className="text-slate-300 dark:text-slate-600" aria-hidden="true" />
                                  </span>
                                  {column.header}
                                  {renderSortIcon(column)}
                                </div>
                                <div
                                  className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-blue-500 active:bg-blue-600"
                                  onMouseDown={(e) => handleMouseDown(e, colKey)}
                                  role="separator"
                                  aria-label={`Resize ${column.header} column`}
                                  tabIndex={0}
                                  onKeyDown={(e) => {
                                    if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
                                      e.preventDefault();
                                      const currentWidth = columnWidths[colKey] || column.width || 150;
                                      const delta = e.key === 'ArrowRight' ? 10 : -10;
                                      setColumnWidths((prev) => ({
                                        ...prev,
                                        [colKey]: Math.max(60, currentWidth + delta),
                                      }));
                                    }
                                  }}
                                />
                              </th>
                            )}
                          </Draggable>
                        );
                      })}
                      {provided.placeholder}
                    </tr>
                  </thead>
                )}
              </Droppable>
              <tbody className={`${vs.border}`}>
                {displayData.length === 0 ? (
                  <tr>
                    <td
                      colSpan={colCount}
                      className="px-4 py-12 text-center text-slate-500 dark:text-slate-300"
                      role="status"
                    >
                      <div className="flex flex-col items-center gap-3">
                        {emptyIcon && (
                          <span className="text-slate-300 dark:text-slate-600" aria-hidden="true">
                            {emptyIcon}
                          </span>
                        )}
                        <span>{search ? 'No results match your search' : emptyMessage}</span>
                      </div>
                    </td>
                  </tr>
                ) : (
                  displayData.map((item, rowIdx) => {
                    const id = keyExtractor(item);
                    const isSelected = selectedIds?.has(id);
                    const customRowClass = rowClassName?.(item) || '';
                    const isEvenRow = rowIdx % 2 === 1;
                    const rowBgClass = isSelected
                      ? 'bg-blue-50 dark:bg-blue-900/20'
                      : isEvenRow && vs.evenRow
                        ? vs.evenRow
                        : vs.bodyRow;

                    return (
                      <tr
                        key={id}
                        className={`group ${rowBgClass} ${
                          onRowClick ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors' : ''
                        } ${customRowClass}`}
                        onMouseEnter={() => onRowHover?.(item)}
                        onClick={() => onRowClick?.(item)}
                        aria-selected={isSelected}
                        aria-rowindex={rowIdx + 1}
                      >
                        {selectable && (
                          <td className="w-10 px-3 py-4" onClick={(e) => e.stopPropagation()}>
                            <button
                              onClick={() => toggleSelect(id)}
                              className="text-slate-500 hover:text-blue-600"
                              aria-label={isSelected ? `Deselect row ${rowIdx + 1}` : `Select row ${rowIdx + 1}`}
                            >
                              {isSelected ? <CheckSquare size={16} className="text-blue-600" aria-hidden="true" /> : <Square size={16} aria-hidden="true" />}
                            </button>
                          </td>
                        )}
                        {visibleColumns.map((column) => {
                          const colKey = String(column.key);
                          const isEditing = editingCell?.rowId === id && editingCell?.colKey === colKey;

                          const startEditing = (e: React.MouseEvent) => {
                            e.stopPropagation();
                            if (!column.editable || !onCellEdit) return;
                            const currentVal = String(getValue(item, colKey) ?? '');
                            setEditValue(currentVal);
                            setEditingCell({ rowId: id, colKey });
                            setTimeout(() => {
                              if (editInputRef.current) editInputRef.current.focus();
                            }, 50);
                          };

                          const saveEdit = async () => {
                            if (!editingCell || !onCellEdit) return;
                            setEditSaving(true);
                            try {
                              const parsedValue = column.editType === 'number' ? Number(editValue) : editValue;
                              await onCellEdit(item, colKey, parsedValue);
                              setEditingCell(null);
                            } catch {
                              setEditingCell(null);
                            } finally {
                              setEditSaving(false);
                            }
                          };

                          const cancelEdit = () => setEditingCell(null);

                          const handleKeyDown = async (e: React.KeyboardEvent) => {
                            if (e.key === 'Enter') await saveEdit();
                            if (e.key === 'Escape') cancelEdit();
                          };

                          return (
                            <td
                              key={colKey}
                              className={`${ds.cell} text-slate-700 dark:text-slate-300 ${alignClasses[column.align || 'left']} ${variant === 'bordered' ? 'border border-slate-200 dark:border-slate-700' : ''} ${
                                column.editable && onCellEdit ? 'group relative cursor-pointer' : ''
                              }`}
                              onDoubleClick={startEditing}
                            >
                              {isEditing ? (
                                <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                                  {column.editType === 'select' && column.editOptions ? (
                                    <select
                                      ref={editInputRef as any}
                                      className="w-full px-2 py-1 text-sm border border-blue-400 rounded bg-white focus:outline-none focus:ring-2 focus:ring-blue-400"
                                      value={editValue}
                                      onChange={(e) => setEditValue(e.target.value)}
                                      onKeyDown={handleKeyDown}
                                      onBlur={saveEdit}
                                      disabled={editSaving}
                                    >
                                      {column.editOptions.map((opt) => (
                                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                                      ))}
                                    </select>
                                  ) : (
                                    <input
                                      ref={editInputRef as any}
                                      type={column.editType === 'number' ? 'number' : 'text'}
                                      className="w-full px-2 py-1 text-sm border border-blue-400 rounded bg-white focus:outline-none focus:ring-2 focus:ring-blue-400"
                                      value={editValue}
                                      onChange={(e) => setEditValue(e.target.value)}
                                      onKeyDown={handleKeyDown}
                                      onBlur={saveEdit}
                                      disabled={editSaving}
                                    />
                                  )}
                                  {editSaving && <span className="text-xs text-blue-500">...</span>}
                                </div>
                              ) : (
                                <div className="flex items-center gap-1">
                                  {column.render ? column.render(item) : String(getValue(item, column.key) ?? '')}
                                  {column.editable && onCellEdit && (
                                    <span className="opacity-0 group-hover:opacity-100 text-[10px] text-blue-400 ml-1 transition-opacity">✎</span>
                                  )}
                                </div>
                              )}
                            </td>
                          );
                        })}
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {pageSizeVal > 0 && totalPages > 1 && (
            <div
              className="flex items-center justify-between px-4 py-3 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900"
              role="navigation"
              aria-label="Pagination"
            >
              <span className="text-xs text-slate-500 dark:text-slate-300">
                {safePage * pageSizeVal + 1}–{Math.min((safePage + 1) * pageSizeVal, filteredData.length)} of {filteredData.length}
              </span>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => setCurrentPage(Math.max(0, safePage - 1))}
                  disabled={safePage === 0}
                  className="p-1 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 disabled:opacity-30 disabled:cursor-not-allowed"
                  aria-label="Previous page"
                >
                  <ChevronLeft size={16} aria-hidden="true" />
                </button>
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  const start = Math.max(0, Math.min(safePage - 2, totalPages - 5));
                  const page = start + i;
                  return (
                    <button
                      key={page}
                      onClick={() => setCurrentPage(page)}
                      className={`w-7 h-7 text-xs rounded ${
                        page === safePage
                          ? 'bg-blue-600 text-white'
                          : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
                      }`}
                      aria-label={`Page ${page + 1}`}
                      aria-current={page === safePage ? 'page' : undefined}
                    >
                      {page + 1}
                    </button>
                  );
                })}
                <button
                  onClick={() => setCurrentPage(Math.min(totalPages - 1, safePage + 1))}
                  disabled={safePage >= totalPages - 1}
                  className="p-1 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 disabled:opacity-30 disabled:cursor-not-allowed"
                  aria-label="Next page"
                >
                  <ChevronRight size={16} aria-hidden="true" />
                </button>
              </div>
            </div>
          )}
        </DragDropContext>
      )}
    </div>
  );
}
