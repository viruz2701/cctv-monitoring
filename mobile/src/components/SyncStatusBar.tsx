import React, { useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ActivityIndicator,
  StyleSheet,
} from 'react-native';
import {
  useBackgroundSync,
  BackgroundSyncState,
  BackgroundSyncActions,
} from '../hooks/useBackgroundSync';

// ──────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────

type SyncStatus = BackgroundSyncState['status'];

interface SyncStatusBarProps {
  /** Показывать кнопку toggle background sync */
  showToggle?: boolean;
  /** Расширенный режим — показывает все детали */
  expanded?: boolean;
  /** onSync callback переопределяет поведение кнопки */
  onSyncPress?: () => void;
}

// ──────────────────────────────────────────────────
// Status config
// ──────────────────────────────────────────────────

const STATUS_CONFIG: Record<
  SyncStatus,
  { icon: string; label: string; bg: string; textColor: string }
> = {
  online: {
    icon: '🟢',
    label: 'Синхронизировано',
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

export default function SyncStatusBar({
  showToggle = false,
  expanded = false,
  onSyncPress,
}: SyncStatusBarProps) {
  const {
    status,
    pendingCount,
    lastSyncAt,
    lastError,
    isRegistered,
    manualSync,
    toggleBackgroundSync,
  } = useBackgroundSync();

  const config = STATUS_CONFIG[status];

  const handleSync = useCallback(async () => {
    if (onSyncPress) {
      onSyncPress();
      return;
    }
    await manualSync();
  }, [onSyncPress, manualSync]);

  // ── Compact mode (одна строка) ────────────────

  if (!expanded) {
    return (
      <TouchableOpacity
        style={[styles.bar, { backgroundColor: config.bg }]}
        onPress={handleSync}
        activeOpacity={0.7}
        disabled={status === 'syncing'}
      >
        <View style={styles.barLeft}>
          {status === 'syncing' ? (
            <ActivityIndicator size="small" color="#92400e" />
          ) : (
            <Text style={styles.icon}>{config.icon}</Text>
          )}
          <Text style={[styles.label, { color: config.textColor }]}>
            {config.label}
          </Text>
        </View>

        <View style={styles.barRight}>
          {/* Queue badge */}
          {pendingCount > 0 && (
            <View style={styles.badge}>
              <Text style={styles.badgeText}>{pendingCount}</Text>
            </View>
          )}

          {/* Last sync */}
          {lastSyncAt && (
            <Text style={[styles.timestamp, { color: config.textColor }]}>
              {formatLastSync(lastSyncAt)}
            </Text>
          )}

          {/* Sync button */}
          {status !== 'syncing' && (
            <Text style={[styles.syncIcon, { color: config.textColor }]}>
              ↻
            </Text>
          )}
        </View>
      </TouchableOpacity>
    );
  }

  // ── Expanded mode ─────────────────────────────

  return (
    <View
      style={[
        styles.expandedContainer,
        { backgroundColor: config.bg },
      ]}
    >
      {/* Status row */}
      <View style={styles.expandedRow}>
        <View style={styles.expandedLeft}>
          {status === 'syncing' ? (
            <ActivityIndicator size="small" color="#92400e" />
          ) : (
            <Text style={styles.iconLarge}>{config.icon}</Text>
          )}
          <View>
            <Text style={[styles.labelLarge, { color: config.textColor }]}>
              {config.label}
            </Text>
            {lastSyncAt && (
              <Text style={[styles.subtext, { color: config.textColor }]}>
                Последняя синхронизация: {formatLastSync(lastSyncAt)}
              </Text>
            )}
          </View>
        </View>
      </View>

      {/* Queue info */}
      <View style={styles.expandedRow}>
        <Text style={[styles.queueLabel, { color: config.textColor }]}>
          Ожидающих мутаций:
        </Text>
        <View style={styles.queueValueRow}>
          <View
            style={[
              styles.queueBadge,
              {
                backgroundColor:
                  pendingCount > 0 ? '#dc2626' : '#16a34a',
              },
            ]}
          >
            <Text style={styles.badgeText}>{pendingCount}</Text>
          </View>
        </View>
      </View>

      {/* Error */}
      {lastError && (
        <View style={styles.errorRow}>
          <Text style={styles.errorText} numberOfLines={2}>
            ⚠️ {lastError}
          </Text>
        </View>
      )}

      {/* Background sync toggle */}
      {showToggle && (
        <View style={styles.expandedRow}>
          <Text style={[styles.queueLabel, { color: config.textColor }]}>
            Фоновая синхронизация:
          </Text>
          <TouchableOpacity
            style={[
              styles.toggleBtn,
              {
                backgroundColor: isRegistered ? '#16a34a' : '#9ca3af',
              },
            ]}
            onPress={toggleBackgroundSync}
            activeOpacity={0.7}
          >
            <Text style={styles.toggleText}>
              {isRegistered ? 'Вкл' : 'Выкл'}
            </Text>
          </TouchableOpacity>
        </View>
      )}

      {/* Manual sync button */}
      <TouchableOpacity
        style={[
          styles.syncButton,
          { opacity: status === 'syncing' ? 0.5 : 1 },
        ]}
        onPress={handleSync}
        disabled={status === 'syncing'}
        activeOpacity={0.7}
      >
        {status === 'syncing' ? (
          <ActivityIndicator size="small" color="#fff" />
        ) : (
          <>
            <Text style={styles.syncButtonIcon}>↻</Text>
            <Text style={styles.syncButtonText}>
              {pendingCount > 0
                ? `Синхронизировать (${pendingCount})`
                : 'Синхронизировать'}
            </Text>
          </>
        )}
      </TouchableOpacity>
    </View>
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
  // Compact bar
  bar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  barLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  barRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  icon: {
    fontSize: 14,
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
  timestamp: {
    fontSize: 11,
    opacity: 0.7,
  },
  syncIcon: {
    fontSize: 16,
    fontWeight: '700',
  },

  // Expanded container
  expandedContainer: {
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderRadius: 12,
    marginHorizontal: 16,
    marginVertical: 8,
    gap: 10,
  },
  expandedRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  expandedLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  iconLarge: {
    fontSize: 20,
  },
  labelLarge: {
    fontSize: 15,
    fontWeight: '700',
  },
  subtext: {
    fontSize: 12,
    opacity: 0.7,
    marginTop: 2,
  },
  queueLabel: {
    fontSize: 13,
    fontWeight: '500',
  },
  queueValueRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  queueBadge: {
    borderRadius: 10,
    minWidth: 24,
    height: 24,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 8,
  },
  errorRow: {
    backgroundColor: 'rgba(220, 38, 38, 0.1)',
    borderRadius: 8,
    padding: 8,
  },
  errorText: {
    color: '#991b1b',
    fontSize: 12,
  },
  toggleBtn: {
    borderRadius: 6,
    paddingVertical: 4,
    paddingHorizontal: 12,
  },
  toggleText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: '700',
  },
  syncButton: {
    backgroundColor: '#1e40af',
    borderRadius: 8,
    paddingVertical: 10,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
  },
  syncButtonIcon: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '700',
  },
  syncButtonText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: '600',
  },
});
