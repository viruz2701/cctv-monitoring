// ═══════════════════════════════════════════════════════════════════════
// Alarms API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface Alarm {
  device_id: string;
  priority: number;
  method: number;
  description: string;
  timestamp: string;
  image_path?: string;
}

// ─── API Methods ────────────────────────────────────────────────────

export const alarmsApi = {
  getAlarms(deviceId?: string): Promise<Alarm[]> {
    const query = deviceId ? `?device_id=${deviceId}` : '';
    return request<Alarm[]>(`/alarms${query}`);
  },

  acknowledgeAlarm(alarmId: string): Promise<void> {
    return request<void>(`/alarms/${alarmId}/acknowledge`, {
      method: 'POST',
    });
  },

  resolveAlarm(alarmId: string): Promise<void> {
    return request<void>(`/alarms/${alarmId}/resolve`, {
      method: 'POST',
    });
  },

  deleteAlarm(alarmId: string): Promise<void> {
    return request<void>(`/alarms/${alarmId}`, {
      method: 'DELETE',
    });
  },

  // External Alarm (for integrations)
  sendExternalAlarm(alarm: {
    device_id: string;
    event_type: string;
    priority: number;
    method: number;
    description: string;
    timestamp?: string;
  }): Promise<void> {
    return request<void>('/external/alarm', {
      method: 'POST',
      body: JSON.stringify(alarm),
    });
  },
};
