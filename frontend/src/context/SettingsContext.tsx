import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback } from 'react';
import type { AppSettings, DashboardLayoutConfig } from '../types';

interface SettingsContextType {
    settings: AppSettings;
    dashboardConfig: DashboardLayoutConfig;
    updateSettings: (updates: Partial<AppSettings>) => void;
    updateDashboardConfig: (updates: Partial<DashboardLayoutConfig>) => void;
}

const SettingsContext = createContext<SettingsContextType | undefined>(undefined);

export function SettingsProvider({ children }: { children: ReactNode }) {
    const [settings, setSettings] = useState<AppSettings>({
        organizationName: 'ACME Corporation',
        systemEmail: 'admin@acme.com',
        timezone: 'EST',
        dateFormat: 'MM/DD/YYYY',
        notifications: {
            deviceOffline: true,
            securityAlerts: true,
            storageWarnings: true,
            dailyReports: false,
            mobilePush: false,
        },
        system: {
            healthCheckInterval: 5,
            sessionTimeout: 30,
            maxRecordingGap: 15,
            alertThreshold: 85,
        },
        security: {
            requires2FA: false,
            passwordPolicy: 'basic',
        }
    });

    const [dashboardConfig, setDashboardConfig] = useState<DashboardLayoutConfig>(() => {
        const saved = localStorage.getItem('dashboardConfig');
        return saved ? JSON.parse(saved) : {
            showStatsRow: true,
            showTicketStats: true,
            showRecentAlerts: true,
            showLatestTickets: true,
            showQuickActions: true
        };
    });

    // Settings Actions — deep-merges all nested sections to avoid dropping sibling keys
    const updateSettings = useCallback((updates: Partial<AppSettings>) => {
        setSettings(prev => ({
            ...prev,
            ...updates,
            notifications: {
                ...prev.notifications,
                ...(updates.notifications || {})
            },
            system: {
                ...prev.system,
                ...(updates.system || {})
            },
            security: {
                ...prev.security,
                ...(updates.security || {})
            }
        }));
    }, []);

    const updateDashboardConfig = useCallback((updates: Partial<DashboardLayoutConfig>) => {
        setDashboardConfig(prev => {
            const next = { ...prev, ...updates };
            localStorage.setItem('dashboardConfig', JSON.stringify(next));
            return next;
        });
    }, []);

    const value = useMemo<SettingsContextType>(() => ({
        settings, dashboardConfig, updateSettings, updateDashboardConfig,
    }), [settings, dashboardConfig, updateSettings, updateDashboardConfig]);

    return (
        <SettingsContext.Provider value={value}>
            {children}
        </SettingsContext.Provider>
    );
}

export function useSettings() {
    const context = useContext(SettingsContext);
    if (context === undefined) {
        throw new Error('useSettings must be used within a SettingsProvider');
    }
    return context;
}
