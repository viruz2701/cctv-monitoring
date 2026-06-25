import React, { useMemo } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
  ScrollView,
  RefreshControl,
} from 'react-native';
import MapView, { Marker, Callout, PROVIDER_GOOGLE } from 'react-native-maps';
import { Ionicons } from '@expo/vector-icons';
import { useOfflineMap } from '../hooks/useOfflineMap';
import OfflineIndicator from '../components/OfflineIndicator';
import { DeviceMapData } from '../api/devices';

// Цвета статусов (соответствуют IEC 62443 SR 3.1 color coding)
const STATUS_COLORS: Record<string, string> = {
  ONLINE: '#059669',   // зелёный
  OFFLINE: '#dc2626',  // красный
  WARNING: '#d97706',  // жёлтый/оранжевый
};

const STATUS_LABELS: Record<string, string> = {
  ONLINE: 'В сети',
  OFFLINE: 'Недоступно',
  WARNING: 'Предупреждение',
};

// Иконки устройств
const DEVICE_ICONS: Record<string, keyof typeof Ionicons.glyphMap> = {
  camera: 'videocam',
  nvr: 'server',
  dvr: 'server',
  switch: 'git-merge',
};

export default function MapScreen() {
  const {
    filteredDevices,
    currentLocation,
    isMapLoading,
    mapError,
    lastSyncAt,
    statusFilter,
    setStatusFilter,
    deviceTypeFilter,
    setDeviceTypeFilter,
    refreshDevices,
    statusCounts,
    isOnline,
  } = useOfflineMap();

  // Начальный регион карты
  const initialRegion = useMemo(() => {
    if (currentLocation) {
      return {
        latitude: currentLocation.latitude,
        longitude: currentLocation.longitude,
        latitudeDelta: 0.05,
        longitudeDelta: 0.05,
      };
    }
    // Центр по устройствам если нет геолокации
    if (filteredDevices.length > 0) {
      const avgLat = filteredDevices.reduce((s, d) => s + d.latitude, 0) / filteredDevices.length;
      const avgLng = filteredDevices.reduce((s, d) => s + d.longitude, 0) / filteredDevices.length;
      return {
        latitude: avgLat,
        longitude: avgLng,
        latitudeDelta: 0.1,
        longitudeDelta: 0.1,
      };
    }
    return {
      latitude: 53.9,  // центр Беларуси
      longitude: 27.57,
      latitudeDelta: 3,
      longitudeDelta: 3,
    };
  }, [currentLocation, filteredDevices]);

  if (isMapLoading) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="large" color="#2563eb" />
        <Text style={styles.loadingText}>Загрузка карты...</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <OfflineIndicator />

      {/* Фильтры статусов */}
      <View style={styles.filterBar}>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filterRow}>
          <FilterChip
            label={`Все (${statusCounts.ONLINE + statusCounts.OFFLINE + statusCounts.WARNING})`}
            active={statusFilter === 'all'}
            onPress={() => setStatusFilter('all')}
          />
          <FilterChip
            label={`🟢 В сети (${statusCounts.ONLINE})`}
            active={statusFilter === 'ONLINE'}
            onPress={() => setStatusFilter('ONLINE')}
            color="#059669"
          />
          <FilterChip
            label={`🔴 Недоступно (${statusCounts.OFFLINE})`}
            active={statusFilter === 'OFFLINE'}
            onPress={() => setStatusFilter('OFFLINE')}
            color="#dc2626"
          />
          <FilterChip
            label={`🟡 Предупреждения (${statusCounts.WARNING})`}
            active={statusFilter === 'WARNING'}
            onPress={() => setStatusFilter('WARNING')}
            color="#d97706"
          />
        </ScrollView>
      </View>

      {/* Фильтры типов устройств */}
      <View style={styles.typeFilterBar}>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.filterRow}>
          <TypeChip
            label="Все"
            active={deviceTypeFilter === 'all'}
            onPress={() => setDeviceTypeFilter('all')}
          />
          <TypeChip
            label="Камеры"
            active={deviceTypeFilter === 'camera'}
            onPress={() => setDeviceTypeFilter('camera')}
          />
          <TypeChip
            label="NVR"
            active={deviceTypeFilter === 'nvr'}
            onPress={() => setDeviceTypeFilter('nvr')}
          />
          <TypeChip
            label="Коммутаторы"
            active={deviceTypeFilter === 'switch'}
            onPress={() => setDeviceTypeFilter('switch')}
          />
        </ScrollView>
      </View>

      {/* Карта */}
      <MapView
        style={styles.map}
        provider={PROVIDER_GOOGLE}
        initialRegion={initialRegion}
        showsUserLocation
        showsMyLocationButton
        showsCompass
        rotateEnabled
        zoomEnabled
        scrollEnabled
      >
        {filteredDevices.map((device) => (
          <Marker
            key={device.device_id}
            coordinate={{
              latitude: device.latitude,
              longitude: device.longitude,
            }}
            pinColor={STATUS_COLORS[device.status] || '#64748b'}
            title={device.name}
            description={`${device.site_name || ''} — ${STATUS_LABELS[device.status] || device.status}`}
          >
            <Callout>
              <View style={styles.callout}>
                <Text style={styles.calloutTitle}>{device.name}</Text>
                {device.site_name ? (
                  <Text style={styles.calloutSite}>{device.site_name}</Text>
                ) : null}
                <View style={styles.calloutStatusRow}>
                  <View
                    style={[
                      styles.statusDot,
                      { backgroundColor: STATUS_COLORS[device.status] || '#64748b' },
                    ]}
                  />
                  <Text style={styles.calloutStatus}>
                    {STATUS_LABELS[device.status] || device.status}
                  </Text>
                </View>
                <Text style={styles.calloutType}>
                  {device.device_type.toUpperCase()}
                  {device.health === 'faulty' ? ' ⚠️' : ''}
                </Text>
              </View>
            </Callout>
          </Marker>
        ))}
      </MapView>

      {/* Информационная панель снизу */}
      <View style={styles.bottomPanel}>
        <View style={styles.panelRow}>
          <Text style={styles.panelText}>
            {filteredDevices.length} устройств
            {!isOnline ? ' (офлайн)' : ''}
          </Text>
          {lastSyncAt && (
            <Text style={styles.syncText}>
              Обновлено: {lastSyncAt.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}
            </Text>
          )}
          <TouchableOpacity style={styles.refreshButton} onPress={refreshDevices}>
            <Ionicons name="refresh" size={20} color="#2563eb" />
          </TouchableOpacity>
        </View>
        {mapError && (
          <Text style={styles.errorText}>{mapError}</Text>
        )}
      </View>
    </View>
  );
}

