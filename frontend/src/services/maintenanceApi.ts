import { request } from './api';

export interface MaintenanceSchedule {
  id: string;
  device_id: string;
  schedule_type: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'custom';
  interval_days: number;
  custom_cron?: string;
  last_completed?: string;
  next_due: string;
  assigned_to?: string;
  checklist: ChecklistItem[];
  estimated_minutes: number;
  priority: 'critical' | 'high' | 'medium' | 'low';
  notes?: string;
  created_at: string;
  updated_at: string;
  device_name?: string;
  assignee_name?: string;
}

export interface ChecklistItem {
  task: string;
  completed: boolean;
}

export interface CreateScheduleRequest {
  device_id: string;
  schedule_type: string;
  interval_days?: number;
  custom_cron?: string;
  next_due: string;
  assigned_to?: string;
  checklist?: ChecklistItem[];
  estimated_minutes?: number;
  priority?: string;
  notes?: string;
}

export const maintenanceApi = {
  getSchedules: (filters?: Record<string, string>) => {
    const params = new URLSearchParams(filters).toString();
    return request<MaintenanceSchedule[]>(`/maintenance/schedules${params ? '?' + params : ''}`);
  },

  getSchedule: (id: string) => {
    return request<MaintenanceSchedule>(`/maintenance/schedules/${id}`);
  },

  createSchedule: (data: CreateScheduleRequest) => {
    return request<MaintenanceSchedule>('/maintenance/schedules', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateSchedule: (id: string, data: Partial<CreateScheduleRequest>) => {
    return request<{ status: string }>(`/maintenance/schedules/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteSchedule: (id: string) => {
    return request<{ status: string }>(`/maintenance/schedules/${id}`, {
      method: 'DELETE',
    });
  },

  getDueSchedules: () => {
    return request<MaintenanceSchedule[]>('/maintenance/schedules/due');
  },

  completeSchedule: (id: string) => {
    return request<{ status: string }>(`/maintenance/schedules/${id}/complete`, {
      method: 'POST',
    });
  },
};
