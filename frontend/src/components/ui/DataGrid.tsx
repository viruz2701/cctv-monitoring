import React, { useState, useRef, useCallback, useMemo } from 'react';
import {
  ChevronUp, ChevronDown, ChevronsUpDown, Download, Columns,
  Search, CheckSquare, Square, FileDown, GripVertical,
  ChevronLeft, ChevronRight, Maximize2, Minus, Type,
} from 'lucide-react';

interface Column<T> {
  key: keyof T | string;
  header: string;
  render?: (item: T) => React.ReactNode;
  sortable?: boolean;
  width?: number;
  minWidth?: number;
  align?: 'left' | 'center' | 'right';
  hideable?: boolean;
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
  pageSize?: number;
  defaultDensity?: DensityMode;
  variant?: DataGridVariant;
  stickyHeader?: boolean;
  rowClassName?: (item: T) => string;
  onColumnReorder?: (columns: Column<T>[]) => void;
}

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
  pageSize,
  defaultDensity = 'standard',
  variant = 'default',
  stickyHeader = false,
  rowClassName,
}: DataGridProps<T>) {
  const [search, setSearch] = useState('');
  const [hiddenColumns, setHiddenColumns] = useState<Set<string>>(new Set());
  const [showColumnMenu, setShowColumnMenu] = useState(false);
  const [showDensityMenu, setShowDensityMenu] = useState(false);
  const [density, setDensity] = useState<DensityMode>(defaultDensity);
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({});
  const [currentPage, setCurrentPage] = useState(0);
  const [draggedCol, setDraggedCol] = useState<number | null>(null);
  const [dragOverCol, setDragOverCol] = useState<number | null>(null);
  const resizingRef = useRef<{ key: string; startX: number; startWidth: number } | null>(null);
  const tableRef = useRef<HTMLDivElement>(null);

  const alignClasses = {
    left: 'text-left',
    center: 'text-center',
    right: 'text-right',
  } as const;

  const [orderedColumnKeys, setOrderedColumnKeys] = useState<string[] | null>(null);

  const visibleColumns = useMemo(() => {
    const visible = columns.filter((c) => !hiddenColumns.has(String(c.key)));
    if (orderedColumnKeys) {
      const orderMap = new Map(columns.map((c, i) => [String(c.key), i]));
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
    a.download = exportFilename;
    a.click();
    URL.revokeObjectURL(url);
  }, [visibleColumns, pageSizeVal, filteredData, displayData, exportFilename]);

  const handleDragStart = useCallback((index: number) => {
    setDraggedCol(index);
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent, index: number) => {
    e.preventDefault();
    setDragOverCol(index);
  }, []);

  const handleDragEnd = useCallback(() => {
    if (draggedCol !== null && dragOverCol !== null && draggedCol !== dragOverCol) {
      const newOrder = [...visibleColumns];
      const [moved] = newOrder.splice(draggedCol, 1);
      newOrder.splice(dragOverCol, 0, moved);
      setOrderedColumnKeys(newOrder.map((c) => String(c.key)));
    }
    setDraggedCol(null);
    setDragOverCol(null);
  }, [draggedCol, dragOverCol, visibleColumns]);

  const renderSortIcon = useCallback((column: Column<T>) => {
    if (!column.sortable) return null;
    const colKey = String(column.key);
    if (sortColumn !== colKey) return <ChevronsUpDown className="w-4 h-4 text-slate-400" aria-hidden="true" />;
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

  return (
    <div
      className={`rounded-xl overflow-hidden ${vs.container}`}
      role="region"
      aria-label="Data grid"
    >
      {/* Toolbar */}
      {(exportable || selectable || toolbar) && (
        <div
          className="flex items-center justify-between px-4 py-2 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 gap-2 flex-wrap"
          role="toolbar"
          aria-label="Data grid toolbar"
        >
          <div className="flex items-center gap-2">
            {selectable && selectedIds && selectedIds.size > 0 && (
              <span className="text-sm text-slate-600 dark:text-slate-400" aria-live="polite">
                {selectedIds.size} selected
              </span>
            )}
            {toolbar}
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" aria-hidden="true" />
              <input
                type="text"
                placeholder="Search..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setCurrentPage(0);
                }}
                className="pl-8 pr-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 w-48 focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Search data grid"
              />
            </div>
            <div className="relative">
              <button
                onClick={() => {
                  setShowColumnMenu(!showColumnMenu);
                  setShowDensityMenu(false);
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
            {exportable && (
              <button
                onClick={exportCSV}
                className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                title="Export CSV"
                aria-label="Export data as CSV"
              >
                <FileDown size={16} aria-hidden="true" />
              </button>
            )}
          </div>
        </div>
      )}

      {/* Table */}
      <div className="overflow-x-auto" ref={tableRef}>
        <table
          className="w-full"
          role="grid"
          aria-label="Data grid table"
          aria-rowcount={displayData.length}
          aria-colcount={colCount}
        >
          <thead
            className={`${vs.headerRow} ${stickyHeader ? 'sticky top-0 z-10' : ''}`}
          >
            <tr>
              {selectable && (
                <th
                  className="w-10 px-3 py-3"
                  scope="col"
                  aria-label="Select all rows"
                >
                  <button
                    onClick={toggleSelectAll}
                    className="text-slate-400 hover:text-blue-600"
                    aria-label={allSelected ? 'Deselect all rows' : 'Select all rows'}
                  >
                    {allSelected ? <CheckSquare size={16} className="text-blue-600" aria-hidden="true" /> : <Square size={16} aria-hidden="true" />}
                  </button>
                </th>
              )}
              {visibleColumns.map((column, colIdx) => {
                const colKey = String(column.key);
                const width = columnWidths[colKey] || column.width;
                const isDragOver = dragOverCol === colIdx;
                return (
                  <th
                    key={colKey}
                    draggable
                    onDragStart={() => handleDragStart(colIdx)}
                    onDragOver={(e) => handleDragOver(e, colIdx)}
                    onDragEnd={handleDragEnd}
                    className={`relative ${ds.header} font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${alignClasses[column.align || 'left']} ${
                      column.sortable ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800' : ''
                    } ${isDragOver ? 'border-l-2 border-l-blue-500' : ''}`}
                    style={{ width: width ? `${width}px` : undefined, minWidth: column.minWidth }}
                    onClick={() => column.sortable && onSort?.(colKey)}
                    scope="col"
                    aria-label={`${column.header}${column.sortable ? ', sortable' : ''}`}
                    aria-sort={
                      sortColumn === colKey
                        ? sortDirection === 'asc'
                          ? 'ascending'
                          : 'descending'
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
                      <GripVertical size={12} className="text-slate-300 dark:text-slate-600 cursor-grab active:cursor-grabbing flex-shrink-0" aria-hidden="true" />
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
                            [colKey]: Math.max(60, (currentWidth) + delta),
                          }));
                        }
                      }}
                    />
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody className={`${vs.border}`}>
            {displayData.length === 0 ? (
              <tr>
                <td
                  colSpan={colCount}
                  className="px-4 py-12 text-center text-slate-500 dark:text-slate-400"
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
                    onClick={() => onRowClick?.(item)}
                    aria-selected={isSelected}
                    aria-rowindex={rowIdx + 1}
                  >
                    {selectable && (
                      <td className="w-10 px-3 py-4" onClick={(e) => e.stopPropagation()}>
                        <button
                          onClick={() => toggleSelect(id)}
                          className="text-slate-400 hover:text-blue-600"
                          aria-label={isSelected ? `Deselect row ${rowIdx + 1}` : `Select row ${rowIdx + 1}`}
                        >
                          {isSelected ? <CheckSquare size={16} className="text-blue-600" aria-hidden="true" /> : <Square size={16} aria-hidden="true" />}
                        </button>
                      </td>
                    )}
                    {visibleColumns.map((column) => (
                      <td
                        key={String(column.key)}
                        className={`${ds.cell} text-slate-700 dark:text-slate-300 ${alignClasses[column.align || 'left']} ${variant === 'bordered' ? 'border border-slate-200 dark:border-slate-700' : ''}`}
                      >
                        {column.render ? column.render(item) : String(getValue(item, column.key) ?? '')}
                      </td>
                    ))}
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
          <span className="text-xs text-slate-500 dark:text-slate-400">
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
                      : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
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
    </div>
  );
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
