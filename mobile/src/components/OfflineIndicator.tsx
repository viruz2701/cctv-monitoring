import React, { useEffect, useState, useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ActivityIndicator,
  StyleSheet,
} from 'react-native';
import { syncService, SyncState, SyncStatus } from '../services/syncService';

// ──────────────────────────────────────────────────
// Props
// ──────────────────────────────────────────────────

interface OfflineIndicatorProps {
  /** Показывать бейдж с количеством ожидающих мутаций */
  showQueueBadge?: boolean;
  /** Показывать при любом статусе (по умолчанию скрыт когда online + нет pending) */
  alwaysVisible?: boolean;
  /** onPress для pull-to-refresh / ручной синхронизации */
  onSyncPress?: () => void;
}

// ──────────────────────────────────────────────────
// Icons / labels
// ──────────────────────────────────────────────────

const STATUS_CONFIG: Record<
  SyncStatus,
  { icon: string; label: string; bg: string; textColor: string }
> = {
  online: {
    icon: '🟢',
    label: 'Online',
    bg: '#d1fae5',
    textColor: '#065f46',
  },
  syncing: {
    icon: '🔄',
    label: 'Синхронизация...',
    bg: '#fef3c7',
    textColor: '#92400e',
  },
  offline: {
    icon: '🔴',
    label: 'Офлайн',
    bg: '#fef2f2',
    textColor: '#991b1b',
  },
};

// ──────────────────────────────────────────────────
// Component
// ──────────────────────────────────────────────────

export default function OfflineIndicator({
  showQueueBadge = true,
  alwaysVisible = false,
  onSyncPress,
}: OfflineIndicatorProps) {
  const [status, setStatus] = useState<SyncStatus>('online');
  const [pendingCount, setPendingCount] = useState(0);
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null);

  useEffect(() => {
    const unsubscribe = syncService.subscribe((state: SyncState) => {
      setStatus(state.status);
      setLastSyncAt(state.lastSyncAt);
    });

    // Обновляем pending count
    const updatePendingCount = async () => {
      const count = await syncService.getPendingCount();
      setPendingCount(count);
    };

    updatePendingCount();
    const interval = setInterval(updatePendingCount, 5000);

    return () => {
      unsubscribe();
      clearInterval(interval);
    };
  }, []);

  const config = STATUS_CONFIG[status];

  // Показываем только если есть что показать
  if (!alwaysVisible && status === 'online' && pendingCount === 0) {
    return null;
  }

  const handleSyncPress = useCallback(() => {
    if (onSyncPress) {
      onSyncPress();
    } else {
      syncService.syncWhenOnline();
    }
  }, [onSyncPress]);

  return (
    <TouchableOpacity
      style={[styles.container, { backgroundColor: config.bg }]}
      onPress={handleSyncPress}
      activeOpacity={0.7}
      disabled={status === 'syncing'}
    >
      <View style={styles.left}>
        {status === 'syncing' ? (
          <ActivityIndicator size="small" color="#92400e" style={styles.spinner} />
        ) : (
          <Text style={styles.icon}>{config.icon}</Text>
        )}
        <Text style={[styles.label, { color: config.textColor }]}>
          {config.label}
        </Text>
      </View>

      <View style={styles.right}>
        {/* Badge с количеством ожидающих мутаций */}
        {showQueueBadge && pendingCount > 0 && (
          <View style={styles.badge}>
            <Text style={styles.badgeText}>{pendingCount}</Text>
          </View>
        )}

        {/* Время последней синхронизации */}
        {lastSyncAt && (
          <Text style={[styles.lastSync, { color: config.textColor }]}>
            {formatLastSync(lastSyncAt)}
          </Text>
        )}
      </View>
    </TouchableOpacity>
  );
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

function formatLastSync(timestamp: number): string {
  const diff = Date.now() - timestamp;
  const seconds = Math.floor(diff / 1000);

  if (seconds < 60) return 'только что';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}м назад`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}ч назад`;
  return `${Math.floor(seconds / 86400)}д назад`;
}

// ──────────────────────────────────────────────────
// Styles
// ──────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  left: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  right: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  icon: {
    fontSize: 14,
  },
  spinner: {
    width: 14,
    height: 14,
  },
  label: {
    fontSize: 13,
    fontWeight: '600',
  },
  badge: {
    backgroundColor: '#dc2626',
    borderRadius: 10,
    minWidth: 20,
    height: 20,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 6,
  },
  badgeText: {
    color: '#fff',
    fontSize: 11,
    fontWeight: '700',
  },
  lastSync: {
    fontSize: 11,
    opacity: 0.7,
  },
});
