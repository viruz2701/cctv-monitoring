// ═══════════════════════════════════════════════════════════════════════
// Reports & Audit Log API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request, requestBlob } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface Report {
  id: string;
  name: string;
  type: string;
  format: string;
  date_range?: string;
  file_url?: string;
  file_name?: string;
  size?: string;
  status: 'ready' | 'expired' | 'generating';
  generated_by?: string;
  generated_at: string;
  expires_at?: string;
}

export interface AuditLogEntry {
  id: string;
  timestamp: string;
  user_id?: string;
  action: string;
  entity_type?: string;
  entity_id?: string;
  old_value?: Record<string, any>;
  new_value?: Record<string, any>;
  ip_address?: string;
}

// ─── Notifications API ──────────────────────────────────────────────

export interface Notification {
  id: string;
  user_id: string;
  title: string;
  message: string;
  type: 'success' | 'warning' | 'error' | 'info';
  link?: string;
  read: boolean;
  created_at: string;
}

export const notificationsApi = {
  getNotifications(): Promise<Notification[]> {
    return request<Notification[]>('/notifications');
  },

  markNotificationRead(notificationId: string): Promise<void> {
    return request<void>(`/notifications/${notificationId}/read`, {
      method: 'POST',
    });
  },

  markAllNotificationsRead(): Promise<void> {
    return request<void>('/notifications/read-all', {
      method: 'POST',
    });
  },

  deleteNotification(notificationId: string): Promise<void> {
    return request<void>(`/notifications/${notificationId}`, {
      method: 'DELETE',
    });
  },

  deleteNotifications(ids: string[]): Promise<void> {
    return request<void>('/notifications/bulk-delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    });
  },
};

// ─── Reports API ────────────────────────────────────────────────────

export const reportsApi = {
  getReports(): Promise<Report[]> {
    return request<Report[]>('/reports');
  },

  generateReport(params: {
    type: string;
    format: string;
    date_range: string;
    filters?: Record<string, any>;
  }): Promise<Report> {
    return request<Report>('/reports/generate', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  getReportFile(reportId: string): Promise<Blob> {
    return requestBlob(`/reports/${reportId}/download`);
  },

  deleteReport(reportId: string): Promise<void> {
    return request<void>(`/reports/${reportId}`, {
      method: 'DELETE',
    });
  },
};

// ─── Audit Log API ──────────────────────────────────────────────────

export const auditLogApi = {
  getAuditLog(params?: {
    user_id?: string;
    action?: string;
    entity_type?: string;
    entity_id?: string;
    time_from?: string;
    time_to?: string;
    limit?: number;
  }): Promise<AuditLogEntry[]> {
    const query = new URLSearchParams();
    if (params?.user_id) query.append('user_id', params.user_id);
    if (params?.action) query.append('action', params.action);
    if (params?.entity_type) query.append('entity_type', params.entity_type);
    if (params?.entity_id) query.append('entity_id', params.entity_id);
    if (params?.time_from) query.append('time_from', params.time_from);
    if (params?.time_to) query.append('time_to', params.time_to);
    if (params?.limit) query.append('limit', String(params.limit));
    return request<AuditLogEntry[]>(`/audit-log?${query.toString()}`);
  },
};
