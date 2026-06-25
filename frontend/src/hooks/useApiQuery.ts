// ═══════════════════════════════════════════════════════════════════════
// React Query Hooks
// ARCH-02: Миграция с React Context провайдеров на React Query для server state.
//
// Каждый query hook:
//   - Изолирован — не вызывает cascading re-renders
//   - Автоматически кэширует и инвалидирует данные
//   - Поддерживает retry, stale-while-revalidate, refetchOnFocus
//   - Типизирован через TypeScript
//
// Compliance:
//   - OWASP ASVS V1.8 (Architecture — stateless design)
//   - IEC 62443 SR 7.1 (Resource availability — async data fetching)
// ═══════════════════════════════════════════════════════════════════════

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../services/api';
import type {
  Device, Site, Ticket, Alarm, Notification as AppNotification,
  User, DashboardStats, Report, AuditLogEntry,
  ServicesSettings,
} from '../services/api';
import type { WorkOrder } from '../services/workOrdersApi';
import { workOrdersApi } from '../services/workOrdersApi';
import { maintenanceApi, type MaintenanceSchedule, type CreateScheduleRequest } from '../services/maintenanceApi';
import { sparePartsApi, type SparePart, type CreateSparePartRequest, type SparePartCategory } from '../services/sparePartsApi';

// ═══════════════════════════════════════════════════════════════════════
// Query Key Factory
// ═══════════════════════════════════════════════════════════════════════

export const queryKeys = {
  devices: {
    all: ['devices'] as const,
    detail: (id: string) => ['devices', id] as const,
  },
  sites: {
    all: ['sites'] as const,
    detail: (id: string) => ['sites', id] as const,
  },
  tickets: {
    all: ['tickets'] as const,
    detail: (id: string) => ['tickets', id] as const,
  },
  alarms: {
    all: ['alarms'] as const,
    byDevice: (deviceId: string) => ['alarms', 'device', deviceId] as const,
  },
  users: {
    all: ['users'] as const,
    me: ['users', 'me'] as const,
    detail: (id: string) => ['users', id] as const,
  },
  workOrders: {
    all: ['workOrders'] as const,
    detail: (id: string) => ['workOrders', id] as const,
  },
  notifications: {
    all: ['notifications'] as const,
  },
  reports: {
    all: ['reports'] as const,
  },
  dashboard: {
    stats: ['dashboard', 'stats'] as const,
  },
  auditLog: {
    all: ['auditLog'] as const,
  },
  services: {
    settings: ['services', 'settings'] as const,
    status: ['services', 'status'] as const,
  },
  maintenance: {
    all: ['maintenance'] as const,
    detail: (id: string) => ['maintenance', id] as const,
  },
  spareParts: {
    all: ['spareParts'] as const,
    lowStock: ['spareParts', 'lowStock'] as const,
    categories: ['spareParts', 'categories'] as const,
    detail: (id: string) => ['spareParts', id] as const,
  },
  predictions: {
    all: ['predictions'] as const,
    stats: ['predictions', 'stats'] as const,
  },
};

// ═══════════════════════════════════════════════════════════════════════
// Devices
// ═══════════════════════════════════════════════════════════════════════

export function useDevices() {
  return useQuery({
    queryKey: queryKeys.devices.all,
    queryFn: () => api.getDevices(),
    staleTime: 30_000, // 30s — данные устройств меняются нечасто
    refetchInterval: 60_000, // background refresh каждую минуту
  });
}

export function useDevice(id: string) {
  return useQuery({
    queryKey: queryKeys.devices.detail(id),
    queryFn: () => api.getDevice(id),
    enabled: !!id,
    staleTime: 30_000,
  });
}

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (device: Partial<import('../services/api').Device>) =>
      api.createDevice(device),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.devices.all });
    },
  });
}

export function useUpdateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<import('../services/api').Device> }) =>
      api.updateDevice(id, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.devices.all });
    },
  });
}

