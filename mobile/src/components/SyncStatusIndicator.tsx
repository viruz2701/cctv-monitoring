// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: SyncStatusIndicator — компонент статуса синхронизации
//
// Отображает текущий статус differential sync:
//   - Иконка статуса (idle/syncing/success/error)
//   - Прогресс-бар для active sync
//   - Количество ожидающих изменений
//   - Кнопка ручной синхронизации
//
// Подписывается на useSyncStore (Zustand).
//
// Соответствует:
//   - IEC 62443-3-3 SR 3.1 (User notification)
// ═══════════════════════════════════════════════════════════════════════════

import React, { useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ActivityIndicator,
  StyleSheet,
} from 'react-native';
import { useSyncStore, SyncStatus } from '../store/syncStore';

// ── Конфигурация статусов ──────────────────────────────────────────────

interface StatusStyle {
  icon: string;
  label: string;
  bg: string;
  textColor: string;
  accentColor: string;
}

const STATUS_STYLES: Record<SyncStatus, StatusStyle> = {
  idle: {
    icon: '🟢',
    label: 'Синхронизировано',
    bg: '#d1fae5',
    textColor: '#065f46',
    accentColor: '#059669',
  },
  syncing: {
    icon: '🔄',
    label: 'Синхронизация...',
    bg: '#fef3c7',
    textColor: '#92400e',
    accentColor: '#d97706',
  },
  success: {
    icon: '✅',
    label: 'Готово',
    bg: '#d1fae5',
    textColor: '#065f46',
    accentColor: '#059669',
  },
  error: {
    icon: '❌',
    label: 'Ошибка синхронизации',
    bg: '#fef2f2',
    textColor: '#991b1b',
    accentColor: '#dc2626',
  },
};

// ── Component ───────────────────────────────────────────────────────────

interface SyncStatusIndicatorProps {
  /** Показывать расширенную информацию */
  expanded?: boolean;
  /** Обработчик нажатия на синхронизацию (переопределяет стандартный) */
  onSyncPress?: () => void;
}

