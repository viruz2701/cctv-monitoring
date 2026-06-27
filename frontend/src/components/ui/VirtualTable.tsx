import React, { useRef, useMemo, useCallback } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import {
  ChevronUp, ChevronDown, ChevronsUpDown, Search,
  CheckSquare, Square, FileDown, Columns,
} from 'lucide-react';

interface Column<T> {
  key: keyof T | string;
  header: string;
  render?: (item: T) => React.ReactNode;
  sortable?: boolean;
  width?: number | string;
  minWidth?: number;
  align?: 'left' | 'center' | 'right';
  hideable?: boolean;
}

interface VirtualTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string;
  onRowClick?: (item: T) => void;
  sortColumn?: string;
  sortDirection?: 'asc' | 'desc';
  onSort?: (column: string) => void;
  emptyMessage?: string;
  loading?: boolean;
  exportable?: boolean;
  exportFilename?: string;
  selectable?: boolean;
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
  toolbar?: React.ReactNode;
  estimateRowHeight?: number;
  maxHeight?: number;
  overscan?: number;
  /** P1-3.3: Prefetch on row hover */
  onRowHover?: (item: T) => void;
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

const ALIGN_CLASSES: Record<string, string> = {
  left: 'text-left justify-start',
  center: 'text-center justify-center',
  right: 'text-right justify-end',
};

