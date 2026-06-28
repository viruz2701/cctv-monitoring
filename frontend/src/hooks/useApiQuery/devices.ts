// ═══════════════════════════════════════════════════════════════════════
// Devices, Sites, Tickets, Alarms — React Query Hooks
// ═══════════════════════════════════════════════════════════════════════

import { useQuery, useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { queryKeys, CACHE } from './shared';
import type { Device, Site, Ticket } from './shared';

// ═══════════════════════════════════════════════════════════════════════
// Devices
// ═══════════════════════════════════════════════════════════════════════

export function useDevices() {
  return useQuery({
    queryKey: queryKeys.devices.all,
    queryFn: () => api.getDevices(),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
    refetchInterval: 60_000,
  });
}

export function useDevice(id: string) {
  return useQuery({
    queryKey: queryKeys.devices.detail(id),
    queryFn: () => api.getDevice(id),
    enabled: !!id,
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
  });
}

export function useCreateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (device: Partial<Device>) =>
      api.createDevice(device),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.devices.all });
    },
  });
}

export function useUpdateDevice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<Device> }) =>
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
// Sites (Reference Data)
// ═══════════════════════════════════════════════════════════════════════

export function useSites() {
  return useQuery({
    queryKey: queryKeys.sites.all,
    queryFn: () => api.getSites(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}

export function useSite(id: string) {
  return useQuery({
    queryKey: queryKeys.sites.detail(id),
    queryFn: () => api.getSite(id),
    enabled: !!id,
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}

export function useCreateSite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (site: Partial<Site>) =>
      api.createSite(site),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.sites.all });
    },
  });
}

export function useUpdateSite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<Site> }) =>
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
// Tickets (List Data)
// ═══════════════════════════════════════════════════════════════════════

export function useTickets() {
  return useQuery({
    queryKey: queryKeys.tickets.all,
    queryFn: () => api.getTickets(),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
    refetchInterval: 120_000,
  });
}

export function useTicket(id: string) {
  return useQuery({
    queryKey: queryKeys.tickets.detail(id),
    queryFn: () => api.getTicket(id),
    enabled: !!id,
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
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
// Alarms (Real-time)
// ═══════════════════════════════════════════════════════════════════════

export function useAlarms(deviceId?: string) {
  return useQuery({
    queryKey: deviceId ? queryKeys.alarms.byDevice(deviceId) : queryKeys.alarms.all,
    queryFn: () => api.getAlarms(deviceId),
    staleTime: CACHE.RT_STALE,
    gcTime: CACHE.RT_GC,
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
// Prefetch utilities
// ═══════════════════════════════════════════════════════════════════════

/**
 * Prefetch device detail on row hover.
 * Использование: onRowHover={(device) => prefetchDevice(queryClient, device.id)}
 */
export function prefetchDevice(client: QueryClient, id: string) {
  if (!id) return;
  client.prefetchQuery({
    queryKey: queryKeys.devices.detail(id),
    queryFn: () => api.getDevice(id),
    staleTime: 30_000,
  });
}
