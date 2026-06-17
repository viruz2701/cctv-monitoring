import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { maintenanceApi, MaintenanceSchedule, CreateScheduleRequest } from '../services/maintenanceApi';
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
  const [schedules, setSchedules] = useState<MaintenanceSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSchedules = useCallback(async (filters?: Record<string, string>) => {
    setLoading(true);
    setError(null);
    try {
      const data = await maintenanceApi.getSchedules(filters);
      setSchedules(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch schedules');
    } finally {
      setLoading(false);
    }
  }, []);

  const createSchedule = async (data: CreateScheduleRequest): Promise<MaintenanceSchedule> => {
    const schedule = await maintenanceApi.createSchedule(data);
    setSchedules((prev) => [...prev, schedule]);
    return schedule;
  };

  const updateSchedule = async (id: string, data: Partial<CreateScheduleRequest>) => {
    await maintenanceApi.updateSchedule(id, data);
    setSchedules((prev) =>
      prev.map((s) => (s.id === id ? { ...s, ...data } as MaintenanceSchedule : s))
    );
  };

  const deleteSchedule = async (id: string) => {
    await maintenanceApi.deleteSchedule(id);
    setSchedules((prev) => prev.filter((s) => s.id !== id));
  };

  const completeSchedule = async (id: string) => {
    await maintenanceApi.completeSchedule(id);
    setSchedules((prev) =>
      prev.map((s) =>
        s.id === id
          ? { ...s, last_completed: new Date().toISOString(), next_due: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString() }
          : s
      )
    );
  };

  useEffect(() => {
    if (!token) return;
    fetchSchedules();
  }, [fetchSchedules, token]);

  return (
    <MaintenanceContext.Provider
      value={{ schedules, loading, error, fetchSchedules, createSchedule, updateSchedule, deleteSchedule, completeSchedule }}
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