export function useDeleteDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteDevice(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.devices.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Sites
// ═══════════════════════════════════════════════════════════════════════

export function useSites() {
  return useQuery({
    queryKey: queryKeys.sites.all,
    queryFn: () => api.getSites(),
    staleTime: 60_000,
  });
}

export function useSite(id: string) {
  return useQuery({
    queryKey: queryKeys.sites.detail(id),
    queryFn: () => api.getSite(id),
    enabled: !!id,
    staleTime: 60_000,
  });
}

export function useCreateSite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (site: Partial<import('../services/api').Site>) =>
      api.createSite(site),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sites.all });
    },
  });
}

export function useUpdateSite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<import('../services/api').Site> }) =>
      api.updateSite(id, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sites.all });
    },
  });
}

export function useDeleteSite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteSite(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sites.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Tickets
// ═══════════════════════════════════════════════════════════════════════

export function useTickets() {
  return useQuery({
    queryKey: queryKeys.tickets.all,
    queryFn: () => api.getTickets(),
    staleTime: 30_000,
    refetchInterval: 120_000,
  });
}

export function useTicket(id: string) {
  return useQuery({
    queryKey: queryKeys.tickets.detail(id),
    queryFn: () => api.getTicket(id),
    enabled: !!id,
    staleTime: 30_000,
  });
}

export function useCreateTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ticket: Partial<Ticket>) => api.createTicket(ticket),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tickets.all });
    },
  });
}

export function useUpdateTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<Ticket> }) =>
      api.updateTicket(id, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tickets.all });
    },
  });
}

export function useDeleteTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteTicket(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tickets.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Alarms
// ═══════════════════════════════════════════════════════════════════════

export function useAlarms(deviceId?: string) {
  return useQuery({
    queryKey: deviceId ? queryKeys.alarms.byDevice(deviceId) : queryKeys.alarms.all,
    queryFn: () => api.getAlarms(deviceId),
    staleTime: 15_000,
    refetchInterval: 30_000,
  });
}

export function useAcknowledgeAlarm() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (alarmId: string) => api.acknowledgeAlarm(alarmId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alarms.all });
    },
  });
}

export function useResolveAlarm() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (alarmId: string) => api.resolveAlarm(alarmId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.alarms.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Notifications
// ═══════════════════════════════════════════════════════════════════════

export function useNotifications() {
  return useQuery({
    queryKey: queryKeys.notifications.all,
    queryFn: () => api.getNotifications(),
    staleTime: 15_000,
    refetchInterval: 30_000,
  });
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.markNotificationRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.notifications.all });
    },
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.markAllNotificationsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.notifications.all });
    },
  });
}

export function useDeleteNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteNotification(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.notifications.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Users
// ═══════════════════════════════════════════════════════════════════════

export function useUsers() {
  return useQuery({
    queryKey: queryKeys.users.all,
    queryFn: () => api.getUsers(),
    staleTime: 60_000,
  });
}

export function useCurrentUser() {
  return useQuery({
    queryKey: queryKeys.users.me,
    queryFn: () => api.getCurrentUser(),
    staleTime: 300_000, // 5 min — пользователь редко меняется
  });
}

// ═══════════════════════════════════════════════════════════════════════
// User Mutations
// ═══════════════════════════════════════════════════════════════════════

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (user: { username: string; password: string; role: string; email?: string }) =>
      api.createUser(user),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<User> }) =>
      api.updateUser(id, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Work Orders
// ═══════════════════════════════════════════════════════════════════════

export function useWorkOrders(filters?: Record<string, string>) {
  return useQuery({
    queryKey: [...queryKeys.workOrders.all, filters],
    queryFn: () => workOrdersApi.getWorkOrders(filters),
    staleTime: 30_000,
    refetchInterval: 120_000,
  });
}

export function useCreateWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: import('../services/workOrdersApi').CreateWorkOrderRequest) =>
      workOrdersApi.createWorkOrder(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    },
  });
}

export function useUpdateWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<WorkOrder> }) =>
      workOrdersApi.updateWorkOrder(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    },
  });
}

export function useDeleteWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => workOrdersApi.deleteWorkOrder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Dashboard
// ═══════════════════════════════════════════════════════════════════════

