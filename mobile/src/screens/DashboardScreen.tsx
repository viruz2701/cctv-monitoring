import React, { useCallback, useEffect, useState } from 'react';
import {
  View,
  Text,
  FlatList,
  RefreshControl,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation } from '@react-navigation/native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { workOrdersApi } from '../api/workOrders';
import { useSyncStore } from '../store/syncStore';
import {
  syncService,
  SyncState,
  SyncStatus,
} from '../services/syncService';
import { RootStackParamList, WorkOrder } from '../types';
import WorkOrderCard from '../components/WorkOrderCard';
import SwipeableCard, { SwipeAction } from '../components/SwipeableCard';
import OfflineIndicator from '../components/OfflineIndicator';
import { getGreeting } from '../utils/dateHelpers';
import { useAuthStore } from '../store/authStore';

const SYNC_STATUS_CONFIG: Record<
  SyncStatus,
  { icon: string; label: string; color: string }
> = {
  online: {
    icon: '🟢',
    label: 'Синхронизировано',
    color: '#065f46',
  },
  syncing: {
    icon: '🔄',
    label: 'Синхронизация...',
    color: '#92400e',
  },
  offline: {
    icon: '🔴',
    label: 'Офлайн',
    color: '#991b1b',
  },
};

// ── Helpers ─────────────────────────────────────────────────

function formatSyncTime(timestamp: number | null): string {
  if (!timestamp) return 'никогда';
  const diff = Date.now() - timestamp;
  const seconds = Math.floor(diff / 1000);

  if (seconds < 60) return 'только что';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}м назад`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}ч назад`;
  return `${Math.floor(seconds / 86400)}д назад`;
}

