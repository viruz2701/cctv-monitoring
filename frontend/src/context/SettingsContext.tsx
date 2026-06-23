import React, { createContext, useContext, useState, ReactNode, useMemo, useCallback, useEffect, useRef } from 'react';
import type { AppSettings, DashboardLayoutConfig, ServicesSettings } from '../types';
import { api } from '../services/api';
import { useAuth } from '../hooks/useAuth';

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
  const statusIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

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
  const [servicesStatus, setServicesStatus] = useState<ServicesStatusMap>({});
  const [servicesStatusLoading, setServicesStatusLoading] = useState(false);

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

  // ── Загрузка настроек сервисов при наличии токена ────
  useEffect(() => {
    if (token) {
      refreshServicesSettings();
      refreshServicesStatus();
    }
    return () => {
      if (statusIntervalRef.current) {
        clearInterval(statusIntervalRef.current);
      }
    };
  }, [token]);

  // ── Polling статуса сервисов каждые 30 секунд ────────
  useEffect(() => {
    if (!token) return;
    // Очищаем предыдущий интервал
    if (statusIntervalRef.current) {
      clearInterval(statusIntervalRef.current);
    }
    // Запускаем новый
    statusIntervalRef.current = setInterval(() => {
      refreshServicesStatus();
    }, 30000); // каждые 30 секунд

    return () => {
      if (statusIntervalRef.current) {
        clearInterval(statusIntervalRef.current);
      }
    };
  }, [token]);

  // ── Methods ────────────────────────────────────────────────────────

  const refreshServicesSettings = async () => {
    setServicesLoading(true);
    try {
      const data = await api.getServicesSettings();
      setServicesSettings(data);
    } catch (error) {
      console.error('Failed to load services settings:', error);
      setServicesSettings({
        services_syslog: { enabled: true, udp_port: 1514, tcp_port: 1514 },
        services_ftp: { enabled: true, port: 2121, user: 'alarm', password: '', root_path: '/var/lib/gb-telemetry/ftp' },
        services_snmp: {
          enabled: true, port: 162, community: 'public', version: 'v2c',
          v1_config: { enabled: true, port: 162, community: 'public' },
          v2c_config: { enabled: true, port: 162, community: 'public' },
          v3_config: { enabled: false, port: 1162, user: '', auth_protocol: 'SHA', auth_password: '', priv_protocol: 'AES', priv_password: '' },
        },
        services_http: { enabled: true, port: 8083 },
        services_dahua: { enabled: true, ports: [37777, 37778] },
        services_hisilicon: { enabled: true, port: 15002 },
        services_tvt: { enabled: true, port: 15003 },
        services_gb28181: {
          enabled: true, host: '0.0.0.0', port: 5060,
          server_id: '34020000002000000001', server_ip: '', realm: '3402000000',
          auth_enabled: false, auth_user: 'admin', auth_password: '',
          auto_catalog: true, auto_device_info: true,
          keepalive_interval: 60, keepalive_timeout: 180,
          max_sub_channels: 64, log_sip_messages: false,
        },
        services_p2p_gateway: {
          enabled: true, url: 'http://localhost:8082', api_key: '',
          hikvision: { username: '', password: '' },
          dahua: { python_path: '/usr/bin/python3', script_path: './bin/dh-p2p/main.py' },
          reolink: { proxy_bin_path: './bin/neolink' },
          xiongmai: { uuid: '', app_key: '', app_secret: '', endpoint: 'api-cn.jftechws.com', region: 'RU', move_card: 2 },
          ezviz: { app_key: '', app_secret: '' },
        }
      } as ServicesSettings);
    } finally {
      setServicesLoading(false);
    }
  };

  const refreshServicesStatus = async () => {
    setServicesStatusLoading(true);
    try {
      const data = await api.getServicesStatus();
      setServicesStatus((data.services || {}) as ServicesStatusMap);
    } catch (error) {
      console.error('Failed to load services status:', error);
    } finally {
      setServicesStatusLoading(false);
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
    servicesStatus,
    servicesStatusLoading,
    updateSettings,
    updateDashboardConfig,
    updateServicesSettings,
    saveServicesSettings,
    refreshServicesSettings,
    refreshServicesStatus,
  }), [settings, dashboardConfig, servicesSettings, servicesLoading, servicesStatus, servicesStatusLoading, updateSettings, updateDashboardConfig, updateServicesSettings, saveServicesSettings, refreshServicesSettings, refreshServicesStatus]);

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
