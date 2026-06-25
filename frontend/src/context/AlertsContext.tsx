// ═══════════════════════════════════════════════════════════════════════
// AlertsContext — Bridge to React Query + Zustand (ARCH-02/03)
//
// Вместо mockData использует API через React Query.
// UI-состояние (выделение, фильтры) — через Zustand alertStore.
//
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext, ReactNode, useMemo, useCallback } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useDevicesSites } from './DevicesSitesContext';
import { useAlarms, useAcknowledgeAlarm, useResolveAlarm } from '../hooks/useApiQuery';
import type { Alarm as APIAlarm } from '../services/api';
import type { Alert, AlertStatus } from '../types';

interface AlertsContextType {
    alerts: Alert[];
    addAlert: (alert: Alert) => void;
    updateAlertStatus: (id: string, status: AlertStatus) => void;
    deleteAlert: (id: string) => void;
}

const AlertsContext = createContext<AlertsContextType | undefined>(undefined);

// ═══ Helper: API Alarm → UI Alert ═══
function mapPriorityToLabel(priority: number): 'critical' | 'high' | 'medium' | 'low' {
    if (priority >= 4) return 'critical';
    if (priority >= 3) return 'high';
    if (priority >= 2) return 'medium';
    return 'low';
}

function mapAlarmToAlert(alarm: APIAlarm): Alert {
    return {
        id: alarm.device_id + '-' + alarm.timestamp,
        deviceId: alarm.device_id,
        deviceName: alarm.device_id, // будет обогащено через useDevices
        type: alarm.priority >= 3 ? 'error' : alarm.priority >= 2 ? 'warning' : 'info',
        message: alarm.description,
        timestamp: alarm.timestamp,
        status: 'active',
        priority: mapPriorityToLabel(alarm.priority),
        source: alarm.device_id,
        siteName: '',
    };
}

export function AlertsProvider({ children }: { children: ReactNode }) {
    const { user } = useAuth();
    const { devices } = useDevicesSites();

    // Используем React Query для получения alarms из API (ARCH-03)
    const { data: apiAlarms = [] } = useAlarms();
    const acknowledgeAlarm = useAcknowledgeAlarm();
    const resolveAlarm = useResolveAlarm();

    // Маппинг API → UI
    const alerts = useMemo(() => {
        const mapped = apiAlarms.map(mapAlarmToAlert);

        // Фильтр по видимым устройствам (role-based access)
        if (!user || user.role === 'admin') return mapped;
        const visibleDeviceIds = devices.map(d => d.id);
        return mapped.filter(alert => visibleDeviceIds.includes(alert.deviceId));
    }, [apiAlarms, user, devices]);

    const addAlert = useCallback((_alert: Alert) => {
        console.warn('addAlert: use create alarm API directly');
    }, []);

    const updateAlertStatus = useCallback((id: string, status: AlertStatus) => {
        // Маппим статус UI → API действие
        if (status === 'acknowledged') {
            acknowledgeAlarm.mutate(id);
        } else if (status === 'resolved') {
            resolveAlarm.mutate(id);
        }
    }, [acknowledgeAlarm, resolveAlarm]);

    const deleteAlert = useCallback((_id: string) => {
        console.warn('deleteAlert: use API alarm delete directly');
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