const VirtualTable = React.memo(function VirtualTableInner<T>({
  data,
  columns,
  keyExtractor,
  onRowClick,
  sortColumn,
  sortDirection,
  onSort,
  emptyMessage = 'No data available',
  loading = false,
  exportable = true,
  exportFilename = 'export.csv',
  selectable = false,
  selectedIds,
  onSelectionChange,
  toolbar,
  estimateRowHeight = 56,
  maxHeight = 600,
  overscan = 5,
  onRowHover,
}: VirtualTableProps<T>) {
  const [search, setSearch] = React.useState('');
  const [hiddenColumns, setHiddenColumns] = React.useState<Set<string>>(new Set());
  const [showColumnMenu, setShowColumnMenu] = React.useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  const visibleColumns = useMemo(
    () => columns.filter((c) => !hiddenColumns.has(String(c.key))),
    [columns, hiddenColumns],
  );

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

  const allSelected = useMemo(() => {
    if (!selectable || !selectedIds) return false;
    return filteredData.length > 0 && filteredData.every((item) => selectedIds.has(keyExtractor(item)));
  }, [filteredData, selectedIds, selectable, keyExtractor]);

  const toggleSelectAll = useCallback(() => {
    if (!onSelectionChange) return;
    if (allSelected) {
      onSelectionChange(new Set());
    } else {
      onSelectionChange(new Set(filteredData.map(keyExtractor)));
    }
  }, [allSelected, filteredData, keyExtractor, onSelectionChange]);

  const toggleSelect = useCallback(
    (id: string) => {
      if (!onSelectionChange || !selectedIds) return;
      const next = new Set(selectedIds);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      onSelectionChange(next);
    },
    [onSelectionChange, selectedIds],
  );

  // ── P1-UX.5: Auto-selection virtualization (>1000 items) ──
  const enableVirtualization = filteredData.length > 1000;
  const rowVirtualizer = useVirtualizer({
    count: filteredData.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimateRowHeight,
    overscan,
    enabled: enableVirtualization,
  });

  const exportCSV = useCallback(() => {
    const header = visibleColumns.map((c) => c.header).join(',');
    const rows = filteredData.map((item) =>
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
  }, [filteredData, visibleColumns, exportFilename]);

  const renderSortIcon = useCallback((column: Column<T>) => {
    if (!column.sortable) return null;
    const colKey = String(column.key);
    if (sortColumn !== colKey) return <ChevronsUpDown className="w-4 h-4 text-slate-400 flex-shrink-0" />;
    return sortDirection === 'asc' ? (
      <ChevronUp className="w-4 h-4 text-blue-600 flex-shrink-0" />
    ) : (
      <ChevronDown className="w-4 h-4 text-blue-600 flex-shrink-0" />
    );
  }, [sortColumn, sortDirection]);

  const gridTemplateColumns = useMemo(() => {
    const parts: string[] = [];
    if (selectable) parts.push('40px');
    visibleColumns.forEach((col) => {
      if (col.width) {
        parts.push(typeof col.width === 'number' ? `${col.width}px` : col.width);
      } else {
        parts.push('minmax(0, 1fr)');
      }
    });
    return parts.join(' ');
  }, [visibleColumns, selectable]);

  if (loading) {
    return (
      <div className="bg-white dark:bg-slate-800 rounded-xl shadow-sm overflow-hidden">
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

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm overflow-hidden">
      {/* Toolbar */}
      {(exportable || selectable || toolbar) && (
        <div className="flex items-center justify-between px-4 py-2 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            {selectable && selectedIds && selectedIds.size > 0 && (
              <span className="text-sm text-slate-600 dark:text-slate-400">
                {selectedIds.size} selected
              </span>
            )}
            {toolbar}
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8 pr-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 w-48 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div className="relative">
              <button
                onClick={() => setShowColumnMenu(!showColumnMenu)}
                className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                title="Toggle columns"
              >
                <Columns size={16} />
              </button>
              {showColumnMenu && (
                <div className="absolute right-0 top-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-2 z-50 min-w-[180px]">
                  {columns.filter((c) => c.hideable !== false).map((col) => (
                    <label
                      key={String(col.key)}
                      className="flex items-center gap-2 px-2 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 rounded cursor-pointer"
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
                      />
                      {col.header}
                    </label>
                  ))}
                </div>
              )}
            </div>
            {exportable && (
              <button
                onClick={exportCSV}
                className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                title="Export CSV"
              >
                <FileDown size={16} />
              </button>
            )}
          </div>
        </div>
      )}

      {/* Grid header */}
      <div
        className="grid bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700 sticky top-0 z-10"
        style={{ gridTemplateColumns }}
      >
        {selectable && (
          <div className="flex items-center justify-center px-2 py-3">
            <button onClick={toggleSelectAll} className="text-slate-400 hover:text-blue-600">
              {allSelected ? <CheckSquare size={16} className="text-blue-600" /> : <Square size={16} />}
            </button>
          </div>
        )}
        {visibleColumns.map((column) => {
          const colKey = String(column.key);
          const align = column.align || 'left';
          return (
            <div
              key={colKey}
              className={`flex items-center gap-1 px-4 py-3 text-xs font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${ALIGN_CLASSES[align]} ${
                column.sortable ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800 select-none' : ''
              }`}
              style={{ minWidth: column.minWidth }}
              onClick={() => column.sortable && onSort?.(colKey)}
            >
              {column.header}
              {renderSortIcon(column)}
            </div>
          );
        })}
      </div>

      {/* Virtualized body (P1-UX.5: auto-selection + seamless fallback) */}
      <div
        ref={scrollRef}
        style={{ height: Math.min(maxHeight, Math.max(200, filteredData.length * estimateRowHeight)), overflow: 'auto' }}
        className="relative"
      >
        {filteredData.length === 0 ? (
          <div className="flex items-center justify-center py-12 text-slate-500 dark:text-slate-400 text-sm">
            {search ? 'No results match your search' : emptyMessage}
          </div>
        ) : enableVirtualization ? (
          /* Virtualized rows for 10k+ items */
          <div style={{ height: rowVirtualizer.getTotalSize(), position: 'relative' }}>
            {rowVirtualizer.getVirtualItems().map((virtualRow) => {
              const item = filteredData[virtualRow.index];
              const id = keyExtractor(item);
              const isSelected = selectedIds?.has(id);
              return (
                <div
                  key={id}
                  data-index={virtualRow.index}
                  ref={rowVirtualizer.measureElement}
                  className={`grid absolute top-0 left-0 w-full border-b border-slate-100 dark:border-slate-700/50 ${
                    onRowClick ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors' : ''
                  } ${isSelected ? 'bg-blue-50 dark:bg-blue-900/20' : ''}`}
                  style={{
                    gridTemplateColumns,
                    minHeight: estimateRowHeight,
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                  onMouseEnter={() => onRowHover?.(item)}
                  onClick={() => onRowClick?.(item)}
                >
                  {selectable && (
                    <div className="flex items-center justify-center px-2" onClick={(e) => e.stopPropagation()}>
                      <button onClick={() => toggleSelect(id)} className="text-slate-400 hover:text-blue-600">
                        {isSelected ? <CheckSquare size={16} className="text-blue-600" /> : <Square size={16} />}
                      </button>
                    </div>
                  )}
                  {visibleColumns.map((column) => {
                    const align = column.align || 'left';
                    return (
                      <div
                        key={String(column.key)}
                        className={`flex items-center px-4 py-4 text-sm text-slate-700 dark:text-slate-300 ${ALIGN_CLASSES[align]}`}
                        style={{ minWidth: column.minWidth }}
                      >
                        {column.render ? column.render(item) : String(getValue(item, column.key) ?? '')}
                      </div>
                    );
                  })}
                </div>
              );
            })}
          </div>
        ) : (
          /* Seamless fallback: direct rendering for <= 1000 items */
          filteredData.map((item, rowIdx) => {
            const id = keyExtractor(item);
            const isSelected = selectedIds?.has(id);
            return (
              <div
                key={id}
                className={`grid w-full border-b border-slate-100 dark:border-slate-700/50 ${
                  onRowClick ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors' : ''
                } ${isSelected ? 'bg-blue-50 dark:bg-blue-900/20' : rowIdx % 2 === 1 ? 'bg-slate-50/50 dark:bg-slate-800/30' : ''}`}
                style={{ gridTemplateColumns, minHeight: estimateRowHeight }}
                onMouseEnter={() => onRowHover?.(item)}
                onClick={() => onRowClick?.(item)}
              >
                {selectable && (
                  <div className="flex items-center justify-center px-2" onClick={(e) => e.stopPropagation()}>
                    <button onClick={() => toggleSelect(id)} className="text-slate-400 hover:text-blue-600">
                      {isSelected ? <CheckSquare size={16} className="text-blue-600" /> : <Square size={16} />}
                    </button>
                  </div>
                )}
                {visibleColumns.map((column) => {
                  const align = column.align || 'left';
                  return (
                    <div
                      key={String(column.key)}
                      className={`flex items-center px-4 py-4 text-sm text-slate-700 dark:text-slate-300 ${ALIGN_CLASSES[align]}`}
                      style={{ minWidth: column.minWidth }}
                    >
                      {column.render ? column.render(item) : String(getValue(item, column.key) ?? '')}
                    </div>
                  );
                })}
              </div>
            );
          })
        )}
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900 text-xs text-slate-500 dark:text-slate-400">
        Showing {filteredData.length} of {data.length} items
      </div>
    </div>
  );
}) as any;

export { VirtualTable };