// ═══════════════════════════════════════════════════════════════════════
// MaintenanceContext — React Query backed
// ARCH-02: Server state managed via React Query, context retained for
// backward compatibility.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext, useCallback } from 'react';
import type { MaintenanceSchedule, CreateScheduleRequest } from '../services/maintenanceApi';
import {
  useMaintenanceSchedules,
  useCreateMaintenanceSchedule,
  useUpdateMaintenanceSchedule,
  useDeleteMaintenanceSchedule,
  useCompleteMaintenanceSchedule,
} from '../hooks/useApiQuery';
import { useAuth } from '../hooks/useAuth';

interface MaintenanceContextType {
  schedules: MaintenanceSchedule[];
  loading: boolean;
  error: string | null;
  fetchSchedules: (filters?: Record<string, string>) => Promise<void>;
  createSchedule: (data: CreateScheduleRequest) => Promise<MaintenanceSchedule>;
  updateSchedule: (id: string, data: Partial<CreateScheduleRequest>) => Promise<void>;
  deleteSchedule: (id: string) => Promise<void>;
  completeSchedule: (id: string) => Promise<void>;
}

const MaintenanceContext = createContext<MaintenanceContextType | undefined>(undefined);

export const MaintenanceProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token } = useAuth();

  const {
    data: schedules = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useMaintenanceSchedules();

  const createMutation = useCreateMaintenanceSchedule();
  const updateMutation = useUpdateMaintenanceSchedule();
  const deleteMutation = useDeleteMaintenanceSchedule();
  const completeMutation = useCompleteMaintenanceSchedule();

  const fetchSchedules = useCallback(async (filters?: Record<string, string>) => {
    await refetch();
  }, [refetch]);

  const createSchedule = useCallback(async (data: CreateScheduleRequest): Promise<MaintenanceSchedule> => {
    return createMutation.mutateAsync(data);
  }, [createMutation]);

  const updateSchedule = useCallback(async (id: string, data: Partial<CreateScheduleRequest>) => {
    await updateMutation.mutateAsync({ id, data });
  }, [updateMutation]);

  const deleteSchedule = useCallback(async (id: string) => {
    await deleteMutation.mutateAsync(id);
  }, [deleteMutation]);

  const completeSchedule = useCallback(async (id: string) => {
    await completeMutation.mutateAsync(id);
  }, [completeMutation]);

  return (
    <MaintenanceContext.Provider
      value={{
        schedules,
        loading,
        error: queryError instanceof Error ? queryError.message : null,
        fetchSchedules,
        createSchedule,
        updateSchedule,
        deleteSchedule,
        completeSchedule,
      }}
    >
      {children}
    </MaintenanceContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useMaintenance = () => {
  const context = useContext(MaintenanceContext);
  if (!context) {
    throw new Error('useMaintenance must be used within MaintenanceProvider');
  }
  return context;
};
