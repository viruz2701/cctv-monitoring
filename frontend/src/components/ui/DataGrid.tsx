import React, { useState, useRef, useCallback, useMemo } from 'react';
import {
  ChevronUp, ChevronDown, ChevronsUpDown, Download, Columns,
  Search, CheckSquare, Square, FileDown,
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

interface DataGridProps<T> {
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
  loading = false,
  exportable = true,
  exportFilename = 'export.csv',
  selectable = false,
  selectedIds,
  onSelectionChange,
  toolbar,
}: DataGridProps<T>) {
  const [search, setSearch] = useState('');
  const [hiddenColumns, setHiddenColumns] = useState<Set<string>>(new Set());
  const [showColumnMenu, setShowColumnMenu] = useState(false);
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({});
  const resizingRef = useRef<{ key: string; startX: number; startWidth: number } | null>(null);
  const tableRef = useRef<HTMLDivElement>(null);

  const alignClasses = {
    left: 'text-left',
    center: 'text-center',
    right: 'text-right',
  };

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

  const toggleSelectAll = () => {
    if (!onSelectionChange) return;
    if (allSelected) {
      onSelectionChange(new Set());
    } else {
      onSelectionChange(new Set(filteredData.map(keyExtractor)));
    }
  };

  const toggleSelect = (id: string) => {
    if (!onSelectionChange || !selectedIds) return;
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onSelectionChange(next);
  };

  const handleMouseDown = (e: React.MouseEvent, colKey: string) => {
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
  };

  const exportCSV = () => {
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
  };

  const renderSortIcon = (column: Column<T>) => {
    if (!column.sortable) return null;
    const colKey = String(column.key);
    if (sortColumn !== colKey) return <ChevronsUpDown className="w-4 h-4 text-slate-400" />;
    return sortDirection === 'asc' ? (
      <ChevronUp className="w-4 h-4 text-blue-600" />
    ) : (
      <ChevronDown className="w-4 h-4 text-blue-600" />
    );
  };

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

      {/* Table */}
      <div className="overflow-x-auto" ref={tableRef}>
        <table className="w-full">
          <thead>
            <tr className="bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700">
              {selectable && (
                <th className="w-10 px-3 py-3">
                  <button onClick={toggleSelectAll} className="text-slate-400 hover:text-blue-600">
                    {allSelected ? <CheckSquare size={16} className="text-blue-600" /> : <Square size={16} />}
                  </button>
                </th>
              )}
              {visibleColumns.map((column) => {
                const colKey = String(column.key);
                const width = columnWidths[colKey] || column.width;
                return (
                  <th
                    key={colKey}
                    className={`relative px-4 py-3 text-xs font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${alignClasses[column.align || 'left']} ${
                      column.sortable ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800' : ''
                    }`}
                    style={{ width: width ? `${width}px` : undefined, minWidth: column.minWidth }}
                    onClick={() => column.sortable && onSort?.(colKey)}
                  >
                    <div className="flex items-center gap-1">
                      {column.header}
                      {renderSortIcon(column)}
                    </div>
                    <div
                      className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-blue-500"
                      onMouseDown={(e) => handleMouseDown(e, colKey)}
                    />
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-700/50">
            {filteredData.length === 0 ? (
              <tr>
                <td
                  colSpan={visibleColumns.length + (selectable ? 1 : 0)}
                  className="px-4 py-12 text-center text-slate-500 dark:text-slate-400"
                >
                  {search ? 'No results match your search' : emptyMessage}
                </td>
              </tr>
            ) : (
              filteredData.map((item) => {
                const id = keyExtractor(item);
                const isSelected = selectedIds?.has(id);
                return (
                  <tr
                    key={id}
                    className={`${
                      onRowClick ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors' : ''
                    } ${isSelected ? 'bg-blue-50 dark:bg-blue-900/20' : ''}`}
                    onClick={() => onRowClick?.(item)}
                  >
                    {selectable && (
                      <td className="w-10 px-3 py-4" onClick={(e) => e.stopPropagation()}>
                        <button onClick={() => toggleSelect(id)} className="text-slate-400 hover:text-blue-600">
                          {isSelected ? <CheckSquare size={16} className="text-blue-600" /> : <Square size={16} />}
                        </button>
                      </td>
                    )}
                    {visibleColumns.map((column) => (
                      <td
                        key={String(column.key)}
                        className={`px-4 py-4 text-sm text-slate-700 dark:text-slate-300 ${alignClasses[column.align || 'left']}`}
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