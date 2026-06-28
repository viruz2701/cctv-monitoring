// ═══════════════════════════════════════════════════════════════════════
// Users, Notifications, Services, Audit Log — React Query Hooks
// ═══════════════════════════════════════════════════════════════════════

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { queryKeys, CACHE } from './shared';
import type { User, AppNotification, ServicesSettings } from './shared';

// ═══════════════════════════════════════════════════════════════════════
// Users (Reference Data)
// ═══════════════════════════════════════════════════════════════════════

export function useUsers() {
  return useQuery({
    queryKey: queryKeys.users.all,
    queryFn: () => api.getUsers(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}

export function useCurrentUser() {
  return useQuery({
    queryKey: queryKeys.users.me,
    queryFn: () => api.getCurrentUser(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
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
// Notifications (Real-time)
// ═══════════════════════════════════════════════════════════════════════

export function useNotifications() {
  return useQuery({
    queryKey: queryKeys.notifications.all,
    queryFn: () =>
      api.getNotifications().catch((err) => {
        if (err instanceof Error && err.message.includes('404')) {
          console.warn('Notifications endpoint not available (404), returning empty array');
          return [];
        }
        throw err;
      }),
    staleTime: CACHE.RT_STALE,
    gcTime: CACHE.RT_GC,
    refetchInterval: 30_000,
    retry: (failureCount, error) => {
      if (error instanceof Error && error.message.includes('404')) return false;
      return failureCount < 3;
    },
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
// Services Settings & Status
// ═══════════════════════════════════════════════════════════════════════

export function useServicesSettings() {
  return useQuery({
    queryKey: queryKeys.services.settings,
    queryFn: () => api.getServicesSettings(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
    retry: 1,
  });
}

export function useServicesStatus() {
  return useQuery({
    queryKey: queryKeys.services.status,
    queryFn: () => api.getServicesStatus(),
    staleTime: CACHE.RT_STALE,
    gcTime: CACHE.RT_GC,
    refetchInterval: 30_000,
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
// Audit Log (Reference Data)
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
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}
