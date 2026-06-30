// ═══════════════════════════════════════════════════════════════════════
// Shared types, query key factory, and cache constants
// ═══════════════════════════════════════════════════════════════════════

import type { QueryClient } from '@tanstack/react-query';

export type {
  Device, Site, Ticket, Alarm, Notification as AppNotification,
  User, DashboardStats, Report, AuditLogEntry,
  ServicesSettings,
} from '../../services/api';

export type { WorkOrder } from '../../services/workOrdersApi';
export type {
  MaintenanceSchedule, CreateScheduleRequest,
} from '../../services/maintenanceApi';
export type {
  SparePart, CreateSparePartRequest, SparePartCategory,
} from '../../services/sparePartsApi';
export type { Prediction } from '../../services/api';

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
  // EDGE-11: Agent Monitoring Dashboard
  agents: {
    all: ['agents'] as const,
    detail: (id: string) => ['agents', id] as const,
  },
};

// ═══════════════════════════════════════════════════════════════════════
// Cache strategy constants
// P1-2.3: React Query Optimization
//   - Reference data → staleTime: 5min, gcTime: 1h
//   - Lists → staleTime: 30s, gcTime: 5min
//   - Real-time → staleTime: 15s, gcTime: 2min
// ═══════════════════════════════════════════════════════════════════════

export const CACHE = {
  /** Reference data: sites, users, categories — rarely changes */
  REF_STALE: 300_000,     // 5 min
  REF_GC: 3_600_000,      // 1 hour
  /** Lists: devices, tickets, work orders — changes infrequently */
  LIST_STALE: 30_000,     // 30 sec
  LIST_GC: 300_000,       // 5 min
  /** Real-time: alarms, notifications, service status — changes frequently */
  RT_STALE: 15_000,       // 15 sec
  RT_GC: 120_000,         // 2 min
} as const;