export function useDashboardStats() {
  return useQuery({
    queryKey: queryKeys.dashboard.stats,
    queryFn: () => api.getDashboardStats(),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Reports
// ═══════════════════════════════════════════════════════════════════════

export function useReports() {
  return useQuery({
    queryKey: queryKeys.reports.all,
    queryFn: () => api.getReports(),
    staleTime: 60_000,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Services Settings & Status
// ═══════════════════════════════════════════════════════════════════════

export function useServicesSettings() {
  return useQuery({
    queryKey: queryKeys.services.settings,
    queryFn: () => api.getServicesSettings(),
    staleTime: 60_000,
    retry: 1,
  });
}

export function useServicesStatus() {
  return useQuery({
    queryKey: queryKeys.services.status,
    queryFn: () => api.getServicesStatus(),
    staleTime: 15_000,
    refetchInterval: 30_000, // polling every 30s
  });
}

export function useUpdateServicesSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (settings: Partial<ServicesSettings>) =>
      api.updateServicesSettings(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.services.settings });
      queryClient.invalidateQueries({ queryKey: queryKeys.services.status });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Maintenance Schedules
// ═══════════════════════════════════════════════════════════════════════

export function useMaintenanceSchedules(filters?: Record<string, string>) {
  return useQuery({
    queryKey: [...queryKeys.maintenance.all, filters],
    queryFn: () => maintenanceApi.getSchedules(filters),
    staleTime: 30_000,
  });
}

export function useCreateMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateScheduleRequest) =>
      maintenanceApi.createSchedule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

export function useUpdateMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateScheduleRequest> }) =>
      maintenanceApi.updateSchedule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

export function useDeleteMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => maintenanceApi.deleteSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

export function useCompleteMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => maintenanceApi.completeSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Parts
// ═══════════════════════════════════════════════════════════════════════

export function useSpareParts(filters?: Record<string, string>) {
  return useQuery({
    queryKey: [...queryKeys.spareParts.all, filters],
    queryFn: () => sparePartsApi.getSpareParts(filters),
    staleTime: 30_000,
  });
}

export function useLowStockParts() {
  return useQuery({
    queryKey: queryKeys.spareParts.lowStock,
    queryFn: () => sparePartsApi.getLowStockParts(),
    staleTime: 30_000,
  });
}

export function useSparePartCategories() {
  return useQuery({
    queryKey: queryKeys.spareParts.categories,
    queryFn: () => sparePartsApi.getCategories(),
    staleTime: 60_000,
  });
}

export function useCreateSparePart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSparePartRequest) =>
      sparePartsApi.createSparePart(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useUpdateSparePart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateSparePartRequest> }) =>
      sparePartsApi.updateSparePart(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useDeleteSparePart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => sparePartsApi.deleteSparePart(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useAdjustStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, quantity }: { id: string; quantity: number }) =>
      sparePartsApi.adjustStock(id, quantity),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useCreateSparePartCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; description?: string; color?: string }) =>
      sparePartsApi.createCategory(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.categories });
    },
  });
}

export function useUpdateSparePartCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name?: string; description?: string; color?: string } }) =>
      sparePartsApi.updateCategory(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.categories });
    },
  });
}

export function useDeleteSparePartCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => sparePartsApi.deleteCategory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.categories });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Audit Log
// ═══════════════════════════════════════════════════════════════════════

export function useAuditLog(params?: {
  user_id?: string;
  action?: string;
  entity_type?: string;
  time_from?: string;
  time_to?: string;
  limit?: number;
}) {
  return useQuery({
    queryKey: [...queryKeys.auditLog.all, params],
    queryFn: () => api.getAuditLog(params),
    staleTime: 60_000,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Predictions / Predictive Maintenance (KF-15.1.3)
// ═══════════════════════════════════════════════════════════════════════

import type { Prediction } from '../services/api';

export function usePredictions(deviceId?: string, limit?: number) {
  return useQuery({
    queryKey: [...queryKeys.predictions.all, deviceId, limit],
    queryFn: () => api.getPredictions(deviceId, limit),
    staleTime: 60_000,
    refetchInterval: 300_000, // auto-refresh каждые 5 минут
  });
}
