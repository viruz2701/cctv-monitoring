import { useState, useEffect, useCallback } from 'react';
import NetInfo from '@react-native-community/netinfo';
import { devicesApi, DeviceMapData } from '../api/devices';
import { useDeviceMapStore } from '../store/deviceMapStore';
import { useLocation } from './useLocation';

type DeviceStatusFilter = 'all' | 'ONLINE' | 'OFFLINE' | 'WARNING';
type DeviceTypeFilter = 'all' | 'camera' | 'nvr' | 'dvr' | 'switch';

interface UseOfflineMapReturn {
  /** Устройства с координатами (из кеша или онлайн) */
  devices: DeviceMapData[];
  /** Текущая позиция пользователя */
  currentLocation: { latitude: number; longitude: number } | null;
  /** Загрузка */
  isMapLoading: boolean;
  /** Ошибка */
  mapError: string | null;
  /** Время последней синхронизации */
  lastSyncAt: Date | null;
  /** Фильтр по статусу */
  statusFilter: DeviceStatusFilter;
  /** Установить фильтр статуса */
  setStatusFilter: (filter: DeviceStatusFilter) => void;
  /** Фильтр по типу устройства */
  deviceTypeFilter: DeviceTypeFilter;
  /** Установить фильтр типа устройства */
  setDeviceTypeFilter: (filter: DeviceTypeFilter) => void;
  /** Отфильтрованные устройства */
  filteredDevices: DeviceMapData[];
  /** Обновить данные с сервера */
  refreshDevices: () => Promise<void>;
  /** Количество устройств по статусам */
  statusCounts: {
    ONLINE: number;
    OFFLINE: number;
    WARNING: number;
  };
  /** Есть ли интернет */
  isOnline: boolean;
}

/**
 * useOfflineMap — хук для загрузки, кеширования и фильтрации устройств на карте.
 *
 * Offline-first стратегия (UX-02):
 * 1. При загрузке — показываем кешированные данные из AsyncStorage
 * 2. Если есть интернет — параллельно загружаем свежие данные с сервера
 * 3. Обновляем кеш при получении свежих данных
 * 4. При офлайн — работаем только с кешем
 */
export function useOfflineMap(): UseOfflineMapReturn {
  const {
    cachedDevices,
    lastSyncAt,
    isLoading: storeLoading,
    error: storeError,
    setCachedDevices,
    loadCachedDevices,
  } = useDeviceMapStore();

  const { latitude, longitude, loading: locationLoading } = useLocation();
  const [isOnline, setIsOnline] = useState(true);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<DeviceStatusFilter>('all');
  const [deviceTypeFilter, setDeviceTypeFilter] = useState<DeviceTypeFilter>('all');

  // Мониторинг сетевого статуса
  useEffect(() => {
    const unsubscribe = NetInfo.addEventListener((state) => {
      setIsOnline(state.isConnected ?? true);
    });
    return () => unsubscribe();
  }, []);

  // Загрузка кеша при монтировании
  useEffect(() => {
    const init = async () => {
      setIsLoading(true);
      await loadCachedDevices();
      setIsLoading(false);
    };
    init();
  }, [loadCachedDevices]);

  // Автоматическая синхронизация при появлении интернета
  useEffect(() => {
    if (isOnline) {
      refreshDevices();
    }
  }, [isOnline]);

  const refreshDevices = useCallback(async () => {
    if (!isOnline) return;

    try {
      setError(null);
      const response = await devicesApi.getDevicesForMap();
      await setCachedDevices(response.devices);
    } catch (err) {
      console.error('Failed to refresh device map data:', err);
      // Graceful degradation: при ошибке сети продолжаем показывать кеш
      setError('Не удалось обновить данные. Используется кеш.');
    }
  }, [isOnline, setCachedDevices]);

  // Фильтрация устройств
  const filteredDevices = cachedDevices.filter((device) => {
    if (statusFilter !== 'all' && device.status !== statusFilter) return false;
    if (deviceTypeFilter !== 'all' && device.device_type !== deviceTypeFilter) return false;
    return true;
  });

  // Количество по статусам
  const statusCounts = {
    ONLINE: cachedDevices.filter((d) => d.status === 'ONLINE').length,
    OFFLINE: cachedDevices.filter((d) => d.status === 'OFFLINE').length,
    WARNING: cachedDevices.filter((d) => d.status === 'WARNING').length,
  };

  return {
    devices: cachedDevices,
    currentLocation: locationLoading ? null : { latitude, longitude },
    isMapLoading: isLoading || storeLoading,
    mapError: error || storeError,
    lastSyncAt: lastSyncAt ? new Date(lastSyncAt) : null,
    statusFilter,
    setStatusFilter,
    deviceTypeFilter,
    setDeviceTypeFilter,
    filteredDevices,
    refreshDevices,
    statusCounts,
    isOnline,
  };
}
