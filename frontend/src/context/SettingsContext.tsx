// ═══════════════════════════════════════════════════════════════════════
// SettingsContext — Bridge (backward compat)
// ARCH-02: Новый код ДОЛЖЕН импортировать напрямую:
//   - useSettingsStore из '../store/settingsStore' (client-side)
//   - useServicesSettings, useServicesStatus из '../hooks/useApiQuery'
//
// Миграция:
//   Было:  import { useSettings } from '../context/SettingsContext'
//   Стало: import { useSettingsStore } from '../store/settingsStore'
//          import { useServicesSettings } from '../hooks/useApiQuery'
//
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect } from 'react';
import { useSettingsStore } from '../store/settingsStore';
import {
  useServicesSettings,
  useServicesStatus,
  useUpdateServicesSettings,
} from '../hooks/useApiQuery';
import type { ServicesSettings } from '../types';

export interface ServiceStatusEntry {
  status: 'running' | 'stopped' | 'disabled' | 'error';
  port: number;
  message?: string;
}

export type ServicesStatusMap = Record<string, ServiceStatusEntry>;

// ── Hook: useSettings — backward compat ─────────────────────────────
// Предоставляет тот же API, что и старый SettingsContext,
// но использует Zustand для локальных настроек + React Query для server state.
// Edit buffer для services settings управляется локально через useState.

export function useSettings() {
  const settings = useSettingsStore((s) => s.settings);
  const dashboardConfig = useSettingsStore((s) => s.dashboardConfig);
  const updateSettings = useSettingsStore((s) => s.updateSettings);
  const updateDashboardConfig = useSettingsStore((s) => s.updateDashboardConfig);

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

  const updateServicesSettingsMutation = useUpdateServicesSettings();

  // ── Edit buffer for services settings ─────────────────────────────
  // Сохраняем копию для редактирования перед сохранением
  const [editBuffer, setEditBuffer] = useState<ServicesSettings | null>(null);

  useEffect(() => {
    if (servicesSettingsData && !editBuffer) {
      setEditBuffer(servicesSettingsData);
    }
  }, [servicesSettingsData, editBuffer]);

  const updateServicesSettings = (updates: Partial<ServicesSettings>) => {
    setEditBuffer((prev) => {
      if (!prev) return prev;
      return { ...prev, ...updates };
    });
  };

  const saveServicesSettings = async () => {
    if (!editBuffer) return;
    await updateServicesSettingsMutation.mutateAsync(editBuffer);
  };

  const handleRefreshServicesSettings = async () => {
    await refreshServicesSettings();
  };

  const handleRefreshServicesStatus = async () => {
    await refreshServicesStatus();
  };

  return {
    settings,
    dashboardConfig,
    servicesSettings: editBuffer,
    servicesLoading,
    servicesStatus: (servicesStatusData?.services || {}) as ServicesStatusMap,
    servicesStatusLoading,
    updateSettings,
    updateDashboardConfig,
    updateServicesSettings,
    saveServicesSettings,
    refreshServicesSettings: handleRefreshServicesSettings,
    refreshServicesStatus: handleRefreshServicesStatus,
  };
}
