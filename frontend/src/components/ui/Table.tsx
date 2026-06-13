import React from 'react';
import { ChevronUp, ChevronDown, ChevronsUpDown } from 'lucide-react';

interface Column<T> {
    key: keyof T | string;
    header: string;
    render?: (item: T) => React.ReactNode;
    sortable?: boolean;
    width?: string;
    align?: 'left' | 'center' | 'right';
}

interface TableProps<T> {
    data: T[];
    columns: Column<T>[];
    keyExtractor: (item: T) => string;
    onRowClick?: (item: T) => void;
    sortColumn?: string;
    sortDirection?: 'asc' | 'desc';
    onSort?: (column: string) => void;
    emptyMessage?: string;
    loading?: boolean;
    expandable?: (item: T) => React.ReactNode;
}

export function Table<T>({
    data,
    columns,
    keyExtractor,
    onRowClick,
    sortColumn,
    sortDirection,
    onSort,
    emptyMessage = 'No data available',
    loading = false,
    expandable,
}: TableProps<T>) {
    const alignClasses = {
        left: 'text-left',
        center: 'text-center',
        right: 'text-right',
    };

    const renderSortIcon = (column: Column<T>) => {
        if (!column.sortable) return null;
        const colKey = String(column.key);

        if (sortColumn !== colKey) {
            return <ChevronsUpDown className="w-4 h-4 text-slate-400" />;
        }

        return sortDirection === 'asc' ? (
            <ChevronUp className="w-4 h-4 text-blue-600" />
        ) : (
            <ChevronDown className="w-4 h-4 text-blue-600" />
        );
    };

    const getValue = (item: T, key: keyof T | string): unknown => {
        if (typeof key === 'string' && key.includes('.')) {
            return key.split('.').reduce<unknown>((acc, part) => {
                if (acc && typeof acc === 'object') {
                    return (acc as Record<string, unknown>)[part];
                }
                return undefined;
            }, item);
        }
        return item[key as keyof T];
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
            <div className="overflow-x-auto">
                <table className="w-full">
                    <thead>
                        <tr className="bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-700">
                            {columns.map((column) => (
                                <th
                                    key={String(column.key)}
                                    className={`px-4 py-3 text-xs font-semibold text-slate-600 dark:text-slate-200 uppercase tracking-wider ${alignClasses[column.align || 'left']
                                        } ${column.sortable ? 'cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800' : ''}`}
                                    style={{ width: column.width }}
                                    onClick={() => column.sortable && onSort?.(String(column.key))}
                                >
                                    <div className="flex items-center gap-1">
                                        {column.header}
                                        {renderSortIcon(column)}
                                    </div>
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 dark:divide-slate-700/50">
                        {data.length === 0 ? (
                            <tr>
                                <td
                                    colSpan={columns.length}
                                    className="px-4 py-12 text-center text-slate-500 dark:text-slate-400"
                                >
                                    {emptyMessage}
                                </td>
                            </tr>
                        ) : (
                            data.map((item) => {
                                const expandedContent = expandable?.(item);
                                return (
                                    <React.Fragment key={keyExtractor(item)}>
                                        <tr
                                            className={`${onRowClick
                                                ? 'cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors'
                                                : ''
                                                } ${expandedContent ? 'bg-slate-50 dark:bg-slate-700/30' : ''}`}
                                            onClick={() => onRowClick?.(item)}
                                        >
                                            {columns.map((column) => (
                                                <td
                                                    key={String(column.key)}
                                                    className={`px-4 py-4 text-sm text-slate-700 dark:text-slate-300 ${alignClasses[column.align || 'left']
                                                        }`}
                                                >
                                                    {column.render
                                                        ? column.render(item)
                                                        : String(getValue(item, column.key) ?? '')}
                                                </td>
                                            ))}
                                        </tr>
                                        {expandedContent && (
                                            <tr>
                                                <td colSpan={columns.length} className="p-0 border-b border-slate-100 dark:border-slate-700/50">
                                                    {expandedContent}
                                                </td>
                                            </tr>
                                        )}
                                    </React.Fragment>
                                );
                            })
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

// Pagination component
interface PaginationProps {
    currentPage: number;
    totalPages: number;
    onPageChange: (page: number) => void;
    totalItems: number;
    itemsPerPage: number;
}

export function Pagination({
    currentPage,
    totalPages,
    onPageChange,
    totalItems,
    itemsPerPage,
}: PaginationProps) {
    const startItem = (currentPage - 1) * itemsPerPage + 1;
    const endItem = Math.min(currentPage * itemsPerPage, totalItems);

    const getPageNumbers = () => {
        const pages: (number | string)[] = [];
        const siblingCount = 1; // Number of pages to show on each side of current page

        // If total pages is small enough, show all
        if (totalPages <= 7) {
            return Array.from({ length: totalPages }, (_, i) => i + 1);
        }

        // Always show first page
        pages.push(1);

        // Determine start and end of sibling window
        const startPage = Math.max(2, currentPage - siblingCount);
        const endPage = Math.min(totalPages - 1, currentPage + siblingCount);

        // Add ellipses or pages before the window
        if (startPage > 2) {
            pages.push('...');
        }

        // Add pages in the window
        for (let i = startPage; i <= endPage; i++) {
            pages.push(i);
        }

        // Add ellipses or pages after the window
        if (endPage < totalPages - 1) {
            pages.push('...');
        }

        // Always show last page
        if (totalPages > 1) {
            pages.push(totalPages);
        }

        return pages;
    };

    return (
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3 px-4 py-3 bg-white dark:bg-slate-800 border-t border-slate-200 dark:border-slate-700">
            <div className="text-sm text-slate-500 dark:text-slate-400 text-center sm:text-left">
                Showing <span className="font-medium">{startItem}</span> to{' '}
                <span className="font-medium">{endItem}</span> of{' '}
                <span className="font-medium">{totalItems}</span> results
            </div>
            <div className="flex items-center gap-2">
                <button
                    onClick={() => onPageChange(currentPage - 1)}
                    disabled={currentPage === 1}
                    className="px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    Previous
                </button>

                {/* Page numbers */}
                <div className="hidden sm:flex items-center gap-1">
                    {getPageNumbers().map((page, index) => {
                        if (page === '...') {
                            return (
                                <span key={`ellipsis-${index}`} className="px-2 text-slate-400 dark:text-slate-500">
                                    ...
                                </span>
                            );
                        }
                        return (
                            <button
                                key={page}
                                onClick={() => onPageChange(page as number)}
                                className={`px-3 py-1.5 text-sm font-medium rounded-lg ${currentPage === page
                                    ? 'bg-blue-600 text-white'
                                    : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
                                    }`}
                            >
                                {page}
                            </button>
                        );
                    })}
                </div>

                {/* Show current page indicator on mobile */}
                <span className="sm:hidden text-sm text-slate-500 dark:text-slate-400">
                    {currentPage} / {totalPages}
                </span>

                <button
                    onClick={() => onPageChange(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    className="px-3 py-1.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    Next
                </button>
            </div>
        </div>
    );
}