export default function SyncStatusIndicator({
  expanded = false,
  onSyncPress,
}: SyncStatusIndicatorProps) {
  const {
    dSyncStatus: status,
    dSyncProgress: progress,
    dSyncLastSyncTime: lastSyncTime,
    dSyncLastChangesCount: lastChangesCount,
    dSyncLastDurationMs: lastDurationMs,
    dSyncError: error,
    dSyncPendingCount: pendingCount,
    startSync,
  } = useSyncStore();

  const styles = STATUS_STYLES[status];

  const handleSync = useCallback(async () => {
    if (onSyncPress) {
      onSyncPress();
      return;
    }
    await startSync();
  }, [onSyncPress, startSync]);

  // ── Compact mode ──────────────────────────────────

  if (!expanded) {
    return (
      <TouchableOpacity
        style={[containerStyles.bar, { backgroundColor: styles.bg }]}
        onPress={handleSync}
        activeOpacity={0.7}
        disabled={status === 'syncing'}
      >
        <View style={containerStyles.barLeft}>
          {status === 'syncing' ? (
            <ActivityIndicator size="small" color={styles.accentColor} />
          ) : (
            <Text style={containerStyles.icon}>{styles.icon}</Text>
          )}
          <Text style={[containerStyles.label, { color: styles.textColor }]}>
            {styles.label}
          </Text>
        </View>

        <View style={containerStyles.barRight}>
          {/* Badge с количеством ожидающих */}
          {pendingCount > 0 && (
            <View style={containerStyles.badge}>
              <Text style={containerStyles.badgeText}>{pendingCount}</Text>
            </View>
          )}

          {/* Кнопка синхронизации */}
          {status !== 'syncing' && (
            <Text style={[containerStyles.syncIcon, { color: styles.textColor }]}>
              ↻
            </Text>
          )}
        </View>
      </TouchableOpacity>
    );
  }

  // ── Expanded mode ─────────────────────────────────

  const entities = Object.values(progress);

  return (
    <View style={[containerStyles.expanded, { backgroundColor: styles.bg }]}>
      {/* Статус */}
      <View style={containerStyles.statusRow}>
        <View style={containerStyles.statusLeft}>
          {status === 'syncing' ? (
            <ActivityIndicator size="small" color={styles.accentColor} />
          ) : (
            <Text style={containerStyles.iconLarge}>{styles.icon}</Text>
          )}
          <View>
            <Text
              style={[containerStyles.labelLarge, { color: styles.textColor }]}
            >
              {styles.label}
            </Text>
            {lastSyncTime && (
              <Text
                style={[
                  containerStyles.subtext,
                  { color: styles.textColor },
                ]}
              >
                Последняя: {formatTimestamp(lastSyncTime)}
              </Text>
            )}
          </View>
        </View>

        {/* Статистика */}
        <View style={containerStyles.stats}>
          {lastChangesCount > 0 && (
            <Text
              style={[containerStyles.statValue, { color: styles.textColor }]}
            >
              {lastChangesCount}
            </Text>
          )}
          {lastDurationMs > 0 && (
            <Text
              style={[containerStyles.statLabel, { color: styles.textColor }]}
            >
              {formatDuration(lastDurationMs)}
            </Text>
          )}
        </View>
      </View>

      {/* Прогресс по entity */}
      {entities.length > 0 && status === 'syncing' && (
        <View style={containerStyles.entityList}>
          {entities.map((entity) => (
            <View key={entity.entity} style={containerStyles.entityRow}>
              <Text
                style={[
                  containerStyles.entityName,
                  { color: styles.textColor },
                ]}
              >
                {getEntityLabel(entity.entity)}
              </Text>
              <Text
                style={[
                  containerStyles.entityStatus,
                  { color: styles.textColor },
                ]}
              >
                {entity.status === 'syncing'
                  ? '⬇ синхр...'
                  : entity.status === 'done'
                    ? `✅ +${entity.changesApplied}`
                    : entity.status === 'error'
                      ? '❌'
                      : '⏳'}
              </Text>
            </View>
          ))}
        </View>
      )}

      {/* Ошибка */}
      {error && (
        <View style={containerStyles.errorBox}>
          <Text style={containerStyles.errorText} numberOfLines={3}>
            ⚠ {error}
          </Text>
        </View>
      )}

      {/* Pending count */}
      {pendingCount > 0 && (
        <View style={containerStyles.pendingRow}>
          <Text style={[containerStyles.subtext, { color: styles.textColor }]}>
            Ожидает отправки: {pendingCount}
          </Text>
        </View>
      )}

      {/* Кнопка синхронизации */}
      <TouchableOpacity
        style={[
          containerStyles.syncButton,
          { backgroundColor: styles.accentColor },
          status === 'syncing' && containerStyles.syncButtonDisabled,
        ]}
        onPress={handleSync}
        disabled={status === 'syncing'}
        activeOpacity={0.7}
      >
        {status === 'syncing' ? (
          <ActivityIndicator size="small" color="#fff" />
        ) : (
          <>
            <Text style={containerStyles.syncButtonIcon}>↻</Text>
            <Text style={containerStyles.syncButtonText}>
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

// ── Helpers ─────────────────────────────────────────────────────────────

/** Форматировать ISO8601 timestamp в относительное время */
function formatTimestamp(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const seconds = Math.floor(diff / 1000);

  if (seconds < 60) return 'только что';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}м назад`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}ч назад`;
  return `${Math.floor(seconds / 86400)}д назад`;
}

/** Форматировать длительность в ms */
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

/** Получить читаемое название сущности */
function getEntityLabel(entity: string): string {
  switch (entity) {
    case 'work_orders':
      return 'Наряды';
    case 'devices':
      return 'Устройства';
    case 'photos':
      return 'Фото';
    case 'audit':
      return 'Аудит';
    default:
      return entity;
  }
}

// ── Styles ──────────────────────────────────────────────────────────────

const containerStyles = StyleSheet.create({
  // Compact
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
  syncIcon: {
    fontSize: 16,
    fontWeight: '700',
  },

  // Expanded
  expanded: {
    paddingVertical: 12,
    paddingHorizontal: 16,
    borderRadius: 12,
    marginHorizontal: 16,
    marginVertical: 8,
    gap: 10,
  },
  statusRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  statusLeft: {
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
  stats: {
    alignItems: 'flex-end',
  },
  statValue: {
    fontSize: 16,
    fontWeight: '700',
  },
  statLabel: {
    fontSize: 11,
    opacity: 0.7,
  },
  entityList: {
    gap: 4,
  },
  entityRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 2,
  },
  entityName: {
    fontSize: 13,
    fontWeight: '500',
  },
  entityStatus: {
    fontSize: 12,
  },
  errorBox: {
    backgroundColor: 'rgba(220, 38, 38, 0.1)',
    borderRadius: 8,
    padding: 8,
  },
  errorText: {
    color: '#991b1b',
    fontSize: 12,
  },
  pendingRow: {
    alignItems: 'center',
  },
  syncButton: {
    borderRadius: 8,
    paddingVertical: 10,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
  },
  syncButtonDisabled: {
    opacity: 0.5,
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
