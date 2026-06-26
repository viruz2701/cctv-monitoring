// ═══════════════════════════════════════════════════════════════════════
// Settings Store (Zustand)
// ARCH-02: Client-side state для настроек и конфигурации дашборда.
// Server state (services settings/status) — через React Query hooks.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AppSettings, DashboardLayoutConfig } from '../types';

// ── Dashboard Config ─────────────────────────────────────────────────

const DEFAULT_DASHBOARD_CONFIG: DashboardLayoutConfig = {
  showStatsRow: true,
  showTicketStats: true,
  showRecentAlerts: true,
  showLatestTickets: true,
  showQuickActions: true,
};

// ── Default App Settings ─────────────────────────────────────────────

const DEFAULT_SETTINGS: AppSettings = {
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
  },
};

// ── Store Interface ──────────────────────────────────────────────────

interface SettingsState {
  // App settings (client-side)
  settings: AppSettings;
  updateSettings: (updates: Partial<AppSettings>) => void;

  // Dashboard layout config (persisted to localStorage)
  dashboardConfig: DashboardLayoutConfig;
  updateDashboardConfig: (updates: Partial<DashboardLayoutConfig>) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      settings: DEFAULT_SETTINGS,
      dashboardConfig: DEFAULT_DASHBOARD_CONFIG,

      updateSettings: (updates) =>
        set((state) => ({
          settings: {
            ...state.settings,
            ...updates,
            notifications: {
              ...state.settings.notifications,
              ...(updates.notifications || {}),
            },
            system: {
              ...state.settings.system,
              ...(updates.system || {}),
            },
            security: {
              ...state.settings.security,
              ...(updates.security || {}),
            },
          },
        })),

      updateDashboardConfig: (updates) =>
        set((state) => ({
          dashboardConfig: { ...state.dashboardConfig, ...updates },
        })),
    }),
    {
      name: 'cctv-settings',
      partialize: (state) => ({
        settings: state.settings,
        dashboardConfig: state.dashboardConfig,
      }),
    }
  )
);
