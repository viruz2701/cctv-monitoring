import React, { useState, useMemo, useRef, useCallback } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import {
    Check, Clock, AlertTriangle, Info, AlertCircle, Bell,
    Trash2, CheckSquare, Square
} from '../components/ui/Icons';
import {
    useNotifications,
    useMarkNotificationRead,
    useMarkAllNotificationsRead,
    useDeleteNotification,
} from '../hooks/useApiQuery';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { SkeletonNotification, SkeletonCard } from '../components/ui';

function startOfDay(date: Date): Date {
    const d = new Date(date);
    d.setHours(0, 0, 0, 0);
    return d;
}

function isToday(date: Date): boolean {
    const today = startOfDay(new Date());
    return startOfDay(date).getTime() === today.getTime();
}

function isYesterday(date: Date): boolean {
    const yesterday = startOfDay(new Date());
    yesterday.setDate(yesterday.getDate() - 1);
    return startOfDay(date).getTime() === yesterday.getTime();
}

function isWithinLast7Days(date: Date): boolean {
    const sevenDaysAgo = new Date();
    sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7);
    sevenDaysAgo.setHours(0, 0, 0, 0);
    return date >= sevenDaysAgo;
}

type FilterType = 'all' | 'unread' | 'error' | 'warning' | 'info';

interface FlatRow {
    __isHeader: boolean;
    key: string;
    label?: string;
    notification?: any;
}

// ── Статические функции (вынесены из компонента для стабильности ссылок) ──

function getIcon(type: string): React.ReactNode {
    switch (type) {
        case 'error': return <AlertCircle className="w-5 h-5 text-red-500" />;
        case 'warning': return <AlertTriangle className="w-5 h-5 text-amber-500" />;
        case 'success': return <Check className="w-5 h-5 text-green-500" />;
        default: return <Info className="w-5 h-5 text-blue-500" />;
    }
}

function getBorderColor(type: string): string {
    switch (type) {
        case 'error': return 'border-l-red-500';
        case 'warning': return 'border-l-amber-500';
        case 'info': return 'border-l-blue-500';
        case 'success': return 'border-l-green-500';
        default: return 'border-l-transparent';
    }
}

function timeAgo(dateStr: string, t: (key: string) => string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
    if (seconds < 60) return t('just_now') || 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}${t('minutes_ago') || 'm ago'}`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}${t('hours_ago') || 'h ago'}`;
    const days = Math.floor(hours / 24);
    return `${days}${t('days_ago') || 'd ago'}`;
}

// ── NotificationRow — мемоизированный компонент для отдельного уведомления ──

interface NotificationRowProps {
    notification: any;
    selectedIds: Set<string>;
    markAsRead: (id: string) => void;
    toggleSelection: (id: string, e: React.MouseEvent) => void;
    navigate: ReturnType<typeof useNavigate>;
    t: (key: string) => string;
}

const NotificationRow = React.memo(function NotificationRow({
    notification,
    selectedIds,
    markAsRead,
    toggleSelection,
    navigate,
    t,
}: NotificationRowProps) {
    return (
        <div
            onClick={() => {
                if (!notification.read) markAsRead(notification.id);
                if (notification.link) navigate(notification.link);
            }}
            className={`group relative flex items-start gap-3 px-4 py-3 transition-all duration-200 cursor-pointer border-l-4 ${getBorderColor(notification.type)} ${notification.read ? 'bg-white dark:bg-slate-900 hover:bg-slate-50 dark:hover:bg-slate-800/50' : 'bg-blue-50/60 dark:bg-blue-900/15 hover:bg-blue-50 dark:hover:bg-blue-900/20'}`}
        >
            <div className="mt-1 flex-shrink-0" onClick={(e) => toggleSelection(notification.id, e)}>
                <div className={`w-5 h-5 rounded border flex items-center justify-center transition-colors ${selectedIds.has(notification.id) ? 'bg-blue-600 border-blue-600' : 'border-slate-300 dark:border-slate-600 hover:border-slate-400'}`}>
                    {selectedIds.has(notification.id) && <Check className="w-3.5 h-3.5 text-white" />}
                </div>
            </div>
            <div className={`mt-0.5 p-2 rounded-full flex-shrink-0 ${notification.read ? 'bg-slate-100 dark:bg-slate-800 text-slate-500' : 'bg-white dark:bg-slate-800 shadow-sm'}`}>
                {getIcon(notification.type)}
            </div>
            <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2">
                    <p className={`text-sm ${notification.read ? 'text-slate-500 dark:text-slate-400 font-normal' : 'text-slate-900 dark:text-white font-bold'}`}>{notification.title}</p>
                    <span className="text-xs text-slate-500 dark:text-slate-400 whitespace-nowrap flex items-center gap-1 font-medium"><Clock className="w-3 h-3" /> {timeAgo(notification.timestamp, t)}</span>
                </div>
                <p className={`mt-1 text-sm ${notification.read ? 'text-slate-500' : 'text-slate-700 dark:text-slate-300'}`}>{notification.message}</p>
                <div className="flex flex-wrap gap-2 mt-1.5">
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                        {t('device')}: {notification.message.split(' - ')[1] || 'Unknown'}
                    </span>
                </div>
                {notification.link && (
                    <Link to={notification.link} className="inline-block mt-2 text-xs font-semibold uppercase tracking-wide text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300" onClick={(e) => e.stopPropagation()}>
                        {t('view_details')} →
                    </Link>
                )}
            </div>
            {!notification.read && <div className="mt-2 w-2 h-2 bg-blue-500 rounded-full flex-shrink-0" title={t('unread')} />}
        </div>
    );
});

