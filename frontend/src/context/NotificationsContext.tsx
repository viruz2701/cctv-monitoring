import React, { createContext, useContext, useState, ReactNode, useCallback } from 'react';
import { alerts as initialAlerts } from '../data/mockData';
import type { Alert, Notification } from '../types';

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

// Helper to convert Alert to Notification
const alertToNotification = (alert: Alert): Notification => ({
    id: alert.id,
    title: alert.type.charAt(0).toUpperCase() + alert.type.slice(1),
    message: `${alert.message} - ${alert.deviceName}`,
    type: alert.type,
    timestamp: alert.timestamp,
    read: false, // Default to unread for now
    link: `/devices/${alert.deviceId}`,
});

export function NotificationsProvider({ children }: { children: ReactNode }) {
    const [notifications, setNotifications] = useState<Notification[]>(() => {
        // Spread notifications across Today, Yesterday, and Last 7 Days
        // so that date grouping is actually useful with mock data
        const hoursOffsets = [1, 3, 5, 28, 52, 72, 96, 120, 144, 168];
        return initialAlerts.map((alert, i) => {
            const offsetHours = hoursOffsets[i % hoursOffsets.length];
            const ts = new Date();
            ts.setHours(ts.getHours() - offsetHours);
            return {
                ...alertToNotification(alert),
                timestamp: ts.toISOString(),
            };
        });
    });

    const unreadCount = notifications.filter(n => !n.read).length;

    const markAsRead = useCallback((id: string) => {
        setNotifications(prev => prev.map(n => n.id === id ? { ...n, read: true } : n));
    }, []);

    const markAllAsRead = useCallback(() => {
        setNotifications(prev => prev.map(n => ({ ...n, read: true })));
    }, []);

    const deleteNotification = useCallback((id: string) => {
        setNotifications(prev => prev.filter(n => n.id !== id));
    }, []);

    const markNotificationsAsRead = useCallback((ids: string[]) => {
        setNotifications(prev => prev.map(n => ids.includes(n.id) ? { ...n, read: true } : n));
    }, []);

    const deleteNotifications = useCallback((ids: string[]) => {
        setNotifications(prev => prev.filter(n => !ids.includes(n.id)));
    }, []);

    const value = {
        notifications,
        unreadCount,
        markAsRead,
        markAllAsRead,
        deleteNotification,
        markNotificationsAsRead,
        deleteNotifications
    };

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
