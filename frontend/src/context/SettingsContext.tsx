import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback } from 'react';
import type { AppSettings, DashboardLayoutConfig, ServicesSettings } from '../types';
import { useAuth } from '../hooks/useAuth';
import {
  useServicesSettings,
  useServicesStatus,
  useUpdateServicesSettings,
} from '../hooks/useApiQuery';

// ── Types ────────────────────────────────────────────────────────────

export interface ServiceStatusEntry {
  status: 'running' | 'stopped' | 'disabled' | 'error';
  port: number;
  message?: string;
}

export type ServicesStatusMap = Record<string, ServiceStatusEntry>;

interface SettingsContextType {
  settings: AppSettings;
  dashboardConfig: DashboardLayoutConfig;
  servicesSettings: ServicesSettings | null;
  servicesLoading: boolean;
  servicesStatus: ServicesStatusMap;
  servicesStatusLoading: boolean;
  updateSettings: (updates: Partial<AppSettings>) => void;
  updateDashboardConfig: (updates: Partial<DashboardLayoutConfig>) => void;
  updateServicesSettings: (updates: Partial<ServicesSettings>) => void;
  saveServicesSettings: () => Promise<void>;
  refreshServicesSettings: () => Promise<void>;
  refreshServicesStatus: () => Promise<void>;
}

const SettingsContext = createContext<SettingsContextType | undefined>(undefined);

export function SettingsProvider({ children }: { children: ReactNode }) {
  const { token } = useAuth();

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
      smsEnabled: false,
      smsForCriticalOnly: false,
      emailForManagers: false,
      rocketsms: { login: '', sender: '', apiUrl: '' },
      smtp: { host: '', port: 587, user: '', from: '' },
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

  const [servicesSettingsState, setServicesSettingsState] = useState<ServicesSettings | null>(null);

  // ── React Query hooks ──────────────────────────────────────────────
  const {
    data: servicesSettingsData,
    isLoading: servicesLoading,
    refetch: refreshServicesSettings,
  } = useServicesSettings();

  const {
    data: servicesStatusData,
    isLoading: servicesStatusLoading,
    refetch: refreshServicesStatus,
  } = useServicesStatus();

  // Sync React Query data to local state for mutation support
  // (React Query provides read data, local state holds pending edits)
  React.useEffect(() => {
    if (servicesSettingsData) {
      setServicesSettingsState(servicesSettingsData);
    }
  }, [servicesSettingsData]);

  const updateServicesSettingsMutation = useUpdateServicesSettings();

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

  // ── Enable query fetching when token is available ──────────────────
  // useServicesSettings and useServicesStatus are enabled by default
  // They will only fetch when the user is authenticated via the token check
  // in the API layer (request function handles 401).

  // ── Methods ────────────────────────────────────────────────────────

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

  const updateServicesSettings = useCallback((updates: Partial<ServicesSettings>) => {
    setServicesSettingsState(prev => {
      if (!prev) return prev;
      return { ...prev, ...updates };
    });
  }, []);

  const saveServicesSettings = async () => {
    if (!servicesSettingsState) return;
    await updateServicesSettingsMutation.mutateAsync(servicesSettingsState);
  };

  // Wrap refetch to match () => Promise<void> signature
  const handleRefreshServicesSettings = useCallback(async () => {
    await refreshServicesSettings();
  }, [refreshServicesSettings]);

  const handleRefreshServicesStatus = useCallback(async () => {
    await refreshServicesStatus();
  }, [refreshServicesStatus]);

  const value = useMemo<SettingsContextType>(() => ({
    settings,
    dashboardConfig,
    servicesSettings: servicesSettingsState,
    servicesLoading,
    servicesStatus: (servicesStatusData?.services || {}) as ServicesStatusMap,
    servicesStatusLoading,
    updateSettings,
    updateDashboardConfig,
    updateServicesSettings,
    saveServicesSettings,
    refreshServicesSettings: handleRefreshServicesSettings,
    refreshServicesStatus: handleRefreshServicesStatus,
  }), [settings, dashboardConfig, servicesSettingsState, servicesLoading, servicesStatusData, servicesStatusLoading, updateSettings, updateDashboardConfig, updateServicesSettings, saveServicesSettings, handleRefreshServicesSettings, handleRefreshServicesStatus]);

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