// ── Filter Chip ──────────────────────────────────────────────────────────

function FilterChip({
  label,
  active,
  onPress,
  color,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
  color?: string;
}) {
  return (
    <TouchableOpacity
      style={[styles.chip, active && styles.chipActive, active && color ? { backgroundColor: color } : null]}
      onPress={onPress}
    >
      <Text style={[styles.chipText, active && styles.chipTextActive]}>
        {label}
      </Text>
    </TouchableOpacity>
  );
}

// ── Type Chip ────────────────────────────────────────────────────────────

function TypeChip({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <TouchableOpacity
      style={[styles.typeChip, active && styles.typeChipActive]}
      onPress={onPress}
    >
      <Text style={[styles.typeChipText, active && styles.typeChipTextActive]}>
        {label}
      </Text>
    </TouchableOpacity>
  );
}

// ── Styles ───────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f1f5f9',
  },
  loadingText: {
    marginTop: 12,
    fontSize: 16,
    color: '#64748b',
  },
  filterBar: {
    backgroundColor: '#fff',
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
    zIndex: 10,
  },
  typeFilterBar: {
    backgroundColor: '#f8fafc',
    paddingVertical: 6,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
    zIndex: 9,
  },
  filterRow: {
    paddingHorizontal: 12,
    gap: 8,
    flexDirection: 'row',
  },
  chip: {
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 20,
    backgroundColor: '#f1f5f9',
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  chipActive: {
    backgroundColor: '#2563eb',
    borderColor: '#2563eb',
  },
  chipText: {
    fontSize: 13,
    color: '#475569',
    fontWeight: '500',
  },
  chipTextActive: {
    color: '#fff',
  },
  typeChip: {
    paddingHorizontal: 12,
    paddingVertical: 4,
    borderRadius: 16,
    backgroundColor: '#f1f5f9',
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  typeChipActive: {
    backgroundColor: '#1e293b',
    borderColor: '#1e293b',
  },
  typeChipText: {
    fontSize: 12,
    color: '#475569',
  },
  typeChipTextActive: {
    color: '#fff',
  },
  map: {
    flex: 1,
  },
  callout: {
    padding: 8,
    minWidth: 160,
  },
  calloutTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#1e293b',
    marginBottom: 2,
  },
  calloutSite: {
    fontSize: 12,
    color: '#64748b',
    marginBottom: 4,
  },
  calloutStatusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    marginBottom: 2,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  calloutStatus: {
    fontSize: 12,
    color: '#475569',
    fontWeight: '500',
  },
  calloutType: {
    fontSize: 11,
    color: '#94a3b8',
    marginTop: 2,
  },
  bottomPanel: {
    backgroundColor: '#fff',
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderTopWidth: 1,
    borderTopColor: '#e2e8f0',
  },
  panelRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  panelText: {
    fontSize: 13,
    color: '#475569',
    fontWeight: '500',
  },
  syncText: {
    fontSize: 11,
    color: '#94a3b8',
  },
  refreshButton: {
    padding: 6,
    borderRadius: 20,
    backgroundColor: '#eff6ff',
  },
  errorText: {
    fontSize: 11,
    color: '#dc2626',
    marginTop: 4,
  },
});
