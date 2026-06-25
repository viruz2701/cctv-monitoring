import React, { createContext, useContext, ReactNode, useMemo, useCallback } from 'react';
import type { Notification } from '../types';
import {
    useNotifications as useNotificationsQuery,
    useMarkNotificationRead,
    useMarkAllNotificationsRead,
    useDeleteNotification,
} from '../hooks/useApiQuery';

interface NotificationsContextType {
    notifications: Notification[];
    unreadCount: number;
    markAsRead: (id: string) => void;
    markAllAsRead: () => void;
    deleteNotification: (id: string) => void;
    markNotificationsAsRead: (ids: string[]) => void;
    deleteNotifications: (ids: string[]) => void;
}

const NotificationsContext = createContext<NotificationsContextType | undefined>(undefined);

/**
 * Map API Notification (snake_case fields, created_at) to local Notification (camelCase, timestamp).
 */
const mapApiNotification = (n: {
    id: string;
    title: string;
    message: string;
    type: 'success' | 'warning' | 'error' | 'info';
    read: boolean;
    created_at: string;
    link?: string;
}): Notification => ({
    id: n.id,
    title: n.title,
    message: n.message,
    type: n.type,
    read: n.read,
    link: n.link,
    timestamp: n.created_at,
});

export function NotificationsProvider({ children }: { children: ReactNode }) {
    const { data: apiNotifications = [] } = useNotificationsQuery();
    const markReadMutation = useMarkNotificationRead();
    const markAllReadMutation = useMarkAllNotificationsRead();
    const deleteMutation = useDeleteNotification();

    const notifications = useMemo(
        () => apiNotifications.map(mapApiNotification),
        [apiNotifications],
    );

    const unreadCount = useMemo(
        () => notifications.filter(n => !n.read).length,
        [notifications],
    );

    const markAsRead = useCallback((id: string) => {
        markReadMutation.mutate(id);
    }, [markReadMutation]);

    const markAllAsRead = useCallback(() => {
        markAllReadMutation.mutate();
    }, [markAllReadMutation]);

    const deleteNotification = useCallback((id: string) => {
        deleteMutation.mutate(id);
    }, [deleteMutation]);

    const markNotificationsAsRead = useCallback((ids: string[]) => {
        ids.forEach(id => markReadMutation.mutate(id));
    }, [markReadMutation]);

    const deleteNotifications = useCallback((ids: string[]) => {
        ids.forEach(id => deleteMutation.mutate(id));
    }, [deleteMutation]);

    const value = useMemo(() => ({
        notifications,
        unreadCount,
        markAsRead,
        markAllAsRead,
        deleteNotification,
        markNotificationsAsRead,
        deleteNotifications,
    }), [
        notifications,
        unreadCount,
        markAsRead,
        markAllAsRead,
        deleteNotification,
        markNotificationsAsRead,
        deleteNotifications,
    ]);

    return (
        <NotificationsContext.Provider value={value}>
            {children}
        </NotificationsContext.Provider>
    );
}

export function useNotifications() {
    const context = useContext(NotificationsContext);
    if (context === undefined) {
        throw new Error('useNotifications must be used within a NotificationsProvider');
    }
    return context;
}
