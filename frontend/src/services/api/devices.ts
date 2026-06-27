// ═══════════════════════════════════════════════════════════════════════
// Devices API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface Device {
  device_id: string;
  owner_id?: string | null;
  name?: string;
  location?: string;
  vendor_type?: string;
  status: string;
  last_seen: string;
  registered_at: string;
  user_agent?: string;
  // P2P fields
  p2p_brand?: string;
  p2p_serial?: string;
  cloud_status?: string;
}

export interface DeviceDetectionResult {
  detected: boolean;
  model?: string;
  vendor?: 'hikvision' | 'dahua' | 'onvif' | 'rtsp' | 'unknown';
  firmware?: string;
  mac_address?: string;
  protocols: string[];
  onvif_profile_s: boolean;
  onvif_profile_t: boolean;
  rtsp_supported: boolean;
  http_api_supported: boolean;
  snapshot_url?: string;
  stream_urls?: string[];
  error?: string;
}

export interface CapacityParams {
  resolution: string;
  fps: number;
  codec: 'H.264' | 'H.265' | 'MJPEG';
  retention_days: number;
  cameras_count: number;
  poe_wattage?: number;
}

export interface CapacityResult {
  bandwidth_mbps: number;
  storage_gb: number;
  poe_budget_watts: number;
  recommended_nvr: string;
  warnings: string[];
}

export interface DashboardStats {
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  warning_devices: number;
  open_tickets: number;
  critical_tickets: number;
  resolution_rate: number;
  avg_response_time_hours: number;
}

// ─── API Methods ────────────────────────────────────────────────────

export const devicesApi = {
  getDevices(): Promise<Device[]> {
    return request<Device[]>('/devices');
  },

  getDevice(deviceId: string): Promise<Device> {
    return request<Device>(`/devices/${deviceId}`);
  },

  getDeviceStatus(deviceId: string): Promise<{ device_id: string; status: string; last_seen: string }> {
    return request<{ device_id: string; status: string; last_seen: string }>(`/devices/${deviceId}/status`);
  },

  createDevice(device: Partial<Device>): Promise<Device> {
    return request<Device>('/devices', {
      method: 'POST',
      body: JSON.stringify(device),
    });
  },

  updateDevice(deviceId: string, updates: Partial<Device>): Promise<Device> {
    return request<Device>(`/devices/${deviceId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  },

  deleteDevice(deviceId: string): Promise<void> {
    return request<void>(`/devices/${deviceId}`, {
      method: 'DELETE',
    });
  },

  getDeviceImages(deviceId: string): Promise<string[]> {
    return request<string[]>(`/images/device/${deviceId}`);
  },

  detectDevice(
    ipOrDomain: string,
    options?: { username?: string; password?: string; port?: number },
  ): Promise<DeviceDetectionResult> {
    const params = new URLSearchParams({ target: ipOrDomain });
    if (options?.username) params.append('username', options.username);
    if (options?.password) params.append('password', options.password);
    if (options?.port) params.append('port', String(options.port));
    return request<DeviceDetectionResult>(`/devices/detect?${params.toString()}`);
  },

  calculateDeviceCapacity(params: CapacityParams): Promise<CapacityResult> {
    return request<CapacityResult>('/devices/calculate-capacity', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },

  getDashboardStats(): Promise<DashboardStats> {
    return request<DashboardStats>('/dashboard/stats');
  },
};
