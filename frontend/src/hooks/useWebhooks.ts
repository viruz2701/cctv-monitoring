// ═══════════════════════════════════════════════════════════════════════
// useWebhooks — React Query хуки для Webhook Builder (P2-3.1)
//
// Следует паттерну useApiQuery.ts:
//   - queryKeys для инвалидации
//   - staleTime/gcTime по типу данных
//   - onSuccess инвалидация списка
//
// Compliance:
//   - OWASP ASVS V7 (Error Handling — traceID в каждом запросе)
//   - IEC 62443 SR 7.1 (Resource availability — async data fetching)
// ═══════════════════════════════════════════════════════════════════════

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { WebhookEndpoint } from '../services/api';
import { api } from '../services/api';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface WebhookLogEntry {
  id: string;
  webhook_id: string;
  event_type: string;
  status: 'success' | 'failed';
  request_url: string;
  request_body: string;
  response_status: number;
  response_body: string;
  duration_ms: number;
  retry_attempt: number;
  error_message?: string;
  created_at: string;
}

export interface WebhookFormData {
  name: string;
  url: string;
  events: string[];
  secret: string;
  active: boolean;
  retry_count: number;
  timeout_seconds: number;
  retry_interval_seconds: number;
  retry_backoff: boolean;
  max_retry_duration_seconds: number;
}

export interface TestWebhookResult {
  status: string;
  status_code: number;
  duration_ms: number;
  response_body: string;
  error?: string;
}