export default function DashboardScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const addToQueue = useSyncStore((s) => s.addToQueue);
  const isOnline = useSyncStore((s) => s.isOnline);
  const [offlineOrders, setOfflineOrders] = useState<WorkOrder[] | null>(null);
  const [isOfflineFallback, setIsOfflineFallback] = useState(false);

  // ── Sync state ─────────────────────────────────────────

  const [syncStatus, setSyncStatus] = useState<SyncStatus>('online');
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null);
  const [pendingCount, setPendingCount] = useState(0);
  const [isManualSyncing, setIsManualSyncing] = useState(false);

  // ── Основной запрос с fallback на SQLite ──────────────

  const {
    data: workOrders,
    isLoading,
    refetch,
    isRefetching,
    isError,
    error,
  } = useQuery({
    queryKey: ['myWorkOrders'],
    queryFn: async () => {
      try {
        // Пробуем получить с сервера
        const orders = await workOrdersApi.getMyWorkOrders();

        // Сохраняем в SQLite кэш
        await syncService.saveWorkOrderLocally(orders[0]); // upsert по одному
        for (const wo of orders) {
          await syncService.saveWorkOrderLocally(wo);
        }

        setIsOfflineFallback(false);
        setOfflineOrders(null);
        return orders;
      } catch (err) {
        // Если сеть есть, но сервер не отвечает — всё равно пытаемся
        if (!isOnline) {
          // Офлайн — загружаем из SQLite
          const local = await syncService.getLocalWorkOrders();
          if (local.length > 0) {
            setOfflineOrders(local);
            setIsOfflineFallback(true);
            return local;
          }
        }
        throw err;
      }
    },
    retry: isOnline ? 2 : 0, // Не ретраим в офлайне
    staleTime: isOnline ? 5 * 60 * 1000 : Infinity, // В офлайне не помечаем как stale
  });

  // ── Подписка на статус синхронизации ─────────────────

  useEffect(() => {
    const unsubscribe = syncService.subscribe((state: SyncState) => {
      setSyncStatus(state.status);
      setLastSyncAt(state.lastSyncAt);
      setPendingCount(state.pendingCount);

      // Автоматический refetch при восстановлении сети
      if (state.status === 'online' && isOfflineFallback) {
        refetch();
      }
    });

    return () => unsubscribe();
  }, [isOfflineFallback, refetch]);

  // ── Ручная синхронизация (pull-to-refresh) ────────────

  const handleManualSync = useCallback(async () => {
    if (isManualSyncing) return;

    setIsManualSyncing(true);
    try {
      await syncService.syncWhenOnline();
      await refetch();
    } catch (error) {
      console.error('[Dashboard] Manual sync failed:', error);
    } finally {
      setIsManualSyncing(false);
    }
  }, [isManualSyncing, refetch]);

  // ── Мутации ───────────────────────────────────────────

  const startMutation = useMutation({
    mutationFn: (id: string) => workOrdersApi.startWorkOrder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['myWorkOrders'] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: (id: string) =>
      workOrdersApi.completeWorkOrder(id, {
        notes: '',
        checklist: [],
        photos: [],
        parts_used: [],
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['myWorkOrders'] });
    },
  });

  // ── Handlers ──────────────────────────────────────────

  const openWorkOrder = useCallback(
    (id: string) => {
      navigation.navigate('WorkOrderDetail', { workOrderId: id });
    },
    [navigation],
  );

  const handleStart = useCallback(
    async (wo: WorkOrder) => {
      const online = useSyncStore.getState().isOnline;

      if (online) {
        startMutation.mutate(wo.id);
      } else {
        // Офлайн: сохраняем в SQLite + pending queue
        await syncService.enqueueMutation({
          entityType: 'work_order',
          entityId: wo.id,
          mutationType: 'update',
          payload: { status: 'in_progress' },
        });

        // Сохраняем локально обновлённый статус
        await syncService.saveWorkOrderLocally({
          ...wo,
          status: 'in_progress',
          updated_at: new Date().toISOString(),
        });

        // Оптимистично обновляем кеш React Query
        queryClient.setQueryData<WorkOrder[]>(['myWorkOrders'], (old) =>
          old?.map((o) =>
            o.id === wo.id ? { ...o, status: 'in_progress' as const } : o,
          ),
        );
      }
    },
    [startMutation, queryClient],
  );

  const handleComplete = useCallback(
    async (wo: WorkOrder) => {
      const online = useSyncStore.getState().isOnline;

      Alert.alert(
        'Завершить наряд',
        `Завершить наряд #${wo.id}?\n\nДля полного завершения откройте наряд и заполните данные.`,
        [
          { text: 'Отмена', style: 'cancel' },
          {
            text: 'Быстрое завершение',
            onPress: async () => {
              if (online) {
                completeMutation.mutate(wo.id);
              } else {
                // Офлайн: сохраняем в SQLite + pending queue
                await syncService.enqueueMutation({
                  entityType: 'work_order',
                  entityId: wo.id,
                  mutationType: 'update',
                  payload: { status: 'completed', payload: { notes: '', checklist: [], photos: [], parts_used: [] } },
                });

                await syncService.saveWorkOrderLocally({
                  ...wo,
                  status: 'completed',
                  updated_at: new Date().toISOString(),
                });

                queryClient.setQueryData<WorkOrder[]>(['myWorkOrders'], (old) =>
                  old?.map((o) =>
                    o.id === wo.id ? { ...o, status: 'completed' as const } : o,
                  ),
                );
              }
            },
          },
        ],
      );
    },
    [completeMutation, queryClient],
  );

  const handleCancel = useCallback(
    async (wo: WorkOrder) => {
      Alert.alert('Отменить наряд', `Отменить наряд #${wo.id}?`, [
        { text: 'Нет', style: 'cancel' },
        {
          text: 'Да, отменить',
          style: 'destructive',
          onPress: async () => {
            await syncService.enqueueMutation({
              entityType: 'work_order',
              entityId: wo.id,
              mutationType: 'update',
              payload: { status: 'cancelled' },
            });

            await syncService.saveWorkOrderLocally({
              ...wo,
              status: 'cancelled',
              updated_at: new Date().toISOString(),
            });

            queryClient.setQueryData<WorkOrder[]>(['myWorkOrders'], (old) =>
              old?.map((o) =>
                o.id === wo.id ? { ...o, status: 'cancelled' as const } : o,
              ),
            );
          },
        },
      ]);
    },
    [queryClient],
  );

  // ── Swipe actions ────────────────────────────────────

  const getSwipeActions = useCallback(
    (wo: WorkOrder): { right?: SwipeAction[]; left?: SwipeAction[] } => {
      switch (wo.status) {
        case 'open':
          return {
            right: [
              {
                key: 'start',
                label: 'В работу',
                color: '#2563eb',
                icon: '▶️',
                onPress: () => handleStart(wo),
              },
            ],
          };
        case 'in_progress':
          return {
            right: [
              {
                key: 'complete',
                label: 'Завершить',
                color: '#059669',
                icon: '✅',
                onPress: () => handleComplete(wo),
              },
            ],
            left: [
              {
                key: 'cancel',
                label: 'Отмена',
                color: '#dc2626',
                icon: '⏹️',
                onPress: () => handleCancel(wo),
              },
            ],
          };
        default:
          return {};
      }
    },
    [handleStart, handleComplete, handleCancel],
  );

  // ── Stats ────────────────────────────────────────────

  const getStatusCounts = () => {
    const orders = workOrders || [];
    return {
      open: orders.filter((wo) => wo.status === 'open').length,
      inProgress: orders.filter((wo) => wo.status === 'in_progress').length,
      completed: orders.filter((wo) => wo.status === 'completed').length,
    };
  };

  const counts = getStatusCounts();

  // ── Render ───────────────────────────────────────────

  const renderItem = useCallback(
    ({ item }: { item: WorkOrder }) => {
      const actions = getSwipeActions(item);
      return (
        <SwipeableCard
          rightActions={actions.right}
          leftActions={actions.left}
          disabled={item.status === 'completed' || item.status === 'cancelled'}
        >
          <WorkOrderCard workOrder={item} onPress={() => openWorkOrder(item.id)} />
        </SwipeableCard>
      );
    },
    [getSwipeActions, openWorkOrder],
  );

  return (
    <View style={styles.container}>
      <OfflineIndicator showQueueBadge alwaysVisible />

      {/* Статус-бар синхронизации */}
      <View style={styles.syncStatusBar}>
        <View style={styles.syncStatusLeft}>
          {syncStatus === 'syncing' || isManualSyncing ? (
            <ActivityIndicator size="small" color="#92400e" style={styles.syncSpinner} />
          ) : (
            <Text style={styles.syncIcon}>
              {SYNC_STATUS_CONFIG[syncStatus].icon}
            </Text>
          )}
          <Text
            style={[
              styles.syncStatusLabel,
              { color: SYNC_STATUS_CONFIG[syncStatus].color },
            ]}
          >
            {SYNC_STATUS_CONFIG[syncStatus].label}
          </Text>
        </View>

        <View style={styles.syncStatusRight}>
          {pendingCount > 0 && (
            <View style={styles.pendingBadge}>
              <Text style={styles.pendingBadgeText}>
                {pendingCount} ожидает
              </Text>
            </View>
          )}
          {lastSyncAt !== null && (
            <Text style={styles.lastSyncText}>
              {formatSyncTime(lastSyncAt)}
            </Text>
          )}
        </View>
      </View>

      <View style={styles.statsRow}>
        <View style={[styles.statCard, { backgroundColor: '#fef2f2' }]}>
          <Text style={[styles.statNumber, { color: '#dc2626' }]}>{counts.open}</Text>
          <Text style={styles.statLabel}>Открытые</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: '#fef3c7' }]}>
          <Text style={[styles.statNumber, { color: '#d97706' }]}>
            {counts.inProgress}
          </Text>
          <Text style={styles.statLabel}>В работе</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: '#d1fae5' }]}>
          <Text style={[styles.statNumber, { color: '#059669' }]}>
            {counts.completed}
          </Text>
          <Text style={styles.statLabel}>Завершено</Text>
        </View>
      </View>

      {/* Офлайн баннер */}
      {isOfflineFallback && (
        <View style={styles.offlineBanner}>
          <Text style={styles.offlineBannerText}>
            🔴 Офлайн режим — показаны кэшированные данные
          </Text>
        </View>
      )}

      <FlatList
        data={workOrders || []}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching || isManualSyncing}
            onRefresh={handleManualSync}
            tintColor="#2563eb"
            title="Синхронизация..."
            titleColor="#64748b"
          />
        }
        ListHeaderComponent={
          <Text style={styles.greeting}>
            {getGreeting()}, {user?.username || 'Техник'}
          </Text>
        }
        ListEmptyComponent={
          !isLoading ? (
            <View style={styles.empty}>
              <Text style={styles.emptyText}>
                {isOfflineFallback
                  ? 'Нет кэшированных данных'
                  : 'Нет назначенных заданий'}
              </Text>
            </View>
          ) : null
        }
      />
    </View>
  );
}