// ── Main Notifications Component ─────────────────────────────────────

export function Notifications() {
    const { t } = useTranslation();
    const { data: apiNotifications = [] } = useNotifications();
    const markRead = useMarkNotificationRead();
    const markAllRead = useMarkAllNotificationsRead();
    const deleteNotif = useDeleteNotification();

    const notifications = useMemo(() => apiNotifications.map((n: any) => ({
        id: n.id,
        title: n.title,
        message: n.message,
        type: n.type,
        read: n.read,
        link: n.link,
        timestamp: n.created_at,
    })), [apiNotifications]);

    const unreadCount = useMemo(() => notifications.filter(n => !n.read).length, [notifications]);

    const markAsRead = (id: string) => markRead.mutate(id);
    const markAllAsRead = () => markAllRead.mutate();
    const markNotificationsAsRead = (ids: string[]) => ids.forEach(id => markRead.mutate(id));
    const deleteNotifications = (ids: string[]) => ids.forEach(id => deleteNotif.mutate(id));

    const [activeFilter, setActiveFilter] = useState<FilterType>('all');
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [pageLoading, setPageLoading] = React.useState(true);
    React.useEffect(() => {
        const timer = setTimeout(() => setPageLoading(false), 500);
        return () => clearTimeout(timer);
    }, []);
    const navigate = useNavigate();

    const filteredNotifications = useMemo(() => {
        let filtered = notifications;
        if (activeFilter === 'unread') {
            filtered = notifications.filter(n => !n.read);
        } else if (activeFilter !== 'all') {
            filtered = notifications.filter(n => n.type === activeFilter);
        }
        return [...filtered].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    }, [notifications, activeFilter]);

    const groupedNotifications = useMemo(() => {
        const groups: Record<string, typeof notifications> = {
            [t('today') || 'Today']: [],
            [t('yesterday') || 'Yesterday']: [],
            [t('last_7_days') || 'Last 7 Days']: [],
            [t('older') || 'Older']: []
        };
        filteredNotifications.forEach(n => {
            const date = new Date(n.timestamp);
            if (isToday(date)) groups[t('today') || 'Today'].push(n);
            else if (isYesterday(date)) groups[t('yesterday') || 'Yesterday'].push(n);
            else if (isWithinLast7Days(date)) groups[t('last_7_days') || 'Last 7 Days'].push(n);
            else groups[t('older') || 'Older'].push(n);
        });
        return groups;
    }, [filteredNotifications, t]);

    // ── P1-UX.5: Flatten grouped notifications into virtual rows ──────
    const totalCount = filteredNotifications.length;
    const enableVirtualization = totalCount > 1000;

    const flatRows = useMemo(() => {
        if (!enableVirtualization) return [] as FlatRow[];
        const rows: FlatRow[] = [];
        Object.entries(groupedNotifications).forEach(([group, groupItems]) => {
            if (groupItems.length === 0) return;
            rows.push({ __isHeader: true, key: `header-${group}`, label: group });
            groupItems.forEach((n: any) => {
                rows.push({ __isHeader: false, key: n.id, notification: n });
            });
        });
        return rows;
    }, [groupedNotifications, enableVirtualization]);

    const scrollRef = useRef<HTMLDivElement>(null);

    const rowVirtualizer = useVirtualizer({
        count: enableVirtualization ? flatRows.length : 0,
        getScrollElement: () => scrollRef.current,
        estimateSize: (index) => {
            if (!enableVirtualization) return 0;
            return flatRows[index]?.__isHeader ? 36 : 120;
        },
        overscan: 5,
        enabled: enableVirtualization,
    });

    // ── Memoized notification renderer for virtual rows ───────────────
    const renderNotificationRow = useCallback((notification: any) => (
        <NotificationRow
            notification={notification}
            selectedIds={selectedIds}
            markAsRead={markAsRead}
            toggleSelection={toggleSelection}
            navigate={navigate}
            t={t}
        />
    ), [selectedIds, markAsRead, navigate, t]);

    const toggleSelection = (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        const newSelected = new Set(selectedIds);
        if (newSelected.has(id)) newSelected.delete(id);
        else newSelected.add(id);
        setSelectedIds(newSelected);
    };

    const toggleSelectAll = () => {
        if (selectedIds.size === filteredNotifications.length && filteredNotifications.length > 0) {
            setSelectedIds(new Set());
        } else {
            setSelectedIds(new Set(filteredNotifications.map(n => n.id)));
        }
    };

    const handleBulkMarkRead = () => {
        markNotificationsAsRead(Array.from(selectedIds));
        setSelectedIds(new Set());
    };

    const handleBulkDelete = () => {
        deleteNotifications(Array.from(selectedIds));
        setSelectedIds(new Set());
    };

    const isAllSelected = filteredNotifications.length > 0 && selectedIds.size === filteredNotifications.length;

    return (
        <div className="space-y-4 sm:space-y-6">
            {pageLoading ? (
                <div className="space-y-6" aria-label="Loading notifications">
                    {/* Header skeleton */}
                    <div className="space-y-2">
                        <div className="h-7 w-48 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                        <div className="h-4 w-64 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                    </div>

                    {/* Filter pills skeleton */}
                    <div className="flex gap-2 overflow-x-auto scrollbar-hide">
                        {Array.from({ length: 4 }, (_, i) => (
                            <div
                                key={i}
                                className="h-8 w-20 bg-slate-200 dark:bg-slate-700 animate-pulse rounded-full flex-shrink-0"
                            />
                        ))}
                    </div>

                    {/* Notifications skeleton */}
                    <SkeletonNotification count={5} />
                </div>
            ) : (
                <>
                    <div className="sticky top-16 z-20 bg-slate-50/95 dark:bg-slate-950/95 backdrop-blur-sm pt-2 sm:pt-3 pb-2 -mx-4 px-4 sm:-mx-6 sm:px-6 border-b border-slate-200 dark:border-slate-800 space-y-2 sm:space-y-3">
                <div className="flex items-center justify-between gap-2">
                    <div className="min-w-0">
                        <h1 className="hidden sm:block text-2xl font-bold text-slate-900 dark:text-white">{t('notifications')}</h1>
                        <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400">
                            {unreadCount} {t('unread')} • {notifications.length} {t('total')}
                        </p>
                    </div>
                    {unreadCount > 0 && (
                        <button onClick={markAllAsRead} className="flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 text-xs sm:text-sm font-medium text-blue-600 bg-white border border-blue-200 hover:bg-blue-50 dark:text-blue-400 dark:bg-slate-800 dark:border-slate-700 dark:hover:bg-slate-700/50 rounded-lg transition-colors shadow-sm">
                            <Check className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
                            <span className="hidden sm:inline">{t('mark_all_read')}</span>
                            <span className="sm:hidden">{t('read_all')}</span>
                        </button>
                    )}
                </div>

                <div className="flex items-center justify-between gap-2">
                    <div className="flex gap-1.5 sm:gap-2 overflow-x-auto scrollbar-hide flex-1 min-w-0">
                        {(['all', 'unread', 'error', 'warning', 'info'] as const).map(filter => (
                            <button key={filter} onClick={() => setActiveFilter(filter)} className={`px-3 sm:px-4 py-1 sm:py-1.5 rounded-full text-xs sm:text-sm font-medium whitespace-nowrap transition-colors ${activeFilter === filter ? 'bg-blue-600 text-white shadow-md shadow-blue-500/20' : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 border border-slate-200 dark:border-slate-700'}`}>
                                {filter === 'all' ? t('all') : t(filter)}
                            </button>
                        ))}
                    </div>
                    <button onClick={toggleSelectAll} className="flex-shrink-0 flex items-center gap-1.5 text-xs sm:text-sm font-semibold text-slate-600 hover:text-slate-800 dark:text-slate-300 dark:hover:text-slate-100 pl-2">
                        {isAllSelected ? <CheckSquare className="w-4 h-4 text-blue-600" /> : <Square className="w-4 h-4" />}
                        <span className="text-[10px] sm:text-sm">{t('all')}</span>
                    </button>
                </div>

                {selectedIds.size > 0 && (
                    <div className="flex items-center justify-between bg-blue-50 dark:bg-blue-900/20 px-3 py-1.5 rounded-lg border border-blue-100 dark:border-blue-900/30">
                        <span className="text-xs sm:text-sm font-medium text-blue-900 dark:text-blue-100">{selectedIds.size} {t('selected')}</span>
                        <div className="flex items-center gap-1.5">
                            <button onClick={handleBulkMarkRead} className="flex items-center gap-1 px-2.5 py-1 text-xs font-medium text-blue-700 hover:bg-blue-100 dark:text-blue-300 dark:hover:bg-blue-800/30 rounded-md transition-colors">
                                <Check className="w-3.5 h-3.5" /> {t('read')}
                            </button>
                            <button onClick={handleBulkDelete} className="flex items-center gap-1 px-2.5 py-1 text-xs font-medium text-red-700 hover:bg-red-100 dark:text-red-300 dark:hover:bg-red-900/30 rounded-md transition-colors">
                                <Trash2 className="w-3.5 h-3.5" /> {t('delete')}
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {/* P1-UX.5: Virtualized list for 10k+ notifications */}
            {enableVirtualization ? (
                <div
                    ref={scrollRef}
                    className="overflow-y-auto"
                    style={{ maxHeight: 'calc(100vh - 280px)', contain: 'strict' }}
                >
                    <div
                        style={{
                            height: `${rowVirtualizer.getTotalSize()}px`,
                            width: '100%',
                            position: 'relative',
                        }}
                    >
                        {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                            const row = flatRows[virtualRow.index];
                            if (row.__isHeader) {
                                return (
                                    <div
                                        key={row.key}
                                        className="sticky top-0 z-10 bg-slate-50 dark:bg-slate-950 py-2 px-1"
                                        style={{
                                            height: `${virtualRow.size}px`,
                                            transform: `translateY(${virtualRow.start}px)`,
                                        }}
                                    >
                                        <h2 className="text-sm font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                            {row.label}
                                        </h2>
                                    </div>
                                );
                            }
                            return (
                                <div
                                    key={row.key}
                                    style={{
                                        height: `${virtualRow.size}px`,
                                        transform: `translateY(${virtualRow.start}px)`,
                                    }}
                                >
                                    <div className="bg-white dark:bg-slate-900 rounded-xl shadow-sm border border-slate-200 dark:border-slate-800 overflow-hidden">
                                        {renderNotificationRow(row.notification)}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                    {filteredNotifications.length === 0 && (
                        <div className="flex flex-col items-center justify-center py-24 px-4 bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 text-center">
                            <div className="bg-slate-100 dark:bg-slate-800 p-4 rounded-full mb-4"><Bell className="w-8 h-8 text-slate-400" /></div>
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white">{t('all_caught_up')}</h3>
                            <p className="text-slate-500 dark:text-slate-400 mt-1 max-w-xs mx-auto">{t('no_notifications_match')}</p>
                            {activeFilter !== 'all' && <button onClick={() => setActiveFilter('all')} className="mt-4 text-blue-600 hover:text-blue-700 font-medium text-sm">{t('clear_filters')}</button>}
                        </div>
                    )}
                </div>
            ) : (
                /* Seamless fallback: grouped rendering for <= 1000 items */
                <div className="space-y-5 sm:space-y-8">
                    {Object.entries(groupedNotifications).map(([group, groupNotifications]) => groupNotifications.length > 0 && (
                        <div key={group} className="space-y-3">
                            <h2 className="text-sm font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider px-1">{group}</h2>
                            <div className="bg-white dark:bg-slate-900 rounded-xl shadow-sm border border-slate-200 dark:border-slate-800 overflow-hidden divide-y divide-slate-100 dark:divide-slate-800">
                                {groupNotifications.map(notification => (
                                    <NotificationRow
                                        key={notification.id}
                                        notification={notification}
                                        selectedIds={selectedIds}
                                        markAsRead={markAsRead}
                                        toggleSelection={toggleSelection}
                                        navigate={navigate}
                                        t={t}
                                    />
                                ))}
                            </div>
                        </div>
                    ))}
                    {filteredNotifications.length === 0 && (
                        <div className="flex flex-col items-center justify-center py-24 px-4 bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 text-center">
                            <div className="bg-slate-100 dark:bg-slate-800 p-4 rounded-full mb-4"><Bell className="w-8 h-8 text-slate-400" /></div>
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white">{t('all_caught_up')}</h3>
                            <p className="text-slate-500 dark:text-slate-400 mt-1 max-w-xs mx-auto">{t('no_notifications_match')}</p>
                            {activeFilter !== 'all' && <button onClick={() => setActiveFilter('all')} className="mt-4 text-blue-600 hover:text-blue-700 font-medium text-sm">{t('clear_filters')}</button>}
                        </div>
                    )}
                </div>
            )}
                </>
            )}
        </div>
    );
}
