// ═══════════════════════════════════════════════════════════════════════
// useSearchEntities — cross-entity search hook (P2-4)
// Searches across WO, Devices, Sites, Parts, Users via API with debounce.
// Uses React Query for caching and deduplication.
// ═══════════════════════════════════════════════════════════════════════

import { useQuery } from '@tanstack/react-query';
import { useMemo, useState, useEffect } from 'react';
import { api } from '../services/api';
import { workOrdersApi } from '../services/workOrdersApi';
import { sparePartsApi } from '../services/sparePartsApi';
import type { Site, Device } from '../types';
import type { WorkOrder } from '../services/workOrdersApi';
import type { SparePart } from '../services/sparePartsApi';

// ── Types ────────────────────────────────────────────────────────────

export interface EntitySearchResult {
  entities: {
    sites: SiteResult[];
    devices: DeviceResult[];
    workOrders: WOResult[];
    spareParts: PartResult[];
    users: UserResult[];
  };
  total: number;
}

export interface SiteResult {
  id: string;
  label: string;
  description: string;
  type: 'site';
  path: string;
}

export interface DeviceResult {
  id: string;
  label: string;
  description: string;
  type: 'device';
  path: string;
}

export interface WOResult {
  id: string;
  label: string;
  description: string;
  type: 'work-order';
  path: string;
}

export interface PartResult {
  id: string;
  label: string;
  description: string;
  type: 'spare-part';
  path: string;
}

export interface UserResult {
  id: string;
  label: string;
  description: string;
  type: 'user';
  path: string;
}

export type EntityResult =
  | SiteResult
  | DeviceResult
  | WOResult
  | PartResult
  | UserResult;

// ── Debounce Hook ────────────────────────────────────────────────────

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

// ── Search Query ─────────────────────────────────────────────────────

export function useSearchEntities(query: string, enabled = true) {
  const debouncedQuery = useDebounce(query, 300);

  const shouldSearch = debouncedQuery.trim().length >= 2 && enabled;

  // Sites search
  const sitesQuery = useQuery({
    queryKey: ['search', 'sites', debouncedQuery],
    queryFn: async (): Promise<SiteResult[]> => {
      const all = await api.getSites();
      const q = debouncedQuery.toLowerCase();
      return all
        .filter((s: any) =>
          (s.name || '').toLowerCase().includes(q) ||
          (s.address || '').toLowerCase().includes(q) ||
          (s.city || '').toLowerCase().includes(q) ||
          (s.organization || '').toLowerCase().includes(q)
        )
        .slice(0, 5)
        .map((s: any) => ({
          id: s.id,
          label: s.name || s.id,
          description: s.city ? `${s.address}, ${s.city}` : s.address || 'Site',
          type: 'site' as const,
          path: `/sites`,
        }));
    },
    enabled: shouldSearch,
    staleTime: 15_000,
  });

  // Devices search
  const devicesQuery = useQuery({
    queryKey: ['search', 'devices', debouncedQuery],
    queryFn: async (): Promise<DeviceResult[]> => {
      const all = await api.getDevices();
      const q = debouncedQuery.toLowerCase();
      return all
        .filter((d: any) =>
          (d.name || '').toLowerCase().includes(q) ||
          (d.device_id || '').toLowerCase().includes(q) ||
          (d.vendor_type || '').toLowerCase().includes(q)
        )
        .slice(0, 5)
        .map((d: any) => ({
          id: d.device_id,
          label: d.name || d.device_id,
          description: `${d.vendor_type || 'device'} — ${d.status || 'unknown'}`,
          type: 'device' as const,
          path: `/devices/${d.device_id}`,
        }));
    },
    enabled: shouldSearch,
    staleTime: 15_000,
  });

  // Work Orders search
  const woQuery = useQuery({
    queryKey: ['search', 'workOrders', debouncedQuery],
    queryFn: async (): Promise<WOResult[]> => {
      const all = await workOrdersApi.getWorkOrders();
      const q = debouncedQuery.toLowerCase();
      return all
        .filter((wo: any) =>
          (wo.id || '').toLowerCase().includes(q) ||
          (wo.device_name || '').toLowerCase().includes(q) ||
          (wo.type || '').toLowerCase().includes(q)
        )
        .slice(0, 5)
        .map((wo: any) => ({
          id: wo.id,
          label: `#${wo.id.slice(0, 8)} — ${wo.type || 'WO'}`,
          description: `${wo.status || 'open'} · ${wo.device_name || wo.device_id || ''}`,
          type: 'work-order' as const,
          path: `/work-orders/${wo.id}`,
        }));
    },
    enabled: shouldSearch,
    staleTime: 15_000,
  });

  // Spare Parts search
  const partsQuery = useQuery({
    queryKey: ['search', 'spareParts', debouncedQuery],
    queryFn: async (): Promise<PartResult[]> => {
      const all = await sparePartsApi.getSpareParts();
      const q = debouncedQuery.toLowerCase();
      return all
        .filter((p: any) =>
          (p.name || '').toLowerCase().includes(q) ||
          (p.sku || '').toLowerCase().includes(q) ||
          (p.category || '').toLowerCase().includes(q)
        )
        .slice(0, 5)
        .map((p: any) => ({
          id: p.id,
          label: p.name || p.id,
          description: `${p.sku || 'no SKU'} · ${p.quantity || 0} in stock`,
          type: 'spare-part' as const,
          path: `/spare-parts`,
        }));
    },
    enabled: shouldSearch,
    staleTime: 15_000,
  });

  // Users search
  const usersQuery = useQuery({
    queryKey: ['search', 'users', debouncedQuery],
    queryFn: async (): Promise<UserResult[]> => {
      const all = await api.getUsers();
      const q = debouncedQuery.toLowerCase();
      return all
        .filter((u: any) =>
          (u.name || '').toLowerCase().includes(q) ||
          (u.username || '').toLowerCase().includes(q) ||
          (u.email || '').toLowerCase().includes(q) ||
          (u.role || '').toLowerCase().includes(q)
        )
        .slice(0, 3)
        .map((u: any) => ({
          id: u.id,
          label: u.name || u.username,
          description: `${u.role || 'user'} · ${u.email || ''}`,
          type: 'user' as const,
          path: `/users`,
        }));
    },
    enabled: shouldSearch,
    staleTime: 15_000,
  });

  // Merged results
  const allQueries = [sitesQuery, devicesQuery, woQuery, partsQuery, usersQuery];
  const isLoading = allQueries.some((q) => q.isLoading);
  const isFetching = allQueries.some((q) => q.isFetching);

  const results = useMemo((): EntityResult[] => {
    const merged: EntityResult[] = [
      ...(sitesQuery.data ?? []),
      ...(devicesQuery.data ?? []),
      ...(woQuery.data ?? []),
      ...(partsQuery.data ?? []),
      ...(usersQuery.data ?? []),
    ];
    return merged;
  }, [sitesQuery.data, devicesQuery.data, woQuery.data, partsQuery.data, usersQuery.data]);

  return {
    results,
    isLoading,
    isFetching,
    total: results.length,
    query: debouncedQuery,
  };
}