export interface WebhookStats {
  total_deliveries_24h: number;
  total_deliveries_7d: number;
  total_deliveries_30d: number;
  success_rate: number;
  avg_latency_ms: number;
  active: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Mock Payloads per Event Type
// ═══════════════════════════════════════════════════════════════════════

export const EVENT_PAYLOADS: Record<string, Record<string, unknown>> = {
  'sla.breached': {
    event: 'sla.breached',
    timestamp: new Date().toISOString(),
    data: {
      work_order_id: 'wo-001',
      site_name: 'Main Facility',
      priority: 'critical',
      sla_deadline: new Date(Date.now() + 3600000).toISOString(),
      response_time_minutes: 45,
      threshold_minutes: 30,
    },
  },
  'sla.at_risk': {
    event: 'sla.at_risk',
    timestamp: new Date().toISOString(),
    data: {
      work_order_id: 'wo-002',
      site_name: 'Secondary Facility',
      priority: 'high',
      sla_deadline: new Date(Date.now() + 7200000).toISOString(),
      remaining_minutes: 25,
    },
  },
  'device.offline': {
    event: 'device.offline',
    timestamp: new Date().toISOString(),
    data: {
      device_id: 'cam-042',
      device_name: 'Parking Lot Camera',
      site_name: 'Main Facility',
      last_seen: new Date(Date.now() - 300000).toISOString(),
      ip_address: '192.168.1.100',
    },
  },
  'device.online': {
    event: 'device.online',
    timestamp: new Date().toISOString(),
    data: {
      device_id: 'cam-042',
      device_name: 'Parking Lot Camera',
      site_name: 'Main Facility',
      ip_address: '192.168.1.100',
    },
  },
  'device.status_changed': {
    event: 'device.status_changed',
    timestamp: new Date().toISOString(),
    data: {
      device_id: 'cam-017',
      device_name: 'Entrance Camera',
      site_name: 'Main Facility',
      previous_status: 'online',
      new_status: 'warning',
      reason: 'high_cpu_usage',
    },
  },
  'work_order.created': {
    event: 'work_order.created',
    timestamp: new Date().toISOString(),
    data: {
      work_order_id: 'wo-003',
      title: 'Emergency Camera Repair',
      site_name: 'Warehouse A',
      priority: 'critical',
      work_type: 'repair',
      assigned_to: 'tech-john',
    },
  },
  'work_order.updated': {
    event: 'work_order.updated',
    timestamp: new Date().toISOString(),
    data: {
      work_order_id: 'wo-003',
      title: 'Emergency Camera Repair',
      status: 'in_progress',
      previous_status: 'open',
      updated_by: 'tech-john',
    },
  },
  'work_order.completed': {
    event: 'work_order.completed',
    timestamp: new Date().toISOString(),
    data: {
      work_order_id: 'wo-003',
      title: 'Emergency Camera Repair',
      completed_at: new Date().toISOString(),
      completed_by: 'tech-john',
      resolution: 'Replaced faulty cable',
    },
  },
  'work_order.cancelled': {
    event: 'work_order.cancelled',
    timestamp: new Date().toISOString(),
    data: {
      work_order_id: 'wo-003',
      title: 'Emergency Camera Repair',
      cancelled_by: 'manager-sarah',
      reason: 'Duplicate request',
    },
  },
  'alarm.created': {
    event: 'alarm.created',
    timestamp: new Date().toISOString(),
    data: {
      alarm_id: 'alarm-081',
      device_id: 'cam-017',
      device_name: 'Entrance Camera',
      severity: 'critical',
      alarm_type: 'motion_detection',
      description: 'Unauthorized movement detected',
    },
  },
  'alarm.resolved': {
    event: 'alarm.resolved',
    timestamp: new Date().toISOString(),
    data: {
      alarm_id: 'alarm-081',
      device_id: 'cam-017',
      resolved_by: 'security-mike',
      resolution: 'False alarm — authorized personnel',
    },
  },
};

// ═══════════════════════════════════════════════════════════════════════
// Event Type Grouping
// ═══════════════════════════════════════════════════════════════════════

export const EVENT_GROUPS = [
  {
    label: 'Work Orders',
    events: [
      { value: 'work_order.created', label: 'WO Created' },
      { value: 'work_order.updated', label: 'WO Updated' },
      { value: 'work_order.completed', label: 'WO Completed' },
      { value: 'work_order.cancelled', label: 'WO Cancelled' },
    ],
  },
  {
    label: 'Alarms',
    events: [
      { value: 'alarm.created', label: 'Alarm Created' },
      { value: 'alarm.resolved', label: 'Alarm Resolved' },
    ],
  },
  {
    label: 'Devices',
    events: [
      { value: 'device.offline', label: 'Device Offline' },
      { value: 'device.online', label: 'Device Online' },
      { value: 'device.status_changed', label: 'Device Status Changed' },
    ],
  },
  {
    label: 'SLA',
    events: [
      { value: 'sla.breached', label: 'SLA Breached' },
      { value: 'sla.at_risk', label: 'SLA At Risk' },
    ],
  },
];

// ═══════════════════════════════════════════════════════════════════════
// Query Key Factory
// ═══════════════════════════════════════════════════════════════════════

export const webhookKeys = {
  all: ['webhooks'] as const,
  detail: (id: string) => ['webhooks', id] as const,
  logs: (id: string) => ['webhooks', id, 'logs'] as const,
  stats: (id: string) => ['webhooks', id, 'stats'] as const,
};

// ═══════════════════════════════════════════════════════════════════════
// Webhook API extensions (routes not in api.ts)
// ═══════════════════════════════════════════════════════════════════════

const WEBHOOK_BASE = '/integrations/extended/webhooks';

async function getWebhookLogs(id: string): Promise<WebhookLogEntry[]> {
  const { request } = await import('../services/api');
  return request<WebhookLogEntry[]>(`${WEBHOOK_BASE}/${id}/logs`);
}

async function getWebhookStats(id: string): Promise<WebhookStats> {
  const { request } = await import('../services/api');
  return request<WebhookStats>(`${WEBHOOK_BASE}/${id}/stats`);
}

async function testWebhookWithPayload(
  id: string,
  payload: Record<string, unknown>
): Promise<TestWebhookResult> {
  const { request } = await import('../services/api');
  return request<TestWebhookResult>(`${WEBHOOK_BASE}/${id}/test`, {
    method: 'POST',
    body: JSON.stringify({ event_type: payload.event, payload }),
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Queries
// ═══════════════════════════════════════════════════════════════════════

/**
 * Получить все вебхуки
 */
export function useWebhooks() {
  return useQuery({
    queryKey: webhookKeys.all,
    queryFn: () => api.getWebhooks(),
    staleTime: 30_000,
    gcTime: 300_000,
  });
}

/**
 * Получить один вебхук по ID
 */
export function useWebhook(id: string | undefined) {
  return useQuery({
    queryKey: webhookKeys.detail(id!),
    queryFn: () => api.getWebhooks().then((list) => list.find((w) => w.id === id)),
    enabled: !!id,
    staleTime: 30_000,
    gcTime: 300_000,
  });
}

/**
 * Получить логи доставки для вебхука
 */
export function useWebhookLogsQuery(id: string | undefined) {
  return useQuery({
    queryKey: webhookKeys.logs(id!),
    queryFn: () => getWebhookLogs(id!),
    enabled: !!id,
    staleTime: 15_000,
    gcTime: 60_000,
    refetchInterval: 30_000,
  });
}

/**
 * Получить статистику вебхука (с автообновлением каждые 30с)
 */
export function useWebhookStats(id: string | undefined) {
  return useQuery({
    queryKey: webhookKeys.stats(id!),
    queryFn: () => getWebhookStats(id!),
    enabled: !!id,
    staleTime: 10_000,
    gcTime: 60_000,
    refetchInterval: 30_000,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Mutations
// ═══════════════════════════════════════════════════════════════════════

/**
 * Создать новый вебхук
 */
export function useCreateWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: WebhookFormData) =>
      api.createWebhook(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.all });
    },
  });
}

/**
 * Обновить существующий вебхук
 */
export function useUpdateWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<WebhookFormData> }) =>
      api.updateWebhook(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.all });
    },
  });
}

/**
 * Удалить вебхук
 */
export function useDeleteWebhook() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteWebhook(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: webhookKeys.all });
    },
  });
}

/**
 * Простой тест вебхука (без кастомного payload)
 */
export function useTestWebhook() {
  return useMutation({
    mutationFn: (id: string) => api.testWebhook(id),
  });
}

/**
 * Тест вебхука с кастомным mock payload
 */
export function useTestWebhookWithPayload() {
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Record<string, unknown> }) =>
      testWebhookWithPayload(id, payload),
  });
}
