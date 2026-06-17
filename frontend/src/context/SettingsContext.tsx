import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback, useEffect } from 'react';
import type { AppSettings, DashboardLayoutConfig, ServicesSettings } from '../types';
import { api } from '../services/api';

interface SettingsContextType {
  settings: AppSettings;
  dashboardConfig: DashboardLayoutConfig;
  servicesSettings: ServicesSettings | null;
  servicesLoading: boolean;
  updateSettings: (updates: Partial<AppSettings>) => void;
  updateDashboardConfig: (updates: Partial<DashboardLayoutConfig>) => void;
  updateServicesSettings: (updates: Partial<ServicesSettings>) => void;
  saveServicesSettings: () => Promise<void>;
  refreshServicesSettings: () => Promise<void>;
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

  const [servicesSettings, setServicesSettings] = useState<ServicesSettings | null>(null);
  const [servicesLoading, setServicesLoading] = useState(false);

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

  // Загрузка настроек сервисов при монтировании
  useEffect(() => {
    refreshServicesSettings();
  }, []);

  const refreshServicesSettings = async () => {
    setServicesLoading(true);
    try {
      const data = await api.getServicesSettings();
      setServicesSettings(data);
    } catch (error) {
      console.error('Failed to load services settings:', error);
      // Устанавливаем дефолтные значения при ошибке
      setServicesSettings({
        services_syslog: { enabled: true, udp_port: 1514, tcp_port: 1514 },
        services_ftp: { enabled: true, port: 2121, user: 'alarm', password: '', root_path: '/var/lib/gb-telemetry/ftp' },
        services_snmp: { enabled: true, port: 162, community: 'public', version: 'v2c' },
        services_http: { enabled: true, port: 8083 },
        services_dahua: { enabled: true, ports: [37777, 37778] },
        services_hisilicon: { enabled: true, port: 15002 },
        services_tvt: { enabled: true, port: 15003 },
        services_sip: { enabled: true, port: 5060, host: '0.0.0.0' },
        services_p2p_gateway: { enabled: true, url: 'http://localhost:8082', api_key: '' }
      });
    } finally {
      setServicesLoading(false);
    }
  };

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
    setServicesSettings(prev => {
      if (!prev) return prev;
      return { ...prev, ...updates };
    });
  }, []);

  const saveServicesSettings = async () => {
    if (!servicesSettings) return;
    try {
      await api.updateServicesSettings(servicesSettings);
      // После успешного сохранения можно показать уведомление
    } catch (error) {
      console.error('Failed to save services settings:', error);
      throw error;
    }
  };

  const value = useMemo<SettingsContextType>(() => ({
    settings,
    dashboardConfig,
    servicesSettings,
    servicesLoading,
    updateSettings,
    updateDashboardConfig,
    updateServicesSettings,
    saveServicesSettings,
    refreshServicesSettings
  }), [settings, dashboardConfig, servicesSettings, servicesLoading, updateSettings, updateDashboardConfig, updateServicesSettings, saveServicesSettings, refreshServicesSettings]);

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