// ──────────────────────────────────────────────────
// Styles
// ──────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },
  // ── Sync status bar ──────────────────────────────
  syncStatusBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 6,
    paddingHorizontal: 16,
    backgroundColor: '#f8fafc',
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
  },
  syncStatusLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  syncStatusRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  syncIcon: {
    fontSize: 12,
  },
  syncSpinner: {
    width: 12,
    height: 12,
  },
  syncStatusLabel: {
    fontSize: 12,
    fontWeight: '600',
  },
  pendingBadge: {
    backgroundColor: '#fef3c7',
    borderRadius: 8,
    paddingHorizontal: 6,
    paddingVertical: 2,
  },
  pendingBadgeText: {
    fontSize: 11,
    fontWeight: '700',
    color: '#92400e',
  },
  lastSyncText: {
    fontSize: 11,
    color: '#94a3b8',
  },
  // ── Greeting ────────────────────────────────────
  greeting: {
    fontSize: 18,
    fontWeight: '600',
    color: '#1e293b',
    marginBottom: 12,
  },
  statsRow: {
    flexDirection: 'row',
    paddingHorizontal: 16,
    paddingBottom: 8,
    gap: 12,
  },
  statCard: {
    flex: 1,
    borderRadius: 12,
    padding: 12,
    alignItems: 'center',
  },
  statNumber: {
    fontSize: 24,
    fontWeight: 'bold',
  },
  statLabel: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 4,
  },
  list: {
    padding: 16,
    paddingTop: 8,
  },
  empty: {
    padding: 40,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 16,
    color: '#94a3b8',
  },
  offlineBanner: {
    backgroundColor: '#fef2f2',
    paddingVertical: 8,
    paddingHorizontal: 16,
    marginHorizontal: 16,
    borderRadius: 8,
    marginBottom: 8,
  },
  offlineBannerText: {
    fontSize: 13,
    color: '#991b1b',
    textAlign: 'center',
    fontWeight: '500',
  },
});
