import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback } from 'react';
import { alerts as initialAlerts } from '../data/mockData';
import type { Alert, AlertStatus } from '../types';
import { useAuth } from '../hooks/useAuth';
import { useDevicesSites } from './DevicesSitesContext';

interface AlertsContextType {
    alerts: Alert[];
    addAlert: (alert: Alert) => void;
    updateAlertStatus: (id: string, status: AlertStatus) => void;
    deleteAlert: (id: string) => void;
}

const AlertsContext = createContext<AlertsContextType | undefined>(undefined);

export function AlertsProvider({ children }: { children: ReactNode }) {
    const { user } = useAuth();
    const { devices } = useDevicesSites();

    // Raw State
    const [rawAlerts, setRawAlerts] = useState<Alert[]>(initialAlerts);

    // 4. Visible Alerts (linked to visible devices)
    const alerts = useMemo(() => {
        if (!user) return [];
        if (user.role === 'admin') return rawAlerts;
        const visibleDeviceIds = devices.map(d => d.id);
        return rawAlerts.filter(alert => visibleDeviceIds.includes(alert.deviceId));
    }, [user, rawAlerts, devices]);

    // Alert Actions
    const addAlert = useCallback((alert: Alert) => {
        setRawAlerts(prev => [alert, ...prev]);
    }, []);

    const updateAlertStatus = useCallback((id: string, status: AlertStatus) => {
        setRawAlerts(prev => prev.map(a => a.id === id ? { ...a, status } : a));
    }, []);

    const deleteAlert = useCallback((id: string) => {
        setRawAlerts(prev => prev.filter(a => a.id !== id));
    }, []);

    const value = useMemo<AlertsContextType>(() => ({
        alerts, addAlert, updateAlertStatus, deleteAlert,
    }), [alerts, addAlert, updateAlertStatus, deleteAlert]);

    return (
        <AlertsContext.Provider value={value}>
            {children}
        </AlertsContext.Provider>
    );
}

export function useAlerts() {
    const context = useContext(AlertsContext);
    if (context === undefined) {
        throw new Error('useAlerts must be used within a AlertsProvider');
    }
    return context;
}
