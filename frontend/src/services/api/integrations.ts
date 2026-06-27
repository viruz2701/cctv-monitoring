// ═══════════════════════════════════════════════════════════════════════
// Integrations API (Webhooks, P2P, Atlas CMMS, Camera Models)
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request, requestBlob } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface WebhookEndpoint {
  id: string;
  name: string;
  url: string;
  events: string[];
  secret?: string;
  active: boolean;
  retry_count: number;
  timeout_seconds: number;
  retry_interval_seconds: number;
  retry_backoff: boolean;
  max_retry_duration_seconds: number;
  last_sent_at?: string;
  last_status?: string;
  created_at: string;
}

export interface CameraSpec {
  id: number;
  brand: string;
  model: string;
  type?: string;
  resolution?: string;
  max_fps?: number;
  lens_mm?: string;
  infrared?: boolean;
  poe?: boolean;
  poe_class?: string;
  power_watts?: number;
  storage_days_estimate?: number;
  bandwidth_mbps?: number;
  protocols?: string[];
  onvif_profile?: string;
  audio_support?: boolean;
  outdoor_rating?: string;
  weight_grams?: number;
  dimensions?: string;
  notes?: string;
  created_at: string;
}

export interface CameraBrand {
  brand: string;
  count: number;
}

export interface CameraModelSummary {
  id: number;
  brand: string;
  model: string;
  type?: string;
  resolution?: string;
}

// ─── Webhooks API ───────────────────────────────────────────────────

export const webhooksApi = {
  getWebhooks(): Promise<WebhookEndpoint[]> {
    return request<WebhookEndpoint[]>('/integrations/extended/webhooks');
  },

  createWebhook(data: { name: string; url: string; events: string[]; secret?: string; active?: boolean }): Promise<WebhookEndpoint> {
    return request<WebhookEndpoint>('/integrations/extended/webhooks', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateWebhook(id: string, data: Partial<WebhookEndpoint>): Promise<WebhookEndpoint> {
    return request<WebhookEndpoint>(`/integrations/extended/webhooks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteWebhook(id: string): Promise<void> {
    return request<void>(`/integrations/extended/webhooks/${id}`, {
      method: 'DELETE',
    });
  },

  testWebhook(id: string): Promise<{ status: string; message: string }> {
    return request<{ status: string; message: string }>(`/integrations/extended/webhooks/${id}/test`, {
      method: 'POST',
    });
  },
};

// ─── P2P API ────────────────────────────────────────────────────────

export const p2pApi = {
  listP2PDevices(): Promise<any[]> {
    return request<any[]>('/p2p/devices');
  },

  registerP2PDevice(data: {
    brand: string;
    serial: string;
    username?: string;
    password?: string;
    security_code?: string;
    ip_address?: string;
  }): Promise<any> {
    return request<any>('/p2p/devices', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  getP2PDeviceStatus(deviceId: string): Promise<{ device_id: string; status: string; rtsp_url: string }> {
    return request<{ device_id: string; status: string; rtsp_url: string }>(`/p2p/status/${deviceId}`);
  },

  sendP2PCommand(deviceId: string, command: { command: string; speed?: number }): Promise<void> {
    return request<void>(`/p2p/command/${deviceId}`, {
      method: 'POST',
      body: JSON.stringify(command),
    });
  },

  getP2PSnapshot(deviceId: string): Promise<Blob> {
    return requestBlob(`/p2p/snapshot/${deviceId}`);
  },
};

// ─── Atlas CMMS API ─────────────────────────────────────────────────

export const atlasApi = {
  healthCheck(): Promise<{ status: string; error?: string; message?: string }> {
    return request<{ status: string; error?: string; message?: string }>('/atlas/health');
  },

  fallbackStatus(): Promise<{ queue_size: number; message?: string }> {
    return request<{ queue_size: number; message?: string }>('/atlas/fallback/status');
  },

  retryFallback(): Promise<{ success: number; failed: number; message?: string }> {
    return request<{ success: number; failed: number; message?: string }>('/atlas/fallback/retry', {
      method: 'POST',
    });
  },

  syncAsset(deviceId: string): Promise<{ status: string; error?: string; message?: string }> {
    return request<{ status: string; error?: string; message?: string }>(`/atlas/sync-asset/${deviceId}`, {
      method: 'POST',
    });
  },
};

// ─── Camera Models API ──────────────────────────────────────────────

export const cameraModelsApi = {
  listBrands(): Promise<{ brands: CameraBrand[] }> {
    return request<{ brands: CameraBrand[] }>('/camera-models/brands');
  },

  listModels(brand: string): Promise<{ brand: string; models: CameraModelSummary[] }> {
    return request<{ brand: string; models: CameraModelSummary[] }>(
      `/camera-models/models?brand=${encodeURIComponent(brand)}`,
    );
  },

  searchModels(query: string): Promise<{ query: string; models: CameraModelSummary[] }> {
    return request<{ query: string; models: CameraModelSummary[] }>(
      `/camera-models/search?q=${encodeURIComponent(query)}`,
    );
  },

  getSpecs(brand: string, model: string): Promise<CameraSpec> {
    return request<CameraSpec>(`/camera-models/${encodeURIComponent(brand)}/${encodeURIComponent(model)}`);
  },

  importSpecs(data: CameraSpec[]): Promise<{ message: string; inserted: number; updated: number; skipped: number; errors: number }> {
    return request('/camera-models/import', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  seedSpecs(): Promise<{ message: string }> {
    return request('/camera-models/seed', {
      method: 'POST',
    });
  },
};
