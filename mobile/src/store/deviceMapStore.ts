import { create } from 'zustand';
import { storage } from '../utils/storage';
import { DeviceMapData } from '../api/devices';

interface DeviceMapState {
  /** Кешированные координаты устройств (offline-first) */
  cachedDevices: DeviceMapData[];
  /** Время последнего обновления кеша */
  lastSyncAt: number | null;
  /** Идёт загрузка */
  isLoading: boolean;
  /** Ошибка загрузки */
  error: string | null;

  /** Установить кеш устройств и сохранить в AsyncStorage */
  setCachedDevices: (devices: DeviceMapData[]) => Promise<void>;
  /** Загрузить кеш из AsyncStorage */
  loadCachedDevices: () => Promise<void>;
  /** Обновить статус устройства в кеше (после синхронизации) */
  updateDeviceStatus: (deviceId: string, status: DeviceMapData['status']) => void;
  /** Сбросить кеш */
  clearCache: () => Promise<void>;
}

const DEVICE_CACHE_KEY = 'deviceMapCache';

export const useDeviceMapStore = create<DeviceMapState>((set, get) => ({
  cachedDevices: [],
  lastSyncAt: null,
  isLoading: false,
  error: null,

  setCachedDevices: async (devices: DeviceMapData[]) => {
    const data = {
      devices,
      lastSyncAt: Date.now(),
    };
    try {
      await storage.setItem(DEVICE_CACHE_KEY, JSON.stringify(data));
    } catch (err) {
      console.error('Failed to cache device map data:', err);
    }
    set({
      cachedDevices: devices,
      lastSyncAt: data.lastSyncAt,
      isLoading: false,
      error: null,
    });
  },

  loadCachedDevices: async () => {
    set({ isLoading: true });
    try {
      const stored = await storage.getItem(DEVICE_CACHE_KEY);
      if (stored) {
        const data = JSON.parse(stored) as {
          devices: DeviceMapData[];
          lastSyncAt: number;
        };
        set({
          cachedDevices: data.devices,
          lastSyncAt: data.lastSyncAt,
          isLoading: false,
          error: null,
        });
      } else {
        set({ isLoading: false });
      }
    } catch (err) {
      console.error('Failed to load cached device map:', err);
      set({
        isLoading: false,
        error: 'Failed to load cached device map',
      });
    }
  },

  updateDeviceStatus: (deviceId: string, status: DeviceMapData['status']) => {
    const devices = get().cachedDevices.map((d) =>
      d.device_id === deviceId ? { ...d, status } : d,
    );
    set({ cachedDevices: devices });
    // Асинхронно сохраняем обновление в кеш
    get().setCachedDevices(devices);
  },

  clearCache: async () => {
    try {
      await storage.removeItem(DEVICE_CACHE_KEY);
    } catch (err) {
      console.error('Failed to clear device map cache:', err);
    }
    set({
      cachedDevices: [],
      lastSyncAt: null,
      error: null,
    });
  },
}));
