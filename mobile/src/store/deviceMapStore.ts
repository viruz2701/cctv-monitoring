import { create } from 'zustand';
import { storage } from '../utils/storage';
import { DeviceMapData } from '../api/devices';
import { CacheMetadata } from '../services/tileCache';

// ──────────────────────────────────────────────────
// Типы
// ──────────────────────────────────────────────────

export interface PreloadSiteStatus {
  /** ID сайта */
  siteId: string;
  /** Статус предзагрузки */
  status: 'pending' | 'preloading' | 'complete' | 'failed';
  /** Прогресс (0–1) */
  progress: number;
  /** Количество загруженных тайлов */
  tileCount: number;
  /** Размер кэша для этого сайта */
  sizeBytes: number;
}

interface DeviceMapState {
  /** Кешированные координаты устройств (offline-first) */
  cachedDevices: DeviceMapData[];
  /** Время последнего обновления кеша */
  lastSyncAt: number | null;
  /** Идёт загрузка */
  isLoading: boolean;
  /** Ошибка загрузки */
  error: string | null;

  // ── Tile cache metadata ──────────────────────────

  /** Список предзагруженных сайтов (из cache_metadata) */
  preloadedSites: CacheMetadata[];
  /** Статус предзагрузки по сайтам */
  preloadStatuses: Record<string, PreloadSiteStatus>;
  /** Общее количество кэшированных тайлов (из SQLite) */
  totalTileCount: number;
  /** Общий размер кэша в байтах (из SQLite) */
  totalCacheSizeBytes: number;
  /** Последнее обновление статистики кэша */
  tileStatsUpdatedAt: number | null;

  /** Установить кеш устройств и сохранить в AsyncStorage */
  setCachedDevices: (devices: DeviceMapData[]) => Promise<void>;
  /** Загрузить кеш из AsyncStorage */
  loadCachedDevices: () => Promise<void>;
  /** Обновить статус устройства в кеше (после синхронизации) */
  updateDeviceStatus: (deviceId: string, status: DeviceMapData['status']) => void;
  /** Сбросить кеш */
  clearCache: () => Promise<void>;

  // ── Tile cache metadata methods ──────────────────

  /** Установить список предзагруженных сайтов */
  setPreloadedSites: (sites: CacheMetadata[]) => void;
  /** Обновить статус предзагрузки для сайта */
  updatePreloadStatus: (siteId: string, update: Partial<PreloadSiteStatus>) => void;
  /** Удалить сайт из предзагруженных */
  removePreloadedSite: (siteId: string) => void;
  /** Обновить общую статистику кэша */
  setTileCacheStats: (tileCount: number, cacheSizeBytes: number) => void;
  /** Получить статус предзагрузки для сайта */
  getPreloadStatus: (siteId: string) => PreloadSiteStatus | undefined;
}

const DEVICE_CACHE_KEY = 'deviceMapCache';

export const useDeviceMapStore = create<DeviceMapState>((set, get) => ({
  cachedDevices: [],
  lastSyncAt: null,
  isLoading: false,
  error: null,

  // ── Tile cache initial state ─────────────────────

  preloadedSites: [],
  preloadStatuses: {},
  totalTileCount: 0,
  totalCacheSizeBytes: 0,
  tileStatsUpdatedAt: null,

  // ── Device cache methods ─────────────────────────

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

  // ── Tile cache metadata methods ──────────────────

  setPreloadedSites: (sites: CacheMetadata[]) => {
    set({ preloadedSites: sites });
  },

  updatePreloadStatus: (siteId: string, update: Partial<PreloadSiteStatus>) => {
    const current = get().preloadStatuses[siteId];
    const existing: PreloadSiteStatus = current || {
      siteId,
      status: 'pending',
      progress: 0,
      tileCount: 0,
      sizeBytes: 0,
    };

    set({
      preloadStatuses: {
        ...get().preloadStatuses,
        [siteId]: { ...existing, ...update },
      },
    });
  },

  removePreloadedSite: (siteId: string) => {
    const { [siteId]: _removed, ...rest } = get().preloadStatuses;
    set({
      preloadedSites: get().preloadedSites.filter((s) => s.siteId !== siteId),
      preloadStatuses: rest,
    });
  },

  setTileCacheStats: (tileCount: number, cacheSizeBytes: number) => {
    set({
      totalTileCount: tileCount,
      totalCacheSizeBytes: cacheSizeBytes,
      tileStatsUpdatedAt: Date.now(),
    });
  },

  getPreloadStatus: (siteId: string) => {
    return get().preloadStatuses[siteId];
  },
}));
