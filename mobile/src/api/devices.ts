import { apiClient } from './client';

// DeviceMapData — лёгкая структура для отображения на карте
// Соответствует MobileDeviceMapData на backend (OWASP ASVS V8 — Data Protection)
export interface DeviceMapData {
  device_id: string;
  name: string;
  latitude: number;
  longitude: number;
  status: 'ONLINE' | 'OFFLINE' | 'WARNING';
  device_type: string;
  site_name?: string;
  health: 'healthy' | 'faulty' | 'degraded';
}

interface MobileDevicesResponse {
  devices: DeviceMapData[];
  total: number;
}

export const devicesApi = {
  /**
   * Получить список устройств с координатами для карты.
   * Поддерживает offline-кеширование на стороне клиента.
   * GET /api/v1/mobile/devices
   */
  getDevicesForMap: async (params?: {
    status?: string;
    device_type?: string;
    site_id?: string;
    search?: string;
  }): Promise<MobileDevicesResponse> => {
    const response = await apiClient.get<MobileDevicesResponse>('/mobile/devices', { params });
    return response.data;
  },
};
